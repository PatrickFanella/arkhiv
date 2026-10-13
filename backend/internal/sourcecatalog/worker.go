package sourcecatalog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/source"
	"github.com/jackc/pgx/v5"
)

func (s *Store) QueuePage(ctx context.Context, expected string, page source.CatalogDiscovery) error {
	tx, err := s.Core.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var cursor string
	var complete bool
	if err = tx.QueryRow(ctx, `SELECT cursor,complete FROM source_catalog_worker_state WHERE collection_id=$1 FOR UPDATE`, s.Collection).Scan(&cursor, &complete); err != nil {
		return err
	}
	if cursor != expected || complete {
		return fmt.Errorf("catalog discovery state changed")
	}
	for _, item := range page.Items {
		if _, err = tx.Exec(ctx, `INSERT INTO source_catalog_queue(collection_id,item_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, s.Collection, item.Identifier); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE source_catalog_worker_state SET cursor=$2,complete=$3,discovered=(SELECT count(*) FROM source_catalog_queue WHERE collection_id=$1),updated_at=now() WHERE collection_id=$1`, s.Collection, page.Cursor, page.Cursor == ""); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Store) Fail(ctx context.Context, id, state string, status int, message string, retry time.Time) error {
	tx, err := s.Core.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE source_catalog_queue SET state=$3,attempts=attempts+1,error=$4,next_attempt=$5,updated_at=now() WHERE collection_id=$1 AND item_id=$2`, s.Collection, id, state, message, retry); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO observations(item_id,status,error) VALUES($1,$2,$3)`, id, status, message); err != nil {
		return err
	}
	if state == "unavailable" || state == "rejected" {
		if _, err = tx.Exec(ctx, `UPDATE source_catalog_items SET unavailable=true,updated_at=now() WHERE collection_id=$1 AND identifier=$2`, s.Collection, id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func (s *Store) Reconcile(ctx context.Context) error {
	tx, err := s.Core.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE source_catalog_worker_state SET cursor='',complete=false,updated_at=now() WHERE collection_id=$1`, s.Collection); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE source_catalog_queue SET state='pending',next_attempt=now(),error='' WHERE collection_id=$1 AND state IN ('imported','unavailable','rejected')`, s.Collection); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Store) Run(ctx context.Context, client *source.Client, pages, batch int, delay time.Duration) (runErr error) {
	if pages < 0 || pages > 100 || batch < 0 || batch > 1000 || delay < 0 || delay > time.Minute {
		return fmt.Errorf("invalid metadata worker bounds")
	}
	if err := s.Init(ctx); err != nil {
		return err
	}
	conn, err := s.Core.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	var locked bool
	if err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended(current_schema() || ':source-catalog:' || $1,7920))`, s.Collection).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		return fmt.Errorf("catalog worker is already running")
	}
	defer conn.Exec(context.Background(), `SELECT pg_advisory_unlock(hashtextextended(current_schema() || ':source-catalog:' || $1,7920))`, s.Collection)
	var retryAt *time.Time
	if err = conn.QueryRow(ctx, `SELECT retry_at FROM source_catalog_worker_state WHERE collection_id=$1`, s.Collection).Scan(&retryAt); err != nil {
		return err
	}
	if retryAt != nil && retryAt.After(client.Clock()) {
		return &source.Backoff{Until: *retryAt, Status: 429}
	}
	defer func() {
		if ctx.Err() != nil {
			return
		}
		save, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if runErr == nil {
			_, err = s.Core.Pool.Exec(save, `UPDATE source_catalog_worker_state SET retry_at=NULL,last_error='' WHERE collection_id=$1`, s.Collection)
		} else {
			until := client.Clock().Add(time.Minute)
			var backoff *source.Backoff
			if errors.As(runErr, &backoff) && backoff.Until.After(until) {
				until = backoff.Until
			}
			_, err = s.Core.Pool.Exec(save, `UPDATE source_catalog_worker_state SET retry_at=$2,last_error=$3 WHERE collection_id=$1`, s.Collection, until, runErr.Error())
		}
		if err != nil && runErr == nil {
			runErr = err
		}
	}()
	var cursor string
	var complete bool
	if err = conn.QueryRow(ctx, `SELECT cursor,complete FROM source_catalog_worker_state WHERE collection_id=$1`, s.Collection).Scan(&cursor, &complete); err != nil {
		return err
	}
	for n := 0; n < pages && !complete; n++ {
		page, response, err := client.DiscoverCatalog(ctx, s.Collection, cursor)
		if err != nil {
			return err
		}
		if err = s.Core.Snapshot(ctx, s.Collection, response.Raw, response.Status, response.ETag, response.Modified, nil); err != nil {
			return err
		}
		if err = s.QueuePage(ctx, cursor, page); err != nil {
			return err
		}
		cursor = page.Cursor
		complete = cursor == ""
		if err = wait(ctx, delay); err != nil {
			return err
		}
	}
	rows, err := s.Core.Pool.Query(ctx, `SELECT item_id FROM source_catalog_queue WHERE collection_id=$1 AND state IN ('pending','retry') AND next_attempt<=now() ORDER BY next_attempt,item_id LIMIT $2`, s.Collection, batch)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	for _, id := range ids {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		etag, modified, err := s.Core.Validators(ctx, id)
		if err != nil {
			return err
		}
		response, fetchErr := client.Metadata(ctx, id, etag, modified)
		state := "retry"
		until := client.Clock().Add(time.Minute)
		var backoff *source.Backoff
		if errors.As(fetchErr, &backoff) {
			until = backoff.Until
		}
		if response.Status == 404 || response.Status == 403 || response.Status == 410 {
			state = "unavailable"
			until = client.Clock().Add(24 * time.Hour)
		}
		if fetchErr == nil {
			if response.Status == 304 {
				response.Raw, _, _, fetchErr = s.Core.RetainedResponse(ctx, id)
			}
			if fetchErr == nil {
				if strings.TrimSpace(string(response.Raw)) == "{}" {
					fetchErr = fmt.Errorf("source metadata is unavailable")
					state = "unavailable"
				} else if _, _, err = Normalize(s.Collection, id, response.Raw); err != nil {
					// Preserve rejected source bytes as well as the rejection observation.
					if saveErr := s.Core.Snapshot(ctx, id, response.Raw, response.Status, response.ETag, response.Modified, nil); saveErr != nil {
						return saveErr
					}
					fetchErr = err
					state = "rejected"
				} else {
					fetchErr = s.Import(ctx, id, response.Raw, response.Status, response.ETag, response.Modified)
				}
			}
		}
		if fetchErr != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err = s.Fail(ctx, id, state, response.Status, fetchErr.Error(), until); err != nil {
				return err
			}
			if backoff != nil {
				return fetchErr
			}
		}
		if err = wait(ctx, delay); err != nil {
			return err
		}
	}
	return nil
}
func wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
