package app

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestProcessedVideoPrivateModesIntegrityAndRevocationIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("disposable TEST_DATABASE_URL absent")
	}
	ctx := context.Background()
	admin, err := catalog.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Pool.Close()
	schema := "processed_" + fmt.Sprint(time.Now().UnixNano())
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
	db := &catalog.DB{Pool: pool}
	if err = db.Migrate(ctx, "../../../migrations"); err != nil {
		t.Fatal(err)
	}
	c := domain.Collection{ID: "processed-fixture", Policy: domain.Local, Budget: 768 << 20, Reserve: 1}
	if err = db.Collection(ctx, c); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	media, derived, receipts := filepath.Join(root, "media"), filepath.Join(root, "derived"), filepath.Join(root, "receipts")
	for _, dir := range []string{media, derived, receipts} {
		if err = os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	original, output := []byte("generated original transport bytes"), []byte("generated DERIVATIVE transport bytes")
	captionBytes := []byte("WEBVTT\n\n00:00.000 --> 00:01.000\nGenerated fixture caption only.\n")
	captionMD5 := md5.Sum(captionBytes)
	for path, data := range map[string][]byte{filepath.Join(media, "original.avi"): original, filepath.Join(derived, "derived.mp4"): output} {
		if err = os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	ep := domain.Episode{ID: "processed-episode", CollectionID: c.ID, Title: "Synthetic processed fixture", Date: "2026-10-06", Synthetic: true, Files: []domain.File{{Name: "original.avi", Source: "original", Bytes: int64(len(original)), MD5: strings.Repeat("a", 32), Duration: 10}}}
	ep.Files = append(ep.Files, domain.File{Name: "fixture.vtt", Bytes: int64(len(captionBytes)), MD5: hex.EncodeToString(captionMD5[:])})
	raw := []byte(`{"metadata":{"identifier":"processed-episode"},"files":[{"name":"original.avi"}],"unknown":"retained"}`)
	if err = db.Snapshot(ctx, ep.ID, raw, 200, "", "", &ep); err != nil {
		t.Fatal(err)
	}
	decision := domain.Decision{Acquisition: true, ProcessingOriginal: true, File: "original.avi", Notes: "Explicit synthetic original approval"}
	if err = db.Decide(ctx, ep.ID, decision); err != nil {
		t.Fatal(err)
	}
	if err = db.BindLocal(ctx, ep.ID, domain.Digest(raw), decision.File, "original.avi", domain.Digest(original), []byte(`{"synthetic":true}`)); err != nil {
		t.Fatal(err)
	}
	r := domain.ProcessedVideoReview{Snapshot: domain.Digest(raw), OriginalFile: decision.File, OriginalSHA256: domain.Digest(original), OriginalBytes: int64(len(original)), Path: "derived.mp4", SHA256: domain.Digest(output), Bytes: int64(len(output)), Duration: 10, Width: 320, Height: 240, ReceiptPath: "processing.json", RecoveryPath: "recovery.json", Published: true, Playback: true, Channel: true, Notes: "Synthetic private derived transport review"}
	r.Captions = []domain.LocalVideoCaption{{CaptionReview: domain.CaptionReview{File: "fixture.vtt", Label: "Generated fixture only", Language: "en"}, Path: "fixture.vtt", Bytes: int64(len(captionBytes)), SHA256: domain.Digest(captionBytes)}}
	if err = os.WriteFile(filepath.Join(media, "fixture.vtt"), captionBytes, 0600); err != nil {
		t.Fatal(err)
	}
	processing, _ := json.Marshal(map[string]any{"original": map[string]any{"sha256": r.OriginalSHA256, "bytes": r.OriginalBytes}, "derivative": map[string]any{"sha256": r.SHA256, "bytes": r.Bytes}, "full_decode_original": true, "full_decode_derivative": true, "tool": "Synthetic fixture tool", "tool_image": strings.Repeat("b", 64), "public": false})
	recovery, _ := json.Marshal(map[string]any{"independent_host": "fixture-host", "fresh_restore_root": "/fixture-restore", "fresh_database": "fixture-db", "all_hashes_match": true, "original_immutable": true, "derivative_and_processing_receipts_restored": true, "original_sha256": r.OriginalSHA256, "derivative_sha256": r.SHA256, "processing_receipt_sha256": domain.Digest(processing), "caption_sha256": []string{r.Captions[0].SHA256}, "public": false})
	for path, data := range map[string][]byte{filepath.Join(receipts, r.ReceiptPath): processing, filepath.Join(receipts, r.RecoveryPath): recovery} {
		if err = os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	r.ReceiptSHA256 = domain.Digest(processing)
	r.ReceiptBytes = int64(len(processing))
	r.RecoverySHA256 = domain.Digest(recovery)
	r.RecoveryBytes = int64(len(recovery))
	decision.ProcessedVideo = &r
	if db.Decide(ctx, ep.ID, decision) == nil {
		t.Fatal("generic review bypassed processed registration")
	}
	decision.ProcessedVideo = nil
	if err = db.ReviewProcessedVideo(ctx, ep.ID, r); err != nil {
		t.Fatal(err)
	}
	page, err := db.Catalog(ctx, catalog.CatalogQuery{Collection: c.ID, Page: 1})
	if err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].Representations[0].File != "original.avi" {
		t.Fatal("processed video missing from private mixed catalog or source identity changed", page, err)
	}
	now := time.Now().UTC()
	schedule := domain.Schedule{Revision: "private-processed", Programs: []domain.Program{{EpisodeID: ep.ID, File: r.File().Name, Policy: domain.Local, Start: now.Add(-time.Second), Duration: 10}}}
	if err = db.SaveSchedule(ctx, schedule); err != nil {
		t.Fatal(err)
	}
	server := &Server{DB: db, Media: media, VideoDerivatives: derived, VideoReceipts: receipts, Collections: domain.CollectionScope{c.ID}, Clock: func() time.Time { return now }}
	get := func(path string, headers map[string]string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", path, nil)
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		server.Handler().ServeHTTP(w, req)
		return w
	}
	for _, path := range []string{"/api/episodes", "/api/episodes/" + ep.ID, "/api/playback/" + ep.ID, "/api/channel"} {
		w := get(path, nil)
		if w.Code != 200 {
			t.Fatal(path, w.Code, w.Body.String())
		}
	}
	w := get("/api/media/"+ep.ID, map[string]string{"Range": "bytes=3-8"})
	if w.Code != 206 || w.Body.String() != string(output[3:9]) || w.Header().Get("ETag") != `"sha256-`+r.SHA256+`"` {
		t.Fatal(w.Code, w.Body.String(), w.Header())
	}
	etag := w.Header().Get("ETag")
	captionPath := "/api/video-captions/" + ep.ID + "/0"
	w = get(captionPath, nil)
	if w.Code != 200 || w.Body.String() != string(captionBytes) || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/vtt") {
		t.Fatal("caption serving", w.Code, w.Body.String(), w.Header())
	}
	captionETag := w.Header().Get("ETag")
	if w = get("/api/media/"+ep.ID, map[string]string{"If-None-Match": etag}); w.Code != 304 {
		t.Fatal(w.Code, w.Body.String())
	}
	if db.ReviewPublicVideo(ctx, ep.ID, true, "Must not activate private processing") == nil {
		t.Fatal("public approval accepted")
	}
	public := &Server{DB: &catalog.DB{Pool: pool, PublicOnly: true}, Media: media, VideoDerivatives: derived, VideoReceipts: receipts, Collections: domain.CollectionScope{c.ID}}
	for _, path := range []string{"/api/episodes/" + ep.ID, "/api/media/" + ep.ID, captionPath} {
		w = httptest.NewRecorder()
		public.Handler().ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 404 {
			t.Fatal("private derivative visible publicly", path, w.Code)
		}
	}
	changed := append([]byte(nil), original...)
	changed[0] ^= 1
	if err = os.WriteFile(filepath.Join(media, "original.avi"), changed, 0600); err != nil {
		t.Fatal(err)
	}
	w = get("/api/media/"+ep.ID, map[string]string{"If-None-Match": etag, "Range": "bytes=0-1"})
	if w.Code != 503 {
		t.Fatal("conditional response bypassed original integrity", w.Code)
	}
	if err = os.WriteFile(filepath.Join(media, "original.avi"), original, 0600); err != nil {
		t.Fatal(err)
	}
	brokenCaption := append([]byte(nil), captionBytes...)
	brokenCaption[len(brokenCaption)-2] ^= 1
	if err = os.WriteFile(filepath.Join(media, "fixture.vtt"), brokenCaption, 0600); err != nil {
		t.Fatal(err)
	}
	w = get(captionPath, map[string]string{"If-None-Match": captionETag})
	if w.Code != 503 {
		t.Fatal("caption cache bypassed modified bytes", w.Code)
	}
	if err = os.WriteFile(filepath.Join(media, "fixture.vtt"), captionBytes, 0600); err != nil {
		t.Fatal(err)
	}
	r.Channel = false
	if err = db.ReviewProcessedVideo(ctx, ep.ID, r); err != nil {
		t.Fatal(err)
	}
	if db.SaveSchedule(ctx, domain.Schedule{Revision: "disallowed", Programs: schedule.Programs}) == nil {
		t.Fatal("on-demand review authorized schedule")
	}
	if w = get("/api/playback/"+ep.ID, nil); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if err = db.Decide(ctx, ep.ID, domain.Decision{File: decision.File, Notes: "Revoke original acquisition and derivative access"}); err != nil {
		t.Fatal(err)
	}
	if w = get("/api/media/"+ep.ID, map[string]string{"If-None-Match": etag}); w.Code != 404 {
		t.Fatal("revoked cached derivative remained available", w.Code)
	}
	if w = get(captionPath, map[string]string{"If-None-Match": captionETag}); w.Code != 404 {
		t.Fatal("revoked caption remained available", w.Code)
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM decision_history WHERE episode_id=$1`, ep.ID).Scan(&count); err != nil || count != 4 {
		t.Fatal("decision history not retained", count, err)
	}
}
