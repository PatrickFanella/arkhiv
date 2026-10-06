package app

import (
	"net/url"
	"testing"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
)

func TestVideoFilters(t *testing.T) {
	for _, query := range []string{"date_from=2026-02-30", "date_from=2026-10-02&date_to=2026-10-01", "duration_min=NaN", "duration_max=Inf", "duration_min=-1", "duration_min=20&duration_max=10", "year=26"} {
		q, _ := url.ParseQuery(query)
		if _, err := parseVideoFilters(q); err == nil {
			t.Fatal("accepted", query)
		}
	}
	q, _ := url.ParseQuery("date_from=2026-01-01&date_to=2026-12-31&subject=Science%20%26%20nature&language=EN&creator=Teacher&duration_min=5&duration_max=15")
	f, err := parseVideoFilters(q)
	if err != nil {
		t.Fatal(err)
	}
	e := domain.Episode{Date: "2026", Subjects: []string{"Science & nature"}, Languages: []string{"en"}, Creator: "Teacher", Video: &domain.VideoDetails{Duration: 10}}
	if !f.matches(e) {
		t.Fatal("combined matching filters rejected")
	}
	e.Date = ""
	if f.matches(e) {
		t.Fatal("unknown date matched range")
	}
	e.Date = "2026-05"
	if !f.matches(e) {
		t.Fatal("contained partial month rejected")
	}
	f.from = "2026-05-10"
	if f.matches(e) {
		t.Fatal("uncertain partial month treated as exact date")
	}
	f.from = "2026-01-01"
	e.Video = nil
	if f.matches(e) {
		t.Fatal("missing duration matched")
	}
}
