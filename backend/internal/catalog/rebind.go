package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
)

// RebindRepresentation moves an existing representation to a newer snapshot of
// the same source, for example one that adds a reviewed OCR companion, and
// applies the new transport review for one collection in the same transaction.
// The source, media and primary file stay fixed and no companion may be
// dropped. Reviews in the representation's other collections bind the old
// snapshot and stop authorizing access until they are reviewed again.
func (d *DB) RebindRepresentation(ctx context.Context, item string, rep domain.Representation, collection string, review domain.TransportReview, approval domain.PreservationApproval, public bool, notes string) error {
	if strings.TrimSpace(notes) == "" {
		return fmt.Errorf("rebinding requires review notes")
	}
	if public && !review.Public {
		return fmt.Errorf("public rebinding requires an explicit public review")
	}
	rep.Collections = []string{collection}
	if err := rep.Validate(); err != nil {
		return err
	}
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var media, source, file, oldHash string
	var oldExtension []byte
	err = tx.QueryRow(ctx, `SELECT r.media,r.source_item_id,r.file,r.source_hash,r.extension FROM catalog_representations r
 JOIN catalog_representation_memberships rm ON rm.item_id=r.item_id AND rm.representation_id=r.id AND rm.collection_id=$3
 WHERE r.item_id=$1 AND r.id=$2 FOR UPDATE OF r`, item, rep.ID, collection).Scan(&media, &source, &file, &oldHash, &oldExtension)
	if err != nil {
		return fmt.Errorf("representation unavailable in collection: %w", err)
	}
	if media != string(rep.Media) || source != rep.SourceItemID || file != rep.File {
		return fmt.Errorf("rebinding cannot change media, source or primary file")
	}
	if oldHash == rep.Snapshot {
		return fmt.Errorf("representation already binds this snapshot")
	}
	var snapshotSource string
	if err = tx.QueryRow(ctx, `SELECT item_id FROM snapshots WHERE sha256=$1`, rep.Snapshot).Scan(&snapshotSource); err != nil || snapshotSource != rep.SourceItemID {
		return fmt.Errorf("new snapshot does not belong to the representation source")
	}
	var old domain.MediaExtension
	if err = json.Unmarshal(oldExtension, &old); err != nil {
		return err
	}
	kept := map[string]bool{}
	files := []string{rep.File}
	if rep.Extension.Print != nil {
		for _, companion := range rep.Extension.Print.Companions {
			kept[companion.File+"\x00"+companion.Role] = true
			files = append(files, companion.File)
		}
	}
	if old.Print != nil {
		for _, companion := range old.Print.Companions {
			if !kept[companion.File+"\x00"+companion.Role] {
				return fmt.Errorf("rebinding cannot drop companion %q", companion.File)
			}
		}
	}
	if (old.Audio != nil) != (rep.Extension.Audio != nil) || (old.Video != nil) != (rep.Extension.Video != nil) || (old.Print != nil) != (rep.Extension.Print != nil) {
		return fmt.Errorf("rebinding cannot change the media extension type")
	}
	for _, name := range files {
		var present bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM snapshots s, jsonb_array_elements(s.parsed->'files') f WHERE s.sha256=$1 AND f->>'name'=$2 AND COALESCE(f->>'private','false') NOT IN ('true','1'))`, rep.Snapshot, name).Scan(&present); err != nil {
			return err
		}
		if !present {
			return fmt.Errorf("file %q absent or restricted in new snapshot", name)
		}
	}
	extension, err := json.Marshal(rep.Extension)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE catalog_representations SET source_hash=$3,extension=$4 WHERE item_id=$1 AND id=$2`, item, rep.ID, rep.Snapshot, extension); err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{"rebind": rep, "previous_snapshot_sha256": oldHash, "previous_extension": json.RawMessage(oldExtension)})
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO catalog_review_history(item_id,payload,notes) VALUES($1,$2,$3)`, item, payload, notes); err != nil {
		return err
	}
	if err = reviewTransportTx(ctx, tx, item, rep.ID, collection, review, approval, public); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
