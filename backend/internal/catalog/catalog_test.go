package catalog

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"git.subcult.tv/subculture-collective/vhness/internal/source"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMetadataIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; database integration unverified")
	}
	ctx := context.Background()
	d, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Pool.Close()
	schema := "vhness_test_" + time.Now().UTC().Format("20060102150405") + "_" + domain.Digest([]byte(t.Name()))[:8]
	admin := d.Pool
	name := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+name); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+name+" CASCADE")
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
	d = &DB{Pool: pool}
	if err = d.Migrate(ctx, "../../../migrations"); err != nil {
		t.Fatal(err)
	}
	if err = d.Migrate(ctx, "../../../migrations"); err != nil {
		t.Fatal("repeat migration", err)
	}
	dir := t.TempDir()
	b, err := os.ReadFile("../../../migrations/0001_metadata.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "0001_metadata.sql"), append(b, []byte("\n-- altered\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if d.Migrate(ctx, dir) == nil {
		t.Fatal("changed migration accepted")
	}
	if err = d.Collection(ctx, domain.Collection{ID: domain.DemocracyNow, Policy: domain.Stream}); err != nil {
		t.Fatal(err)
	}
	if _, err = d.Pool.Exec(ctx, `UPDATE collections SET policy='local_preserve',budget=10,reserve=1 WHERE id=$1`, domain.DemocracyNow); err == nil {
		t.Fatal("DB Democracy policy protection missing")
	}
	raw := []byte("{\n\"metadata\":{\"identifier\":\"item\",\"mediatype\":\"movies\",\"collection\":[\"democracy_now_vid\"],\"title\":\"Title\",\"unknown\":42},\"files\":[{\"name\":\"a.mp4\",\"size\":\"123\",\"length\":\"60\",\"custom\":true}]}\n")
	e, err := source.Normalize("item", domain.DemocracyNow, raw)
	if err != nil {
		t.Fatal(err)
	}
	if err = d.Snapshot(ctx, e.ID, raw, 200, "etag", "", &e); err != nil {
		t.Fatal(err)
	}
	decision := domain.Decision{Published: true, Playback: true, Channel: true, File: "a.mp4", Title: "Editorial title", Notes: "Private streaming review"}
	if err = d.Decide(ctx, e.ID, decision); err != nil {
		t.Fatal(err)
	}
	var recovered []byte
	if err = d.Pool.QueryRow(ctx, `SELECT raw FROM snapshots WHERE sha256=$1`, domain.Digest(raw)).Scan(&recovered); err != nil || string(recovered) != string(raw) {
		t.Fatal("raw snapshot fidelity", err)
	}
	var changed map[string]any
	if err = json.Unmarshal(raw, &changed); err != nil {
		t.Fatal(err)
	}
	changed["metadata"].(map[string]any)["title"] = "Source changed"
	newRaw, _ := json.Marshal(changed)
	e, err = source.Normalize("item", domain.DemocracyNow, newRaw)
	if err != nil {
		t.Fatal(err)
	}
	if err = d.Snapshot(ctx, e.ID, newRaw, 200, "new", "", &e); err != nil {
		t.Fatal(err)
	}
	entries, err := d.Entries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if entries["item"].Episode.Title != "Editorial title" || entries["item"].Decision.Notes != decision.Notes {
		t.Fatal("refresh overwrote review")
	}
	var count int
	if err = d.Pool.QueryRow(ctx, `SELECT count(*) FROM snapshots`).Scan(&count); err != nil || count != 2 {
		t.Fatal("snapshot history", err, count)
	}
	schedule := domain.Schedule{Revision: "one", Programs: []domain.Program{{EpisodeID: "item", File: "a.mp4", Policy: domain.Stream, Start: time.Now().UTC(), Duration: 60}}}
	if err = d.SaveSchedule(ctx, schedule); err != nil {
		t.Fatal(err)
	}
	if d.SaveSchedule(ctx, schedule) == nil {
		t.Fatal("schedule revision overwritten")
	}
	restored, err := d.Schedule(ctx)
	if err != nil || restored.Revision != "one" {
		t.Fatal("schedule persistence", err)
	}
	if err = d.NotModified(ctx, "item", "", ""); err != nil {
		t.Fatal(err)
	}
	var observations int
	if err = d.Pool.QueryRow(ctx, `SELECT count(*) FROM observations WHERE status=304`).Scan(&observations); err != nil || observations != 1 {
		t.Fatal("304 history", err, observations)
	}
	if d.NotModified(ctx, "unknown", "", "") == nil {
		t.Fatal("unbacked 304 accepted")
	}
	if err = d.QueuePage(ctx, domain.DemocracyNow, 2, 2, []string{"item", "item", "other"}, true); err != nil {
		t.Fatal(err)
	}
	counts, err := d.Counts(ctx, domain.DemocracyNow)
	if err != nil || counts["pending"] != 2 {
		t.Fatal("discovery deduplication", err, counts)
	}
}
