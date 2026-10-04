// Adapted from Dedhed at e5843b17fd33d2e09c88fe3e5022153a0c9e8adc (GPL-3.0-only).
// Package music owns audio contracts independently of document catalog models.
package music

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"
)

type File struct {
	Name   string `json:"name"`
	Format string `json:"format"`
	Bytes  int64  `json:"bytes"`
	MD5    string `json:"md5,omitempty"`
}
type Track struct {
	Title    string  `json:"title"`
	Position int     `json:"position"`
	Start    float64 `json:"start"`
	Duration float64 `json:"duration"`
	File     File    `json:"file"`
	Variants []File  `json:"variants"`
}
type Set struct {
	Title    string `json:"title"`
	First    int    `json:"first"`
	Last     int    `json:"last"`
	Reviewed bool   `json:"reviewed"`
}
type Recording struct {
	ID         string   `json:"id"`
	ConcertID  string   `json:"concert_id"`
	Artist     string   `json:"artist"`
	Date       string   `json:"date"`
	Venue      string   `json:"venue"`
	Location   string   `json:"location"`
	Title      string   `json:"title"`
	Kind       string   `json:"kind"`
	Source     string   `json:"source"`
	Taper      string   `json:"taper"`
	Transferer string   `json:"transferer"`
	Tracks     []Track  `json:"tracks"`
	Sets       []Set    `json:"sets"`
	Duration   float64  `json:"duration"`
	Warnings   []string `json:"warnings"`
	Synthetic  bool     `json:"synthetic"`
}
type Review struct {
	Published   bool   `json:"published"`
	Acquisition bool   `json:"acquisition"`
	Playback    bool   `json:"playback"`
	Relay       bool   `json:"relay"`
	PrivateOnly bool   `json:"private_only"`
	Kind        string `json:"kind"`
	Notes       string `json:"notes"`
	Sets        []Set  `json:"sets,omitempty"`
	Title       string `json:"title,omitempty"`
	ConcertID   string `json:"concert_id,omitempty"`
}
type Entry struct {
	Recording  Recording `json:"recording"`
	Review     Review    `json:"review"`
	Media      string    `json:"media"`
	SHA256     string    `json:"sha256"`
	SourceHash string    `json:"source_hash"`
}
type Program struct {
	RecordingID string    `json:"recording_id"`
	Start       time.Time `json:"start"`
	Duration    float64   `json:"duration"`
}
type Schedule struct {
	Revision string    `json:"revision"`
	Programs []Program `json:"programs"`
}
type Now struct {
	ServerTime time.Time `json:"server_time"`
	Revision   string    `json:"revision"`
	Program    Program   `json:"program"`
	Offset     float64   `json:"offset"`
	Track      int       `json:"track"`
	Next       *Program  `json:"next,omitempty"`
}

func Digest(b []byte) string          { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func Identity(parts ...string) string { return Digest([]byte(strings.Join(parts, "\x00")))[:24] }
func (r Recording) Validate() error {
	if r.ID == "" || (r.ConcertID == "" && (strings.TrimSpace(r.Kind) == "" || r.Kind == "concert")) || len(r.Tracks) == 0 || len(r.Tracks) > 1000 || r.Duration <= 0 || r.Duration > 86400 || math.IsNaN(r.Duration) || math.IsInf(r.Duration, 0) {
		return fmt.Errorf("invalid recording identity or duration")
	}
	end := 0.0
	for i, t := range r.Tracks {
		if t.Position != i || t.Duration <= 0 || math.IsNaN(t.Duration) || math.IsInf(t.Duration, 0) || math.IsNaN(t.Start) || math.IsInf(t.Start, 0) || math.Abs(t.Start-end) > 0.01 {
			return fmt.Errorf("invalid track order or duration at %d", i)
		}
		end += t.Duration
	}
	if math.Abs(end-r.Duration) > 0.05 {
		return fmt.Errorf("recording duration differs from track sum")
	}
	return nil
}
func ValidateSets(sets []Set, n int) error {
	end := -1
	for _, s := range sets {
		if !s.Reviewed || strings.TrimSpace(s.Title) == "" || s.First < 0 || s.Last < s.First || s.Last >= n || s.First <= end {
			return fmt.Errorf("sets require reviewed, ordered, nonoverlapping track ranges")
		}
		end = s.Last
	}
	return nil
}
func (s Schedule) Validate() error {
	if s.Revision == "" || len(s.Programs) == 0 {
		return fmt.Errorf("empty schedule")
	}
	var end time.Time
	for _, p := range s.Programs {
		if p.RecordingID == "" || p.Start.IsZero() || p.Duration <= 0 || math.IsNaN(p.Duration) || math.IsInf(p.Duration, 0) || p.Duration > 86400 || p.Start.Before(end) {
			return fmt.Errorf("invalid or overlapping program")
		}
		end = p.Start.Add(time.Duration(p.Duration * float64(time.Second)))
	}
	return nil
}
func (s Schedule) At(now time.Time, records map[string]Entry) (Now, error) {
	if err := s.Validate(); err != nil {
		return Now{}, err
	}
	for i, p := range s.Programs {
		offset := now.Sub(p.Start).Seconds()
		if offset < 0 || offset >= p.Duration {
			continue
		}
		e, ok := records[p.RecordingID]
		if !ok || !e.Review.Published || !e.Review.Playback || e.Media == "" {
			return Now{}, fmt.Errorf("scheduled recording unavailable")
		}
		n := Now{ServerTime: now, Revision: s.Revision, Program: p, Offset: offset, Track: 0}
		for j, t := range e.Recording.Tracks {
			if offset >= t.Start {
				n.Track = j
			}
		}
		if i+1 < len(s.Programs) {
			n.Next = &s.Programs[i+1]
		}
		return n, nil
	}
	return Now{}, fmt.Errorf("station has no current program")
}
