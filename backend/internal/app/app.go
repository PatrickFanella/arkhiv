package app

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
)

type Server struct {
	DB    *catalog.DB
	Clock func() time.Time
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
	mux.HandleFunc("GET /api/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := s.DB.Pool.Ping(r.Context()); err != nil {
			write(w, 503, map[string]string{"error": "database unavailable"})
			return
		}
		write(w, 200, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/episodes", s.list)
	mux.HandleFunc("GET /api/episodes/{id}", s.episode)
	mux.HandleFunc("GET /api/playback/{id}", s.playback)
	mux.HandleFunc("GET /api/channel", s.channel)
	mux.HandleFunc("GET /api/status", s.status)
	return mux
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	entries, err := s.DB.Entries(r.Context())
	if err != nil {
		write(w, 503, map[string]string{"error": "catalog unavailable"})
		return
	}
	list := []domain.Episode{}
	query := strings.ToLower(r.URL.Query().Get("q"))
	year := r.URL.Query().Get("year")
	for _, e := range entries {
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
	sort.Slice(list, func(i, j int) bool {
		if list[i].Date == list[j].Date {
			return list[i].ID < list[j].ID
		}
		return list[i].Date > list[j].Date
	})
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
	entries, err := s.DB.Entries(r.Context())
	if err != nil {
		write(w, 503, map[string]string{"error": "catalog unavailable"})
		return
	}
	e, ok := entries[r.PathValue("id")]
	if !ok || !e.Decision.Published {
		write(w, 404, map[string]string{"error": "episode unavailable"})
		return
	}
	write(w, 200, e)
}
func (s *Server) playback(w http.ResponseWriter, r *http.Request) {
	entries, err := s.DB.Entries(r.Context())
	if err != nil {
		write(w, 503, map[string]string{"error": "catalog unavailable"})
		return
	}
	e, ok := entries[r.PathValue("id")]
	if !ok {
		write(w, 404, map[string]string{"error": "episode unavailable"})
		return
	}
	p, err := domain.Resolve(e, r.URL.Query().Get("mode") == "channel")
	if err != nil {
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
		write(w, 200, domain.Now{ServerTime: now.UTC(), State: "unavailable", Reason: "channel has no schedule"})
		return
	}
	entries, err := s.DB.Entries(r.Context())
	if err != nil {
		write(w, 503, map[string]string{"error": "catalog unavailable"})
		return
	}
	write(w, 200, schedule.At(now, entries))
}
func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("collection")
	if id == "" {
		id = domain.DemocracyNow
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
	write(w, 200, map[string]any{"worker": state, "items": counts, "video_storage": "disabled for source_stream", "playback_proof": "server status does not prove client playback"})
}
