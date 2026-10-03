package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/app"
	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"git.subcult.tv/subculture-collective/vhness/internal/source"
	"git.subcult.tv/subculture-collective/vhness/internal/worker"
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
		return fmt.Errorf("usage: vhness migrate|collection|import|fetch|inspect|review|schedule|discover|refresh|report|seed|serve")
	}
	cmd := os.Args[1]
	f := flag.NewFlagSet(cmd, flag.ContinueOnError)
	id := f.String("id", "", "source item identity")
	collection := f.String("collection", domain.DemocracyNow, "collection identity")
	file := f.String("file", "", "JSON input")
	policy := f.String("policy", string(domain.Stream), "delivery policy")
	budget := f.Int64("budget", 0, "local byte budget")
	reserve := f.Int64("reserve", 0, "free-space reserve")
	pages := f.Int("pages", 1, "bounded discovery pages (0–100)")
	items := f.Int("items", 20, "bounded metadata fetches (0–1000)")
	timeout := f.Duration("timeout", 10*time.Minute, "run deadline")
	if err := f.Parse(os.Args[2:]); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	d, err := catalog.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer d.Pool.Close()
	switch cmd {
	case "migrate":
		return d.Migrate(ctx, env("VHNESS_MIGRATIONS", "../migrations"))
	case "collection":
		return d.Collection(ctx, domain.Collection{ID: *collection, Policy: domain.Policy(*policy), Budget: *budget, Reserve: *reserve})
	case "import":
		raw, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		e, err := source.Normalize(*id, *collection, raw)
		if err != nil {
			return err
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
		return d.SaveSchedule(ctx, s)
	case "discover", "refresh":
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
		server := &http.Server{Addr: env("API_ADDR", "127.0.0.1:8080"), Handler: (&app.Server{DB: d, Web: os.Getenv("VHNESS_WEB")}).Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
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
