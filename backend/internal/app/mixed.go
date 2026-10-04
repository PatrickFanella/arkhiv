package app

import (
	"net/http"
	"strconv"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
)

func (s *Server) mixedCatalog(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := 1
	if q.Has("page") {
		var err error
		page, err = strconv.Atoi(q.Get("page"))
		if err != nil || page < 1 || page > 100000 {
			write(w, 400, map[string]string{"error": "invalid page"})
			return
		}
	}
	media := domain.Media(q.Get("media"))
	if (media != "" && !media.Valid()) || len(q.Get("q")) > 1000 {
		write(w, 400, map[string]string{"error": "invalid catalog filter"})
		return
	}
	var scope []string
	if s.Collection != "" {
		scope = []string{s.Collection}
	}
	result, err := s.DB.Catalog(r.Context(), catalog.CatalogQuery{Scope: scope, Collection: q.Get("collection"), Media: media, Search: q.Get("q"), ID: r.PathValue("id"), Page: page})
	if err != nil {
		write(w, 503, map[string]string{"error": "catalog unavailable"})
		return
	}
	if r.PathValue("id") != "" {
		if len(result.Items) != 1 {
			write(w, 404, map[string]string{"error": "catalog item unavailable"})
			return
		}
		write(w, 200, result.Items[0])
		return
	}
	write(w, 200, result)
}
