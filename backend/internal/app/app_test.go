package app

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPrivatePlaybackAndMediaIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL absent; HTTP/database integration unverified")
	}
	ctx := context.Background()
	admin, err := catalog.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Pool.Close()
	schema := "vhness_http_" + time.Now().Format("20060102150405")
	name := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Pool.Exec(ctx, "CREATE SCHEMA "+name); err != nil {
		t.Fatal(err)
	}
	defer admin.Pool.Exec(ctx, "DROP SCHEMA "+name+" CASCADE")
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	d := &catalog.DB{Pool: pool}
	if err = d.Migrate(ctx, "../../../migrations"); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "index.html"), []byte("VHNESS application"), 0600)
	clock := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	server := &Server{DB: d, Web: root, Media: root, Clock: func() time.Time { return clock }}
	handler := server.Handler()
	get := func(path string, headers map[string]string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := get("/library", nil); w.Code != 200 || w.Body.String() != "VHNESS application" {
		t.Fatal("static fallback", w.Code, w.Body.String())
	}
	if w := get("/api/unknown", nil); w.Code != 404 {
		t.Fatal("unknown API fell through", w.Code)
	}
	c := domain.Collection{ID: "generated", Policy: domain.Local, Budget: 1 << 20, Reserve: 1}
	if err = d.Collection(ctx, c); err != nil {
		t.Fatal(err)
	}
	content := []byte("generated-video-original")
	hash := domain.Digest(content)
	e := domain.Episode{ID: "generated-one", CollectionID: c.ID, Title: "Generated fixture", Synthetic: true, Files: []domain.File{{Name: "original.mp4", Bytes: int64(len(content)), Duration: 10}}}
	raw := []byte(`{"generated":true}`)
	if err = d.Snapshot(ctx, e.ID, raw, 200, "", "", &e); err != nil {
		t.Fatal(err)
	}
	if w := get("/api/playback/"+e.ID, nil); w.Code != 404 {
		t.Fatal("unreviewed playback exposed")
	}
	decision := domain.Decision{Acquisition: true, Published: true, Playback: true, Channel: true, File: "original.mp4", Notes: "Generated original private review"}
	if err = d.Decide(ctx, e.ID, decision); err != nil {
		t.Fatal(err)
	}
	if w := get("/api/playback/"+e.ID, nil); w.Code != 404 {
		t.Fatal("unacquired local exposed")
	}
	os.WriteFile(filepath.Join(root, "original.mp4"), content, 0600)
	if err = d.BindLocal(ctx, e.ID, domain.Digest(raw), decision.File, "original.mp4", hash, []byte(`{"fixture":true}`)); err != nil {
		t.Fatal(err)
	}
	if w := get("/api/media/"+e.ID, map[string]string{"Range": "bytes=2-6"}); w.Code != 206 || w.Body.String() != string(content[2:7]) {
		t.Fatal("range", w.Code, w.Body.String())
	}
	response := get("/api/media/"+e.ID, nil)
	etag := response.Header().Get("ETag")
	if w := get("/api/media/"+e.ID, map[string]string{"If-None-Match": etag}); w.Code != 304 {
		t.Fatal("conditional", w.Code)
	}
	schedule := domain.Schedule{Revision: "generated-proof", Programs: []domain.Program{{EpisodeID: e.ID, File: decision.File, Policy: c.Policy, Start: clock.Add(-2 * time.Second), Duration: 10}}}
	if err = d.SaveSchedule(ctx, schedule); err != nil {
		t.Fatal(err)
	}
	var now domain.Now
	w := get("/api/channel", nil)
	if err = json.Unmarshal(w.Body.Bytes(), &now); err != nil || now.Offset != 2 || now.State != "playing" || !now.ServerTime.Equal(clock) {
		t.Fatal("channel clock", err, now)
	}
	if now.Episode == nil || len(now.Episode.Files) != 0 || now.Playback == nil || now.Playback.File.Name != decision.File {
		t.Fatal("channel should retain the selected asset without repeating the complete source inventory")
	}
	decision.Playback = false
	decision.Channel = false
	decision.Published = false
	if err = d.Decide(ctx, e.ID, decision); err != nil {
		t.Fatal(err)
	}
	if w := get("/api/media/"+e.ID, map[string]string{"If-None-Match": etag}); w.Code != 404 {
		t.Fatal("withdrawn conditional leaked", w.Code)
	}
	if err = d.Decide(ctx, e.ID, domain.Decision{Acquisition: true, Published: true, Playback: true, Channel: true, File: "original.mp4", Notes: "Restore fixture review"}); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "original.mp4"), []byte("corrupted"), 0600)
	if w := get("/api/media/"+e.ID, nil); w.Code != 503 {
		t.Fatal("corrupt original served", w.Code)
	}
}
