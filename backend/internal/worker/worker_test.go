package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"git.subcult.tv/subculture-collective/vhness/internal/source"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWorkerResumeAndPersistentBackoff(t *testing.T) {
	database := os.Getenv("TEST_DATABASE_URL")
	if database == "" {
		t.Skip("TEST_DATABASE_URL not set; worker integration unverified")
	}
	ctx := context.Background()
	admin, err := catalog.Open(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Pool.Close()
	schema := "vhness_worker_" + time.Now().UTC().Format("20060102150405")
	name := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Pool.Exec(ctx, "CREATE SCHEMA "+name); err != nil {
		t.Fatal(err)
	}
	defer admin.Pool.Exec(ctx, "DROP SCHEMA "+name+" CASCADE")
	config, err := pgxpool.ParseConfig(database)
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
	if err = d.Collection(ctx, domain.Collection{ID: domain.DemocracyNow, Policy: domain.Stream}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	requests := 0
	limited := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/metadata/democracy_now_vid":
			w.Write([]byte(`{"metadata":{"identifier":"democracy_now_vid","mediatype":"collection"},"files":[]}`))
		case "/advancedsearch.php":
			w.Write([]byte(`{"response":{"numFound":2,"docs":[{"identifier":"item"},{"identifier":"item"}]}}`))
		case "/metadata/item":
			if limited {
				w.Header().Set("Retry-After", "7200")
				w.WriteHeader(429)
				return
			}
			w.Write([]byte(`{"metadata":{"identifier":"item","mediatype":"movies","collection":"democracy_now_vid","title":"Sample"},"files":[{"name":"a.mp4","size":"10","length":"60"}]}`))
		default:
			t.Errorf("worker requested media: %s", r.URL.Path)
			w.WriteHeader(400)
		}
	}))
	defer server.Close()
	client := &source.Client{Base: server.URL, HTTP: server.Client(), Clock: func() time.Time { return now }}
	w := Worker{DB: d, Source: client}
	result, err := w.Run(ctx, domain.DemocracyNow, 1, 10, time.Minute)
	if err != nil || result.Backoff == "" {
		t.Fatalf("run %v %v", result, err)
	}
	state, err := d.State(ctx, domain.DemocracyNow)
	if err != nil || state.RetryAt == nil || !state.RetryAt.Equal(now.Add(2*time.Hour)) || !state.Complete {
		t.Fatalf("state %v %v", state, err)
	}
	before := requests
	result, err = w.Run(ctx, domain.DemocracyNow, 1, 10, time.Minute)
	if err != nil || requests != before || result.Backoff == "" {
		t.Fatal("restart ignored persistent backoff")
	}
	limited = false
	now = now.Add(3 * time.Hour)
	if _, err = d.Pool.Exec(ctx, `UPDATE metadata_queue SET next_attempt=now() WHERE collection_id=$1`, domain.DemocracyNow); err != nil {
		t.Fatal(err)
	}
	result, err = w.Run(ctx, domain.DemocracyNow, 1, 10, time.Minute)
	if err != nil || result.Imported != 1 || result.Pages != 0 {
		t.Fatalf("resume %v %v", result, err)
	}
	counts, err := d.Counts(ctx, domain.DemocracyNow)
	if err != nil || counts["imported"] != 1 {
		t.Fatalf("counts %v %v", counts, err)
	}
	var snapshots int
	if err = d.Pool.QueryRow(ctx, `SELECT count(*) FROM snapshots`).Scan(&snapshots); err != nil || snapshots != 2 {
		t.Fatal("collection/item metadata not preserved", err, snapshots)
	}
}
