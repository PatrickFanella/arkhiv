package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
)

const MaxResponse = 16 << 20

type Client struct {
	HTTP  *http.Client
	Base  string
	Clock func() time.Time
}
type Response struct {
	Raw      []byte
	Status   int
	ETag     string
	Modified string
}
type Backoff struct {
	Until  time.Time
	Status int
}

func (b *Backoff) Error() string {
	return fmt.Sprintf("source backoff until %s (HTTP %d)", b.Until.UTC().Format(time.RFC3339), b.Status)
}
func Allowed(u *url.URL) bool {
	h := strings.ToLower(u.Hostname())
	return u.Scheme == "https" && u.User == nil && (h == "archive.org" || strings.HasSuffix(h, ".us.archive.org") || strings.HasSuffix(h, ".ca.archive.org")) && (u.Port() == "" || u.Port() == "443")
}
func New() *Client {
	return &Client{Base: "https://archive.org", Clock: time.Now, HTTP: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) > 5 || !Allowed(r.URL) || (!strings.HasPrefix(r.URL.Path, "/metadata/") && r.URL.Path != "/advancedsearch.php" && r.URL.Path != "/services/search/v1/scrape") {
			return fmt.Errorf("metadata redirect rejected")
		}
		return nil
	}}}
}
func RetryDeadline(header string, now time.Time) time.Time {
	if n, err := strconv.ParseInt(header, 10, 64); err == nil && n >= 0 && n < (1<<63-1)-now.Unix() {
		return time.Unix(now.Unix()+n, int64(now.Nanosecond()))
	}
	if t, err := http.ParseTime(header); err == nil && t.After(now) {
		return t
	}
	return now.Add(30 * time.Second)
}
func (c *Client) get(ctx context.Context, path, etag, modified string) (Response, error) {
	if !strings.HasPrefix(path, "/metadata/") && !strings.HasPrefix(path, "/advancedsearch.php?") && !strings.HasPrefix(path, "/services/search/v1/scrape?") {
		return Response{}, fmt.Errorf("metadata client cannot request media")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+path, nil)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("User-Agent", "VHNESS/0.1 (metadata-only catalog)")
	req.Header.Set("Accept-Encoding", "identity")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if modified != "" {
		req.Header.Set("If-Modified-Since", modified)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	r := Response{Status: resp.StatusCode, ETag: resp.Header.Get("ETag"), Modified: resp.Header.Get("Last-Modified")}
	if resp.StatusCode == 429 || resp.StatusCode >= 500 {
		return r, &Backoff{Until: RetryDeadline(resp.Header.Get("Retry-After"), c.Clock()), Status: resp.StatusCode}
	}
	if resp.StatusCode == 304 {
		return r, nil
	}
	if resp.StatusCode != 200 {
		return r, fmt.Errorf("source metadata status %d", resp.StatusCode)
	}
	r.Raw, err = io.ReadAll(io.LimitReader(resp.Body, MaxResponse+1))
	if err != nil {
		return r, err
	}
	if len(r.Raw) > MaxResponse {
		return r, fmt.Errorf("metadata response exceeds limit")
	}
	if !json.Valid(r.Raw) {
		return r, fmt.Errorf("invalid source JSON")
	}
	return r, nil
}
func (c *Client) Metadata(ctx context.Context, id, etag, modified string) (Response, error) {
	if !domain.ValidID(id) {
		return Response{}, fmt.Errorf("invalid item identity")
	}
	return c.get(ctx, "/metadata/"+url.PathEscape(id), etag, modified)
}
func (c *Client) Discover(ctx context.Context, collection string, page, rows int) ([]string, int64, Response, error) {
	if !domain.ValidID(collection) || page < 1 || rows < 1 || rows > 100 {
		return nil, 0, Response{}, fmt.Errorf("invalid discovery bounds")
	}
	q := url.Values{"q": {"collection:" + collection + " AND mediatype:movies"}, "fl[]": {"identifier"}, "sort[]": {"identifier asc"}, "rows": {strconv.Itoa(rows)}, "page": {strconv.Itoa(page)}, "output": {"json"}}
	r, err := c.get(ctx, "/advancedsearch.php?"+q.Encode(), "", "")
	if err != nil {
		return nil, 0, r, err
	}
	var d struct {
		Response *struct {
			Count *int64 `json:"numFound"`
			Docs  []struct {
				ID string `json:"identifier"`
			} `json:"docs"`
		} `json:"response"`
	}
	if err = json.Unmarshal(r.Raw, &d); err != nil {
		return nil, 0, r, err
	}
	if d.Response == nil || d.Response.Count == nil || *d.Response.Count < 0 || len(d.Response.Docs) > rows {
		return nil, 0, r, fmt.Errorf("invalid discovery response")
	}
	ids := []string{}
	for _, doc := range d.Response.Docs {
		if !domain.ValidID(doc.ID) {
			return nil, 0, r, fmt.Errorf("invalid discovery identifier")
		}
		ids = append(ids, doc.ID)
	}
	return ids, *d.Response.Count, r, nil
}
