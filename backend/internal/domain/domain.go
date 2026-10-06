// Package domain defines contracts shared by source and local delivery.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const DemocracyNow = "democracy_now_vid"

// BrowseLess retains broadcast-date ordering for news and the source's numbered
// file order for episodes materialized from the same complete-series item.
func BrowseLess(a, b Episode) bool {
	if a.SourceItemID != b.SourceItemID {
		return a.SourceItemID < b.SourceItemID
	}
	if a.SourceItemID != "" {
		prefixA, _, _ := strings.Cut(a.SourceFile, " ")
		prefixB, _, _ := strings.Cut(b.SourceFile, " ")
		ai, errA := strconv.Atoi(prefixA)
		bi, errB := strconv.Atoi(prefixB)
		if (errA == nil) != (errB == nil) {
			return errA == nil
		}
		if errA == nil && errB == nil && ai != bi {
			return ai < bi
		}
		if a.SourceFile != b.SourceFile {
			return a.SourceFile < b.SourceFile
		}
	}
	if a.Date != b.Date {
		return a.Date > b.Date
	}
	return a.ID < b.ID
}

type Policy string

const (
	Stream Policy = "source_stream"
	Local  Policy = "local_preserve"
)

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,200}$`)

func ValidID(s string) bool   { return idPattern.MatchString(s) }
func Digest(b []byte) string  { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func Duration(n float64) bool { return n > 0 && n <= 86400 && !math.IsInf(n, 0) && !math.IsNaN(n) }

type Collection struct {
	ID         string `json:"id"`
	Policy     Policy `json:"policy"`
	Budget     int64  `json:"budget_bytes"`
	Reserve    int64  `json:"reserve_bytes"`
	SourceItem string `json:"source_item,omitempty"`
}

func (c Collection) Validate() error {
	if !ValidID(c.ID) || (c.Policy != Stream && c.Policy != Local) {
		return fmt.Errorf("invalid collection or delivery policy")
	}
	if c.SourceItem != "" && (!ValidID(c.SourceItem) || c.Policy != Local || c.SourceItem == DemocracyNow) {
		return fmt.Errorf("file-based item requires an explicitly local source item")
	}
	if c.ID == DemocracyNow && (c.Policy != Stream || c.SourceItem != "") {
		return fmt.Errorf("Democracy Now! must remain source-streaming only")
	}
	if c.Policy == Local && (c.Budget <= 0 || c.Reserve <= 0) {
		return fmt.Errorf("local preservation needs positive byte budget and free-space reserve")
	}
	return nil
}

type File struct {
	Name     string          `json:"name"`
	Format   string          `json:"format"`
	Source   string          `json:"source"`
	Original string          `json:"original,omitempty"`
	Bytes    int64           `json:"bytes"`
	SHA1     string          `json:"sha1,omitempty"`
	MD5      string          `json:"md5,omitempty"`
	Duration float64         `json:"duration"`
	Width    int             `json:"width"`
	Height   int             `json:"height"`
	Private  bool            `json:"private"`
	Raw      json.RawMessage `json:"raw"`
}

func (f File) Candidate() bool {
	return !f.Private && strings.HasSuffix(strings.ToLower(f.Name), ".mp4") && f.Bytes > 0 && Duration(f.Duration) && SafeFile(f.Name)
}

// PreservationCandidate allows a checksummed source AVI to be retained for an
// explicitly approved processing job. It does not establish playback eligibility.
func (f File) PreservationCandidate() bool {
	if f.Candidate() {
		return true
	}
	return !f.Private && f.Source == "original" && strings.HasSuffix(strings.ToLower(f.Name), ".avi") && f.Bytes > 0 && Duration(f.Duration) && SafeFile(f.Name) && (validSourceChecksum(f.MD5, 32) || validSourceChecksum(f.SHA1, 40))
}
func validSourceChecksum(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, c := range value {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}
func SafeFile(s string) bool { return s != "" && s != "." && !strings.ContainsAny(s, "/\\\x00") }

type Episode struct {
	ID                string        `json:"id"`
	CollectionID      string        `json:"collection_id"`
	SourceItemID      string        `json:"source_item_id,omitempty"`
	SourceCollections []string      `json:"source_collections,omitempty"`
	SourceFile        string        `json:"source_file,omitempty"`
	Title             string        `json:"title"`
	Date              string        `json:"date"`
	Description       string        `json:"description"`
	Creator           string        `json:"creator,omitempty"`
	Rights            string        `json:"rights,omitempty"`
	LicenseURL        string        `json:"license_url,omitempty"`
	Files             []File        `json:"files"`
	Synthetic         bool          `json:"synthetic"`
	Subjects          []string      `json:"subjects,omitempty"`
	Languages         []string      `json:"languages,omitempty"`
	Artwork           *Artwork      `json:"artwork,omitempty"`
	Video             *VideoDetails `json:"video,omitempty"`
}

func (e Episode) Validate() error {
	if (e.SourceItemID != "" && (!ValidID(e.SourceItemID) || !SafeFile(e.SourceFile))) || !ValidID(e.ID) || !ValidID(e.CollectionID) || strings.TrimSpace(e.Title) == "" {
		return fmt.Errorf("invalid episode identity")
	}
	if e.Date != "" {
		layout := "2006-01-02"
		if len(e.Date) == 4 {
			layout = "2006"
		} else if len(e.Date) == 7 {
			layout = "2006-01"
		}
		if _, err := time.Parse(layout, e.Date); err != nil {
			return fmt.Errorf("invalid broadcast date")
		}
	}
	return nil
}

type Decision struct {
	ProcessingOriginal bool            `json:"processing_original,omitempty"`
	Acquisition        bool            `json:"acquisition"`
	Published          bool            `json:"published"`
	Playback           bool            `json:"playback"`
	Channel            bool            `json:"channel"`
	Public             bool            `json:"public"`
	File               string          `json:"file"`
	Title              string          `json:"title,omitempty"`
	Notes              string          `json:"notes"`
	Captions           []CaptionReview `json:"captions,omitempty"`
	Artwork            *ArtworkReview  `json:"artwork,omitempty"`
}

// CaptionReview records a browser-qualified source WebVTT file. Discovery never
// enables a track. Local sidecar preservation needs its own acquisition budget.
type CaptionReview struct {
	File     string `json:"file"`
	Label    string `json:"label"`
	Language string `json:"language"`
}
type Caption struct {
	CaptionReview
	URL string `json:"url"`
}

var languagePattern = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

func (e Entry) reviewedCaptions(d Decision) ([]Caption, error) {
	if len(d.Captions) > 16 || (len(d.Captions) > 0 && e.Policy != Stream) {
		return nil, fmt.Errorf("source captions require a streaming policy and at most 16 reviewed tracks")
	}
	var tracks []Caption
	seen := make(map[string]bool)
	for _, review := range d.Captions {
		if !SafeFile(review.File) || !strings.HasSuffix(strings.ToLower(review.File), ".vtt") || strings.TrimSpace(review.Label) == "" || len(review.Label) > 200 || !languagePattern.MatchString(review.Language) || seen[review.File] {
			return nil, fmt.Errorf("invalid reviewed WebVTT identity, label, or language")
		}
		found := false
		for _, file := range e.Episode.Files {
			if file.Name == review.File && !file.Private && file.Bytes > 0 {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("reviewed caption is absent or restricted")
		}
		seen[review.File] = true
		tracks = append(tracks, Caption{CaptionReview: review, URL: "https://archive.org/download/" + url.PathEscape(e.Episode.ID) + "/" + url.PathEscape(review.File)})
	}
	return tracks, nil
}

type Entry struct {
	SourceTitle   string   `json:"-"`
	Episode       Episode  `json:"episode"`
	Decision      Decision `json:"decision"`
	Policy        Policy   `json:"policy"`
	SourceHash    string   `json:"source_hash"`
	Unavailable   bool     `json:"unavailable"`
	LocalPath     string   `json:"-"`
	LocalSHA256   string   `json:"-"`
	LocalFile     string   `json:"-"`
	LocalSnapshot string   `json:"-"`
}

func (e Entry) Selected() (File, bool) {
	if e.Episode.SourceFile != "" && e.Decision.File != e.Episode.SourceFile {
		return File{}, false
	}
	for _, f := range e.Episode.Files {
		if f.Name == e.Decision.File && f.Candidate() {
			return f, true
		}
	}
	return File{}, false
}

// SelectedOriginal requires a distinct processing decision for an AVI original.
// Selected remains limited to browser candidates, including after local binding.
func (e Entry) SelectedOriginal() (File, bool) {
	if f, ok := e.Selected(); ok {
		return f, true
	}
	if e.Policy != Local || e.Episode.DemocracySource() || !e.Decision.Acquisition || !e.Decision.ProcessingOriginal || e.Decision.Playback || e.Decision.Published || e.Decision.Channel || e.Decision.Public || (e.Episode.SourceFile != "" && e.Decision.File != e.Episode.SourceFile) {
		return File{}, false
	}
	for _, f := range e.Episode.Files {
		if f.Name == e.Decision.File && f.PreservationCandidate() {
			return f, true
		}
	}
	return File{}, false
}
func (e Entry) Watchable() bool {
	_, ok := e.Selected()
	return ok && e.Decision.Published && e.Decision.Playback && !e.Unavailable && (e.Policy == Stream || (e.LocalPath != "" && e.LocalSHA256 != "" && e.LocalFile == e.Decision.File && e.LocalSnapshot == e.SourceHash))
}
func (e Entry) ValidateDecision(d Decision) error {
	if strings.TrimSpace(d.Notes) == "" {
		return fmt.Errorf("decision requires review notes")
	}
	if d.Public {
		return fmt.Errorf("public release is not enabled")
	}
	if d.Acquisition && (e.Policy != Local || e.Episode.DemocracySource()) {
		return fmt.Errorf("source-stream collections cannot acquire media")
	}
	copy := e
	copy.Decision = d
	if d.ProcessingOriginal && (e.Policy != Local || e.Episode.DemocracySource() || !d.Acquisition || d.Playback || d.Published || d.Channel || d.Public) {
		return fmt.Errorf("processing-original approval requires private acquisition only")
	}
	if d.Acquisition {
		if _, ok := copy.SelectedOriginal(); !ok {
			return fmt.Errorf("acquisition requires an explicitly reviewed eligible original")
		}
	}
	if d.Playback || d.Published || d.Channel {
		if _, ok := copy.Selected(); !ok {
			return fmt.Errorf("review requires an eligible selected file")
		}
	}
	if d.Published && !d.Playback || d.Channel && !d.Playback {
		return fmt.Errorf("publication and channel use require playback permission")
	}
	if _, err := e.reviewedCaptions(d); err != nil {
		return err
	}
	if _, err := e.reviewedArtwork(d); err != nil {
		return err
	}
	return nil
}

type Playback struct {
	ID        string    `json:"id"`
	Policy    Policy    `json:"policy"`
	URL       string    `json:"url"`
	SourceURL string    `json:"source_url"`
	File      File      `json:"file"`
	Snapshot  string    `json:"snapshot_sha256"`
	Captions  []Caption `json:"captions,omitempty"`
}

func Resolve(e Entry, channel bool) (Playback, error) {
	if !e.Watchable() || (channel && !e.Decision.Channel) {
		return Playback{}, fmt.Errorf("episode unavailable or not approved for this viewing mode")
	}
	f, _ := e.Selected()
	tracks, err := e.reviewedCaptions(e.Decision)
	if err != nil {
		return Playback{}, err
	}
	sourceID := e.Episode.ID
	if e.Episode.SourceItemID != "" {
		sourceID = e.Episode.SourceItemID
	}
	p := Playback{ID: e.Episode.ID, Policy: e.Policy, SourceURL: "https://archive.org/details/" + url.PathEscape(sourceID), File: f, Snapshot: e.SourceHash, Captions: tracks}
	if e.Policy == Stream {
		p.URL = "https://archive.org/download/" + url.PathEscape(sourceID) + "/" + url.PathEscape(f.Name)
	} else {
		p.URL = "/api/media/" + url.PathEscape(e.Episode.ID)
	}
	return p, nil
}

type Program struct {
	EpisodeID string    `json:"episode_id"`
	File      string    `json:"file"`
	Policy    Policy    `json:"policy"`
	Start     time.Time `json:"start"`
	Duration  float64   `json:"duration"`
}
type Schedule struct {
	Revision string    `json:"revision"`
	Programs []Program `json:"programs"`
}
type Now struct {
	ServerTime time.Time `json:"server_time"`
	Revision   string    `json:"revision"`
	State      string    `json:"state"`
	Reason     string    `json:"reason,omitempty"`
	Program    *Program  `json:"program,omitempty"`
	Offset     float64   `json:"offset"`
	Episode    *Episode  `json:"episode,omitempty"`
	Playback   *Playback `json:"playback,omitempty"`
	Next       *Program  `json:"next,omitempty"`
}

func (s Schedule) Validate() error {
	if s.Revision == "" || len(s.Programs) == 0 || len(s.Programs) > 1000 {
		return fmt.Errorf("empty or oversized schedule")
	}
	var end time.Time
	for _, p := range s.Programs {
		if !ValidID(p.EpisodeID) || !SafeFile(p.File) || (p.Policy != Stream && p.Policy != Local) || p.Start.IsZero() || !Duration(p.Duration) || p.Start.Before(end) {
			return fmt.Errorf("invalid or overlapping program")
		}
		end = p.Start.Add(time.Duration(p.Duration * float64(time.Second)))
	}
	return nil
}
func (s Schedule) At(t time.Time, entries map[string]Entry) Now {
	n := Now{ServerTime: t.UTC(), Revision: s.Revision, State: "unavailable", Reason: "channel schedule exhausted"}
	if err := s.Validate(); err != nil {
		n.Reason = err.Error()
		return n
	}
	for i, p := range s.Programs {
		if t.Before(p.Start) {
			next := p
			n.Next = &next
			n.Reason = "channel schedule gap"
			return n
		}
		offset := t.Sub(p.Start).Seconds()
		if offset >= p.Duration {
			continue
		}
		current := p
		n.Program = &current
		n.Offset = offset
		if i+1 < len(s.Programs) {
			next := s.Programs[i+1]
			n.Next = &next
		}
		e, ok := entries[p.EpisodeID]
		f, eligible := e.Selected()
		if !ok || !eligible || f.Name != p.File || e.Policy != p.Policy || p.Duration > f.Duration+0.1 {
			n.Reason = "scheduled asset no longer eligible"
			return n
		}
		resolved, err := Resolve(e, true)
		if err != nil {
			n.Reason = err.Error()
			return n
		}
		n.State = "playing"
		n.Reason = ""
		n.Episode = &e.Episode
		n.Playback = &resolved
		return n
	}
	return n
}

func (e Episode) DemocracySource() bool {
	if e.CollectionID == DemocracyNow || e.SourceItemID == DemocracyNow {
		return true
	}
	for _, id := range e.SourceCollections {
		if id == DemocracyNow {
			return true
		}
	}
	return false
}
