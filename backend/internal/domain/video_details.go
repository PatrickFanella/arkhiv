package domain

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// MetadataStrings accepts source strings or string arrays, never objects or
// arbitrary unknown fields. Compound strings remain intact as supplied.
func MetadataStrings(raw []byte) []string {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return nil
	}
	var values []any
	if s, ok := value.(string); ok {
		values = []any{s}
	} else {
		values, _ = value.([]any)
	}
	var out []string
	seen := map[string]bool{}
	for _, v := range values {
		s, ok := v.(string)
		s = strings.TrimSpace(s)
		if ok && s != "" && !seen[s] {
			out = append(out, s)
			seen[s] = true
		}
	}
	return out
}

// ArtworkReview authorizes a source reference, never acquisition. The snapshot
// pins the image inventory; an unrelated later refresh must not reuse approval.
type ArtworkReview struct {
	File        string `json:"file"`
	Snapshot    string `json:"snapshot_sha256"`
	EpisodeFile string `json:"episode_file,omitempty"`
}

type Artwork struct {
	ArtworkReview
	URL          string `json:"url"`
	SourceItemID string `json:"source_item_id"`
	Scope        string `json:"scope"`
}

type VideoDetails struct {
	SourceTitle string  `json:"source_title"`
	TitleOrigin string  `json:"title_origin"`
	SourceURL   string  `json:"source_url"`
	Snapshot    string  `json:"snapshot_sha256"`
	File        string  `json:"file,omitempty"`
	Format      string  `json:"format,omitempty"`
	Duration    float64 `json:"duration,omitempty"`
	Width       int     `json:"width,omitempty"`
	Height      int     `json:"height,omitempty"`
	Bytes       int64   `json:"bytes,omitempty"`
}

func (e Episode) SourceID() string {
	if e.SourceItemID != "" {
		return e.SourceItemID
	}
	return e.ID
}

func SafeArtworkFile(file string) bool {
	if strings.ContainsAny(file, "\\\x00") {
		return false
	}
	for _, part := range strings.Split(file, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func ArtworkPath(file string) string {
	parts := strings.Split(file, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func (e Entry) reviewedArtwork(d Decision) (*Artwork, error) {
	if d.Artwork == nil {
		return nil, nil
	}
	r := *d.Artwork
	if !SafeArtworkFile(r.File) || r.Snapshot != e.SourceHash || !ValidID(e.Episode.SourceID()) || r.EpisodeFile != "" && (r.EpisodeFile != e.Episode.SourceFile || !SafeFile(r.EpisodeFile)) {
		return nil, fmt.Errorf("artwork must pin the current source snapshot and a safe file")
	}
	lower := strings.ToLower(r.File)
	if !strings.HasSuffix(lower, ".jpg") && !strings.HasSuffix(lower, ".jpeg") && !strings.HasSuffix(lower, ".png") && !strings.HasSuffix(lower, ".webp") {
		return nil, fmt.Errorf("artwork requires a supported raster image")
	}
	for _, f := range e.Episode.Files {
		if f.Name == r.File && !f.Private && f.Bytes > 0 {
			scope := "episode"
			if e.Episode.SourceItemID != "" && r.EpisodeFile == "" {
				scope = "series"
			}
			return &Artwork{ArtworkReview: r, URL: "https://archive.org/download/" + url.PathEscape(e.Episode.SourceID()) + "/" + ArtworkPath(r.File), SourceItemID: e.Episode.SourceID(), Scope: scope}, nil
		}
	}
	return nil, fmt.Errorf("artwork is absent, empty or restricted")
}

// ViewerEpisode exposes an allowlisted projection. Raw inventory and editorial
// notes remain retained internally, not copied into the public viewer response.
func (e Entry) ViewerEpisode(sourceTitle string) Episode {
	ep := e.Episode
	ep.Files = nil
	ep.Artwork, _ = e.reviewedArtwork(e.Decision)
	origin := "source"
	if e.Decision.Title != "" {
		origin = "editorial"
	}
	ep.Video = &VideoDetails{SourceTitle: sourceTitle, TitleOrigin: origin, SourceURL: "https://archive.org/details/" + url.PathEscape(ep.SourceID()), Snapshot: e.SourceHash}
	if f, ok := e.Selected(); ok {
		ep.Video.File = f.Name
		ep.Video.Format = f.Format
		ep.Video.Duration = f.Duration
		ep.Video.Width = f.Width
		ep.Video.Height = f.Height
		ep.Video.Bytes = f.Bytes
	}
	return ep
}
