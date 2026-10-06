package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReviewedArtworkAndSafeDetails(t *testing.T) {
	e := Entry{SourceTitle: "Source title", SourceHash: Digest([]byte("snapshot")), Episode: Episode{ID: "episode", SourceItemID: "series", Title: "Reviewed title", Files: []File{{Name: "cover.jpg", Bytes: 20}, {Name: "video.mp4", Bytes: 100, Duration: 10, Width: 640, Height: 480, Raw: json.RawMessage(`{"private_note":"secret"}`)}}}, Decision: Decision{Title: "Reviewed title", File: "video.mp4", Notes: "private operator notes"}}
	e.Decision.Artwork = &ArtworkReview{File: "cover.jpg", Snapshot: e.SourceHash}
	ep := e.ViewerEpisode(e.SourceTitle)
	if ep.Artwork == nil || ep.Artwork.Scope != "series" || ep.Artwork.URL != "https://archive.org/download/series/cover.jpg" || ep.Video.SourceTitle != "Source title" || ep.Video.TitleOrigin != "editorial" || ep.Video.Duration != 10 {
		t.Fatalf("projection: %+v", ep)
	}
	b, _ := json.Marshal(ep)
	if strings.Contains(string(b), "secret") || strings.Contains(string(b), "private operator") || ep.Files != nil {
		t.Fatal("private values exposed", string(b))
	}
	for _, mutate := range []func(*Entry){
		func(e *Entry) { e.SourceHash = Digest([]byte("changed")) },
		func(e *Entry) { e.Episode.Files[0].Private = true },
		func(e *Entry) { e.Decision.Artwork.File = "../cover.jpg" },
		func(e *Entry) { e.Decision.Artwork.File = "cover.svg" },
	} {
		copy := e
		copy.Episode.Files = append([]File(nil), e.Episode.Files...)
		review := *e.Decision.Artwork
		copy.Decision.Artwork = &review
		mutate(&copy)
		if _, err := copy.reviewedArtwork(copy.Decision); err == nil {
			t.Fatal("invalid artwork accepted")
		}
		if copy.ViewerEpisode(copy.SourceTitle).Artwork != nil {
			t.Fatal("stale/restricted artwork exposed")
		}
	}
}

func TestMetadataStrings(t *testing.T) {
	for raw, want := range map[string]int{`"en"`: 1, `["en","en",42,{"private":"secret"},"fr"]`: 2, `{"private":"secret"}`: 0, `null`: 0} {
		if len(MetadataStrings([]byte(raw))) != want {
			t.Fatal(raw)
		}
	}
}
