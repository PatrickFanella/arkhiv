package app

import (
	"fmt"
	"git.subcult.tv/subculture-collective/vhness/internal/music"
	"net/url"
	"testing"
)

func TestLegacyRecordingFiltersBeforePagination(t *testing.T) {
	entries := map[string]music.Entry{}
	for i := 0; i < 29; i++ {
		id := fmt.Sprintf("tape-%02d", i)
		recording := music.Recording{ID: id, Date: "1977-05-08", Venue: "Barton Hall", Taper: "Fixture taper"}
		if i == 27 {
			recording.Date = "1977"
		}
		if i == 28 {
			recording.Date = ""
		}
		entries[id] = music.Entry{Recording: recording}
	}
	query := url.Values{"q": {"fixture taper"}, "venue": {"barton"}, "date_from": {"1977-05-08"}, "date_to": {"1977-05-08"}, "page": {"2"}}
	items, total, page, err := legacyRecordingPage(entries, query)
	if err != nil || total != 27 || page != 2 || len(items) != 7 || items[0].Recording.ID != "tape-20" {
		t.Fatal("combined audio filters after pagination", items, total, page, err)
	}
	delete(query, "date_from")
	delete(query, "date_to")
	if _, total, _, err := legacyRecordingPage(entries, query); err != nil || total != 29 {
		t.Fatal("unknown source dates should remain without bounds", total, err)
	}
	for _, invalid := range []url.Values{{"year": {"77"}}, {"date_from": {"1977-02-30"}}, {"date_from": {"1977-05-09"}, "date_to": {"1977-05-08"}}, {"page": {"10001"}}} {
		if _, _, _, err := legacyRecordingPage(entries, invalid); err == nil {
			t.Fatal("accepted invalid audio filter", invalid)
		}
	}
}
