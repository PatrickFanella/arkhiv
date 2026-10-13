package app

import (
	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"net/http/httptest"
	"testing"
)

func TestSourceCatalogIsScopedAndPrivate(t *testing.T) {
	for _, path := range []string{"/api/source-catalog/v1/items", "/api/source-catalog/v1/items/source", "/api/source-catalog/v1/status", "/api/source-catalog/v1/facets"} {
		w := httptest.NewRecorder()
		(&Server{DB: &catalog.DB{PublicOnly: true}, Collection: "etree"}).Handler().ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 404 {
			t.Fatalf("unreviewed public source inventory: %s %d", path, w.Code)
		}
		w = httptest.NewRecorder()
		(&Server{DB: &catalog.DB{}, Collection: "etree"}).Handler().ServeHTTP(w, httptest.NewRequest("GET", path+"?collection=other", nil))
		if w.Code != 404 {
			t.Fatalf("foreign source inventory: %s %d", path, w.Code)
		}
	}
}
