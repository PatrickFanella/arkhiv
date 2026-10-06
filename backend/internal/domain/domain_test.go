package domain

import (
	"sort"
	"testing"
	"time"
)

func TestBrowseOrderPreservesNumberedSeriesAndNewsDates(t *testing.T) {
	series := []Episode{{ID: "opaque-a", SourceItemID: "series", SourceFile: "100 Motion.mp4"}, {ID: "opaque-b", SourceItemID: "series", SourceFile: "02 Earth.mp4"}, {ID: "opaque-c", SourceItemID: "series", SourceFile: "01 Flight.mp4"}, {ID: "opaque-d", SourceItemID: "series", SourceFile: "09 Wind.mp4"}}
	sort.Slice(series, func(i, j int) bool { return BrowseLess(series[i], series[j]) })
	for i, want := range []string{"01 Flight.mp4", "02 Earth.mp4", "09 Wind.mp4", "100 Motion.mp4"} {
		if series[i].SourceFile != want {
			t.Fatal("source episode order", series)
		}
	}
	news := []Episode{{ID: "earlier", Date: "2026-10-01"}, {ID: "later", Date: "2026-10-02"}}
	if !BrowseLess(news[1], news[0]) {
		t.Fatal("news no longer sorts newest first")
	}
}

func TestProtectedCollectionAndDecisions(t *testing.T) {
	if err := (Collection{ID: DemocracyNow, Policy: Local, Budget: 10, Reserve: 1}).Validate(); err == nil {
		t.Fatal("Democracy Now acquisition enabled")
	}
	e := Entry{Episode: Episode{ID: "sample", Files: []File{{Name: "a.mp4", Bytes: 1, Duration: 60}}}, Policy: Stream}
	if err := e.ValidateDecision(Decision{Acquisition: true, Notes: "test"}); err == nil {
		t.Fatal("stream acquisition enabled")
	}
	if err := e.ValidateDecision(Decision{Playback: true, File: "../a.mp4", Notes: "test"}); err == nil {
		t.Fatal("invalid file eligible")
	}
	e.Decision = Decision{Published: true, Playback: true, Channel: false, File: "a.mp4", Notes: "private sample"}
	if _, err := Resolve(e, true); err == nil {
		t.Fatal("channel permission ignored")
	}
	p, err := Resolve(e, false)
	if err != nil || p.URL != "https://archive.org/download/sample/a.mp4" {
		t.Fatalf("resolution %v %v", p, err)
	}
}
func TestScheduleBoundariesAndWithdrawal(t *testing.T) {
	start := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	e := Entry{Episode: Episode{ID: "sample", Files: []File{{Name: "a.mp4", Bytes: 1, Duration: 60}}}, Policy: Stream, Decision: Decision{Published: true, Playback: true, Channel: true, File: "a.mp4"}}
	entries := map[string]Entry{"sample": e}
	s := Schedule{Revision: "one", Programs: []Program{{EpisodeID: "sample", File: "a.mp4", Policy: Stream, Start: start, Duration: 60}, {EpisodeID: "sample", File: "a.mp4", Policy: Stream, Start: start.Add(60 * time.Second), Duration: 60}}}
	n := s.At(start.Add(30*time.Second), entries)
	if n.State != "playing" || n.Offset != 30 {
		t.Fatalf("midpoint %#v", n)
	}
	n = s.At(start.Add(60*time.Second), entries)
	if n.State != "playing" || n.Offset != 0 || !n.Program.Start.Equal(start.Add(60*time.Second)) {
		t.Fatalf("boundary %#v", n)
	}
	e.Decision.Channel = false
	entries["sample"] = e
	n = s.At(start.Add(30*time.Second), entries)
	if n.State != "unavailable" || n.Program == nil || n.Next == nil {
		t.Fatalf("withdrawal lost slot %#v", n)
	}
	if s.At(start.Add(120*time.Second), entries).State != "unavailable" {
		t.Fatal("exhausted schedule loops")
	}
	s.Programs[1].Start = start.Add(59 * time.Second)
	if s.Validate() == nil {
		t.Fatal("overlap allowed")
	}
}

func TestSourceCaptionsRequireExplicitReviewAndCurrentEligibility(t *testing.T) {
	e := Entry{Episode: Episode{ID: "sample", Files: []File{{Name: "a.mp4", Bytes: 10, Duration: 60}, {Name: "English captions.vtt", Bytes: 100}}}, Policy: Stream, Decision: Decision{Published: true, Playback: true, File: "a.mp4", Notes: "Browser qualified sample"}}
	p, err := Resolve(e, false)
	if err != nil || len(p.Captions) != 0 {
		t.Fatal("discovered captions were automatically enabled", p, err)
	}
	review := CaptionReview{File: "English captions.vtt", Label: "English", Language: "en"}
	e.Decision.Captions = []CaptionReview{review}
	if err = e.ValidateDecision(e.Decision); err != nil {
		t.Fatal(err)
	}
	p, err = Resolve(e, false)
	if err != nil || len(p.Captions) != 1 || p.Captions[0].URL != "https://archive.org/download/sample/English%20captions.vtt" {
		t.Fatal("reviewed caption identity", p, err)
	}
	for _, invalid := range []CaptionReview{{File: "missing.vtt", Label: "English", Language: "en"}, {File: "../English captions.vtt", Label: "English", Language: "en"}, {File: "a.mp4", Label: "English", Language: "en"}, {File: review.File, Label: "English", Language: "unknown language"}} {
		decision := e.Decision
		decision.Captions = []CaptionReview{invalid}
		if e.ValidateDecision(decision) == nil {
			t.Fatal("invalid caption accepted", invalid)
		}
	}
	e.Episode.Files[1].Private = true
	if _, err = Resolve(e, false); err == nil {
		t.Fatal("restricted caption after source refresh remained eligible")
	}
	e.Episode.Files[1].Private = false
	e.Policy = Local
	if e.ValidateDecision(e.Decision) == nil {
		t.Fatal("source captions enabled without independent local sidecar preservation")
	}
}

func TestProcessingOriginalDoesNotGrantPlayback(t *testing.T) {
	f := File{Name: "original.avi", Source: "original", Bytes: 100, Duration: 60, MD5: "91b3b5e772b76c3a03b5238103ff114a"}
	e := Entry{Policy: Local, Episode: Episode{ID: "avi", CollectionID: "private-processing", SourceFile: f.Name, Files: []File{f}}, Decision: Decision{Acquisition: true, ProcessingOriginal: true, File: f.Name, Notes: "Owner approved bounded private processing"}}
	if err := e.ValidateDecision(e.Decision); err != nil {
		t.Fatal(err)
	}
	if _, ok := e.SelectedOriginal(); !ok {
		t.Fatal("explicit original unavailable")
	}
	e.LocalPath = "original.avi"
	e.LocalSHA256 = Digest([]byte("retained"))
	e.LocalFile = f.Name
	e.LocalSnapshot = e.SourceHash
	if _, ok := e.Selected(); ok || e.Watchable() {
		t.Fatal("AVI treated as browser media")
	}
	for _, permission := range []string{"playback", "published", "channel", "public"} {
		d := e.Decision
		switch permission {
		case "playback":
			d.Playback = true
		case "published":
			d.Published = true
		case "channel":
			d.Channel = true
		case "public":
			d.Public = true
		}
		if e.ValidateDecision(d) == nil {
			t.Fatal("processing original granted", permission)
		}
	}
	d := e.Decision
	d.ProcessingOriginal = false
	if e.ValidateDecision(d) == nil {
		t.Fatal("implicit AVI acquisition allowed")
	}
	e.Policy = Stream
	if e.ValidateDecision(e.Decision) == nil {
		t.Fatal("stream processing allowed")
	}
	e.Policy = Local
	e.Episode.SourceCollections = []string{DemocracyNow}
	if e.ValidateDecision(e.Decision) == nil {
		t.Fatal("relabeled Democracy processing allowed")
	}
}
