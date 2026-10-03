package domain

import (
	"testing"
	"time"
)

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
