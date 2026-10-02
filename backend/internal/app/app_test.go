package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if recorder.Code != http.StatusOK || recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected response: %d %v", recorder.Code, recorder.Header())
	}
	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["service"] != "arkhiv" || body["status"] != "ok" || body["stage"] != "bootstrap" {
		t.Fatalf("unexpected health: %v", body)
	}
}

func TestRouteBoundaries(t *testing.T) {
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/api/missing", http.StatusNotFound},
		{"POST", "/api/health", http.StatusMethodNotAllowed},
	} {
		r := httptest.NewRecorder()
		Handler().ServeHTTP(r, httptest.NewRequest(tc.method, tc.path, nil))
		if r.Code != tc.status {
			t.Errorf("%s %s: got %d, want %d", tc.method, tc.path, r.Code, tc.status)
		}
	}
}
