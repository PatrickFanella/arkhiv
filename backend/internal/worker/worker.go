package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/source"
)

type Worker struct {
	DB     *catalog.DB
	Source *source.Client
}
type Result struct {
	Pages    int    `json:"pages"`
	Imported int    `json:"imported"`
	Failed   int    `json:"failed"`
	Backoff  string `json:"backoff,omitempty"`
}

func (w *Worker) Run(ctx context.Context, id string, pages, items int, deadline time.Duration) (Result, error) {
	var result Result
	if pages < 0 || pages > 100 || items < 0 || items > 1000 || deadline <= 0 || deadline > 30*time.Minute {
		return result, fmt.Errorf("invalid worker bounds")
	}
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	conn, err := w.DB.Pool.Acquire(ctx)
	if err != nil {
		return result, err
	}
	defer conn.Release()
	var locked bool
	if err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended($1,7920))`, id).Scan(&locked); err != nil {
		return result, err
	}
	if !locked {
		return result, fmt.Errorf("collection worker is already running")
	}
	defer conn.Exec(context.Background(), `SELECT pg_advisory_unlock(hashtextextended($1,7920))`, id)
	state, err := w.DB.State(ctx, id)
	if err != nil {
		return result, err
	}
	if state.RetryAt != nil && state.RetryAt.After(w.Source.Clock()) {
		result.Backoff = state.RetryAt.Format(time.RFC3339)
		return result, nil
	}
	handleBackoff := func(err error) bool {
		var b *source.Backoff
		if errors.As(err, &b) {
			result.Backoff = b.Until.Format(time.RFC3339)
			err = w.DB.Backoff(ctx, id, b.Until, b.Error())
			return err == nil
		}
		return false
	}
	if pages > 0 && state.Page == 1 {
		response, fetchErr := w.Source.Metadata(ctx, id, "", "")
		if fetchErr != nil {
			if handleBackoff(fetchErr) {
				return result, nil
			}
			return result, fetchErr
		}
		if err = w.DB.Snapshot(ctx, id, response.Raw, response.Status, response.ETag, response.Modified, nil); err != nil {
			return result, err
		}
	}
	for n := 0; n < pages && !state.Complete; n++ {
		ids, count, discoveryResponse, fetchErr := w.Source.Discover(ctx, id, state.Page, 100)
		if fetchErr != nil {
			if handleBackoff(fetchErr) {
				return result, nil
			}
			return result, fetchErr
		}
		if err = w.DB.Snapshot(ctx, id, discoveryResponse.Raw, discoveryResponse.Status, discoveryResponse.ETag, discoveryResponse.Modified, nil); err != nil {
			return result, err
		}
		complete := len(ids) < 100 || int64(state.Page*100) >= count
		if err = w.DB.QueuePage(ctx, id, state.Page+1, count, ids, complete); err != nil {
			return result, err
		}
		state.Page++
		state.Complete = complete
		result.Pages++
	}
	pending, err := w.DB.Pending(ctx, id, items)
	if err != nil {
		return result, err
	}
	for _, item := range pending {
		etag, modified, err := w.DB.Validators(ctx, item)
		if err != nil {
			return result, err
		}
		response, fetchErr := w.Source.Metadata(ctx, item, etag, modified)
		if fetchErr != nil {
			var b *source.Backoff
			if errors.As(fetchErr, &b) {
				if err = w.DB.FinishItem(ctx, id, item, "retry", fetchErr.Error(), b.Until, response.Status); err != nil {
					return result, err
				}
				if handleBackoff(fetchErr) {
					return result, nil
				}
				return result, fetchErr
			}
			status := "retry"
			retry := w.Source.Clock().Add(5 * time.Minute)
			if response.Status == 404 || response.Status == 403 {
				status = "unavailable"
				retry = w.Source.Clock().Add(24 * time.Hour)
			}
			if err = w.DB.FinishItem(ctx, id, item, status, fetchErr.Error(), retry, response.Status); err != nil {
				return result, err
			}
			result.Failed++
			continue
		}
		if response.Status == 304 {
			var priorETag, priorModified string
			response.Raw, priorETag, priorModified, err = w.DB.RetainedResponse(ctx, item)
			if err != nil {
				return result, err
			}
			if response.ETag == "" {
				response.ETag = priorETag
			}
			if response.Modified == "" {
				response.Modified = priorModified
			}
		}
		{
			e, err := source.Normalize(item, id, response.Raw)
			if err != nil {
				if snapshotErr := w.DB.Snapshot(ctx, item, response.Raw, response.Status, response.ETag, response.Modified, nil); snapshotErr != nil {
					return result, snapshotErr
				}
				if err = w.DB.FinishItem(ctx, id, item, "rejected", err.Error(), w.Source.Clock().Add(24*time.Hour), response.Status); err != nil {
					return result, err
				}
				result.Failed++
				continue
			}
			if err = w.DB.Snapshot(ctx, item, response.Raw, response.Status, response.ETag, response.Modified, &e); err != nil {
				return result, err
			}
		}
		if err = w.DB.FinishItem(ctx, id, item, "imported", "", w.Source.Clock().Add(24*time.Hour), response.Status); err != nil {
			return result, err
		}
		result.Imported++
	}
	return result, w.DB.Success(ctx, id)
}
