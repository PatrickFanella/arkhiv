// Source inventory contract retained from Dedhed (GPL-3.0-only).
package music

import "time"

type SourceItem struct {
	ID         string    `json:"id"`
	ShowDate   string    `json:"show_date"` // YYYY-MM-DD, or "" when the source month or day is unknown
	DateText   string    `json:"date_text"`
	Artist     string    `json:"artist"`
	Venue      string    `json:"venue"`
	Location   string    `json:"location"`
	Title      string    `json:"title"`
	Kind       string    `json:"kind"`
	Source     string    `json:"source"`
	Lineage    string    `json:"lineage"`
	Taper      string    `json:"taper"`
	Transferer string    `json:"transferer"`
	Downloads  int       `json:"downloads"`
	AvgRating  float64   `json:"avg_rating"`
	NumReviews int       `json:"num_reviews"`
	ItemSize   int64     `json:"item_size"`
	AddedAt    time.Time `json:"added_at"`
	Tracks     []Track   `json:"tracks"`
	Songs      string    `json:"songs"`
	Duration   float64   `json:"duration"`
	Warnings   []string  `json:"warnings"`
	Summary    []byte    `json:"-"`
	Snapshot   string    `json:"snapshot_sha256"`
	ObservedAt time.Time `json:"observed_at"`
}
