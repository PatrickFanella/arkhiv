// Dedhed contract regressions retained from the pinned GPL donor.
package music

import (
	"math"
	"testing"
	"time"
)

func TestRecordingRejectsNonFiniteTrackStart(t *testing.T) {
	r := Recording{ID: "recording", ConcertID: "concert", Duration: 1, Tracks: []Track{{Position: 0, Start: math.NaN(), Duration: 1}}}
	if r.Validate() == nil {
		t.Fatal("NaN bypassed ordered track validation")
	}
	r.Tracks[0].Start = math.Inf(1)
	if r.Validate() == nil {
		t.Fatal("infinite track start accepted")
	}
}

func TestNonConcertRecordingHasNoInventedConcertIdentity(t *testing.T) {
	r := Recording{ID: "lecture", Kind: "spoken-word", Duration: 1, Tracks: []Track{{Position: 0, Duration: 1}}}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	r.Kind = "concert"
	if r.Validate() == nil {
		t.Fatal("concert without grouping identity accepted")
	}
}

func TestScheduleBoundariesAndMissingSources(t *testing.T) {
	start := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	r := Recording{ID: "a", ConcertID: "concert", Duration: 20, Tracks: []Track{{Position: 0, Duration: 10}, {Position: 1, Start: 10, Duration: 10}}}
	e := Entry{Recording: r, Review: Review{Published: true, Playback: true}, Media: "a.wav"}
	records := map[string]Entry{"a": e}
	s := Schedule{Revision: "1", Programs: []Program{{RecordingID: "a", Start: start, Duration: 20}, {RecordingID: "a", Start: start.Add(20 * time.Second), Duration: 20}}}
	for _, tc := range []struct {
		sec    int
		track  int
		offset float64
	}{{0, 0, 0}, {10, 1, 10}, {19, 1, 19}, {20, 0, 0}, {39, 1, 19}} {
		n, err := s.At(start.Add(time.Duration(tc.sec)*time.Second), records)
		if err != nil || n.Track != tc.track || n.Offset != tc.offset {
			t.Fatalf("%+v: %+v %v", tc, n, err)
		}
	}
	if _, err := s.At(start.Add(40*time.Second), records); err == nil {
		t.Fatal("exhausted schedule accepted")
	}
	if _, err := s.At(start, map[string]Entry{}); err == nil {
		t.Fatal("missing source accepted")
	}
	s.Programs[1].Start = start.Add(19 * time.Second)
	if s.Validate() == nil {
		t.Fatal("overlap accepted")
	}
}
func TestReviewedSets(t *testing.T) {
	for _, s := range [][]Set{{{Title: "Disc", First: 0, Last: 2}}, {{Title: "I", First: 0, Last: 2, Reviewed: true}, {Title: "II", First: 2, Last: 4, Reviewed: true}}, {{Title: "I", First: 0, Last: 5, Reviewed: true}}} {
		if ValidateSets(s, 5) == nil {
			t.Fatal("invalid boundary accepted")
		}
	}
	if err := ValidateSets([]Set{{Title: "I", First: 0, Last: 2, Reviewed: true}, {Title: "II", First: 3, Last: 4, Reviewed: true}}, 5); err != nil {
		t.Fatal(err)
	}
}
