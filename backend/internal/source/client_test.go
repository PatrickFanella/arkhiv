package source

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestMetadataBackoffAndPayloadLimits(t *testing.T) {
	now := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metadata/limited":
			w.Header().Set("Retry-After", "86400")
			w.WriteHeader(429)
		case "/metadata/oversized":
			w.Write([]byte(strings.Repeat(" ", MaxResponse+1)))
		case "/metadata/unchanged":
			if r.Header.Get("If-None-Match") != "saved" {
				t.Error("validator missing")
			}
			w.WriteHeader(304)
		default:
			t.Errorf("unexpected media request %s", r.URL.Path)
			w.WriteHeader(400)
		}
	}))
	defer server.Close()
	c := &Client{Base: server.URL, HTTP: server.Client(), Clock: func() time.Time { return now }}
	_, err := c.Metadata(context.Background(), "limited", "", "")
	var b *Backoff
	if !errors.As(err, &b) || !b.Until.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("backoff truncated: %v", err)
	}
	if _, err = c.Metadata(context.Background(), "oversized", "", ""); err == nil {
		t.Fatal("oversized response accepted")
	}
	r, err := c.Metadata(context.Background(), "unchanged", "saved", "")
	if err != nil || r.Status != 304 {
		t.Fatalf("conditional %v %v", r, err)
	}
	if _, err = c.get(context.Background(), "/download/item/video.mp4", "", ""); err == nil {
		t.Fatal("metadata client fetched video")
	}
}
func TestRedirectHostAndPathPolicy(t *testing.T) {
	for _, s := range []string{"http://archive.org/metadata/item", "https://archive.org.evil.test/metadata/item", "https://user:pass@archive.org/metadata/item", "https://archive.org:444/metadata/item"} {
		u, _ := url.Parse(s)
		if Allowed(u) {
			t.Fatalf("unsafe source %s", s)
		}
	}
	c := New()
	for _, s := range []string{"https://archive.org/download/item/video.mp4", "https://evil.test/metadata/item"} {
		req, _ := http.NewRequest("GET", s, nil)
		if c.HTTP.CheckRedirect(req, nil) == nil {
			t.Fatalf("unsafe redirect %s", s)
		}
	}
}
