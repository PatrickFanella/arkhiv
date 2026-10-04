package app

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"git.subcult.tv/subculture-collective/vhness/internal/preservation"
)

func (s *Server) authorizedTransport(r *http.Request) (catalog.TransportEntry, domain.Action, string, error) {
	collection := r.URL.Query().Get("collection")
	if collection == "" {
		collection = s.configuredCollection()
	}
	if !s.allowsCollection(collection) {
		return catalog.TransportEntry{}, "", "", fmt.Errorf("collection unavailable")
	}
	e, err := s.DB.Transport(r.Context(), r.PathValue("id"), r.PathValue("representation"), collection)
	if err != nil {
		return e, "", "", err
	}
	action := domain.Action(r.URL.Query().Get("action"))
	if action == "" {
		if e.Representation.Media == domain.PrintMedia {
			action = domain.ReadAction
		} else {
			action = domain.PlayAction
		}
	}
	if action != domain.ReadAction && action != domain.PlayAction && action != domain.DownloadAction {
		return e, action, "", fmt.Errorf("unsupported viewer action")
	}
	file := r.URL.Query().Get("file")
	if file == "" {
		file = e.Representation.File
	}
	if representationMIME(e.Representation.Media, file) == "" {
		return e, action, file, fmt.Errorf("unsupported representation file")
	}
	err = domain.AuthorizeTransport(e.Collection, e.Representation, e.Raw, e.Review, e.Preservation, action, file)
	return e, action, file, err
}

func (s *Server) resolveRepresentation(w http.ResponseWriter, r *http.Request) {
	e, action, file, err := s.authorizedTransport(r)
	if err != nil {
		write(w, 404, map[string]string{"error": "representation unavailable"})
		return
	}
	source := "https://archive.org/details/" + url.PathEscape(e.Representation.SourceItemID)
	asset := "https://archive.org/download/" + url.PathEscape(e.Representation.SourceItemID) + "/" + url.PathEscape(file)
	if e.Collection.Policy == domain.Local {
		q := url.Values{"collection": {e.Collection.ID}, "action": {string(action)}, "file": {file}}
		asset = "/api/catalog/v1/items/" + url.PathEscape(r.PathValue("id")) + "/representations/" + url.PathEscape(r.PathValue("representation")) + "/media?" + q.Encode()
	}
	write(w, 200, map[string]any{"media": e.Representation.Media, "policy": e.Collection.Policy, "url": asset, "source_url": source, "file": file, "snapshot_sha256": e.Representation.Snapshot})
}

func representationMIME(media domain.Media, file string) string {
	ext := strings.ToLower(filepath.Ext(file))
	if media == domain.PrintMedia && ext == ".pdf" {
		return "application/pdf"
	}
	if media == domain.VideoMedia && ext == ".mp4" {
		return "video/mp4"
	}
	if media == domain.AudioMedia {
		switch ext {
		case ".mp3":
			return "audio/mpeg"
		case ".flac":
			return "audio/flac"
		case ".ogg":
			return "audio/ogg"
		case ".wav":
			return "audio/wav"
		case ".m4a":
			return "audio/mp4"
		}
	}
	return ""
}

func (s *Server) representationMedia(w http.ResponseWriter, r *http.Request) {
	e, action, file, err := s.authorizedTransport(r)
	if err != nil || e.Collection.Policy != domain.Local || s.Media == "" {
		write(w, 404, map[string]string{"error": "representation unavailable"})
		return
	}
	contentType := representationMIME(e.Representation.Media, file)
	if contentType == "" {
		write(w, 404, map[string]string{"error": "unsupported representation file"})
		return
	}
	var original domain.LocalOriginal
	for _, candidate := range e.Preservation.Manifest {
		if candidate.File == file {
			original = candidate
			break
		}
	}
	f, err := preservation.Verify(s.Media, original.Path, original.SHA256, original.Bytes)
	if err != nil {
		write(w, 503, map[string]string{"error": "local original failed integrity check"})
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		write(w, 503, map[string]string{"error": "local original unavailable"})
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("ETag", `"sha256-`+original.SHA256+`"`)
	disposition := "inline"
	if action == domain.DownloadAction {
		disposition = "attachment"
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": file}))
	http.ServeContent(w, r, file, info.ModTime(), f)
}
