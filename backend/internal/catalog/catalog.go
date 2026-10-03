package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct{ Pool *pgxpool.Pool }

func Open(ctx context.Context, url string) (*DB, error) {
	if url == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	p, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err = p.Ping(ctx); err != nil {
		p.Close()
		return nil, err
	}
	return &DB{p}, nil
}
func (d *DB) Migrate(ctx context.Context, dir string) error {
	c, err := d.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer c.Release()
	if _, err = c.Exec(ctx, `SELECT pg_advisory_lock(79202173)`); err != nil {
		return err
	}
	defer c.Exec(context.Background(), `SELECT pg_advisory_unlock(79202173)`)
	if _, err = c.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_versions (name text PRIMARY KEY,sha256 text NOT NULL)`); err != nil {
		return err
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("no migrations")
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		name := filepath.Base(f)
		hash := domain.Digest(b)
		var old string
		err = c.QueryRow(ctx, `SELECT sha256 FROM schema_versions WHERE name=$1`, name).Scan(&old)
		if err == nil {
			if old != hash {
				return fmt.Errorf("applied migration checksum changed: %s", name)
			}
			continue
		}
		if err != pgx.ErrNoRows {
			return err
		}
		tx, err := c.Begin(ctx)
		if err != nil {
			return err
		}
		sql := string(b)
		if _, err = tx.Exec(ctx, sql); err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO schema_versions VALUES($1,$2)`, name, hash)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}
func (d *DB) Collection(ctx context.Context, c domain.Collection) error {
	if err := c.Validate(); err != nil {
		return err
	}
	// Existing policy is immutable; changing storage ownership requires a new reviewed migration.
	var existing string
	err := d.Pool.QueryRow(ctx, `SELECT policy FROM collections WHERE id=$1`, c.ID).Scan(&existing)
	if err == nil && existing != string(c.Policy) {
		return fmt.Errorf("existing collection policy cannot be changed implicitly")
	}
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	_, err = d.Pool.Exec(ctx, `INSERT INTO collections(id,policy,budget,reserve) VALUES($1,$2,$3,$4) ON CONFLICT(id) DO NOTHING`, c.ID, c.Policy, c.Budget, c.Reserve)
	if err != nil {
		return err
	}
	_, err = d.Pool.Exec(ctx, `INSERT INTO worker_state(collection_id) VALUES($1) ON CONFLICT DO NOTHING`, c.ID)
	return err
}
func (d *DB) GetCollection(ctx context.Context, id string) (domain.Collection, error) {
	var c domain.Collection
	err := d.Pool.QueryRow(ctx, `SELECT id,policy,budget,reserve FROM collections WHERE id=$1`, id).Scan(&c.ID, &c.Policy, &c.Budget, &c.Reserve)
	return c, err
}
func (d *DB) Snapshot(ctx context.Context, item string, raw []byte, status int, etag, modified string, e *domain.Episode) error {
	if !domain.ValidID(item) || !json.Valid(raw) {
		return fmt.Errorf("invalid source snapshot")
	}
	if e != nil {
		if e.ID != item {
			return fmt.Errorf("snapshot episode identity mismatch")
		}
		if err := e.Validate(); err != nil {
			return err
		}
	}
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	hash := domain.Digest(raw)
	if _, err = tx.Exec(ctx, `INSERT INTO snapshots(sha256,item_id,raw,parsed) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, hash, item, raw, json.RawMessage(raw)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO observations(item_id,snapshot_hash,status,etag,modified) VALUES($1,$2,$3,$4,$5)`, item, hash, status, etag, modified); err != nil {
		return err
	}
	if e != nil {
		b, err := json.Marshal(e)
		if err != nil {
			return err
		}
		var prior string
		err = tx.QueryRow(ctx, `SELECT collection_id FROM episodes WHERE id=$1`, item).Scan(&prior)
		if err == nil && prior != e.CollectionID {
			return fmt.Errorf("episode collection identity changed")
		}
		if err != nil && err != pgx.ErrNoRows {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO episodes(id,collection_id,source_hash,source) VALUES($1,$2,$3,$4) ON CONFLICT(id) DO UPDATE SET source_hash=excluded.source_hash,source=excluded.source,unavailable=false,updated_at=now()`, item, e.CollectionID, hash, b)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func (d *DB) Entries(ctx context.Context) (map[string]domain.Entry, error) {
	rows, err := d.Pool.Query(ctx, `SELECT e.source,e.decision,c.policy,e.source_hash,e.unavailable,e.local_path,e.local_sha256 FROM episodes e JOIN collections c ON c.id=e.collection_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]domain.Entry{}
	for rows.Next() {
		var e domain.Entry
		var s, b []byte
		if err = rows.Scan(&s, &b, &e.Policy, &e.SourceHash, &e.Unavailable, &e.LocalPath, &e.LocalSHA256); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(s, &e.Episode); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &e.Decision); err != nil {
			return nil, err
		}
		if e.Decision.Title != "" {
			e.Episode.Title = e.Decision.Title
		}
		out[e.Episode.ID] = e
	}
	return out, rows.Err()
}
func (d *DB) Decide(ctx context.Context, id string, decision domain.Decision) error {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var source []byte
	var policy domain.Policy
	err = tx.QueryRow(ctx, `SELECT e.source,c.policy FROM episodes e JOIN collections c ON c.id=e.collection_id WHERE e.id=$1 FOR UPDATE OF e`, id).Scan(&source, &policy)
	if err != nil {
		return err
	}
	var e domain.Entry
	e.Policy = policy
	if err = json.Unmarshal(source, &e.Episode); err != nil {
		return err
	}
	if err = e.ValidateDecision(decision); err != nil {
		return err
	}
	b, err := json.Marshal(decision)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE episodes SET decision=$2,updated_at=now() WHERE id=$1`, id, b); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO decision_history(episode_id,decision) VALUES($1,$2)`, id, b); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (d *DB) SaveSchedule(ctx context.Context, s domain.Schedule) error {
	if err := s.Validate(); err != nil {
		return err
	}
	entries, err := d.Entries(ctx)
	if err != nil {
		return err
	}
	for _, p := range s.Programs {
		e, ok := entries[p.EpisodeID]
		f, eligible := e.Selected()
		if !ok || !eligible || !e.Watchable() || !e.Decision.Channel || f.Name != p.File || p.Policy != e.Policy || p.Duration > f.Duration+0.1 {
			return fmt.Errorf("schedule references ineligible asset")
		}
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO schedules(revision,payload) VALUES($1,$2)`, s.Revision, b); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO current_schedule(id,revision) VALUES(1,$1) ON CONFLICT(id) DO UPDATE SET revision=excluded.revision`, s.Revision); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (d *DB) Schedule(ctx context.Context) (domain.Schedule, error) {
	var s domain.Schedule
	var b []byte
	err := d.Pool.QueryRow(ctx, `SELECT payload FROM schedules JOIN current_schedule USING(revision) WHERE id=1`).Scan(&b)
	if err == nil {
		err = json.Unmarshal(b, &s)
	}
	return s, err
}

type WorkerState struct {
	CollectionID string     `json:"collection_id"`
	Page         int        `json:"page"`
	Complete     bool       `json:"discovery_complete"`
	Count        int64      `json:"source_count"`
	RetryAt      *time.Time `json:"retry_at"`
	LastSuccess  *time.Time `json:"last_success"`
	LastError    string     `json:"last_error"`
}

func (d *DB) State(ctx context.Context, id string) (WorkerState, error) {
	var s WorkerState
	err := d.Pool.QueryRow(ctx, `SELECT collection_id,page,discovery_complete,source_count,retry_at,last_success,last_error FROM worker_state WHERE collection_id=$1`, id).Scan(&s.CollectionID, &s.Page, &s.Complete, &s.Count, &s.RetryAt, &s.LastSuccess, &s.LastError)
	return s, err
}
func (d *DB) QueuePage(ctx context.Context, id string, page int, count int64, ids []string, complete bool) error {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, item := range ids {
		if !domain.ValidID(item) {
			return fmt.Errorf("invalid discovered item")
		}
		if _, err = tx.Exec(ctx, `INSERT INTO metadata_queue(collection_id,item_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, id, item); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE worker_state SET page=$2,source_count=$3,discovery_complete=$4,last_error='',updated_at=now() WHERE collection_id=$1`, id, page, count, complete)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (d *DB) Pending(ctx context.Context, id string, n int) ([]string, error) {
	rows, err := d.Pool.Query(ctx, `SELECT item_id FROM metadata_queue WHERE collection_id=$1 AND state IN ('pending','retry') AND next_attempt<=now() ORDER BY discovered_at,item_id LIMIT $2`, id, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var item string
		if err = rows.Scan(&item); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
func (d *DB) FinishItem(ctx context.Context, collection, item, state, msg string, retry time.Time, status int) error {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `UPDATE metadata_queue SET state=$3,error=$4,attempts=attempts+1,next_attempt=$5 WHERE collection_id=$1 AND item_id=$2`, collection, item, state, msg, retry)
	if err != nil {
		return err
	}
	if state == "unavailable" {
		if _, err = tx.Exec(ctx, `UPDATE episodes SET unavailable=true WHERE id=$1`, item); err != nil {
			return err
		}
	}
	if state == "imported" {
		if _, err = tx.Exec(ctx, `UPDATE episodes SET unavailable=false WHERE id=$1`, item); err != nil {
			return err
		}
	}
	if state != "imported" {
		if _, err = tx.Exec(ctx, `INSERT INTO observations(item_id,status,error) VALUES($1,$2,$3)`, item, status, msg); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func (d *DB) Backoff(ctx context.Context, id string, until time.Time, msg string) error {
	_, err := d.Pool.Exec(ctx, `UPDATE worker_state SET retry_at=$2,last_error=$3,updated_at=now() WHERE collection_id=$1`, id, until, msg)
	return err
}
func (d *DB) Success(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `UPDATE worker_state SET retry_at=NULL,last_success=now(),last_error='',updated_at=now() WHERE collection_id=$1`, id)
	return err
}
func (d *DB) Reconcile(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `UPDATE worker_state SET page=1,discovery_complete=false WHERE collection_id=$1`, id)
	return err
}
func (d *DB) RefreshKnown(ctx context.Context, id string, limit int) error {
	_, err := d.Pool.Exec(ctx, `UPDATE metadata_queue SET state='pending',next_attempt=now() WHERE (collection_id,item_id) IN (SELECT collection_id,item_id FROM metadata_queue WHERE collection_id=$1 AND state IN ('imported','unavailable','rejected') ORDER BY next_attempt,item_id LIMIT $2)`, id, limit)
	return err
}
func (d *DB) Validators(ctx context.Context, id string) (string, string, error) {
	var e, m string
	err := d.Pool.QueryRow(ctx, `SELECT etag,modified FROM observations WHERE item_id=$1 AND snapshot_hash IS NOT NULL ORDER BY id DESC LIMIT 1`, id).Scan(&e, &m)
	if err == pgx.ErrNoRows {
		return "", "", nil
	}
	return e, m, err
}
func (d *DB) Counts(ctx context.Context, id string) (map[string]int64, error) {
	rows, err := d.Pool.Query(ctx, `SELECT state,count(*) FROM metadata_queue WHERE collection_id=$1 GROUP BY state`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var s string
		var n int64
		if err = rows.Scan(&s, &n); err != nil {
			return nil, err
		}
		out[s] = n
	}
	return out, rows.Err()
}
