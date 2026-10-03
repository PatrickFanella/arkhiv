// Package updates persists complete source inventories without altering publication.
package updates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"git.subcult.tv/subculture-collective/arkhiv/internal/catalog"
	"git.subcult.tv/subculture-collective/arkhiv/internal/config"
	ia "git.subcult.tv/subculture-collective/arkhiv/internal/source/internetarchive"
	"git.subcult.tv/subculture-collective/arkhiv/internal/storage"
	"github.com/jackc/pgx/v5"
)

type Inventory struct {
	Metadata map[string]any     `json:"metadata"`
	Files    map[string]ia.File `json:"files"`
}
type Change struct {
	File   string   `json:"file"`
	Kind   string   `json:"kind"`
	Before *ia.File `json:"before,omitempty"`
	After  *ia.File `json:"after,omitempty"`
}
type Result struct {
	Run       string     `json:"run"`
	Checked   int        `json:"checked"`
	Added     int        `json:"added"`
	Content   int        `json:"content"`
	Metadata  int        `json:"metadata"`
	Removed   int        `json:"removed"`
	Uncertain int        `json:"uncertain"`
	Pending   int        `json:"pending"`
	Status    string     `json:"status"`
	RetryAt   *time.Time `json:"retry_at,omitempty"`
}

func Complete(m ia.Metadata) (Inventory, error) {
	inv := Inventory{Metadata: m.Metadata, Files: map[string]ia.File{}}
	if m.Metadata == nil || m.Error != "" || len(m.Files) == 0 {
		return inv, fmt.Errorf("incomplete source inventory")
	}
	for _, f := range m.Files {
		if f.Name == "" || strings.HasPrefix(f.Name, "/") || strings.Contains(f.Name, "\\") || strings.Contains("/"+f.Name+"/", "/../") {
			return inv, fmt.Errorf("unsafe source filename")
		}
		if _, exists := inv.Files[f.Name]; exists {
			return inv, fmt.Errorf("duplicate source filename")
		}
		inv.Files[f.Name] = f
	}
	return inv, nil
}
func equal(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}
func Diff(before, after Inventory) []Change {
	out := []Change{}
	names := map[string]bool{}
	for n := range before.Files {
		names[n] = true
	}
	for n := range after.Files {
		names[n] = true
	}
	ordered := []string{}
	for n := range names {
		ordered = append(ordered, n)
	}
	sort.Strings(ordered)
	for _, n := range ordered {
		a, old := before.Files[n]
		b, newer := after.Files[n]
		kind := ""
		switch {
		case !old:
			kind = "added"
		case !newer:
			kind = "removed"
		case a.Size != b.Size || a.MD5 != b.MD5 || a.SHA1 != b.SHA1:
			kind = "content"
		case !equal(a, b):
			kind = "metadata"
		case b.MD5 == "" && b.SHA1 == "":
			kind = "uncertain"
		}
		if kind != "" {
			change := Change{File: n, Kind: kind}
			if old {
				change.Before = &a
			}
			if newer {
				change.After = &b
			}
			out = append(out, change)
		}
	}
	if !equal(before.Metadata, after.Metadata) {
		out = append(out, Change{File: "", Kind: "metadata"})
	}
	return out
}

// Refresh advances the baseline only after a complete validated item response and
// one transaction that also creates review events. A failed response never infers removals.
func Refresh(ctx context.Context, c config.Collection, client *ia.Client, store *storage.Store, db *catalog.DB) (result Result, err error) {
	result.Run = catalog.ID(c.ID, time.Now().UTC().String())
	result.Status = "running"
	if err = db.Collection(ctx, c); err != nil {
		return
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO source_refresh_runs(id,collection_id,status) VALUES($1,$2,'running')`, result.Run, c.ID); err != nil {
		return
	}
	defer func() {
		if err != nil {
			result.Status = "failed"
			var backoff *ia.RetryError
			if errors.As(err, &backoff) {
				result.RetryAt = &backoff.At
			}
			b, _ := json.Marshal(result)
			cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			db.Pool.Exec(cleanup, `UPDATE source_refresh_runs SET status='failed',finished_at=now(),result=$2 WHERE id=$1`, result.Run, b)
		}
	}()
	m, raw, e := client.Metadata(ctx, c.Item)
	if e != nil {
		err = e
		return
	}
	current, e := Complete(m)
	if e != nil {
		err = e
		return
	}
	snap := catalog.ID(c.Item, string(raw))
	if err = store.WriteBytes("snapshots/"+snap+".json", raw); err != nil {
		return
	}
	tx, e := db.Pool.Begin(ctx)
	if e != nil {
		err = e
		return
	}
	defer tx.Rollback(ctx)
	// Also serialize callers that do not use the CLI filesystem lock.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,74209114))`, c.ID); err != nil {
		return
	}
	previous := Inventory{Files: map[string]ia.File{}}
	var old []byte
	e = tx.QueryRow(ctx, `SELECT inventory FROM source_inventories WHERE collection_id=$1 AND source_item=$2`, c.ID, c.Item).Scan(&old)
	if e != nil && e != pgx.ErrNoRows {
		err = e
		return
	}
	if e == nil {
		if err = json.Unmarshal(old, &previous); err != nil {
			return
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO source_snapshots(id,item_id,raw) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, snap, c.Item, raw); err != nil {
		return
	}
	for _, change := range Diff(previous, current) {
		result.Checked++
		switch change.Kind {
		case "added":
			result.Added++
		case "content":
			result.Content++
		case "metadata":
			result.Metadata++
		case "removed":
			result.Removed++
		case "uncertain":
			result.Uncertain++
		}
		detail, _ := json.Marshal(change)
		id := catalog.ID(c.ID, c.Item, string(detail))
		if _, err = tx.Exec(ctx, `INSERT INTO source_changes(id,collection_id,source_item,filename,kind,snapshot_id,detail) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`, id, c.ID, c.Item, change.File, change.Kind, snap, detail); err != nil {
			return
		}
	}
	result.Checked = len(current.Files)
	encoded, _ := json.Marshal(current)
	if _, err = tx.Exec(ctx, `INSERT INTO source_inventories(collection_id,source_item,snapshot_id,inventory) VALUES($1,$2,$3,$4) ON CONFLICT(collection_id,source_item) DO UPDATE SET snapshot_id=excluded.snapshot_id,inventory=excluded.inventory,checked_at=now()`, c.ID, c.Item, snap, encoded); err != nil {
		return
	}
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM source_changes WHERE collection_id=$1 AND decision='pending'`, c.ID).Scan(&result.Pending); err != nil {
		return
	}
	result.Status = "complete"
	receipt, _ := json.Marshal(result)
	if _, err = tx.Exec(ctx, `UPDATE source_refresh_runs SET status='complete',finished_at=now(),result=$2 WHERE id=$1`, result.Run, receipt); err != nil {
		return
	}
	err = tx.Commit(ctx)
	return
}

func Queue(ctx context.Context, db *catalog.DB, collection string) ([]json.RawMessage, error) {
	return QueuePage(ctx, db, collection, 0)
}
func QueuePage(ctx context.Context, db *catalog.DB, collection string, offset int) ([]json.RawMessage, error) {
	if offset < 0 {
		return nil, fmt.Errorf("invalid queue offset")
	}
	rows, err := db.Pool.Query(ctx, `SELECT jsonb_build_object('id',id,'item',source_item,'file',filename,'kind',kind,'decision',decision,'detail',detail,'review',review_note,'document',document_id) FROM source_changes WHERE collection_id=$1 ORDER BY created_at,id LIMIT 500 OFFSET $2`, collection, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []json.RawMessage{}
	for rows.Next() {
		var b []byte
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func Review(ctx context.Context, db *catalog.DB, id, decision, note string) error {
	if (decision != "approve" && decision != "reject" && decision != "hold") || strings.TrimSpace(note) == "" {
		return fmt.Errorf("approve, reject or hold and a review note required")
	}
	command, err := db.Pool.Exec(ctx, `UPDATE source_changes SET decision=$2,review_note=$3,reviewed_at=now() WHERE id=$1 AND decision<>'acquired'`, id, decision, note)
	if err == nil && command.RowsAffected() != 1 {
		return os.ErrNotExist
	}
	return err
}
