package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Media string

const (
	PrintMedia Media = "print"
	AudioMedia Media = "audio"
	VideoMedia Media = "video"
)

func (m Media) Valid() bool { return m == PrintMedia || m == AudioMedia || m == VideoMedia }

type PrintExtension struct {
	DocumentID string           `json:"document_id"`
	Variant    string           `json:"variant"`
	Pages      int              `json:"pages"`
	Companions []PrintCompanion `json:"companions,omitempty"`
}
type PrintCompanion struct {
	File string `json:"file"`
	Role string `json:"role"` // ocr or cover, never an independent catalog item.
}
type AudioTrack struct {
	ID          string `json:"id"`
	File        string `json:"file"`
	Title       string `json:"title"`
	Set         string `json:"set,omitempty"`
	SetReviewed bool   `json:"set_reviewed"`
}
type AudioExtension struct {
	RecordingID string            `json:"recording_id"`
	ConcertID   string            `json:"concert_id,omitempty"`
	Tracks      []AudioTrack      `json:"tracks"`
	Credits     map[string]string `json:"credits"`
}
type VideoExtension struct {
	EpisodeID string          `json:"episode_id"`
	Captions  []CaptionReview `json:"captions,omitempty"`
}
type MediaExtension struct {
	Print *PrintExtension `json:"print,omitempty"`
	Audio *AudioExtension `json:"audio,omitempty"`
	Video *VideoExtension `json:"video,omitempty"`
}

// Catalog listings do not grant transport, acquisition, public or relay rights.
// Module resolvers must independently validate those decisions and exact assets.
type Representation struct {
	ID              string         `json:"id"`
	Media           Media          `json:"media"`
	SourceItemID    string         `json:"source_item_id"`
	Snapshot        string         `json:"snapshot_sha256"`
	File            string         `json:"file"`
	Extension       MediaExtension `json:"extension"`
	Collections     []string       `json:"collections"`
	LegacyEpisodeID string         `json:"legacy_episode_id,omitempty"`
	SearchCoverage  []string       `json:"search_coverage"`
}
type CatalogItem struct {
	ID              string           `json:"id"`
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	Date            string           `json:"date"`
	Creator         string           `json:"creator"`
	Editorial       json.RawMessage  `json:"editorial,omitempty"`
	Collections     []string         `json:"collections"`
	Representations []Representation `json:"representations"`
}

func (r Representation) Validate() error {
	if !ValidID(r.ID) || !r.Media.Valid() || !ValidID(r.SourceItemID) || !SafeFile(r.File) || len(r.Snapshot) != 64 || r.LegacyEpisodeID != "" {
		return fmt.Errorf("invalid representation identity")
	}
	for _, c := range r.Snapshot {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return fmt.Errorf("invalid snapshot digest")
		}
	}
	count := 0
	if r.Extension.Print != nil {
		count++
	}
	if r.Extension.Audio != nil {
		count++
	}
	if r.Extension.Video != nil {
		count++
	}
	if count != 1 {
		return fmt.Errorf("representation requires exactly one typed extension")
	}
	switch r.Media {
	case PrintMedia:
		p := r.Extension.Print
		if p == nil || !ValidID(p.DocumentID) || strings.TrimSpace(p.Variant) == "" || p.Pages < 0 {
			return fmt.Errorf("invalid print extension")
		}
		seen := map[string]bool{r.File: true}
		if len(p.Companions) > 16 {
			return fmt.Errorf("too many print companions")
		}
		for _, companion := range p.Companions {
			if !SafeFile(companion.File) || seen[companion.File] || (companion.Role != "ocr" && companion.Role != "cover") {
				return fmt.Errorf("invalid print companion")
			}
			seen[companion.File] = true
		}
	case AudioMedia:
		a := r.Extension.Audio
		if a == nil || !ValidID(a.RecordingID) || (a.ConcertID != "" && !ValidID(a.ConcertID)) || len(a.Tracks) == 0 || len(a.Tracks) > 1000 {
			return fmt.Errorf("invalid audio extension")
		}
		seen := map[string]bool{}
		for _, track := range a.Tracks {
			if !ValidID(track.ID) || !SafeFile(track.File) || seen[track.ID] || (!track.SetReviewed && track.Set != "") {
				return fmt.Errorf("invalid track identity or unreviewed set")
			}
			seen[track.ID] = true
		}
	case VideoMedia:
		if r.Extension.Video == nil || !ValidID(r.Extension.Video.EpisodeID) {
			return fmt.Errorf("invalid video extension")
		}
	}
	return nil
}

func (i CatalogItem) Validate() error {
	if !ValidID(i.ID) || strings.TrimSpace(i.Title) == "" || len(i.Representations) == 0 || len(i.Representations) > 64 || len(i.Collections) == 0 || len(i.Collections) > 64 {
		return fmt.Errorf("invalid catalog item")
	}
	members := map[string]bool{}
	for _, id := range i.Collections {
		if !ValidID(id) || members[id] {
			return fmt.Errorf("invalid membership")
		}
		members[id] = true
	}
	seen := map[string]bool{}
	for _, r := range i.Representations {
		if err := r.Validate(); err != nil {
			return err
		}
		if seen[r.ID] || len(r.Collections) == 0 {
			return fmt.Errorf("invalid representation membership")
		}
		seen[r.ID] = true
		rseen := map[string]bool{}
		for _, id := range r.Collections {
			if !members[id] || rseen[id] {
				return fmt.Errorf("representation outside item membership")
			}
			rseen[id] = true
		}
	}
	if len(i.Editorial) > 0 && (!json.Valid(i.Editorial) || len(i.Editorial) > 1<<20) {
		return fmt.Errorf("invalid editorial metadata")
	}
	return nil
}
