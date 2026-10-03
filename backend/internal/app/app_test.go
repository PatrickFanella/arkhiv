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
	config.MaxConns = 2
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
	// The legacy projection retains stable video links and reads decisions live.
	var mixedPage catalog.CatalogPage
	if err = json.Unmarshal(get("/api/catalog/v1/items?media=video", nil).Body.Bytes(), &mixedPage); err != nil || mixedPage.Total != 1 || mixedPage.Items[0].Representations[0].LegacyEpisodeID != e.ID {
		t.Fatal("legacy catalog projection", err, mixedPage)
	}
	if err = d.Collection(ctx, domain.Collection{ID: "mixed-fixture", Policy: domain.Stream}); err != nil {
		t.Fatal(err)
	}
	sourceRaw := []byte(`{"metadata":{"identifier":"multi-source","unknown":"retained"},"files":[{"name":"talk.pdf"},{"name":"talk.flac"}]}`)
	if err = d.Snapshot(ctx, "multi-source", sourceRaw, 200, "", "", nil); err != nil {
		t.Fatal(err)
	}
	item := domain.CatalogItem{ID: "multi-talk", Title: "A mixed talk", Collections: []string{c.ID, "mixed-fixture"}, Representations: []domain.Representation{
		{ID: "print", Media: domain.PrintMedia, SourceItemID: "multi-source", Snapshot: domain.Digest(sourceRaw), File: "talk.pdf", Collections: []string{c.ID, "mixed-fixture"}, Extension: domain.MediaExtension{Print: &domain.PrintExtension{DocumentID: "doc", Variant: "original", Pages: 2}}},
		{ID: "audio", Media: domain.AudioMedia, SourceItemID: "multi-source", Snapshot: domain.Digest(sourceRaw), File: "talk.flac", Collections: []string{"mixed-fixture"}, Extension: domain.MediaExtension{Audio: &domain.AudioExtension{RecordingID: "recording", Tracks: []domain.AudioTrack{{ID: "track", File: "talk.flac"}}}}},
	}}
	item.Representations[0].File = "absent.pdf"
	if err = d.InsertCatalog(ctx, item, "Must reject a fabricated source file"); err == nil {
		t.Fatal("absent source file accepted")
	}
	item.Representations[0].File = "talk.pdf"
	if err = d.InsertCatalog(ctx, item, "Synthetic metadata-only catalog mapping"); err != nil {
		t.Fatal(err)
	}
	server.Collection = c.ID
	if err = json.Unmarshal(get("/api/catalog/v1/items/multi-talk", nil).Body.Bytes(), &item); err != nil || len(item.Representations) != 1 || len(item.Collections) != 1 || item.Collections[0] != c.ID {
		t.Fatal("representation scope leaked", err, item)
	}
	if w := get("/api/catalog/v1/items?collection=mixed-fixture", nil); w.Code != 404 {
		t.Fatal("foreign catalog scope", w.Code)
	}
	server.Collection = ""
	if err = json.Unmarshal(get("/api/catalog/v1/items?media=audio&q=mixed", nil).Body.Bytes(), &mixedPage); err != nil || mixedPage.Total != 1 || len(mixedPage.Items[0].Representations) != 1 || mixedPage.SearchCoverage[0] != "metadata" {
		t.Fatal("typed search", err, mixedPage)
	}
	if err = json.Unmarshal(get("/api/catalog/v1/items?media=audio&page=999", nil).Body.Bytes(), &mixedPage); err != nil || mixedPage.Total != 1 || len(mixedPage.Items) != 0 {
		t.Fatal("empty page lost total", err, mixedPage)
	}
	for _, path := range []string{"/api/catalog/v1/items?page=0", "/api/catalog/v1/items?media=pdf", "/api/catalog/v1/items?page=999999999999999999999"} {
		if w := get(path, nil); w.Code != 400 {
			t.Fatal("invalid catalog query", path, w.Code)
		}
	}
	if _, err = d.Pool.Exec(ctx, `UPDATE catalog_representation_memberships SET listed=false WHERE item_id='multi-talk' AND representation_id='audio'`); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(get("/api/catalog/v1/items?media=audio", nil).Body.Bytes(), &mixedPage); err != nil || mixedPage.Total != 0 {
		t.Fatal("revoked representation remained searchable", err, mixedPage)
	}
	if _, err = d.Pool.Exec(ctx, `UPDATE catalog_memberships SET listed=false WHERE item_id='multi-talk'`); err != nil {
		t.Fatal(err)
	}
	if w := get("/api/catalog/v1/items/multi-talk", nil); w.Code != 404 {
		t.Fatal("revoked item remained addressable", w.Code)
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
	// A product instance must not expose another collection even when its
	// database contains retained migration/recovery records for that source.
	server.Collection = domain.DemocracyNow
	for _, path := range []string{"/api/episodes/" + e.ID, "/api/playback/" + e.ID, "/api/media/" + e.ID, "/api/episodes?collection=" + c.ID, "/api/status?collection=" + c.ID} {
		if w := get(path, nil); w.Code != 404 {
			t.Fatal("cross-collection response", path, w.Code)
		}
	}
	var catalogResponse struct {
		Episodes []domain.Episode `json:"episodes"`
		Total    int              `json:"total"`
	}
	if err = json.Unmarshal(get("/api/episodes", nil).Body.Bytes(), &catalogResponse); err != nil || catalogResponse.Total != 0 {
		t.Fatal("unscoped query leaked another collection", err, catalogResponse)
	}
	now = domain.Now{}
	if err = json.Unmarshal(get("/api/channel", nil).Body.Bytes(), &now); err != nil || now.State != "unavailable" || now.Episode != nil {
		t.Fatal("cross-collection schedule leaked", err, now)
	}
	var collections []catalog.CollectionSummary
	if err = json.Unmarshal(get("/api/collections", nil).Body.Bytes(), &collections); err != nil || len(collections) != 0 {
		t.Fatal("collections leaked", err, collections)
	}
	server.Collection = c.ID
	if w := get("/api/playback/"+e.ID, nil); w.Code != 200 {
		t.Fatal("configured collection was blocked", w.Code)
	}
	server.Collection = ""
	if err = d.Collection(ctx, domain.Collection{ID: domain.DemocracyNow, Policy: domain.Stream}); err != nil {
		t.Fatal(err)
	}
	other := domain.Episode{ID: "other-collection", CollectionID: domain.DemocracyNow, Title: "Other source fixture", Synthetic: true, Files: []domain.File{{Name: "original.mp4", Bytes: 1, Duration: 10}}}
	if err = d.Snapshot(ctx, other.ID, []byte(`{"other_source_fixture":true}`), 200, "", "", &other); err != nil {
		t.Fatal(err)
	}
	if err = d.Decide(ctx, other.ID, domain.Decision{Published: true, Playback: true, Channel: true, File: "original.mp4", Notes: "Synthetic source fixture review"}); err != nil {
		t.Fatal(err)
	}
	mixed := domain.Schedule{Revision: "mixed-scope-proof", Programs: append(append([]domain.Program{}, schedule.Programs...), domain.Program{EpisodeID: other.ID, File: "original.mp4", Policy: domain.Stream, Start: clock.Add(8 * time.Second), Duration: 10})}
	currentEntries, err := d.CurrentEntries(ctx, mixed, clock)
	if err != nil || len(currentEntries) != 1 || currentEntries[e.ID].Episode.ID != e.ID {
		t.Fatal("poll loaded future inventories", err, currentEntries)
	}
	if within, err := d.ScheduleWithinCollection(ctx, mixed, c.ID); err != nil || within {
		t.Fatal("future program scope bypass", err, within)
	}
	if err = d.SaveSchedule(ctx, mixed); err != nil {
		t.Fatal(err)
	}
	server.Collection = c.ID
	now = domain.Now{}
	if err = json.Unmarshal(get("/api/channel", nil).Body.Bytes(), &now); err != nil || now.State != "unavailable" || now.Episode != nil {
		t.Fatal("foreign future slot exposed through current-only poll", err, now)
	}
	server.Collection = ""
	schedule.Revision = "generated-proof-after-scope"
	if err = d.SaveSchedule(ctx, schedule); err != nil {
		t.Fatal(err)
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
	if w := get("/api/catalog/v1/items/legacy-video:"+e.ID, nil); w.Code != 404 {
		t.Fatal("legacy withdrawal not reflected", w.Code)
	}
	if err = d.Decide(ctx, e.ID, domain.Decision{Acquisition: true, Published: true, Playback: true, Channel: true, File: "original.mp4", Notes: "Restore fixture review"}); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "original.mp4"), []byte("corrupted"), 0600)
	if w := get("/api/media/"+e.ID, nil); w.Code != 503 {
		t.Fatal("corrupt original served", w.Code)
	}
	// Reproduce a DB operation unable to acquire a connection, without
	// disrupting another runtime or changing the disposable database itself.
	held := make([]*pgxpool.Conn, 0, 2)
	for range 2 {
		connection, err := pool.Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		held = append(held, connection)
	}
	began := time.Now()
	response = get("/api/ready", nil)
	for _, connection := range held {
		connection.Release()
	}
	if response.Code != 503 || time.Since(began) > 4*time.Second {
		t.Fatal("blocked database readiness must fail within its deadline", response.Code, time.Since(began))
	}
	if w := get("/api/ready", nil); w.Code != 200 {
		t.Fatal("readiness did not recover after DB capacity returned", w.Code)
	}
}
