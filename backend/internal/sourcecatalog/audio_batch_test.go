package sourcecatalog

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
)

func TestBatchAudioCoverageResumeAndIndependentDecision(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	for _, id := range []string{"nested", "unsupported", "held", "mpeg4"} {
		raw := fixture(id, "Room")
		if id == "nested" {
			raw = []byte(strings.ReplaceAll(string(raw), "song.mp3", "disc 1/01  Song.mp3"))
		}
		if id == "unsupported" {
			raw = []byte(strings.ReplaceAll(string(raw), "song.mp3", "song.shn"))
		}
		if id == "mpeg4" {
			raw = []byte(strings.ReplaceAll(string(raw), "song.mp3", "song.mp4"))
		}
		if err := s.Import(ctx, id, raw, 200, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.ReviewAudio(ctx, "held", "Independent manual decision"); err != nil {
		t.Fatal(err)
	}
	item := "audio-" + domain.Digest([]byte("etree\x00held"))[:24]
	entry, err := s.Core.Transport(ctx, item, "audio", "etree")
	if err != nil {
		t.Fatal(err)
	}
	entry.Review.Playback = false
	if err = s.Core.ReviewTransport(ctx, item, "audio", "etree", entry.Review, domain.PreservationApproval{}); err != nil {
		t.Fatal(err)
	}
	c, err := s.ReviewAudioBatch(ctx, 100, "Owner approved private source playback")
	if err != nil || c.Reviewed != 2 || c.Unsupported != 1 || c.Held != 1 || c.Pending != 0 {
		t.Fatal(c, err)
	}
	nested := "audio-" + domain.Digest([]byte("etree\x00nested"))[:24]
	e, err := s.Core.Transport(ctx, nested, "audio", "etree")
	if err != nil {
		t.Fatal(err)
	}
	if err = domain.AuthorizeTransport(e.Collection, e.Representation, e.Raw, e.Review, e.Preservation, domain.PlayAction, "disc 1/01  Song.mp3"); err != nil {
		t.Fatal(err)
	}
	for _, action := range []domain.Action{domain.DownloadAction, domain.AcquireAction, domain.RelayAction, domain.ScheduleAction} {
		if domain.AuthorizeTransport(e.Collection, e.Representation, e.Raw, e.Review, e.Preservation, action, e.Representation.File) == nil {
			t.Fatal("batch granted unrelated action", action)
		}
	}
	if e.Review.Public {
		t.Fatal("batch published recording")
	}
	before := 0
	s.Core.Pool.QueryRow(ctx, `SELECT sum(attempts) FROM source_catalog_audio_reviews`).Scan(&before)
	c, err = s.ReviewAudioBatch(ctx, 100, "Repeated approval")
	var after int
	s.Core.Pool.QueryRow(ctx, `SELECT sum(attempts) FROM source_catalog_audio_reviews`).Scan(&after)
	if err != nil || before != after {
		t.Fatal("repeat batch re-reviewed completed sources", c, err, before, after)
	}
	// Changed metadata can renew the batch's own decision, but cannot renew a
	// separately revoked or independently reviewed source.
	raw := []byte(strings.ReplaceAll(string(fixture("nested", "Changed room")), "song.mp3", "disc 1/01  Song.mp3"))
	if err = s.Import(ctx, "nested", raw, 200, "", ""); err != nil {
		t.Fatal(err)
	}
	c, err = s.ReviewAudioBatch(ctx, 100, "Owner approved private source playback")
	if err != nil || c.Reviewed != 2 {
		t.Fatal("changed batch source did not renew", c, err)
	}
	e, err = s.Core.Transport(ctx, nested, "audio", "etree")
	if err != nil {
		t.Fatal(err)
	}
	e.Review.Playback = false
	e.Review.Notes = "Owner revoked playback"
	if err = s.Core.ReviewTransport(ctx, nested, "audio", "etree", e.Review, domain.PreservationApproval{}); err != nil {
		t.Fatal(err)
	}
	if err = s.Import(ctx, "nested", []byte(strings.ReplaceAll(string(raw), "Changed room", "New room")), 200, "", ""); err != nil {
		t.Fatal(err)
	}
	c, err = s.ReviewAudioBatch(ctx, 100, "Owner approved private source playback")
	if err != nil || c.Reviewed != 1 || c.Held != 2 {
		t.Fatal("batch overwrote independent revocation", c, err)
	}
	if _, err = s.ReviewAudioBatch(ctx, 100, ""); err == nil {
		t.Fatal("implicit approval accepted")
	}
	if _, err = s.ReviewAudioBatch(ctx, 1001, "notes"); err == nil {
		t.Fatal("unbounded review accepted")
	}
	detail, err := s.Detail(ctx, "nested")
	if err != nil {
		t.Fatal(err)
	}
	var bindings []json.RawMessage
	json.Unmarshal(detail["playback_bindings"].(json.RawMessage), &bindings)
	if len(bindings) != 0 {
		t.Fatal("held source remained playable")
	}
}
