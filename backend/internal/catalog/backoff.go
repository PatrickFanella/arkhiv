package catalog

import (
	"context"
	"encoding/json"
	"time"
)

// SourceBackoff records a failed source operation without advancing inventories.
func (d *DB) SourceBackoff(ctx context.Context, collection, phase string, at time.Time) error {
	raw, _ := json.Marshal(map[string]any{"status": "failed", "phase": phase, "retry_at": at})
	_, err := d.Pool.Exec(ctx, `INSERT INTO source_refresh_runs(id,collection_id,status,finished_at,result) VALUES($1,$2,'failed',now(),$3)`, ID(collection, phase, time.Now().UTC().String()), collection, raw)
	return err
}
