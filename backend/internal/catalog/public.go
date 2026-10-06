package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
)

// ReviewPublicVideo records publication independently of private playback and
// acquisition decisions. Source refresh or a different selected file invalidates
// the approval without deleting its history.
func (d *DB) ReviewPublicVideo(ctx context.Context, id string, approved bool, notes string) error {
	if strings.TrimSpace(notes) == "" {
		return fmt.Errorf("publication requires authorization notes")
	}
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var e domain.Entry
	var source, decision []byte
	err = tx.QueryRow(ctx, `SELECT e.source,e.decision,e.source_hash,c.policy,e.unavailable,e.local_path,e.local_sha256,e.local_file,e.local_snapshot FROM episodes e JOIN collections c ON c.id=e.collection_id WHERE e.id=$1 FOR UPDATE OF e`, id).Scan(&source, &decision, &e.SourceHash, &e.Policy, &e.Unavailable, &e.LocalPath, &e.LocalSHA256, &e.LocalFile, &e.LocalSnapshot)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(source, &e.Episode); err != nil {
		return err
	}
	if err = json.Unmarshal(decision, &e.Decision); err != nil {
		return err
	}
	if approved {
		if e.Decision.ProcessedVideo != nil {
			return fmt.Errorf("processed-video pilot has no public activation approval")
		}
		if _, err = domain.Resolve(e, false); err != nil {
			return fmt.Errorf("publication needs reviewed playable video: %w", err)
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO episode_public_reviews(episode_id,source_hash,file,approved,notes) VALUES($1,$2,$3,$4,$5) ON CONFLICT(episode_id) DO UPDATE SET source_hash=excluded.source_hash,file=excluded.file,approved=excluded.approved,notes=excluded.notes,updated_at=now()`, id, e.SourceHash, e.Decision.File, approved, notes); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO episode_public_history(episode_id,source_hash,file,approved,notes) VALUES($1,$2,$3,$4,$5)`, id, e.SourceHash, e.Decision.File, approved, notes); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
