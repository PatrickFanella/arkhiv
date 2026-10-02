package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"git.subcult.tv/subculture-collective/arkhiv/internal/catalog"
	"git.subcult.tv/subculture-collective/arkhiv/internal/config"
	"git.subcult.tv/subculture-collective/arkhiv/internal/fixtures"
	"git.subcult.tv/subculture-collective/arkhiv/internal/index"
	"git.subcult.tv/subculture-collective/arkhiv/internal/recovery"
	"git.subcult.tv/subculture-collective/arkhiv/internal/storage"
	"github.com/jackc/pgx/v5"
	"net/http/httptest"
	urlpkg "net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestArchiveIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("disposable PostgreSQL companion gate requires TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cfg, e := pgx.ParseConfig(url)
	if e != nil {
		t.Fatal("invalid test database URL")
	}
	if cfg.Host != "db" || cfg.User != "dev" || cfg.Password != "disposable-dev-only" {
		t.Fatal("test database must be the isolated launcher service")
	}
	admin, e := pgx.ConnectConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close(context.Background())
	create := func(label string) string {
		name := fmt.Sprintf("arkhiv_test_%s_%d", label, time.Now().UnixNano())
		if _, e := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() {
			admin.Exec(context.Background(), "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
		})
		u, _ := urlpkg.Parse(url)
		u.Path = "/" + name
		return u.String()
	}
	sourceURL := create("source")
	db, e := catalog.Open(ctx, sourceURL)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Pool.Close()
	migrations := "../../../migrations"
	if e = db.Migrate(ctx, migrations); e != nil {
		t.Fatal(e)
	}
	if e = db.Migrate(ctx, migrations); e != nil {
		t.Fatal(e)
	}
	store, e := storage.Open(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	t.Setenv("ARKHIV_FIXTURE", "1")
	if e = fixtures.Seed(ctx, db, store); e != nil {
		t.Fatal(e)
	}

	blank, e := db.Get(ctx, catalog.ID("fixture-zines", "synthetic", "No-text fixture"), false)
	if e != nil || blank.Variants[0].TextStatus != "no-text" {
		t.Fatal("missing text state", e)
	}
	companion := []byte("OCR fallback unpagedword")
	ch, cn, _ := storage.Digest(bytes.NewReader(companion))
	os.WriteFile(filepath.Join(store.Path, "ocr.part"), companion, 0600)
	if e = store.Promote("ocr.part", ch, cn); e != nil {
		t.Fatal(e)
	}
	_, e = db.Pool.Exec(ctx, `INSERT INTO assets(sha256,size) VALUES($1,$2)`, ch, cn)
	if e != nil {
		t.Fatal(e)
	}
	snapOCR, _ := db.Snapshot(ctx, "synthetic", []byte(`{"ocr":true}`))
	_, e = db.Pool.Exec(ctx, `INSERT INTO asset_companions(asset_sha,companion_sha,filename,source_snapshot) VALUES($1,$2,'fixture_djvu.txt',$3)`, blank.Variants[0].SHA, ch, snapOCR)
	if e != nil {
		t.Fatal(e)
	}
	if e = index.Extract(ctx, db, store, blank.Variants[0].SHA); e != nil {
		t.Fatal(e)
	}
	var page int
	if e = db.Pool.QueryRow(ctx, `SELECT page FROM page_text WHERE asset_sha=$1`, blank.Variants[0].SHA).Scan(&page); e != nil || page != 0 {
		t.Fatal("OCR invented a page mapping", e)
	}
	handler := ArchiveHandler(db, store)
	request := func(method, path, rangeValue string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		if rangeValue != "" {
			r.Header.Set("Range", rangeValue)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	id := catalog.ID("fixture-zines", "synthetic", "Mutual aid fixture")
	doc, e := db.Get(ctx, id, true)
	if e != nil {
		t.Fatal(e)
	}
	asset := doc.Variants[0]
	w := request("GET", "/api/documents?collection=fixture-zines&q=knownword", "")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var result struct {
		Total     int
		Documents []catalog.Document
	}
	if e = json.Unmarshal(w.Body.Bytes(), &result); e != nil {
		t.Fatal(e)
	}
	if result.Total != 1 || len(result.Documents[0].Hits) == 0 || result.Documents[0].Hits[0].Page != 2 {
		t.Fatalf("page-search result: %s", w.Body.String())
	}
	privateID := catalog.ID("fixture-zines", "synthetic", "Private fixture")
	private, e := db.Get(ctx, privateID, false)
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range []string{"/api/documents/" + privateID, "/api/assets/" + private.Variants[0].ID, "/api/assets/invalid"} {
		if request("GET", path, "").Code != 404 {
			t.Fatal("unpublished asset leak", path)
		}
	}
	for _, path := range []string{"/api/documents?limit=51", "/api/documents?offset=-1"} {
		if request("GET", path, "").Code != 400 {
			t.Fatal("invalid query accepted")
		}
	}
	w = request("GET", "/api/assets/"+asset.ID, "bytes=0-4")
	if w.Code != 206 || w.Body.String() != "%PDF-" || !strings.HasPrefix(w.Header().Get("Content-Range"), "bytes 0-4/") {
		t.Fatal("range contract", w.Code, w.Body.String())
	}
	w = request("HEAD", "/api/assets/"+asset.ID, "")
	if w.Code != 200 || w.Body.Len() != 0 || w.Header().Get("Content-Type") != "application/pdf" {
		t.Fatal("HEAD contract")
	}
	w = request("GET", "/api/assets/"+asset.ID+"?download=1", "")
	if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment") {
		t.Fatal("download contract")
	}
	h, n, _ := storage.Digest(strings.NewReader(w.Body.String()))
	if h != asset.SHA || n != asset.Size {
		t.Fatal("download differs from original")
	}
	if e = db.Edit(ctx, id, map[string]any{"title": "Reviewed title"}, "", "", nil); e != nil {
		t.Fatal(e)
	}
	c := config.Collection{ID: "fixture-zines", Item: "synthetic"}
	snap, _ := db.Snapshot(ctx, "synthetic", []byte(`{"refresh":true}`))
	if e = db.Import(ctx, c, config.Selection{Key: "Mutual aid fixture", File: asset.Filename, Title: "Source refreshed"}, asset.SHA, asset.Size, snap); e != nil {
		t.Fatal(e)
	}
	refreshed, e := db.Get(ctx, id, true)
	if e != nil || refreshed.Metadata["title"] != "Reviewed title" || !refreshed.Published || len(refreshed.Variants) != 1 {
		t.Fatal("refresh lost editorial state", e)
	}
	// Missing derivatives must prevent publication even with a review note.
	_, e = db.Pool.Exec(ctx, `DELETE FROM derivatives WHERE asset_sha=$1`, private.Variants[0].SHA)
	if e != nil {
		t.Fatal(e)
	}
	yes := true
	if db.Edit(ctx, privateID, nil, "", "review", &yes) == nil {
		t.Fatal("published incomplete document")
	}
	if db.Group(ctx, private.Variants[0].ID, id, "review") == nil {
		t.Fatal("grouped into a published document")
	}
	no := false
	if e = db.Edit(ctx, id, nil, "", "", &no); e != nil {
		t.Fatal(e)
	}
	if e = db.Group(ctx, private.Variants[0].ID, id, "variant review"); e != nil {
		t.Fatal(e)
	}
	backup := filepath.Join(t.TempDir(), "backup")
	if e = recovery.Backup(ctx, db, store, sourceURL, backup); e != nil {
		t.Fatal(e)
	}
	restoredURL := create("restore")
	destination := filepath.Join(t.TempDir(), "restored")
	if e = recovery.Restore(ctx, restoredURL, backup, destination); e != nil {
		t.Fatal(e)
	}
	restored, e := catalog.Open(ctx, restoredURL)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Pool.Close()
	d, e := restored.Get(ctx, id, false)
	if e != nil || d.Metadata["title"] != "Reviewed title" || len(d.Variants) != 2 {
		t.Fatal("restore lost records", e)
	}
	if recovery.Restore(ctx, restoredURL, backup, destination) == nil {
		t.Fatal("restored over existing state")
	}
	os.WriteFile(filepath.Join(backup, "catalog.dump"), []byte("corrupt"), 0600)
	if recovery.Restore(ctx, create("corrupt"), backup, filepath.Join(t.TempDir(), "restore")) == nil {
		t.Fatal("accepted corrupt catalog backup")
	}
}
