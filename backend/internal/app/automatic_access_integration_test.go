package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"git.subcult.tv/subculture-collective/arkhiv/internal/catalog"
	"git.subcult.tv/subculture-collective/arkhiv/internal/config"
	"git.subcult.tv/subculture-collective/arkhiv/internal/fixtures"
	"git.subcult.tv/subculture-collective/arkhiv/internal/index"
	"git.subcult.tv/subculture-collective/arkhiv/internal/storage"
)

func checkAutomaticAccess(t *testing.T, ctx context.Context, db *catalog.DB, store *storage.Store) {
	t.Helper()
	c := config.Collection{ID: "automatic-fixture", Item: "synthetic", Name: "Automatic test collection", Accent: "#123456", Limits: config.Limits{Documents: 20, TotalBytes: 1 << 20, FileBytes: 1 << 20, Concurrency: 1}}
	if err := db.Collection(ctx, c); err != nil {
		t.Fatal(err)
	}
	snapshot, err := db.Snapshot(ctx, c.Item, []byte(`{"synthetic":true}`))
	if err != nil {
		t.Fatal(err)
	}
	variants := map[string]string{}
	hashes := map[string]string{}
	for _, name := range []string{"allowed", "bad-reader", "corrupt-original", "revoked", "withdrawn", "foreign-source", "held"} {
		pdf := fixtures.PDF("Automatic policy fixture " + name)
		sha, size, err := storage.Digest(bytes.NewReader(pdf))
		if err != nil {
			t.Fatal(err)
		}
		part := "auto-" + name + ".part"
		if err = os.WriteFile(filepath.Join(store.Path, part), pdf, 0600); err != nil {
			t.Fatal(err)
		}
		if err = store.Promote(part, sha, size); err != nil {
			t.Fatal(err)
		}
		item := c
		if name == "foreign-source" {
			item.Item = "other-source"
		}
		selection := config.Selection{Key: name, File: name + ".pdf", Title: name, Rights: "unknown"}
		if err = db.Import(ctx, item, selection, sha, size, snapshot); err != nil {
			t.Fatal(err)
		}
		if err = index.Extract(ctx, db, store, sha); err != nil {
			t.Fatal(err)
		}
		docID := catalog.ID(c.ID, item.Item, name)
		variants[name] = catalog.ID(docID, selection.File, sha)
		hashes[name] = sha
		if name == "withdrawn" {
			no := false
			if err = db.Edit(ctx, docID, nil, "", "Explicit owner withdrawal", &no); err != nil {
				t.Fatal(err)
			}
		}
	}
	if result, err := db.ApproveAutomatic(ctx, store, c); err != nil || result.Approved != 0 {
		t.Fatal("manual collection auto-approved", result, err)
	}
	if err = db.RevokePublic(ctx, variants["revoked"], "Explicit removal request"); err != nil {
		t.Fatal(err)
	}
	if err = store.WriteJSON("derivatives/"+hashes["bad-reader"]+"/manifest.json", map[string]any{"input_sha256": "wrong"}); err != nil {
		t.Fatal(err)
	}
	original, _ := storage.Original(hashes["corrupt-original"])
	if err = os.Chmod(filepath.Join(store.Path, original), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(store.Path, original), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO source_changes(id,collection_id,source_item,filename,kind,snapshot_id,detail,decision,review_note) VALUES('automatic-held',$1,$2,'held.pdf','added',$3,'{}','hold','Explicit source hold')`, c.ID, c.Item, snapshot); err != nil {
		t.Fatal(err)
	}
	c.PublicAccess = config.PublicAccess{Policy: "verified-automatic", Item: c.Item, Review: "Synthetic owner access decision", PublicURL: "https://example.org", Contact: "info@example.org"}
	result, err := db.ApproveAutomatic(ctx, store, c)
	if err != nil || result.Approved != 1 || result.Published != 1 || len(result.Skipped) != 5 {
		t.Fatal("approval boundaries", result, err)
	}
	var count int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM variants WHERE public_approved AND document_id IN (SELECT id FROM documents WHERE collection_id=$1)`, c.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("wrong variants approved", count, err)
	}
	again, err := db.ApproveAutomatic(ctx, store, c)
	if err != nil || again.Approved != 0 {
		t.Fatal("repeat approval not idempotent", again, err)
	}
	if err = db.RevokePublic(ctx, variants["allowed"], "Remove formerly approved fixture"); err != nil {
		t.Fatal(err)
	}
	again, err = db.ApproveAutomatic(ctx, store, c)
	if err != nil || again.Approved != 0 {
		t.Fatal("automatic policy undid revocation", again, err)
	}
	var events int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM public_access_events WHERE variant_id=$1 AND approved`, variants["allowed"]).Scan(&events); err != nil || events != 1 {
		t.Fatal("duplicate or missing access event", events, err)
	}
	for _, name := range []string{"allowed", "withdrawn"} {
		pdf := fixtures.PDF("New source revision after withdrawal: " + name)
		sha, size, e := storage.Digest(bytes.NewReader(pdf))
		if e != nil {
			t.Fatal(e)
		}
		part := "auto-revision-" + name + ".part"
		if e = os.WriteFile(filepath.Join(store.Path, part), pdf, 0600); e != nil {
			t.Fatal(e)
		}
		if e = store.Promote(part, sha, size); e != nil {
			t.Fatal(e)
		}
		selected := config.Selection{Key: "revision-" + name, File: name + ".pdf", Title: "New edition " + name, Rights: "unknown"}
		if e = db.Import(ctx, c, selected, sha, size, snapshot); e != nil {
			t.Fatal(e)
		}
		if e = index.Extract(ctx, db, store, sha); e != nil {
			t.Fatal(e)
		}
	}
	again, err = db.ApproveAutomatic(ctx, store, c)
	if err != nil || again.Approved != 0 || again.Published != 0 {
		t.Fatal("new source hash bypassed prior withdrawal", again, err)
	}

}
