package internetarchive

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBoundedCompleteDiscovery(t *testing.T) {
	mode := "complete"
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		page := r.URL.Query().Get("page")
		start := 0
		docs := []map[string]string{{"identifier": "a"}, {"identifier": "b"}}
		found := 3
		if page == "2" {
			start = 2
			docs = []map[string]string{{"identifier": "c"}}
		}
		switch mode {
		case "partial":
			if page == "2" {
				docs = nil
			}
		case "duplicate":
			if page == "2" {
				docs = []map[string]string{{"identifier": "b"}}
			}
		case "changed":
			if page == "2" {
				found = 4
			}
		case "over-limit":
			found = 100
		case "malformed":
			w.Write([]byte(`{"response":`))
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"response": map[string]any{"numFound": found, "start": start, "docs": docs}})
	}))
	defer server.Close()
	client := &Client{Base: server.URL, HTTP: server.Client()}
	result, err := client.Discover(context.Background(), "collection:zines", 10, 2)
	if err != nil || len(result.Items) != 3 || len(result.Pages) != 2 {
		t.Fatal(result, err)
	}
	for _, test := range []string{"partial", "duplicate", "changed", "over-limit", "malformed"} {
		mode = test
		if _, err = client.Discover(context.Background(), "zines", 10, 2); err == nil {
			t.Fatal("accepted", test)
		}
	}
	mode = "complete"
	client.LimitRequests(1)
	requests = 0
	if _, err = client.Discover(context.Background(), "zines", 10, 2); err == nil || requests != 1 {
		t.Fatal("ignored request budget", requests, err)
	}
}
