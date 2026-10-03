package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/app"
	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"git.subcult.tv/subculture-collective/vhness/internal/preservation"
	printmodule "git.subcult.tv/subculture-collective/vhness/internal/print"
	"git.subcult.tv/subculture-collective/vhness/internal/source"
	"git.subcult.tv/subculture-collective/vhness/internal/worker"
	"github.com/jackc/pgx/v5"
)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func load(p string, v any) error {
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: vhness migrate|collection|review-budget|review-library|plan-replay|import|import-item|import-catalog|review-transport|index-print|fetch|inspect|acquire|audit|review|schedule|discover|refresh|report|seed|serve|healthcheck")
	}
	cmd := os.Args[1]
	if cmd == "healthcheck" {
		address := strings.Replace(env("API_ADDR", "127.0.0.1:8080"), "0.0.0.0:", "127.0.0.1:", 1)
		client := &http.Client{Timeout: 4 * time.Second}
		response, err := client.Get("http://" + address + "/api/ready")
		if err != nil {
			return fmt.Errorf("readiness request failed")
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("database not ready")
		}
		return nil
	}
	f := flag.NewFlagSet(cmd, flag.ContinueOnError)
	id := f.String("id", "", "source item identity")
	representation := f.String("representation", "", "catalog representation identity")
	scope := os.Getenv("VHNESS_COLLECTION")
	collection := f.String("collection", env("VHNESS_COLLECTION", domain.DemocracyNow), "collection identity")
	file := f.String("file", "", "JSON input")
	sourceItem := f.String("source-item", "", "explicit multi-file source movie item")
	episodeFile := f.String("episode-file", "", "exact file to materialize from a selected source item")
	synthetic := f.Bool("synthetic", false, "explicit generated local fixture metadata")
	fixtureOriginal := f.String("fixture-original", "", "generated fixture original, only for vhness-local-fixture")
	policy := f.String("policy", string(domain.Stream), "delivery policy")
	budget := f.Int64("budget", 0, "local byte budget")
	reserve := f.Int64("reserve", 0, "free-space reserve")
	expectedBudget := f.Int64("expected-budget", 0, "previous reviewed budget required for explicit increase")
	expectedReserve := f.Int64("expected-reserve", 0, "previous reviewed reserve required for explicit increase")
	notes := f.String("notes", "", "authorization notes for explicit budget review")
	after := f.String("after-id", "", "bounded library review cursor")
	days := f.Int("days", 7, "explicit replay horizon, 1 through 7 days")
	pages := f.Int("pages", 1, "bounded discovery pages (0–100)")
	items := f.Int("items", 20, "bounded metadata fetches (0–1000)")
	timeout := f.Duration("timeout", 10*time.Minute, "run deadline")
	if err := f.Parse(os.Args[2:]); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if scope != "" && (!domain.ValidID(scope) || *collection != scope) {
		return fmt.Errorf("collection outside configured instance scope")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	databaseURL := os.Getenv("DATABASE_URL")
	if path := os.Getenv("DATABASE_URL_FILE"); path != "" {
		secret, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("database connection file unavailable")
		}
		databaseURL = strings.TrimSpace(string(secret))
	}
	d, err := catalog.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer d.Pool.Close()
	if scope != "" && (cmd == "review" || cmd == "inspect") {
		entry, err := d.Entry(ctx, *id)
		if err != nil || entry.Episode.CollectionID != scope {
			return fmt.Errorf("episode outside configured instance scope")
		}
	}
	switch cmd {
	case "index-print":
		return printmodule.Index(ctx, d, os.Getenv("VHNESS_MEDIA"), os.Getenv("VHNESS_DERIVATIVES"), *id, *representation, *collection)
	case "import-catalog":
		var item domain.CatalogItem
		if err := load(*file, &item); err != nil {
			return err
		}
		if scope != "" {
			for _, c := range item.Collections {
				if c != scope {
					return fmt.Errorf("catalog mapping outside configured scope")
				}
			}
		}
		return d.InsertCatalog(ctx, item, *notes)
	case "review-transport":
		var input struct {
			Review       domain.TransportReview      `json:"review"`
			Preservation domain.PreservationApproval `json:"preservation"`
		}
		if err := load(*file, &input); err != nil {
			return err
		}
		return d.ReviewTransport(ctx, *id, *representation, *collection, input.Review, input.Preservation)
	case "migrate":
		return d.Migrate(ctx, env("VHNESS_MIGRATIONS", "../migrations"))
	case "collection":
		return d.Collection(ctx, domain.Collection{ID: *collection, Policy: domain.Policy(*policy), Budget: *budget, Reserve: *reserve, SourceItem: *sourceItem})
	case "review-budget":
		return d.ReviewBudget(ctx, *collection, *expectedBudget, *expectedReserve, *budget, *reserve, *notes)
	case "review-library":
		if strings.TrimSpace(*notes) == "" || *items < 1 || *items > 1000 {
			return fmt.Errorf("bounded library review needs authorization notes and 1 through 1000 items")
		}
		c, err := d.GetCollection(ctx, *collection)
		if err != nil || c.Policy != domain.Stream {
			return fmt.Errorf("library review requires an existing source-streaming collection")
		}
		rows, err := d.Pool.Query(ctx, `SELECT id FROM episodes WHERE collection_id=$1 AND id>$2 AND coalesce(decision->>'notes','')='' ORDER BY id LIMIT $3`, *collection, *after, *items)
		if err != nil {
			return err
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		selected := map[string]string{}
		skipped := map[string]string{}
		cursor := *after
		for _, id := range ids {
			cursor = id
			e, err := d.Entry(ctx, id)
			if err != nil {
				return err
			}
			if e.Unavailable {
				skipped[id] = "source unavailable"
				continue
			}
			files := []domain.File{}
			for _, file := range e.Episode.Files {
				if file.Candidate() {
					files = append(files, file)
				}
			}
			score := func(file domain.File) int {
				if file.Height == 480 {
					return 0
				}
				if file.Height > 0 && file.Height <= 1080 {
					return 1
				}
				return 2
			}
			sort.Slice(files, func(i, j int) bool {
				if score(files[i]) != score(files[j]) {
					return score(files[i]) < score(files[j])
				}
				if files[i].Bytes != files[j].Bytes {
					return files[i].Bytes < files[j].Bytes
				}
				return files[i].Name < files[j].Name
			})
			if len(files) == 0 {
				skipped[id] = "no eligible MP4 with a reported duration"
				continue
			}
			decision := domain.Decision{Published: true, Playback: true, Channel: true, File: files[0].Name, Notes: *notes + "; source snapshot " + e.SourceHash + "; private playback eligibility from source inventory, individual decoding unverified"}
			if err = d.Decide(ctx, id, decision); err != nil {
				return err
			}
			selected[id] = decision.File
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"reviewed": selected, "skipped": skipped, "after_id": cursor, "visited": len(ids)})
	case "plan-replay":
		if *days < 1 || *days > 7 {
			return fmt.Errorf("replay horizon must be 1 through 7 days")
		}
		rows, err := d.Pool.Query(ctx, `SELECT id FROM episodes WHERE collection_id=$1 AND decision->>'channel'='true' ORDER BY source->>'date' DESC,id LIMIT 500`, *collection)
		if err != nil {
			return err
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		eligible := []domain.Entry{}
		for _, id := range ids {
			entry, err := d.Entry(ctx, id)
			if err != nil {
				return err
			}
			if _, err = domain.Resolve(entry, true); err == nil {
				file, _ := entry.Selected()
				entry.Episode.Files = []domain.File{file}
				eligible = append(eligible, entry)
			}
		}
		if len(eligible) == 0 {
			return fmt.Errorf("no reviewed channel-eligible episodes")
		}
		sort.Slice(eligible, func(i, j int) bool {
			a, b := eligible[i].Episode, eligible[j].Episode
			if a.SourceFile != "" && b.SourceFile != "" {
				ai, _ := strconv.Atoi(strings.Fields(a.SourceFile)[0])
				bi, _ := strconv.Atoi(strings.Fields(b.SourceFile)[0])
				if ai != bi {
					return ai < bi
				}
			}
			if a.Date != b.Date {
				return a.Date > b.Date
			}
			return a.ID < b.ID
		})
		now := time.Now().UTC()
		horizon := now.Add(time.Duration(*days) * 24 * time.Hour)
		planned := domain.Schedule{Revision: "replay-" + now.Format("20060102T150405.000000000Z")}
		existing, scheduleErr := d.Schedule(ctx)
		if scheduleErr != nil && !errors.Is(scheduleErr, pgx.ErrNoRows) {
			return scheduleErr
		}
		if scheduleErr == nil {
			for _, program := range existing.Programs {
				entry, entryErr := d.Entry(ctx, program.EpisodeID)
				if entryErr != nil || entry.Episode.CollectionID != *collection {
					return fmt.Errorf("existing schedule outside configured collection; review its replacement explicitly")
				}
				if program.Start.Add(time.Duration(program.Duration * float64(time.Second))).After(now) {
					planned.Programs = append(planned.Programs, program)
				}
			}
		}
		start := now
		index := 0
		if len(planned.Programs) > 0 {
			last := planned.Programs[len(planned.Programs)-1]
			start = last.Start.Add(time.Duration(last.Duration * float64(time.Second)))
			for i, entry := range eligible {
				if entry.Episode.ID == last.EpisodeID {
					index = (i + 1) % len(eligible)
					break
				}
			}
		}
		for start.Before(horizon) {
			if len(planned.Programs) >= 1000 {
				return fmt.Errorf("replay exceeds bounded program count; reduce horizon")
			}
			entry := eligible[index%len(eligible)]
			file, _ := entry.Selected()
			planned.Programs = append(planned.Programs, domain.Program{EpisodeID: entry.Episode.ID, File: file.Name, Policy: entry.Policy, Start: start, Duration: file.Duration})
			start = start.Add(time.Duration(file.Duration * float64(time.Second)))
			index++
		}
		if err = d.SaveSchedule(ctx, planned); err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"revision": planned.Revision, "programs": len(planned.Programs), "horizon": start})
	case "import-item":
		c, err := d.GetCollection(ctx, *collection)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		e, err := source.ItemFile(c, *id, *episodeFile, raw)
		if err != nil {
			return err
		}
		if err = d.Snapshot(ctx, *id, raw, 200, "", "", nil); err != nil {
			return err
		}
		if err = d.ItemEpisode(ctx, e, domain.Digest(raw)); err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"episode_id": e.ID, "source_item": e.SourceItemID, "file": *episodeFile})
	case "import":
		raw, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		e, err := source.Normalize(*id, *collection, raw)
		if err != nil {
			return err
		}
		if *synthetic {
			if *collection != "vhness-local-fixture" {
				return fmt.Errorf("synthetic import requires vhness-local-fixture")
			}
			e.Synthetic = true
		}
		return d.Snapshot(ctx, *id, raw, 200, "", "", &e)
	case "fetch":
		response, err := source.New().Metadata(ctx, *id, "", "")
		if err != nil {
			return err
		}
		e, err := source.Normalize(*id, *collection, response.Raw)
		if err != nil {
			return err
		}
		return d.Snapshot(ctx, *id, response.Raw, response.Status, response.ETag, response.Modified, &e)
	case "audit":
		c, err := d.GetCollection(ctx, *collection)
		if err != nil {
			return err
		}
		if c.Policy != domain.Local {
			return fmt.Errorf("integrity audit requires a local collection")
		}
		entries, err := d.LocalEntries(ctx, *collection)
		if err != nil {
			return err
		}
		failures := 0
		results := map[string]string{}
		for id, e := range entries {
			raw, checkErr := d.PreservationReceipt(ctx, id)
			var receipt preservation.Receipt
			if checkErr == nil {
				checkErr = json.Unmarshal(raw, &receipt)
			}
			if checkErr == nil && (receipt.Path != e.LocalPath || receipt.SHA256 != e.LocalSHA256 || receipt.Manifest.Snapshot != e.LocalSnapshot || receipt.Manifest.File != e.LocalFile || receipt.Manifest.ID != id) {
				checkErr = fmt.Errorf("preservation lineage mismatch")
			}
			if checkErr == nil {
				f, verifyErr := preservation.Verify(os.Getenv("VHNESS_MEDIA"), e.LocalPath, e.LocalSHA256, receipt.Manifest.Bytes)
				checkErr = verifyErr
				if f != nil {
					f.Close()
				}
			}
			if checkErr != nil {
				failures++
				results[id] = checkErr.Error()
			} else {
				results[id] = "verified"
			}
		}
		if err = json.NewEncoder(os.Stdout).Encode(map[string]any{"originals": len(entries), "failures": failures, "results": results}); err != nil {
			return err
		}
		if failures > 0 {
			return fmt.Errorf("integrity audit failed for %d originals", failures)
		}
		return nil
	case "acquire":
		if *timeout <= 0 || *timeout > 30*time.Minute {
			return fmt.Errorf("acquisition deadline must be within 30 minutes")
		}
		var manifest preservation.Manifest
		if err = load(*file, &manifest); err != nil {
			return err
		}
		if scope != "" && manifest.Collection != scope {
			return fmt.Errorf("manifest outside configured instance scope")
		}
		c, err := d.GetCollection(ctx, manifest.Collection)
		if err != nil {
			return err
		}
		e, err := d.Entry(ctx, manifest.ID)
		if err != nil {
			return err
		}
		acquisitionCtx, cancel := context.WithTimeout(ctx, *timeout)
		defer cancel()
		mediaRoot := os.Getenv("VHNESS_MEDIA")
		if !filepath.IsAbs(mediaRoot) {
			return fmt.Errorf("absolute VHNESS_MEDIA required")
		}
		collectionRoot := filepath.Join(mediaRoot, c.ID)
		acquirer := preservation.New(collectionRoot)
		if *fixtureOriginal != "" {
			acquirer = preservation.GeneratedFixture(collectionRoot, *fixtureOriginal)
		}
		receipt, err := acquirer.Acquire(acquisitionCtx, c, e, manifest)
		if err != nil {
			return err
		}
		receipt.Path = filepath.Join(c.ID, receipt.Path)
		raw, err := json.Marshal(receipt)
		if err != nil {
			return err
		}
		if err = d.BindLocal(ctx, manifest.ID, manifest.Snapshot, manifest.File, receipt.Path, receipt.SHA256, raw); err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(receipt)
	case "review":
		var decision domain.Decision
		if err := load(*file, &decision); err != nil {
			return err
		}
		return d.Decide(ctx, *id, decision)
	case "schedule":
		var s domain.Schedule
		if err := load(*file, &s); err != nil {
			return err
		}
		if scope != "" {
			for _, program := range s.Programs {
				entry, err := d.Entry(ctx, program.EpisodeID)
				if err != nil || entry.Episode.CollectionID != scope {
					return fmt.Errorf("scheduled episode outside configured instance scope")
				}
			}
		}
		return d.SaveSchedule(ctx, s)
	case "discover", "refresh":
		if c, collectionErr := d.GetCollection(ctx, *collection); collectionErr != nil {
			return collectionErr
		} else if c.SourceItem != "" {
			return fmt.Errorf("file-based sources require explicit import-item selection")
		}
		if _, err = d.GetCollection(ctx, *collection); err != nil {
			return err
		}
		if cmd == "refresh" {
			if err = d.Reconcile(ctx, *collection); err != nil {
				return err
			}
			if err = d.RefreshKnown(ctx, *collection, *items); err != nil {
				return err
			}
		}
		client := source.New()
		w := worker.Worker{DB: d, Source: client}
		result, err := w.Run(ctx, *collection, *pages, *items, *timeout)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(result)
	case "report":
		entries, err := d.Entries(ctx)
		if err != nil {
			return err
		}
		state, err := d.State(ctx, *collection)
		if err != nil {
			return err
		}
		counts, err := d.Counts(ctx, *collection)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"episodes": len(entries), "worker": state, "counts": counts})
	case "inspect":
		entries, err := d.Entries(ctx)
		if err != nil {
			return err
		}
		e, ok := entries[*id]
		if !ok {
			return fmt.Errorf("episode not found")
		}
		return json.NewEncoder(os.Stdout).Encode(e)
	case "seed":
		if scope != "" && scope != domain.DemocracyNow {
			return fmt.Errorf("synthetic seed requires Democracy Now! instance scope")
		}
		if err = d.Collection(ctx, domain.Collection{ID: domain.DemocracyNow, Policy: domain.Stream}); err != nil {
			return err
		}
		for i := 1; i <= 3; i++ {
			item := fmt.Sprintf("vhness-synthetic-%d", i)
			payload := map[string]any{"metadata": map[string]any{"identifier": item, "mediatype": "movies", "collection": []string{domain.DemocracyNow}, "title": fmt.Sprintf("Synthetic episode %02d — development fixture", i), "date": fmt.Sprintf("2026-10-%02d", i), "description": "Clearly labeled synthetic metadata. No video is stored or acquired."}, "files": []map[string]any{{"name": "synthetic.mp4", "format": "h.264", "size": "1", "length": "60"}}}
			raw, _ := json.Marshal(payload)
			e, err := source.Normalize(item, domain.DemocracyNow, raw)
			if err != nil {
				return err
			}
			e.Synthetic = true
			if err = d.Snapshot(ctx, item, raw, 200, "", "", &e); err != nil {
				return err
			}
			if err = d.Decide(ctx, item, domain.Decision{Published: true, Playback: true, Channel: true, File: "synthetic.mp4", Notes: "Synthetic metadata fixture; no real media"}); err != nil {
				return err
			}
		}
		return nil
	case "serve":
		server := &http.Server{Addr: env("API_ADDR", "127.0.0.1:8080"), Handler: (&app.Server{DB: d, Web: os.Getenv("VHNESS_WEB"), Media: os.Getenv("VHNESS_MEDIA"), Derivatives: os.Getenv("VHNESS_DERIVATIVES"), Collection: scope}).Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
		go func() {
			<-ctx.Done()
			shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdown)
		}()
		log.Printf("VHNESS API listening on %s; data root %s", server.Addr, filepath.Clean(env("VHNESS_DATA", "../var")))
		err = server.ListenAndServe()
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}
func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
