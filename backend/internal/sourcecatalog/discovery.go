package sourcecatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var regexpMonth = regexp.MustCompile(`^(0[1-9]|1[0-2])$`)

func validCompleteDate(value string) bool {
	_, err := time.Parse("2006-01-02", value)
	return len(value) == 10 && err == nil
}

func validateDiscovery(day, from, to string) error {
	if day != "" {
		if len(day) != 5 || !validCompleteDate("2000-"+day) {
			return fmt.Errorf("use a valid month and day in MM-DD form")
		}
	}
	if (from == "") != (to == "") {
		return fmt.Errorf("choose both ends of the year range or all years")
	}
	if from != "" && (!Year.MatchString(from) || !Year.MatchString(to) || from > to) {
		return fmt.Errorf("choose an ordered year range between 1900 and 2099")
	}
	return nil
}

func ValidateArtist(artist string) error {
	if strings.TrimSpace(artist) == "" || len(artist) > 300 {
		return fmt.Errorf("a bounded artist name is required")
	}
	return nil
}

type CalendarQuery struct{ Month, YearFrom, YearTo, Artist string }

func (q CalendarQuery) Validate() error {
	month, err := strconv.Atoi(q.Month)
	if err != nil || month < 1 || month > 12 {
		return fmt.Errorf("choose a month from 1 to 12")
	}
	if len(q.Artist) > 300 {
		return fmt.Errorf("artist name is too long")
	}
	return validateDiscovery("", q.YearFrom, q.YearTo)
}

type CalendarDay struct {
	Date       string `json:"date"`
	Recordings int64  `json:"recordings"`
}

type Calendar struct {
	Days     []CalendarDay `json:"days"`
	Month    string        `json:"month"`
	YearFrom string        `json:"year_from"`
	YearTo   string        `json:"year_to"`
}

func (s *Store) Calendar(ctx context.Context, q CalendarQuery) (Calendar, error) {
	result := Calendar{Days: []CalendarDay{}, Month: q.Month, YearFrom: q.YearFrom, YearTo: q.YearTo}
	if err := q.Validate(); err != nil {
		return result, err
	}
	month, _ := strconv.Atoi(q.Month)
	rows, err := s.Core.Pool.Query(ctx, `SELECT effective->>'date',count(*) FROM source_catalog_items
 WHERE collection_id=$1 AND NOT excluded AND source_catalog_complete_date(effective->>'date')
 AND substring(effective->>'date',6,2)=$2
 AND ($3='' OR left(effective->>'date',4)>=$3) AND ($4='' OR left(effective->>'date',4)<=$4)
 AND ($5='' OR lower(effective->>'creator')=lower($5)) GROUP BY 1 ORDER BY 1`, s.Collection, fmt.Sprintf("%02d", month), q.YearFrom, q.YearTo, q.Artist)
	if err != nil {
		return result, err
	}
	result.Days, err = pgx.CollectRows(rows, pgx.RowToStructByPos[CalendarDay])
	return result, err
}

func (s *Store) Artist(ctx context.Context, artist string) (map[string]any, error) {
	if err := ValidateArtist(artist); err != nil {
		return nil, err
	}
	var count int64
	var first, last, name string
	err := s.Core.Pool.QueryRow(ctx, `SELECT count(*),COALESCE(min(effective->>'date') FILTER(WHERE source_catalog_complete_date(effective->>'date')),''),
 COALESCE(max(effective->>'date') FILTER(WHERE source_catalog_complete_date(effective->>'date')),''),COALESCE(min(effective->>'creator'),'')
 FROM source_catalog_items WHERE collection_id=$1 AND NOT excluded AND lower(effective->>'creator')=lower($2)`, s.Collection, artist).Scan(&count, &first, &last, &name)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, pgx.ErrNoRows
	}
	var profile []byte
	var updated time.Time
	err = s.Core.Pool.QueryRow(ctx, `SELECT profile,updated_at FROM source_artist_profiles WHERE collection_id=$1 AND source_artist=lower($2)`, s.Collection, name).Scan(&profile, &updated)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}
	result := map[string]any{"artist": name, "recordings": count, "first_recording": first, "last_recording": last, "profile": nil}
	if len(profile) > 0 {
		result["profile"] = json.RawMessage(profile)
		result["profile_updated_at"] = updated
	}
	return result, nil
}

type ArtistPhotoEntry struct {
	Artist string          `json:"artist"`
	Photo  json.RawMessage `json:"photo"`
}

// ArtistPhotos lists attributed photographs for bound artists that still have
// visible recordings. Artist is the lowercased source artist key; consumers
// match it case-insensitively against recording creators.
func (s *Store) ArtistPhotos(ctx context.Context) ([]ArtistPhotoEntry, error) {
	rows, err := s.Core.Pool.Query(ctx, `SELECT p.source_artist,p.profile->'photo' FROM source_artist_profiles p
 WHERE p.collection_id=$1 AND jsonb_typeof(p.profile->'photo')='object'
 AND EXISTS(SELECT 1 FROM source_catalog_items i WHERE i.collection_id=p.collection_id AND NOT i.excluded AND lower(i.effective->>'creator')=p.source_artist)
 ORDER BY p.source_artist`, s.Collection)
	if err != nil {
		return nil, err
	}
	entries, err := pgx.CollectRows(rows, pgx.RowToStructByPos[ArtistPhotoEntry])
	if entries == nil {
		entries = []ArtistPhotoEntry{}
	}
	return entries, err
}
