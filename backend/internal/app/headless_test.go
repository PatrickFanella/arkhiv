package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHeadlessHTTPContractWithoutFrontend(t *testing.T) {
	// No web directory, frontend build or client package is needed. These public
	// discovery endpoints and collection guards work before catalog access.
	handler := (&Server{Collection: "headless-fixture"}).Handler()
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w
	}
	for _, path := range []string{"/", "/library", "/assets/frontend.js", "/api/unknown"} {
		if w := get(path); w.Code != http.StatusNotFound {
			t.Fatalf("headless route %s served frontend content: %d", path, w.Code)
		}
	}
	if w := get("/api/health"); w.Code != http.StatusOK {
		t.Fatalf("health unavailable without a frontend: %d", w.Code)
	}
	var version struct {
		API         string            `json:"api_version"`
		Collections []string          `json:"collections"`
		Modules     map[string]string `json:"modules"`
	}
	w := get("/api/version")
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &version) != nil || version.API != "1" {
		t.Fatalf("plain HTTP capability discovery failed: %s", w.Body.String())
	}
	for _, module := range []string{"catalog", "print", "audio", "video"} {
		if version.Modules[module] != "1" {
			t.Fatalf("missing independent media contract: %s", module)
		}
	}
	if len(version.Collections) != 1 || version.Collections[0] != "headless-fixture" {
		t.Fatalf("headless scope differs: %+v", version.Collections)
	}
	if w := get("/api/catalog/v1/items?collection=foreign"); w.Code != http.StatusNotFound {
		t.Fatalf("headless request escaped collection policy: %d", w.Code)
	}
}
