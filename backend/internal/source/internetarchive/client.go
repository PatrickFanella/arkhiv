// Package internetarchive reads item metadata and bounded original-file transfers.
package internetarchive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const UserAgent = "Arkhiv/0.1 (document preservation; info@subcult.tv)"

type RetryError struct{ At time.Time }

func (e *RetryError) Error() string { return "source backoff until " + e.At.UTC().Format(time.RFC3339) }

type File struct {
	Name    string `json:"name"`
	Format  string `json:"format"`
	Size    string `json:"size"`
	MD5     string `json:"md5"`
	SHA1    string `json:"sha1"`
	Source  string `json:"source"`
	Private string `json:"private"`
}

func (f File) Bytes() int64 { n, _ := strconv.ParseInt(f.Size, 10, 64); return n }

type Metadata struct {
	Metadata map[string]any `json:"metadata"`
	Files    []File         `json:"files"`
	Error    string         `json:"error"`
}
type Client struct {
	HTTP  *http.Client
	Base  string
	mu    sync.Mutex
	cache map[string]metadataCache
}

type metadataCache struct {
	ETag string
	Raw  []byte
}

func Allowed(u *url.URL) bool {
	h := strings.ToLower(u.Hostname())
	return u.Scheme == "https" && u.User == nil && (h == "archive.org" || strings.HasSuffix(h, ".us.archive.org")) && (u.Port() == "" || u.Port() == "443")
}
func New() *Client {
	return &Client{Base: "https://archive.org", HTTP: &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) > 5 || !Allowed(r.URL) {
			return fmt.Errorf("source redirect rejected")
		}
		return nil
	}}}
}
func (c *Client) Request(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	for attempt := 0; attempt < 4; attempt++ {
		r, e := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+path, nil)
		if e != nil {
			return nil, e
		}
		r.Header.Set("User-Agent", UserAgent)
		r.Header.Set("Accept-Encoding", "identity")
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		resp, e := c.HTTP.Do(r)
		if e != nil {
			return nil, e
		}
		if resp.StatusCode != 429 && resp.StatusCode < 500 {
			return resp, nil
		}
		resp.Body.Close()
		delay := time.Duration(1<<attempt) * time.Second
		if s := resp.Header.Get("Retry-After"); s != "" {
			if n, e := strconv.Atoi(s); e == nil {
				delay = time.Duration(n) * time.Second
			} else if t, e := http.ParseTime(s); e == nil {
				delay = time.Until(t)
			}
		}
		if delay < 0 {
			delay = 0
		}
		if delay > 60*time.Second || attempt == 3 {
			return nil, &RetryError{At: time.Now().Add(delay)}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
	return nil, fmt.Errorf("unreachable retry state")
}
func (c *Client) Metadata(ctx context.Context, item string) (Metadata, []byte, error) {
	c.mu.Lock()
	cached := c.cache[item]
	c.mu.Unlock()
	headers := map[string]string{}
	if cached.ETag != "" {
		headers["If-None-Match"] = cached.ETag
	}
	resp, e := c.Request(ctx, "/metadata/"+url.PathEscape(item), headers)
	if e != nil {
		return Metadata{}, nil, e
	}
	defer resp.Body.Close()
	if resp.StatusCode == 304 && len(cached.Raw) > 0 {
		var m Metadata
		e := json.Unmarshal(cached.Raw, &m)
		return m, append([]byte(nil), cached.Raw...), e
	}
	if resp.StatusCode != 200 {
		return Metadata{}, nil, fmt.Errorf("metadata status %d", resp.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, (16<<20)+1))
	if e != nil {
		return Metadata{}, nil, e
	}
	if len(b) > 16<<20 {
		return Metadata{}, nil, fmt.Errorf("metadata exceeds bound")
	}
	var m Metadata
	if e = json.Unmarshal(b, &m); e != nil {
		return m, b, e
	}
	if m.Error != "" || m.Metadata == nil || len(m.Files) == 0 {
		return m, b, fmt.Errorf("item unavailable or missing files")
	}
	c.mu.Lock()
	if c.cache == nil {
		c.cache = map[string]metadataCache{}
	}
	c.cache[item] = metadataCache{resp.Header.Get("ETag"), append([]byte(nil), b...)}
	c.mu.Unlock()
	return m, b, nil
}
func FilePath(item, file string) string {
	parts := strings.Split(file, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return "/download/" + url.PathEscape(item) + "/" + strings.Join(parts, "/")
}
