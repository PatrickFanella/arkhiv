package app

import (
	"net/http"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
)

func (s *Server) collectionScope() domain.CollectionScope {
	if s.Collection != "" {
		return domain.CollectionScope{s.Collection}
	}
	if s.DB != nil && s.DB.PublicOnly && s.Collections == nil {
		return domain.CollectionScope{}
	}
	return s.Collections
}
func (s *Server) allowsCollection(id string) bool { return s.collectionScope().Allows(id) }
func (s *Server) defaultCollection() string       { return s.collectionScope().Default() }
func (s *Server) configuredCollection() string {
	if s.collectionScope() == nil {
		return ""
	}
	return s.collectionScope().Default()
}
func (s *Server) allowsRequestCollection(r *http.Request, id string) bool {
	selected := r.URL.Query().Get("collection")
	return s.allowsCollection(id) && (selected == "" || selected == id)
}
func (s *Server) channelScope(r *http.Request) domain.CollectionScope {
	if selected := r.URL.Query().Get("collection"); selected != "" {
		return domain.CollectionScope{selected}
	}
	return s.collectionScope()
}
