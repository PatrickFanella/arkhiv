package sourcecatalog

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestCalendarRangesLeapDaysAndExclusions(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	for id, date := range map[string]string{"leap": "2000-02-29", "leap2": "2004-02-29", "invalid": "2001-02-29", "other": "2010-02-28", "excluded": "2008-02-29"} {
		raw := []byte(strings.ReplaceAll(string(fixture(id, "Room")), "2003-09-23", date))
		if err := s.Import(ctx, id, raw, 200, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	listed := false
	if err := s.SetListing(ctx, ListingDecision{Identifiers: []string{"excluded"}, Listed: &listed}, "Exclude fixture"); err != nil {
		t.Fatal(err)
	}
	calendar, err := s.Calendar(ctx, CalendarQuery{Month: "2"})
	if err != nil || len(calendar.Days) != 3 {
		t.Fatal(calendar, err)
	}
	calendar, err = s.Calendar(ctx, CalendarQuery{Month: "02", YearFrom: "2000", YearTo: "2004"})
	if err != nil || len(calendar.Days) != 2 {
		t.Fatal(calendar, err)
	}
	calendar, err = s.Calendar(ctx, CalendarQuery{Month: "2", YearFrom: "2004", YearTo: "2004"})
	if err != nil || len(calendar.Days) != 1 || calendar.Days[0].Date != "2004-02-29" {
		t.Fatal(calendar, err)
	}
	monthPage, monthErr := s.Search(ctx, Query{Month: "02", YearFrom: "2004", YearTo: "2004", Sort: "date_desc"})
	if monthErr != nil || monthPage.Total != 1 {
		t.Fatal(monthPage, monthErr)
	}
	page, err := s.Search(ctx, Query{MonthDay: "02-29", Sort: "date_desc"})
	if err != nil || page.Total != 2 || page.Items[0]["identifier"] != "leap2" {
		t.Fatal(page, err)
	}
	page, err = s.Search(ctx, Query{MonthDay: "02-29", YearFrom: "2000", YearTo: "2000", Sort: "date_desc"})
	if err != nil || page.Total != 1 {
		t.Fatal(page, err)
	}
	for _, q := range []Query{{MonthDay: "02-30"}, {MonthDay: "2-9"}, {YearFrom: "2004", YearTo: "2000"}, {YearFrom: "2004"}, {Sort: "unknown"}} {
		if q.Validate() == nil {
			t.Fatal("accepted invalid discovery filters", q)
		}
	}
	artist, err := s.Artist(ctx, "ween")
	if err != nil || artist["recordings"] != int64(4) || artist["first_recording"] != "2000-02-29" || artist["last_recording"] != "2010-02-28" {
		t.Fatal(artist, err)
	}
}

func TestDatePaginationUsesDateAndIdentifier(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	for i := 0; i < 102; i++ {
		id := fmt.Sprintf("show%03d", i)
		date := "2000-05-08"
		if i == 101 {
			date = "2001-05-08"
		}
		if err := s.Import(ctx, id, []byte(strings.ReplaceAll(string(fixture(id, "Room")), "2003-09-23", date)), 200, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	first, err := s.Search(ctx, Query{MonthDay: "05-08", Sort: "date_desc"})
	if err != nil || len(first.Items) != 100 || first.Items[0]["identifier"] != "show101" || first.Cursor == "" {
		t.Fatal(first, err)
	}
	second, err := s.Search(ctx, Query{MonthDay: "05-08", Sort: "date_desc", Cursor: first.Cursor})
	if err != nil || len(second.Items) != 2 || second.Items[0]["identifier"] != "show099" || second.Cursor != "" {
		t.Fatal(second, err)
	}
	if _, err = s.Search(ctx, Query{MonthDay: "05-09", Sort: "date_desc", Cursor: first.Cursor}); err == nil {
		t.Fatal("cursor accepted after date filter changed")
	}
}

func TestArtistEnrichmentRetainsSourcesAndRecordingIdentity(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	raw := fixture("show", "Original Room")
	if err := s.Import(ctx, "show", raw, 200, "", ""); err != nil {
		t.Fatal(err)
	}
	id := "00000000-0000-0000-0000-000000000001"
	mb := []byte(`{"id":"` + id + `","name":"Verified fixture artist","type":"Group","country":"US"}`)
	bundle := ArtistBundle{MusicBrainzID: id, Sources: []ArtistSource{{URL: "https://musicbrainz.org/ws/2/artist/" + id + "?fmt=json", Raw: base64.StdEncoding.EncodeToString(mb), FetchedAt: time.Now()}}}
	if err := s.ImportArtist(ctx, "Ween", bundle, "Verified fixture binding; test data only"); err != nil {
		t.Fatal(err)
	}
	artist, err := s.Artist(ctx, "WEEN")
	if err != nil || artist["profile"] == nil {
		t.Fatal(artist, err)
	}
	var profile map[string]any
	if err = json.Unmarshal(artist["profile"].(json.RawMessage), &profile); err != nil || profile["musicbrainz_id"] != id {
		t.Fatal(profile, err)
	}
	if retained, err := s.Raw(ctx, "show"); err != nil || string(retained) != string(raw) {
		t.Fatal("artist enrichment changed recording metadata", err)
	}
	var saved []byte
	if err = s.Core.Pool.QueryRow(ctx, `SELECT raw FROM source_artist_snapshots`).Scan(&saved); err != nil || string(saved) != string(mb) {
		t.Fatal("artist snapshot bytes changed", err)
	}
	bundle.MusicBrainzID = "00000000-0000-0000-0000-000000000002"
	if err = s.ImportArtist(ctx, "Ween", bundle, "Mismatched source"); err == nil {
		t.Fatal("mismatched MusicBrainz identity accepted")
	}
	if err = s.ImportArtist(ctx, "Absent artist", bundle, "No recordings"); err == nil {
		t.Fatal("foreign artist bound")
	}
	if photos, err := s.ArtistPhotos(ctx); err != nil || len(photos) != 0 {
		t.Fatal("artist without a photograph listed", photos, err)
	}
	bundle.MusicBrainzID = id
	bundle.Photo = &ArtistPhoto{URL: "https://upload.wikimedia.org/wikipedia/commons/a/ab/Fixture.jpg", Source: "https://commons.wikimedia.org/wiki/File:Fixture.jpg", Credit: "Fixture photographer", License: "CC BY-SA 4.0", LicenseURL: "https://creativecommons.org/licenses/by-sa/4.0"}
	if err = s.ImportArtist(ctx, "Ween", bundle, "Verified fixture photograph; test data only"); err != nil {
		t.Fatal(err)
	}
	photos, err := s.ArtistPhotos(ctx)
	if err != nil || len(photos) != 1 || photos[0].Artist != "ween" {
		t.Fatal(photos, err)
	}
	var photo ArtistPhoto
	if err = json.Unmarshal(photos[0].Photo, &photo); err != nil || photo.Credit != "Fixture photographer" {
		t.Fatal(photo, err)
	}
	listed := false
	if err = s.SetListing(ctx, ListingDecision{Identifiers: []string{"show"}, Listed: &listed}, "Exclude only recording"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Artist(ctx, "Ween"); err == nil {
		t.Fatal("excluded-only artist leaked")
	}
	if photos, err = s.ArtistPhotos(ctx); err != nil || len(photos) != 0 {
		t.Fatal("excluded-only artist photograph leaked", photos, err)
	}
}
