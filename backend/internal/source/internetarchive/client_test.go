package internetarchive

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestMetadataRetryCacheAndRefresh(t *testing.T) {
	calls := 0
	etag := `"one"`
	title := "first"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/metadata/item" {
			t.Error("unexpected payload request")
		}
		if r.Header.Get("User-Agent") != UserAgent {
			t.Error("missing identification")
		}
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(429)
			return
		}
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(304)
			return
		}
		w.Header().Set("ETag", etag)
		fmt.Fprintf(w, `{"metadata":{"title":%q},"files":[{"name":"one.pdf","size":"4"},{"name":"two.pdf","size":"8","md5":"checksum"}]}`, title)
	}))
	defer server.Close()
	c := &Client{Base: server.URL, HTTP: server.Client()}
	m, raw, e := c.Metadata(context.Background(), "item")
	if e != nil || len(m.Files) != 2 || m.Files[0].MD5 != "" {
		t.Fatal("metadata inventory", e)
	}
	_, cached, e := c.Metadata(context.Background(), "item")
	if e != nil || string(cached) != string(raw) || calls != 3 {
		t.Fatal("cache did not preserve raw response")
	}
	etag = `"two"`
	title = "changed"
	m, _, e = c.Metadata(context.Background(), "item")
	if e != nil || m.Metadata["title"] != "changed" {
		t.Fatal("missed source refresh")
	}
}
func TestMalformedMetadataAndRestrictedHosts(t *testing.T) {
	for _, body := range []string{`{`, `{}`, `{"metadata":{},"files":[]}`, `{"error":"missing"}`} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		c := &Client{Base: s.URL, HTTP: s.Client()}
		if _, _, e := c.Metadata(context.Background(), "item"); e == nil {
			t.Fatal("accepted unavailable metadata", body)
		}
		s.Close()
	}
	for _, v := range []struct {
		u       string
		allowed bool
	}{{"https://archive.org/metadata/item", true}, {"https://ia801234.us.archive.org/a.pdf", true}, {"http://archive.org/file", false}, {"https://archive.org.evil.invalid/file", false}, {"https://user@archive.org/file", false}, {"https://archive.org:8080/file", false}} {
		u, _ := url.Parse(v.u)
		if Allowed(u) != v.allowed {
			t.Fatal("host constraint", v.u)
		}
	}
}
