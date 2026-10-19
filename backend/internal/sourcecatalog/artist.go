package sourcecatalog

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type ArtistSource struct {
	URL       string    `json:"url"`
	Raw       string    `json:"raw_base64"`
	FetchedAt time.Time `json:"fetched_at"`
}

type ArtistBiography struct {
	Text    string `json:"text"`
	Source  string `json:"source"`
	License string `json:"license"`
}

type ArtistPhoto struct {
	URL        string `json:"url"`
	Source     string `json:"source"`
	Credit     string `json:"credit"`
	License    string `json:"license"`
	LicenseURL string `json:"license_url"`
}

type ArtistBundle struct {
	MusicBrainzID string           `json:"musicbrainz_id"`
	Biography     *ArtistBiography `json:"biography,omitempty"`
	Photo         *ArtistPhoto     `json:"photo,omitempty"`
	Sources       []ArtistSource   `json:"sources"`
}

var artistMBID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func externalArtistURL(raw string, hosts ...string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || len(raw) > 2000 {
		return false
	}
	for _, host := range hosts {
		if u.Hostname() == host {
			return true
		}
	}
	return false
}

// ImportArtist binds an explicitly verified source artist to an MBID. Public
// page reads never search for matches or fetch third-party information.
func (s *Store) ImportArtist(ctx context.Context, artist string, bundle ArtistBundle, notes string) error {
	if err := ValidateArtist(artist); err != nil {
		return err
	}
	if !artistMBID.MatchString(bundle.MusicBrainzID) || len(bundle.Sources) < 1 || len(bundle.Sources) > 6 || strings.TrimSpace(notes) == "" || len(notes) > 4000 {
		return fmt.Errorf("verified MusicBrainz identity, bounded sources and review notes required")
	}
	if b := bundle.Biography; b != nil && (len(b.Text) > 16000 || strings.TrimSpace(b.Text) == "" || b.License != "CC BY-SA 4.0" || !externalArtistURL(b.Source, "en.wikipedia.org")) {
		return fmt.Errorf("invalid attributed biography")
	}
	if p := bundle.Photo; p != nil && (!externalArtistURL(p.URL, "upload.wikimedia.org") || !externalArtistURL(p.Source, "commons.wikimedia.org") || p.Credit == "" || len(p.Credit) > 2000 || p.License == "" || len(p.License) > 200 || !externalArtistURL(p.LicenseURL, "creativecommons.org", "commons.wikimedia.org")) {
		return fmt.Errorf("invalid attributed artist photograph")
	}
	var sourceCount int
	if err := s.Core.Pool.QueryRow(ctx, `SELECT count(*) FROM source_catalog_items WHERE collection_id=$1 AND effective->>'creator'=$2`, s.Collection, artist).Scan(&sourceCount); err != nil {
		return err
	}
	if sourceCount == 0 {
		return fmt.Errorf("artist is not present in this collection")
	}
	type retained struct {
		source ArtistSource
		raw    []byte
		hash   string
	}
	retainedSources := []retained{}
	var mb map[string]any
	for _, source := range bundle.Sources {
		if !externalArtistURL(source.URL, "musicbrainz.org", "www.wikidata.org", "en.wikipedia.org", "commons.wikimedia.org") || source.FetchedAt.IsZero() || source.FetchedAt.After(time.Now().Add(5*time.Minute)) {
			return fmt.Errorf("invalid artist source provenance")
		}
		raw, err := base64.StdEncoding.DecodeString(source.Raw)
		if err != nil || len(raw) > 2<<20 || !json.Valid(raw) {
			return fmt.Errorf("invalid or oversized artist snapshot")
		}
		hash := sha256.Sum256(raw)
		retainedSources = append(retainedSources, retained{source, raw, hex.EncodeToString(hash[:])})
		u, _ := url.Parse(source.URL)
		if u.Hostname() == "musicbrainz.org" && u.Path == "/ws/2/artist/"+bundle.MusicBrainzID {
			if err = json.Unmarshal(raw, &mb); err != nil {
				return err
			}
		}
	}
	if mb == nil || mb["id"] != bundle.MusicBrainzID || Text(mb["name"]) == "" {
		return fmt.Errorf("retained MusicBrainz response does not match the reviewed identity")
	}
	profile := map[string]any{"musicbrainz_id": bundle.MusicBrainzID, "musicbrainz_url": "https://musicbrainz.org/artist/" + bundle.MusicBrainzID, "name": mb["name"], "type": mb["type"], "country": mb["country"], "area": mb["area"], "begin-area": mb["begin-area"], "life-span": mb["life-span"], "disambiguation": mb["disambiguation"], "aliases": mb["aliases"], "biography": bundle.Biography, "photo": bundle.Photo}
	payload, _ := json.Marshal(profile)
	tx, err := s.Core.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	hashes := []string{}
	for _, retained := range retainedSources {
		hashes = append(hashes, retained.hash)
		if _, err = tx.Exec(ctx, `INSERT INTO source_artist_snapshots(sha256,url,raw,fetched_at) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, retained.hash, retained.source.URL, retained.raw, retained.source.FetchedAt); err != nil {
			return err
		}
	}
	sources, _ := json.Marshal(hashes)
	if _, err = tx.Exec(ctx, `INSERT INTO source_artist_profiles(collection_id,source_artist,musicbrainz_id,profile,source_hashes,notes) VALUES($1,lower($2),$3,$4,$5,$6)
 ON CONFLICT(collection_id,source_artist) DO UPDATE SET musicbrainz_id=excluded.musicbrainz_id,profile=excluded.profile,source_hashes=excluded.source_hashes,notes=excluded.notes,updated_at=now()`, s.Collection, artist, bundle.MusicBrainzID, payload, sources, notes); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO source_artist_profile_history(collection_id,source_artist,musicbrainz_id,profile,source_hashes,notes) VALUES($1,$2,$3,$4,$5,$6)`, s.Collection, artist, bundle.MusicBrainzID, payload, sources, notes); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
