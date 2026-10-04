package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"git.subcult.tv/subculture-collective/vhness/internal/preservation"
	"github.com/jackc/pgx/v5"
)

type Server struct {
	DB                *catalog.DB
	Clock             func() time.Time
	Media             string
	Derivatives       string
	Web               string
	Collection        string
	Collections       domain.CollectionScope
	failures          sync.Mutex
	resolutionFailure *ResolutionFailure
}
type ResolutionFailure struct {
	ID     string    `json:"episode_id"`
	Mode   string    `json:"mode"`
	Reason string    `json:"reason"`
	Time   time.Time `json:"at"`
	Count  int64     `json:"count_since_start"`
}

func (s *Server) failure(r *http.Request, reason string) {
	s.failures.Lock()
	defer s.failures.Unlock()
	count := int64(1)
	if s.resolutionFailure != nil {
		count += s.resolutionFailure.Count
	}
	s.resolutionFailure = &ResolutionFailure{r.PathValue("id"), r.URL.Query().Get("mode"), reason, time.Now().UTC(), count}
}
func (s *Server) failureStatus() *ResolutionFailure {
	s.failures.Lock()
	defer s.failures.Unlock()
	if s.resolutionFailure == nil {
		return nil
	}
	copy := *s.resolutionFailure
	return &copy
}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) { write(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		write(w, 200, map[string]any{"api_version": "1", "collection": s.Collection, "collections": s.collectionScope()})
	})
	mux.HandleFunc("GET /api/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := s.DB.Pool.Ping(r.Context()); err != nil {
			write(w, 503, map[string]string{"error": "database unavailable"})
			return
		}
		write(w, 200, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/episodes", s.list)
	mux.HandleFunc("GET /api/catalog/v1/items", s.mixedCatalog)
	mux.HandleFunc("GET /api/catalog/v1/collections", s.mixedCollections)
	mux.HandleFunc("GET /api/catalog/v1/items/{id}", s.mixedCatalog)
	mux.HandleFunc("GET /api/catalog/v1/items/{id}/representations/{representation}/resolve", s.resolveRepresentation)
	mux.HandleFunc("GET /api/catalog/v1/items/{id}/representations/{representation}/media", s.representationMedia)
	mux.HandleFunc("GET /api/catalog/v1/items/{id}/representations/{representation}/print", s.printStatus)
	mux.HandleFunc("GET /api/catalog/v1/items/{id}/representations/{representation}/search", s.printSearch)
	mux.HandleFunc("GET /api/catalog/v1/items/{id}/representations/{representation}/cover", s.printCover)
	mux.HandleFunc("GET /api/catalog/v1/items/{id}/representations/{representation}/audio/resolve", s.audioResolve)
	mux.HandleFunc("GET /api/catalog/v1/items/{id}/representations/{representation}/audio/media", s.audioMedia)
	mux.HandleFunc("GET /api/audio/channel", s.audioChannel)
	mux.HandleFunc("GET /api/collections", func(w http.ResponseWriter, r *http.Request) {
		collections, err := s.DB.Collections(r.Context())
		if err != nil {
			write(w, 503, map[string]string{"error": "collections unavailable"})
			return
		}
		if s.collectionScope() != nil {
			selected := []catalog.CollectionSummary{}
			for _, collection := range collections {
				if s.allowsCollection(collection.ID) {
					selected = append(selected, collection)
				}
			}
			collections = selected
		}
		write(w, 200, collections)
	})
	mux.HandleFunc("GET /api/episodes/{id}", s.episode)
	mux.HandleFunc("GET /api/playback/{id}", s.playback)
	mux.HandleFunc("GET /api/channel", s.channel)
	mux.HandleFunc("GET /api/media/{id}", s.media)
	mux.HandleFunc("GET /api/status", s.status)
	if s.Web != "" {
		files := http.FileServer(http.Dir(s.Web))
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				write(w, 404, map[string]string{"error": "unknown API route"})
				return
			}
			if r.URL.Path != "/" {
				if _, err := os.Stat(filepath.Join(s.Web, filepath.Clean(r.URL.Path))); err != nil {
					http.ServeFile(w, r, filepath.Join(s.Web, "index.html"))
					return
				}
			}
			files.ServeHTTP(w, r)
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if selected := r.URL.Query().Get("collection"); selected != "" && !s.allowsCollection(selected) {
			write(w, 404, map[string]string{"error": "collection unavailable"})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			// A connected but unresponsive database must not hold readiness or
			// the single in-flight channel poll indefinitely. Local file serving
			// keeps its normal streaming lifetime; only DB work uses this context.
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()
			r = r.WithContext(ctx)
		}
		mux.ServeHTTP(w, r)
	})
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	entries, err := s.DB.Summaries(r.Context())
	if err != nil {
		write(w, 503, map[string]string{"error": "catalog unavailable"})
		return
	}
	list := []domain.Episode{}
	query := strings.ToLower(r.URL.Query().Get("q"))
	year := r.URL.Query().Get("year")
	for _, e := range entries {
		if !s.allowsRequestCollection(r, e.Episode.CollectionID) {
			continue
		}
		if selected := r.URL.Query().Get("collection"); selected != "" && e.Episode.CollectionID != selected {
			continue
		}
		if !e.Decision.Published {
			continue
		}
		ep := e.Episode
		if query != "" && !strings.Contains(strings.ToLower(ep.Title+" "+ep.Description), query) {
			continue
		}
		if year != "" && !strings.HasPrefix(ep.Date, year) {
			continue
		}
		ep.Files = nil
		list = append(list, ep)
	}
	sort.Slice(list, func(i, j int) bool { return domain.BrowseLess(list[i], list[j]) })
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	total := len(list)
	start := (page - 1) * 24
	if start > total || start < 0 {
		start = total
	}
	end := start + 24
	if end > total {
		end = total
	}
	write(w, 200, map[string]any{"episodes": list[start:end], "total": total, "page": page})
}
func (s *Server) episode(w http.ResponseWriter, r *http.Request) {
	e, err := s.DB.Entry(r.Context(), r.PathValue("id"))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		write(w, 503, map[string]string{"error": "catalog unavailable"})
		return
	}
	if err != nil || !e.Decision.Published || !s.allowsRequestCollection(r, e.Episode.CollectionID) {
		write(w, 404, map[string]string{"error": "episode unavailable"})
		return
	}
	write(w, 200, e)
}
func (s *Server) playback(w http.ResponseWriter, r *http.Request) {
	e, err := s.DB.Entry(r.Context(), r.PathValue("id"))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		write(w, 503, map[string]string{"error": "catalog unavailable"})
		return
	}
	if err != nil || !s.allowsRequestCollection(r, e.Episode.CollectionID) {
		s.failure(r, "episode unavailable")
		write(w, 404, map[string]string{"error": "episode unavailable"})
		return
	}
	p, err := domain.Resolve(e, r.URL.Query().Get("mode") == "channel")
	if err != nil {
		s.failure(r, err.Error())
		write(w, 404, map[string]string{"error": err.Error()})
		return
	}
	write(w, 200, p)
}
func (s *Server) channel(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	if s.Clock != nil {
		now = s.Clock()
	}
	schedule, err := s.DB.Schedule(r.Context())
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			write(w, 503, map[string]string{"error": "schedule unavailable"})
			return
		}
		write(w, 200, domain.Now{ServerTime: now.UTC(), State: "unavailable", Reason: "channel has no schedule"})
		return
	}
	within, err := s.DB.ScheduleWithinScope(r.Context(), schedule, s.channelScope(r))
	if err != nil {
		write(w, 503, map[string]string{"error": "catalog unavailable"})
		return
	}
	if !within {
		write(w, 200, domain.Now{ServerTime: now.UTC(), State: "unavailable", Reason: "schedule contains a different collection"})
		return
	}
	entries, err := s.DB.CurrentEntries(r.Context(), schedule, now)
	if err != nil {
		write(w, 503, map[string]string{"error": "catalog unavailable"})
		return
	}
	state := schedule.At(now, entries)
	if state.Episode != nil {
		// Polling viewers need identity and attribution, not the parent item's
		// entire file inventory. Complete inventories remain in the catalog.
		state.Episode.Files = nil
	}
	write(w, 200, state)
}
func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("collection")
	if id == "" {
		id = s.defaultCollection()
	}
	if !s.allowsCollection(id) {
		write(w, 404, map[string]string{"error": "select an available collection"})
		return
	}
	state, err := s.DB.State(r.Context(), id)
	if err != nil {
		write(w, 503, map[string]string{"error": "worker status unavailable"})
		return
	}
	counts, err := s.DB.Counts(r.Context(), id)
	if err != nil {
		write(w, 503, map[string]string{"error": "queue status unavailable"})
		return
	}
	inventory, err := s.DB.Inventory(r.Context(), id)
	if err != nil {
		write(w, 503, map[string]string{"error": "inventory status unavailable"})
		return
	}
	collection, err := s.DB.GetCollection(r.Context(), id)
	if err != nil {
		write(w, 503, map[string]string{"error": "collection status unavailable"})
		return
	}
	now := time.Now().UTC()
	if s.Clock != nil {
		now = s.Clock().UTC()
	}
	channel := map[string]any{"state": "unavailable"}
	if schedule, err := s.DB.Schedule(r.Context()); err == nil {
		within, scopeErr := s.DB.ScheduleWithinScope(r.Context(), schedule, domain.CollectionScope{id})
		entries, err := s.DB.CurrentEntries(r.Context(), schedule, now)
		if err == nil && scopeErr == nil && within {
			state := schedule.At(now, entries)
			gaps := 0
			for i := 1; i < len(schedule.Programs); i++ {
				prior := schedule.Programs[i-1]
				if schedule.Programs[i].Start.After(prior.Start.Add(time.Duration(prior.Duration * float64(time.Second)))) {
					gaps++
				}
			}
			last := schedule.Programs[len(schedule.Programs)-1]
			channel = map[string]any{"state": state.State, "reason": state.Reason, "revision": state.Revision, "horizon": last.Start.Add(time.Duration(last.Duration * float64(time.Second))), "gaps": gaps}
		}
	}
	write(w, 200, map[string]any{"worker": state, "items": counts, "inventory": inventory, "collection": collection, "channel": channel, "metadata_retention": "all snapshots, observations, decisions, and schedules retained; no automatic pruning", "metadata_growth_alert": inventory.DatabaseBytes > 1<<30, "client_playback_status": "unverified by server; consult browser qualification", "last_server_resolution_failure": s.failureStatus(), "video_storage": collection.Policy})
}

func (s *Server) media(w http.ResponseWriter, r *http.Request) {
	e, err := s.DB.Entry(r.Context(), r.PathValue("id"))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		write(w, 503, map[string]string{"error": "catalog unavailable"})
		return
	}
	if err != nil || e.Policy != domain.Local || !e.Watchable() || s.Media == "" || !s.allowsRequestCollection(r, e.Episode.CollectionID) {
		write(w, 404, map[string]string{"error": "episode unavailable"})
		return
	}
	selected, _ := e.Selected()
	f, err := preservation.Verify(s.Media, e.LocalPath, e.LocalSHA256, selected.Bytes)
	if err != nil {
		s.failure(r, "local original failed integrity check")
		write(w, 503, map[string]string{"error": "local original failed integrity check"})
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		write(w, 503, map[string]string{"error": "local original unavailable"})
		return
	}
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("ETag", `"sha256-`+e.LocalSHA256+`"`)
	http.ServeContent(w, r, selected.Name, info.ModTime(), f)
}
