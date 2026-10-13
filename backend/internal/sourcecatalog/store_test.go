package sourcecatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	sourcepkg "git.subcult.tv/subculture-collective/vhness/internal/source"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testDB(t *testing.T) *Store {
	t.Helper()
	address := os.Getenv("TEST_DATABASE_URL")
	if address == "" {
		t.Skip("set TEST_DATABASE_URL to an explicitly disposable PostgreSQL database")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, address)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("test_live_music_%d", time.Now().UnixNano())
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	config, err := pgxpool.ParseConfig(address)
	if err != nil {
		t.Fatal(err)
	}
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET search_path TO "+quoted)
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	core := &catalog.DB{Pool: pool}
	db := &Store{Core: core, Collection: "etree"}
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+quoted+" CASCADE")
		_ = admin.Close(context.Background())
	})
	if err = db.Core.Migrate(ctx, "../../../migrations"); err != nil {
		t.Fatal(err)
	}
	if err := db.Core.Collection(ctx, domain.Collection{ID: "etree", Policy: domain.Stream}); err != nil {
		t.Fatal(err)
	}
	if err := db.Init(ctx); err != nil {
		t.Fatal(err)
	}
	return db
}
func fixture(id, venue string) []byte {
	return []byte(fmt.Sprintf(`{"unknown":{"keep":true},"metadata":{"identifier":%q,"collection":["etree"],"creator":"Ween","title":"Show","venue":%q,"date":"2003-09-23","extra":"preserved"},"files":[{"name":"song.mp3","title":"Ocean Man"},{"name":"private.flac","private":"true","custom":42}]}`, id, venue))
}
func TestImportReimportAndEditorialSearch(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	raw := fixture("show", "Original Room")
	if err := db.Import(ctx, "show", raw, 200, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := db.Edit(ctx, "show", map[string]any{"venue": "Corrected Room"}, "venue confirmed by taper notes"); err != nil {
		t.Fatal(err)
	}
	if err := db.Import(ctx, "show", fixture("show", "Updated Source Room"), 200, "", ""); err != nil {
		t.Fatal(err)
	}
	detail, err := db.Detail(ctx, "show")
	if err != nil {
		t.Fatal(err)
	}
	var effective map[string]string
	if err = json.Unmarshal(detail["metadata"].(json.RawMessage), &effective); err != nil {
		t.Fatal(err)
	}
	if effective["venue"] != "Corrected Room" || detail["source_metadata"].(map[string]any)["venue"] != "Updated Source Room" {
		t.Fatal(detail)
	}
	files := detail["files"].([]any)
	if len(files) != 2 || files[1].(map[string]any)["custom"] != float64(42) || detail["unknown"] == nil {
		t.Fatal("source inventory or unknown fields lost")
	}
	page, err := db.Search(ctx, Query{Search: "Corrected Ocean", Year: "2003", Artist: "Ween", Venue: "corrected", From: "2003-09-01", To: "2003-10-01"})
	if err != nil || page.Total != 1 {
		t.Fatal(page, err)
	}
	var snapshots int
	if err = db.Core.Pool.QueryRow(ctx, `SELECT count(*) FROM snapshots`).Scan(&snapshots); err != nil || snapshots != 2 {
		t.Fatal(snapshots, err)
	}
	if err = db.Edit(ctx, "show", map[string]any{"venue": nil}, "restore source"); err != nil {
		t.Fatal(err)
	}
	page, err = db.Search(ctx, Query{Venue: "updated source"})
	if err != nil || page.Total != 1 {
		t.Fatal(page, err)
	}
	if err = db.Core.Migrate(ctx, "../../../migrations"); err != nil {
		t.Fatal("migration is not idempotent", err)
	}
	if _, err = db.Core.Pool.Exec(ctx, `UPDATE schema_versions SET sha256='changed'`); err != nil {
		t.Fatal(err)
	}
	if err = db.Core.Migrate(ctx, "../../../migrations"); err == nil {
		t.Fatal("changed migration checksum accepted")
	}
}
func TestKeysetPagination(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	for i := 0; i < 103; i++ {
		id := fmt.Sprintf("show%03d", i)
		if err := db.Import(ctx, id, fixture(id, "Room"), 200, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	first, err := db.Search(ctx, Query{})
	if err != nil || len(first.Items) != 100 || first.Total != 103 || first.Cursor == "" {
		t.Fatal(first, err)
	}
	next, err := db.Search(ctx, Query{Cursor: first.Cursor})
	if err != nil || len(next.Items) != 3 || next.Total != 103 || next.Cursor != "" {
		t.Fatal(next, err)
	}
	if next.Items[0]["identifier"] != "show100" {
		t.Fatal(next.Items)
	}
	if _, err = db.Search(ctx, Query{Search: "room", Cursor: first.Cursor}); err == nil {
		t.Fatal("accepted cursor from another query")
	}
}
func TestWorkerMetadataOnlyAndResume(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		switch r.URL.Path {
		case "/services/search/v1/scrape":
			if r.URL.Query().Get("cursor") == "" {
				_, _ = w.Write([]byte(`{"items":[{"identifier":"show1"},{"identifier":"show2"}],"cursor":"second","total":3}`))
			} else {
				_, _ = w.Write([]byte(`{"items":[{"identifier":"show3"}],"total":1}`))
			}
		case "/metadata/show1", "/metadata/show2", "/metadata/show3":
			_, _ = w.Write(fixture(strings.TrimPrefix(r.URL.Path, "/metadata/"), "Room"))
		default:
			t.Errorf("worker requested media: %s", r.URL.Path)
			w.WriteHeader(400)
		}
	}))
	defer server.Close()
	client := &sourcepkg.Client{Base: server.URL, HTTP: server.Client(), Clock: time.Now}
	if err := db.Run(ctx, client, 1, 1, 0); err != nil {
		t.Fatal(err)
	}
	status, err := db.Status(ctx)
	if err != nil || status.Imported != 1 || status.Pending != 1 || status.DiscoveryComplete {
		t.Fatal(status, err)
	}
	if err := db.Run(ctx, client, 1, 10, 0); err != nil {
		t.Fatal(err)
	}
	status, err = db.Status(ctx)
	if err != nil || status.Imported != 3 || status.Pending != 0 || !status.DiscoveryComplete {
		t.Fatal(status, err)
	}
	var raw []byte
	raw, err = db.Raw(ctx, "show1")
	if err != nil || string(raw) != string(fixture("show1", "Room")) {
		t.Fatal("source bytes changed", err)
	}
	if err := db.Edit(ctx, "show1", map[string]any{"venue": "Correction"}, "evidence"); err != nil {
		t.Fatal(err)
	}
	if err := db.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.Run(ctx, client, 1, 10, 0); err != nil {
		t.Fatal(err)
	}
	page, err := db.Search(ctx, Query{Venue: "Correction"})
	if err != nil || page.Total != 1 {
		t.Fatal("reconciliation overwrote editorial state", page, err)
	}
	for _, path := range requests {
		if strings.HasPrefix(path, "/download") {
			t.Fatal(path)
		}
	}
}
func TestWithdrawalAndTransientRetry(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	if err := db.Import(ctx, "show", fixture("show", "Room"), 200, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := db.Fail(ctx, "show", "retry", 503, "source down", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	page, err := db.Search(ctx, Query{Availability: "available"})
	if err != nil || page.Total != 1 {
		t.Fatal(page, err)
	}
	if err := db.Fail(ctx, "show", "unavailable", 404, "withdrawn", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	page, err = db.Search(ctx, Query{Availability: "unavailable"})
	if err != nil || page.Total != 1 {
		t.Fatal(page, err)
	}
	if _, err := db.Raw(ctx, "show"); err != nil {
		t.Fatal("withdrawal lost metadata", err)
	}
	if err := db.Import(ctx, "show", fixture("show", "Room"), 200, "", ""); err != nil {
		t.Fatal(err)
	}
	page, err = db.Search(ctx, Query{Availability: "available"})
	if err != nil || page.Total != 1 {
		t.Fatal(page, err)
	}
}

func TestPlaybackRequiresSeparateReviewAndRefreshInvalidatesIt(t *testing.T) {
	store := testDB(t)
	ctx := context.Background()
	raw := fixture("show", "Room")
	if err := store.Import(ctx, "show", raw, 200, "", ""); err != nil {
		t.Fatal(err)
	}
	detail, err := store.Detail(ctx, "show")
	if err != nil {
		t.Fatal(err)
	}
	if string(detail["playback_bindings"].(json.RawMessage)) != "[]" {
		t.Fatal("metadata import granted playback")
	}
	if err := store.ReviewAudio(ctx, "show", "Owner approved private source-streaming sample; no acquisition or relay"); err != nil {
		t.Fatal(err)
	}
	detail, err = store.Detail(ctx, "show")
	if err != nil {
		t.Fatal(err)
	}
	var bindings []struct {
		Item           string `json:"item"`
		Representation string `json:"representation"`
	}
	if err = json.Unmarshal(detail["playback_bindings"].(json.RawMessage), &bindings); err != nil || len(bindings) != 1 {
		t.Fatal(bindings, err)
	}
	binding := bindings[0]
	entry, err := store.Core.Transport(ctx, binding.Item, binding.Representation, "etree")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Review.Acquisition || entry.Review.Relay || entry.Review.Public || entry.Review.Schedule {
		t.Fatal("source playback granted unrelated rights")
	}
	if err := store.Import(ctx, "show", fixture("show", "Changed source"), 200, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Core.Transport(ctx, binding.Item, binding.Representation, "etree"); err == nil {
		t.Fatal("stale review survived changed source snapshot")
	}
	if err := store.ReviewAudio(ctx, "show", "Re-reviewed changed source metadata"); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Core.Transport(ctx, binding.Item, binding.Representation, "etree"); err != nil {
		t.Fatal("explicit re-review failed", err)
	}
	if err := store.Fail(ctx, "show", "unavailable", 404, "source removed", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Core.Transport(ctx, binding.Item, binding.Representation, "etree"); err == nil {
		t.Fatal("withdrawn source retained transport")
	}
}
func TestSourceCatalogCollectionIsolation(t *testing.T) {
	store := testDB(t)
	ctx := context.Background()
	if err := store.Import(ctx, "show", fixture("show", "Room"), 200, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.Core.Collection(ctx, domain.Collection{ID: "other", Policy: domain.Stream}); err != nil {
		t.Fatal(err)
	}
	other := &Store{Core: store.Core, Collection: "other"}
	if err := other.Init(ctx); err != nil {
		t.Fatal(err)
	}
	page, err := other.Search(ctx, Query{})
	if err != nil || page.Total != 0 {
		t.Fatal("foreign collection leaked", page, err)
	}
	if _, err := other.Detail(ctx, "show"); err == nil {
		t.Fatal("foreign source detail leaked")
	}
	if err := other.Edit(ctx, "show", map[string]any{"venue": "wrong"}, "wrong collection"); err == nil {
		t.Fatal("foreign correction accepted")
	}
}

func TestAudioReviewPreservesExactFilename(t *testing.T) {
	store := testDB(t)
	ctx := context.Background()
	raw := []byte(strings.ReplaceAll(string(fixture("show", "Room")), "song.mp3", "01  Song.mp3"))
	if err := store.Import(ctx, "show", raw, 200, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.ReviewAudio(ctx, "show", "Review exact original filename"); err != nil {
		t.Fatal(err)
	}
	detail, err := store.Detail(ctx, "show")
	if err != nil {
		t.Fatal(err)
	}
	var bindings []struct {
		Files []string `json:"files"`
	}
	if err = json.Unmarshal(detail["playback_bindings"].(json.RawMessage), &bindings); err != nil || len(bindings) != 1 || len(bindings[0].Files) != 1 || bindings[0].Files[0] != "01  Song.mp3" {
		t.Fatal(bindings, err)
	}
}
