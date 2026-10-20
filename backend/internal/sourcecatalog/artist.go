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

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
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

// ArtistPhoto is the attributed photograph in a reviewed bundle. Data holds
// the image bytes fetched from DataURL; the engine stores renditions of them
// so pages do not load the provider's copy.
type ArtistPhoto struct {
	URL        string `json:"url"`
	Source     string `json:"source"`
	Credit     string `json:"credit"`
	License    string `json:"license"`
	LicenseURL string `json:"license_url"`
	DataURL    string `json:"data_url,omitempty"`
	Data       string `json:"data_base64,omitempty"`
}

// ArtistBundle binds a source artist to a reviewed MusicBrainz identity, to
// the Internet Archive collection that holds the artist's recordings, or both.
type ArtistBundle struct {
	MusicBrainzID     string           `json:"musicbrainz_id,omitempty"`
	ArchiveCollection string           `json:"archive_collection,omitempty"`
	Biography         *ArtistBiography `json:"biography,omitempty"`
	Photo             *ArtistPhoto     `json:"photo,omitempty"`
	Sources           []ArtistSource   `json:"sources"`
}

// Text and images published on an Internet Archive collection page carry no
// separate license. They are labelled as the collection's own material.
const (
	ArchiveDescriptionLicense = "Internet Archive collection description"
	ArchiveImageLicense       = "Internet Archive collection image"
)

func archiveDetails(collection string) string { return "https://archive.org/details/" + collection }

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

func nullable(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func containsText(value any, want string) bool {
	switch v := value.(type) {
	case string:
		return v == want
	case []any:
		for _, item := range v {
			if item == want {
				return true
			}
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
	mbid, archive := bundle.MusicBrainzID, bundle.ArchiveCollection
	if (mbid == "" && archive == "") || (mbid != "" && !artistMBID.MatchString(mbid)) || (archive != "" && (!domain.ValidID(archive) || len(archive) > 100)) || len(bundle.Sources) < 1 || len(bundle.Sources) > 8 || strings.TrimSpace(notes) == "" || len(notes) > 4000 {
		return fmt.Errorf("verified MusicBrainz or archive collection identity, bounded sources and review notes required")
	}
	if b := bundle.Biography; b != nil {
		wikipedia := b.License == "CC BY-SA 4.0" && externalArtistURL(b.Source, "en.wikipedia.org")
		collectionPage := archive != "" && b.License == ArchiveDescriptionLicense && b.Source == archiveDetails(archive)
		if len(b.Text) > 16000 || strings.TrimSpace(b.Text) == "" || !(wikipedia || collectionPage) {
			return fmt.Errorf("invalid attributed biography")
		}
	}
	var display, thumbnail ArtistMedia
	var originalHash string
	if p := bundle.Photo; p != nil {
		commons := externalArtistURL(p.URL, "upload.wikimedia.org") && externalArtistURL(p.Source, "commons.wikimedia.org") && p.License != "" && len(p.License) <= 200 && externalArtistURL(p.LicenseURL, "creativecommons.org", "commons.wikimedia.org") && (p.DataURL == "" || externalArtistURL(p.DataURL, "upload.wikimedia.org"))
		collectionImage := archive != "" && strings.HasPrefix(p.URL, "https://archive.org/download/"+archive+"/") && externalArtistURL(p.URL, "archive.org") && p.Source == archiveDetails(archive) && p.License == ArchiveImageLicense && p.LicenseURL == "" && p.DataURL == p.URL && p.Data != ""
		if p.Credit == "" || len(p.Credit) > 2000 || !(commons || collectionImage) || (p.Data == "") != (p.DataURL == "") {
			return fmt.Errorf("invalid attributed artist photograph")
		}
		if p.Data != "" {
			raw, err := base64.StdEncoding.DecodeString(p.Data)
			if err != nil {
				return fmt.Errorf("invalid artist image encoding")
			}
			if display, thumbnail, err = artistRenditions(raw); err != nil {
				return err
			}
			sum := sha256.Sum256(raw)
			originalHash = hex.EncodeToString(sum[:])
		}
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
	var mb, ia map[string]any
	for _, source := range bundle.Sources {
		if !externalArtistURL(source.URL, "musicbrainz.org", "www.wikidata.org", "en.wikipedia.org", "commons.wikimedia.org", "archive.org") || source.FetchedAt.IsZero() || source.FetchedAt.After(time.Now().Add(5*time.Minute)) {
			return fmt.Errorf("invalid artist source provenance")
		}
		raw, err := base64.StdEncoding.DecodeString(source.Raw)
		if err != nil || len(raw) > 2<<20 || !json.Valid(raw) {
			return fmt.Errorf("invalid or oversized artist snapshot")
		}
		hash := sha256.Sum256(raw)
		retainedSources = append(retainedSources, retained{source, raw, hex.EncodeToString(hash[:])})
		u, _ := url.Parse(source.URL)
		if mbid != "" && u.Hostname() == "musicbrainz.org" && u.Path == "/ws/2/artist/"+mbid {
			if err = json.Unmarshal(raw, &mb); err != nil {
				return err
			}
		}
		if archive != "" && u.Hostname() == "archive.org" && u.Path == "/metadata/"+archive {
			var response map[string]any
			if err = json.Unmarshal(raw, &response); err != nil {
				return err
			}
			ia, _ = response["metadata"].(map[string]any)
		}
	}
	if mbid != "" && (mb == nil || mb["id"] != mbid || Text(mb["name"]) == "") {
		return fmt.Errorf("retained MusicBrainz response does not match the reviewed identity")
	}
	profile := map[string]any{"biography": bundle.Biography}
	if archive != "" {
		// The collection must be this catalog's own subcollection and hold at
		// least one of the artist's visible recordings.
		if ia == nil || ia["identifier"] != archive || ia["mediatype"] != "collection" || Text(ia["title"]) == "" || !containsText(ia["collection"], s.Collection) {
			return fmt.Errorf("retained archive collection response does not match the reviewed identity")
		}
		var holds bool
		if err := s.Core.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM source_catalog_items i JOIN snapshots x ON x.sha256=i.source_hash
 WHERE i.collection_id=$1 AND NOT i.excluded AND lower(i.effective->>'creator')=lower($2)
 AND (x.parsed->'metadata'->'collection' ? $3 OR x.parsed->'metadata'->>'collection'=$3))`, s.Collection, artist, archive).Scan(&holds); err != nil {
			return err
		}
		if !holds {
			return fmt.Errorf("archive collection holds none of this artist's recordings")
		}
		profile["archive_collection"], profile["archive_url"], profile["name"] = archive, archiveDetails(archive), Text(ia["title"])
	}
	if mbid != "" {
		for key, value := range map[string]any{"musicbrainz_id": mbid, "musicbrainz_url": "https://musicbrainz.org/artist/" + mbid, "name": mb["name"], "type": mb["type"], "country": mb["country"], "area": mb["area"], "begin-area": mb["begin-area"], "life-span": mb["life-span"], "disambiguation": mb["disambiguation"], "aliases": mb["aliases"]} {
			profile[key] = value
		}
	}
	if p := bundle.Photo; p != nil {
		photo := map[string]any{"url": p.URL, "source": p.Source, "credit": p.Credit, "license": p.License, "license_url": p.LicenseURL}
		if p.Data != "" {
			photo["image"], photo["thumbnail"], photo["data_url"], photo["data_sha256"] = display, thumbnail, p.DataURL, originalHash
		}
		profile["photo"] = photo
	}
	payload, _ := json.Marshal(profile)
	tx, err := s.Core.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, media := range []ArtistMedia{display, thumbnail} {
		if media.data == nil {
			continue
		}
		if _, err = tx.Exec(ctx, `INSERT INTO source_artist_media(sha256,content_type,width,height,data) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`, media.SHA256, media.ContentType, media.Width, media.Height, media.data); err != nil {
			return err
		}
	}
	hashes := []string{}
	for _, retained := range retainedSources {
		hashes = append(hashes, retained.hash)
		if _, err = tx.Exec(ctx, `INSERT INTO source_artist_snapshots(sha256,url,raw,fetched_at) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, retained.hash, retained.source.URL, retained.raw, retained.source.FetchedAt); err != nil {
			return err
		}
	}
	sources, _ := json.Marshal(hashes)
	if _, err = tx.Exec(ctx, `INSERT INTO source_artist_profiles(collection_id,source_artist,musicbrainz_id,archive_collection,profile,source_hashes,notes) VALUES($1,lower($2),$3,$4,$5,$6,$7)
 ON CONFLICT(collection_id,source_artist) DO UPDATE SET musicbrainz_id=excluded.musicbrainz_id,archive_collection=excluded.archive_collection,profile=excluded.profile,source_hashes=excluded.source_hashes,notes=excluded.notes,updated_at=now()`, s.Collection, artist, nullable(mbid), nullable(archive), payload, sources, notes); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO source_artist_profile_history(collection_id,source_artist,musicbrainz_id,archive_collection,profile,source_hashes,notes) VALUES($1,$2,$3,$4,$5,$6,$7)`, s.Collection, artist, nullable(mbid), nullable(archive), payload, sources, notes); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
