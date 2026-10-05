// Adapted from Dedhed's retained show-index queries (GPL-3.0-only).
package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"git.subcult.tv/subculture-collective/vhness/internal/music"
	"github.com/jackc/pgx/v5"
	"regexp"
	"strings"
	"time"
)

var dateShape = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func (d *DB) retainedAudioPredicate(collection string) string {
	// Collection is an operator-configured safe identifier; queries still bind it
	// as a parameter separately below rather than interpolating request data.
	return "collection_id=$1 AND (NOT $2 OR id IN (SELECT v.source_item_id FROM public_catalog_representations v JOIN representation_transport_reviews t ON t.item_id=v.item_id AND t.representation_id=v.representation_id AND t.collection_id=v.collection_id WHERE v.collection_id=$1 AND v.media='audio' AND t.review->>'playback'='true'))"
}

// YearCount is the number of items and distinct show dates in one year.
type YearCount struct {
	Year  int `json:"year"`
	Items int `json:"items"`
	Dates int `json:"dates"`
}

// DayCount is the number of items on one show date, with its venue for labels.
type DayCount struct {
	Date  string `json:"date"`
	Items int    `json:"items"`
	Venue string `json:"venue"`
}

func (d *DB) LegacyYears(ctx context.Context, collection string) ([]YearCount, error) {
	rows, err := d.Pool.Query(ctx, `SELECT EXTRACT(YEAR FROM show_date)::int, count(*)::int, count(DISTINCT show_date)::int FROM retained_audio_items WHERE `+d.retainedAudioPredicate(collection)+` AND show_date IS NOT NULL GROUP BY 1 ORDER BY 1`, collection, d.PublicOnly)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[YearCount])
}

// Days lists every show date in a year with its item count and most common venue.
func (d *DB) LegacyDays(ctx context.Context, collection string, year int) ([]DayCount, error) {
	if year < 1960 || year > 2100 {
		return nil, fmt.Errorf("year out of range")
	}
	rows, err := d.Pool.Query(ctx, `SELECT to_char(show_date,'YYYY-MM-DD'), count(*)::int, mode() WITHIN GROUP (ORDER BY venue) FROM retained_audio_items WHERE `+d.retainedAudioPredicate(collection)+` AND show_date >= make_date($3,1,1) AND show_date < make_date($3+1,1,1) GROUP BY show_date ORDER BY show_date`, collection, d.PublicOnly, year)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[DayCount])
}

// OnThisDay lists every show date sharing a month and day, across years.
func (d *DB) LegacyOnThisDay(ctx context.Context, collection string, month, day int) ([]DayCount, error) {
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return nil, fmt.Errorf("month or day out of range")
	}
	rows, err := d.Pool.Query(ctx, `SELECT to_char(show_date,'YYYY-MM-DD'), count(*)::int, mode() WITHIN GROUP (ORDER BY venue) FROM retained_audio_items WHERE `+d.retainedAudioPredicate(collection)+` AND EXTRACT(MONTH FROM show_date)=$3 AND EXTRACT(DAY FROM show_date)=$4 GROUP BY show_date ORDER BY show_date`, collection, d.PublicOnly, month, day)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[DayCount])
}

const itemColumns = `id, coalesce(to_char(show_date,'YYYY-MM-DD'),''), date_text, artist, venue, location, title, kind, source, lineage, taper, transferer, downloads, avg_rating::float8, num_reviews, item_size, coalesce(added_at, 'epoch'::timestamptz), tracks, songs, duration, warnings, snapshot_sha256, observed_at`

func scanItem(rows pgx.CollectableRow) (music.SourceItem, error) {
	var it music.SourceItem
	var tracks, warnings []byte
	err := rows.Scan(&it.ID, &it.ShowDate, &it.DateText, &it.Artist, &it.Venue, &it.Location, &it.Title, &it.Kind, &it.Source, &it.Lineage, &it.Taper, &it.Transferer, &it.Downloads, &it.AvgRating, &it.NumReviews, &it.ItemSize, &it.AddedAt, &tracks, &it.Songs, &it.Duration, &warnings, &it.Snapshot, &it.ObservedAt)
	if err != nil {
		return it, err
	}
	if it.AddedAt.Unix() == 0 {
		it.AddedAt = time.Time{}
	}
	if err = json.Unmarshal(tracks, &it.Tracks); err != nil {
		return it, err
	}
	if it.Tracks == nil {
		it.Tracks = []music.Track{}
	}
	if err = json.Unmarshal(warnings, &it.Warnings); err != nil {
		return it, err
	}
	if it.Warnings == nil {
		it.Warnings = []string{}
	}
	return it, nil
}

// ItemsOn returns every item for one show date, best rated and most downloaded first.
func (d *DB) LegacyItemsOn(ctx context.Context, collection string, date string) ([]music.SourceItem, error) {
	if !dateShape.MatchString(date) {
		return nil, fmt.Errorf("invalid date")
	}
	rows, err := d.Pool.Query(ctx, `SELECT `+itemColumns+` FROM retained_audio_items WHERE `+d.retainedAudioPredicate(collection)+` AND show_date=$3::date ORDER BY avg_rating DESC, downloads DESC, id`, collection, d.PublicOnly, date)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanItem)
}

// SearchHit is one show date matched by a search, with its items.
type SearchHit struct {
	Date  string             `json:"date"`
	Venue string             `json:"venue"`
	Items []music.SourceItem `json:"items"`
}

// Search runs a full-text query over titles, venues, places, credits and song
// titles, grouping hits by show date. Dates without a month or day are excluded.
func (d *DB) LegacySearch(ctx context.Context, collection string, q, year string, page, size int) ([]SearchHit, int, error) {
	if page < 1 || size < 1 || size > 100 {
		return nil, 0, fmt.Errorf("invalid page")
	}
	words := []string{}
	for _, w := range strings.Fields(strings.ToLower(q)) {
		w = regexp.MustCompile(`[^\p{L}\p{N}]+`).ReplaceAllString(w, "")
		if w != "" {
			words = append(words, w+":*")
		}
	}
	if len(words) > 12 {
		return nil, 0, fmt.Errorf("query too long")
	}
	query := strings.Join(words, " & ")
	where := d.retainedAudioPredicate(collection) + ` AND show_date IS NOT NULL`
	args := []any{collection, d.PublicOnly}
	if query != "" {
		args = append(args, query)
		where += fmt.Sprintf(` AND search @@ to_tsquery('simple', $%d)`, len(args))
	}
	if year != "" {
		if !regexp.MustCompile(`^\d{4}$`).MatchString(year) {
			return nil, 0, fmt.Errorf("invalid year")
		}
		args = append(args, year)
		where += fmt.Sprintf(` AND EXTRACT(YEAR FROM show_date)=$%d::int`, len(args))
	}
	var total int
	if err := d.Pool.QueryRow(ctx, `SELECT count(DISTINCT show_date) FROM retained_audio_items WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := d.Pool.Query(ctx, `SELECT `+itemColumns+` FROM retained_audio_items WHERE show_date IN (SELECT show_date FROM retained_audio_items WHERE `+where+fmt.Sprintf(` GROUP BY show_date ORDER BY show_date LIMIT $%d OFFSET $%d) AND `, len(args)-1, len(args))+where+` ORDER BY show_date, avg_rating DESC, downloads DESC, id`, args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, scanItem)
	if err != nil {
		return nil, 0, err
	}
	hits := []SearchHit{}
	for _, it := range items {
		if len(hits) == 0 || hits[len(hits)-1].Date != it.ShowDate {
			hits = append(hits, SearchHit{Date: it.ShowDate, Venue: it.Venue})
		}
		hits[len(hits)-1].Items = append(hits[len(hits)-1].Items, it)
	}
	return hits, total, nil
}

// ItemCount reports how many items are indexed and how many have fetched metadata.
func (d *DB) LegacyItemCount(ctx context.Context, collection string) (int, int, error) {
	var items, fetched int
	err := d.Pool.QueryRow(ctx, `SELECT count(*)::int, count(*) FILTER (WHERE snapshot_sha256<>'')::int FROM retained_audio_items WHERE collection_id=$1`, collection).Scan(&items, &fetched)
	return items, fetched, err
}
