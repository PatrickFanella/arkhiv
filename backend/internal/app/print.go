package app

import (
	"fmt"
	"net/http"
	"strings"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"git.subcult.tv/subculture-collective/vhness/internal/preservation"
)

func (s *Server) authorizedPrint(r *http.Request) (domain.PrintIndex, string, error) {
	e, action, file, err := s.authorizedTransport(r)
	if err != nil || action != domain.ReadAction || e.Representation.Media != domain.PrintMedia || e.Collection.Policy != domain.Local || s.Media == "" {
		return domain.PrintIndex{}, file, fmt.Errorf("print unavailable")
	}
	index, err := s.DB.PrintIndex(r.Context(), r.PathValue("id"), r.PathValue("representation"), file)
	if err != nil {
		return index, file, err
	}
	if index.TextStatus == "document-text" {
		if index.TextFile == "" || domain.AuthorizeTransport(e.Collection, e.Representation, e.Raw, e.Review, e.Preservation, domain.ReadAction, index.TextFile) != nil {
			return index, file, fmt.Errorf("OCR companion reading unavailable")
		}
		verified := false
		for _, asset := range e.Preservation.Manifest {
			if asset.File == index.TextFile && asset.SHA256 == index.TextSHA256 {
				f, err := preservation.Verify(s.Media, asset.Path, asset.SHA256, asset.Bytes)
				if err != nil {
					return index, file, err
				}
				f.Close()
				verified = true
			}
		}
		if !verified {
			return index, file, fmt.Errorf("stale OCR index")
		}
	}
	for _, asset := range e.Preservation.Manifest {
		if asset.File == file {
			if index.Snapshot != e.Representation.Snapshot || index.OriginalSHA256 != asset.SHA256 {
				return index, file, fmt.Errorf("stale print index")
			}
			f, err := preservation.Verify(s.Media, asset.Path, asset.SHA256, asset.Bytes)
			if err != nil {
				return index, file, err
			}
			f.Close()
			return index, file, nil
		}
	}
	return index, file, fmt.Errorf("print outside manifest")
}

func (s *Server) printStatus(w http.ResponseWriter, r *http.Request) {
	index, _, err := s.authorizedPrint(r)
	if err != nil {
		write(w, 404, map[string]string{"error": "print index unavailable"})
		return
	}
	write(w, 200, index)
}
func (s *Server) printSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if len(query) > 200 || strings.TrimSpace(query) == "" {
		write(w, 400, map[string]string{"error": "document search requires a bounded query"})
		return
	}
	index, file, err := s.authorizedPrint(r)
	if err != nil || (index.TextStatus != "page-text" && index.TextStatus != "document-text") {
		write(w, 404, map[string]string{"error": "document text unavailable"})
		return
	}
	hits, err := s.DB.PrintSearch(r.Context(), r.PathValue("id"), r.PathValue("representation"), file, query)
	if err != nil {
		write(w, 503, map[string]string{"error": "document search unavailable"})
		return
	}
	write(w, 200, hits)
}
func (s *Server) printCover(w http.ResponseWriter, r *http.Request) {
	index, _, err := s.authorizedPrint(r)
	if err != nil || s.Derivatives == "" || index.CoverPath == "" || index.CoverBytes <= 0 {
		write(w, 404, map[string]string{"error": "document cover unavailable"})
		return
	}
	f, err := preservation.Verify(s.Derivatives, index.CoverPath, index.CoverSHA256, index.CoverBytes)
	if err != nil {
		write(w, 404, map[string]string{"error": "document cover unavailable"})
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		write(w, 404, map[string]string{"error": "document cover unavailable"})
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, "cover.png", info.ModTime(), f)
}
