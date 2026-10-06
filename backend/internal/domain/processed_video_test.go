package domain

import (
	"strings"
	"testing"
	"time"
)

func processedFixture() Entry {
	f := File{Name: "source.avi", Source: "original", Bytes: 100, Duration: 10, Width: 320, Height: 240, MD5: strings.Repeat("a", 32)}
	e := Entry{Episode: Episode{ID: "processed", CollectionID: "private-video", Title: "Source title", Files: []File{f}}, Policy: Local, Budget: 1024, SourceHash: strings.Repeat("b", 64), LocalSnapshot: strings.Repeat("b", 64), LocalFile: f.Name, LocalPath: "original.avi", LocalSHA256: strings.Repeat("c", 64), Decision: Decision{Acquisition: true, ProcessingOriginal: true, File: f.Name, Notes: "Original acquisition only"}}
	e.Decision.ProcessedVideo = &ProcessedVideoReview{Snapshot: e.SourceHash, OriginalFile: f.Name, OriginalSHA256: e.LocalSHA256, OriginalBytes: f.Bytes, Path: "derived.mp4", SHA256: strings.Repeat("d", 64), Bytes: 100, Duration: 10, Width: 320, Height: 240, ReceiptPath: "processing.json", ReceiptSHA256: strings.Repeat("e", 64), ReceiptBytes: 100, RecoveryPath: "restore.json", RecoverySHA256: strings.Repeat("f", 64), RecoveryBytes: 100, Published: true, Playback: true, Channel: true, Notes: "Owner-approved private qualification"}
	return e
}

func TestProcessedVideoDoesNotRewriteSourceOrOriginalPermissions(t *testing.T) {
	e := processedFixture()
	if err := e.ValidateDecision(e.Decision); err != nil {
		t.Fatal(err)
	}
	f, ok := e.Selected()
	if !ok || f.Source != "local_processing" || f.Original != "source.avi" {
		t.Fatal(f, ok)
	}
	original, ok := e.SelectedOriginal()
	if !ok || original.Name != "source.avi" {
		t.Fatal(original, ok)
	}
	if e.Decision.Playback || e.Decision.Published || e.Decision.Channel || len(e.Episode.Files) != 1 || e.Episode.Files[0].Name != "source.avi" {
		t.Fatal("original permissions or inventory changed")
	}
	p, err := Resolve(e, true)
	if err != nil || !p.Processed || p.URL != "/api/media/processed" {
		t.Fatal(p, err)
	}
	start := time.Now().Add(-time.Second)
	s := Schedule{Revision: "private-derived", Programs: []Program{{EpisodeID: e.Episode.ID, File: f.Name, Policy: Local, Start: start, Duration: 10}}}
	if n := s.At(start.Add(time.Second), map[string]Entry{e.Episode.ID: e}); n.State != "playing" || n.Playback == nil || !n.Playback.Processed {
		t.Fatal(n)
	}
	e.Decision.ProcessedVideo.Channel = false
	if _, err = Resolve(e, true); err == nil {
		t.Fatal("on-demand approval authorized channel")
	}
	if _, err = Resolve(e, false); err != nil {
		t.Fatal(err)
	}
}

func TestProcessedVideoRejectsStaleBindingsRightsAndLimits(t *testing.T) {
	changes := []func(*Entry){
		func(e *Entry) { e.SourceHash = strings.Repeat("1", 64) }, func(e *Entry) { e.LocalSHA256 = strings.Repeat("2", 64) },
		func(e *Entry) { e.Decision.Acquisition = false }, func(e *Entry) { e.Decision.ProcessingOriginal = false },
		func(e *Entry) { e.Episode.SourceCollections = []string{DemocracyNow} }, func(e *Entry) { e.Policy = Stream },
		func(e *Entry) { e.Decision.ProcessedVideo.Public = true }, func(e *Entry) { e.Decision.ProcessedVideo.Playback = false },
		func(e *Entry) { e.Decision.ProcessedVideo.Path = "../derived.mp4" }, func(e *Entry) { e.Decision.ProcessedVideo.Bytes = 384<<20 + 1 },
		func(e *Entry) { e.Budget = 100 }, func(e *Entry) { e.Decision.ProcessedVideo.OriginalFile = "different.avi" },
		func(e *Entry) { e.Decision.ProcessedVideo.RecoverySHA256 = "" }, func(e *Entry) { e.Decision.ProcessedVideo.Duration = 11 },
	}
	for i, change := range changes {
		e := processedFixture()
		change(&e)
		if e.ValidateProcessedVideo(*e.Decision.ProcessedVideo) == nil || e.Watchable() {
			t.Fatal("invalid binding accepted", i)
		}
	}
}

func TestLocalCaptionsNeedExactSourceSelectionAndReviewedBudget(t *testing.T) {
	e := processedFixture()
	c := LocalVideoCaption{CaptionReview: CaptionReview{File: "English.vtt", Label: "English fixture", Language: "en"}, Path: "captions/English.vtt", Bytes: 50, SHA256: strings.Repeat("1", 64)}
	e.Episode.Files = append(e.Episode.Files, File{Name: c.File, Bytes: c.Bytes, MD5: strings.Repeat("a", 32)})
	e.Decision.ProcessedVideo.Captions = []LocalVideoCaption{c}
	if err := e.ValidateProcessedVideo(*e.Decision.ProcessedVideo); err != nil {
		t.Fatal(err)
	}
	p, err := Resolve(e, false)
	if err != nil || len(p.Captions) != 1 || p.Captions[0].URL != "/api/video-captions/processed/0" {
		t.Fatal(p, err)
	}
	e.Episode.Files[1].Private = true
	if e.ValidateProcessedVideo(*e.Decision.ProcessedVideo) == nil {
		t.Fatal("restricted caption accepted")
	}
	e.Episode.Files[1].Private = false
	e.Episode.Files[1].MD5 = ""
	if e.ValidateProcessedVideo(*e.Decision.ProcessedVideo) == nil {
		t.Fatal("unpinned caption accepted")
	}
	e.Episode.Files[1].MD5 = strings.Repeat("a", 32)
	e.Budget = 200
	if e.ValidateProcessedVideo(*e.Decision.ProcessedVideo) == nil {
		t.Fatal("caption exceeded allowance")
	}
}
