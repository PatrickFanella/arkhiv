package app

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
)

type videoFilters struct {
	from, to, subject, language, creator string
	min, max                             *float64
}

func parseVideoFilters(q url.Values) (videoFilters, error) {
	f := videoFilters{from: q.Get("date_from"), to: q.Get("date_to"), subject: q.Get("subject"), language: q.Get("language"), creator: q.Get("creator")}
	for _, date := range []string{f.from, f.to} {
		if date != "" {
			if _, err := time.Parse("2006-01-02", date); err != nil {
				return f, fmt.Errorf("dates must use YYYY-MM-DD")
			}
		}
	}
	if f.from != "" && f.to != "" && f.from > f.to {
		return f, fmt.Errorf("date range is reversed")
	}
	if year := q.Get("year"); year != "" {
		if _, err := time.Parse("2006", year); err != nil {
			return f, fmt.Errorf("year must use YYYY")
		}
	}
	for key, target := range map[string]**float64{"duration_min": &f.min, "duration_max": &f.max} {
		if q.Get(key) == "" {
			continue
		}
		n, err := strconv.ParseFloat(q.Get(key), 64)
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n < 0 {
			return f, fmt.Errorf("duration bounds must be finite nonnegative seconds")
		}
		*target = &n
	}
	if f.min != nil && f.max != nil && *f.min > *f.max {
		return f, fmt.Errorf("duration range is reversed")
	}
	for _, v := range []string{f.subject, f.language, f.creator, q.Get("q")} {
		if len(v) > 500 {
			return f, fmt.Errorf("filter exceeds 500 bytes")
		}
	}
	return f, nil
}

func (f videoFilters) matches(e domain.Episode) bool {
	// Partial dates represent uncertain intervals. Date-range matching accepts
	// only intervals completely contained in the requested bounds.
	if f.from != "" || f.to != "" {
		lo, hi := e.Date, e.Date
		switch len(e.Date) {
		case 4:
			lo += "-01-01"
			hi += "-12-31"
		case 7:
			month, err := time.Parse("2006-01", e.Date)
			if err != nil {
				return false
			}
			lo += "-01"
			hi = month.AddDate(0, 1, -1).Format("2006-01-02")
		case 10:
		default:
			return false
		}
		if f.from != "" && lo < f.from || f.to != "" && hi > f.to {
			return false
		}
	}
	contains := func(values []string, wanted string) bool {
		if wanted == "" {
			return true
		}
		for _, v := range values {
			if strings.EqualFold(v, wanted) {
				return true
			}
		}
		return false
	}
	if !contains(e.Subjects, f.subject) || !contains(e.Languages, f.language) {
		return false
	}
	if f.creator != "" && !strings.EqualFold(e.Creator, f.creator) {
		return false
	}
	if f.min != nil || f.max != nil {
		if e.Video == nil || !domain.Duration(e.Video.Duration) {
			return false
		}
		if f.min != nil && e.Video.Duration < *f.min || f.max != nil && e.Video.Duration > *f.max {
			return false
		}
	}
	return true
}
