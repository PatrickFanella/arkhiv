package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestVideoViewerProjectionIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL absent; viewer database integration unverified")
	}
	ctx := context.Background()
	admin, err := catalog.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Pool.Close()
	schema := "viewer_" + fmt.Sprint(time.Now().UnixNano())
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
	for _, id := range []string{"viewer-fixture", "foreign-fixture"} {
		if err = d.Collection(ctx, domain.Collection{ID: id, Policy: domain.Stream}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 30; i++ {
		id := fmt.Sprintf("episode-%02d", i)
		raw := []byte(fmt.Sprintf(`{"metadata":{"identifier":%q,"subject":["Science & nature"],"language":"en","private":"secret source"},"files":[{"name":"movie.mp4"},{"name":"cover.jpg"}]}`, id))
		e := domain.Episode{ID: id, CollectionID: "viewer-fixture", Title: id, Date: "2026-10-05", Creator: "Teacher", Files: []domain.File{{Name: "movie.mp4", Bytes: 100, Duration: 10, Format: "MPEG4", Raw: json.RawMessage(`{"private":"secret file"}`)}, {Name: "cover.jpg", Bytes: 20}}}
		if i == 29 {
			e.CollectionID = "foreign-fixture"
		}
		if err = d.Snapshot(ctx, id, raw, 200, "", "", &e); err != nil {
			t.Fatal(err)
		}
		decision := domain.Decision{Published: true, Playback: true, Channel: true, File: "movie.mp4", Title: "Editorial " + id, Notes: "secret operator", Artwork: &domain.ArtworkReview{File: "cover.jpg", Snapshot: domain.Digest(raw)}}
		if err = d.Decide(ctx, id, decision); err != nil {
			t.Fatal(err)
		}
		if i != 28 {
			if err = d.ReviewPublicVideo(ctx, id, true, "Synthetic fixture approval"); err != nil {
				t.Fatal(err)
			}
		}
	}
	server := &Server{DB: &catalog.DB{Pool: pool, PublicOnly: true}, Collections: domain.CollectionScope{"viewer-fixture"}}
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}
	query := "/api/episodes?collection=viewer-fixture&subject=Science%20%26%20nature&language=en&creator=Teacher&duration_min=5&duration_max=15&date_from=2026-01-01&date_to=2026-12-31"
	var page struct {
		Episodes []domain.Episode `json:"episodes"`
		Total    int              `json:"total"`
	}
	w := get(query + "&page=2")
	if err = json.Unmarshal(w.Body.Bytes(), &page); err != nil || w.Code != 200 || page.Total != 28 || len(page.Episodes) != 4 {
		t.Fatal(w.Code, w.Body.String(), err)
	}
	if page.Episodes[0].Video == nil || page.Episodes[0].Artwork == nil || page.Episodes[0].Video.TitleOrigin != "editorial" {
		t.Fatal("summary lost selected file, artwork or provenance")
	}
	w = get("/api/episodes/episode-00")
	var detail struct {
		Episode domain.Episode `json:"episode"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || len(detail.Episode.Subjects) != 1 || detail.Episode.Subjects[0] != "Science & nature" || strings.Contains(w.Body.String(), "secret") || strings.Contains(w.Body.String(), `"decision"`) {
		t.Fatal("unsafe detail", w.Code, w.Body.String())
	}
	for _, path := range []string{"/api/episodes/episode-28", "/api/episodes/episode-29"} {
		if get(path).Code != 404 {
			t.Fatal("private/foreign detail exposed", path)
		}
	}
	if get("/api/episodes?duration_min=NaN").Code != 400 {
		t.Fatal("invalid filters accepted")
	}
	w = get(query + "&q=missing")
	json.Unmarshal(w.Body.Bytes(), &page)
	if page.Total != 0 {
		t.Fatal("combined query not applied")
	}
	// Withdrawal and source refresh invalidate publication without deleting data.
	if err = d.ReviewPublicVideo(ctx, "episode-00", false, "Synthetic withdrawal"); err != nil {
		t.Fatal(err)
	}
	if get("/api/episodes/episode-00").Code != 404 {
		t.Fatal("withdrawn detail exposed")
	}
	stored, err := d.Entry(ctx, "episode-00")
	if err != nil || len(stored.Episode.Files) != 2 || stored.Decision.Notes != "secret operator" {
		t.Fatal("internal history/inventory changed", err)
	}
}
