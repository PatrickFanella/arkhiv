package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"git.subcult.tv/subculture-collective/arkhiv/internal/config"
	"git.subcult.tv/subculture-collective/arkhiv/internal/storage"
)

type AutomaticAccess struct {
	Approved  int             `json:"approved_variants"`
	Published int             `json:"published_documents"`
	Skipped   []AutomaticSkip `json:"skipped"`
}
type AutomaticSkip struct {
	Document string `json:"document"`
	Reason   string `json:"reason"`
}

// ApproveAutomatic applies an explicit owner policy only to its exact collection
// and source item. A withdrawal is sticky until a separate manual decision.
func (d *DB) ApproveAutomatic(ctx context.Context, store *storage.Store, c config.Collection) (result AutomaticAccess, err error) {
	result.Skipped = []AutomaticSkip{}
	if !c.PublicAccess.Automatic() {
		return result, nil
	}
	if err = c.PublicAccess.Validate(c); err != nil {
		return result, err
	}
	rows, err := d.Pool.Query(ctx, `SELECT id FROM documents WHERE collection_id=$1 AND source_item=$2 AND EXISTS(SELECT 1 FROM variants WHERE document_id=documents.id AND NOT public_approved) ORDER BY id`, c.ID, c.PublicAccess.Item)
	if err != nil {
		return result, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return result, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	for _, id := range ids {
		approved, published, reason, e := d.approveAutomaticDocument(ctx, store, c, id)
		if e != nil {
			return result, e
		}
		result.Approved += approved
		if published {
			result.Published++
		}
		if reason != "" {
			result.Skipped = append(result.Skipped, AutomaticSkip{id, reason})
		}
	}
	return result, nil
}

func (d *DB) approveAutomaticDocument(ctx context.Context, store *storage.Store, c config.Collection, id string) (int, bool, string, error) {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return 0, false, "", err
	}
	defer tx.Rollback(ctx)
	var item, rights, review string
	var published bool
	if err = tx.QueryRow(ctx, `SELECT source_item,rights,review_note,published FROM documents WHERE id=$1 AND collection_id=$2 AND source_item=$3 FOR UPDATE`, id, c.ID, c.PublicAccess.Item).Scan(&item, &rights, &review, &published); err != nil {
		return 0, false, "", err
	}
	var withdrawn bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM review_events WHERE document_id=$1 AND detail->>'published'='false')`, id).Scan(&withdrawn); err != nil {
		return 0, false, "", err
	}
	if withdrawn {
		return 0, false, "explicit document withdrawal", nil
	}
	rows, err := tx.Query(ctx, `SELECT v.id,v.filename,v.asset_sha,a.size,COALESCE(x.pages,0),COALESCE(x.status,'failed'),v.public_approved,EXISTS(SELECT 1 FROM variants prior JOIN documents prior_doc ON prior_doc.id=prior.document_id WHERE prior.filename=v.filename AND prior_doc.collection_id=$2 AND prior_doc.source_item=$3 AND (EXISTS(SELECT 1 FROM public_access_events WHERE variant_id=prior.id AND NOT approved) OR EXISTS(SELECT 1 FROM review_events WHERE document_id=prior_doc.id AND detail->>'published'='false'))),EXISTS(SELECT 1 FROM source_changes WHERE collection_id=$2 AND source_item=$3 AND decision IN ('hold','reject') AND filename=v.filename) FROM variants v JOIN assets a ON a.sha256=v.asset_sha LEFT JOIN derivatives x ON x.asset_sha=v.asset_sha WHERE v.document_id=$1 ORDER BY v.id FOR UPDATE OF v`, id, c.ID, item)
	if err != nil {
		return 0, false, "", err
	}
	type checkedVariant struct {
		Variant
		pages                   int
		status                  string
		approved, revoked, held bool
	}
	variants := []checkedVariant{}
	for rows.Next() {
		var v checkedVariant
		if err = rows.Scan(&v.ID, &v.Filename, &v.SHA, &v.Size, &v.pages, &v.status, &v.approved, &v.revoked, &v.held); err != nil {
			rows.Close()
			return 0, false, "", err
		}
		variants = append(variants, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, false, "", err
	}
	if len(variants) == 0 {
		return 0, false, "no preserved PDF variants", nil
	}
	eligible := []Variant{}
	for _, v := range variants {
		if v.held {
			return 0, false, "source change explicitly held or rejected", nil
		}
		if v.pages < 1 || v.status == "failed" {
			return 0, false, "reader extraction incomplete or failed", nil
		}
		if err = store.Verify(v.SHA, v.Size); err != nil {
			return 0, false, "original integrity check failed", nil
		}
		file, readErr := store.Read(v.SHA)
		if readErr != nil {
			return 0, false, "original unavailable", nil
		}
		header := make([]byte, 5)
		_, readErr = io.ReadFull(file, header)
		file.Close()
		if readErr != nil || string(header) != "%PDF-" {
			return 0, false, "original is not a PDF", nil
		}
		var derivative struct {
			Input  string `json:"input_sha256"`
			Pages  int    `json:"pages"`
			Status string `json:"status"`
		}
		if err = store.JSON("derivatives/"+v.SHA+"/manifest.json", &derivative); err != nil || derivative.Input != v.SHA || derivative.Pages != v.pages || derivative.Status != v.status {
			return 0, false, "reader derivative manifest mismatch", nil
		}
		if !v.approved && !v.revoked {
			v.Variant.PublicApproved = true
			eligible = append(eligible, v.Variant)
		}
	}
	if len(eligible) == 0 {
		return 0, false, "remaining variants or prior source editions explicitly withdrawn", nil
	}
	if rights == "" {
		rights = "unknown"
	}
	manifest := PublicReview{Format: 1, Collection: c.ID, PublicURL: c.PublicAccess.PublicURL, Contact: c.PublicAccess.Contact, Approval: c.PublicAccess.Review, Documents: []PublicDocument{{ID: id, Source: item, Approved: true, Rights: rights, Variants: eligible}}}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return 0, false, "", err
	}
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])
	if err = store.WriteBytes("public-access/automatic-"+hash+".json", raw); err != nil {
		return 0, false, "", err
	}
	if review == "" {
		review = c.PublicAccess.Review
	}
	if _, err = tx.Exec(ctx, `UPDATE documents SET published=true,rights=$2,review_note=$3,updated_at=now() WHERE id=$1`, id, rights, review); err != nil {
		return 0, false, "", err
	}
	for _, v := range eligible {
		if _, err = tx.Exec(ctx, `UPDATE variants SET public_approved=true WHERE id=$1`, v.ID); err != nil {
			return 0, false, "", err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO public_access_events(variant_id,asset_sha,approved,review,manifest_sha256) VALUES($1,$2,true,$3,$4)`, v.ID, v.SHA, c.PublicAccess.Review, hash); err != nil {
			return 0, false, "", err
		}
	}
	event, _ := json.Marshal(map[string]any{"policy": c.PublicAccess.Policy, "review": c.PublicAccess.Review, "published": true, "manifest_sha256": hash, "variants": len(eligible)})
	if _, err = tx.Exec(ctx, `INSERT INTO review_events(document_id,action,detail) VALUES($1,'automatic-public-access',$2)`, id, event); err != nil {
		return 0, false, "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, false, "", fmt.Errorf("automatic access transaction: %w", err)
	}
	return len(eligible), !published, "", nil
}
