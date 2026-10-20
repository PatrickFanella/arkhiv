package sourcecatalog

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
	"time"
)

func testJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 90, 255})
		}
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, nil); err != nil {
		t.Fatal(err)
	}
	// Insert an EXIF segment carrying a location marker after the SOI marker.
	exif := append([]byte{0xff, 0xe1, 0x00, 0x14}, []byte("Exif\x00\x00GPS-MARKER-01")...)
	raw := out.Bytes()
	return append(append(append([]byte{}, raw[:2]...), exif...), raw[2:]...)
}

func TestArtistRenditionsBoundSizeAndDropMetadata(t *testing.T) {
	raw := testJPEG(t, 2000, 1000)
	if !bytes.Contains(raw, []byte("GPS-MARKER-01")) {
		t.Fatal("fixture lacks embedded metadata")
	}
	display, thumbnail, err := artistRenditions(raw)
	if err != nil {
		t.Fatal(err)
	}
	if display.Width != ArtistDisplayWidth || display.Height != 640 || thumbnail.Width != ArtistThumbnailWidth || thumbnail.Height != 165 {
		t.Fatal(display, thumbnail)
	}
	for _, media := range []ArtistMedia{display, thumbnail} {
		if media.ContentType != "image/jpeg" || bytes.Contains(media.data, []byte("GPS-MARKER-01")) || !ValidMediaID(media.SHA256) {
			t.Fatal("rendition kept metadata or lacks an address", media.ContentType)
		}
	}
	small, smallThumb, err := artistRenditions(testJPEG(t, 200, 100))
	if err != nil || small.Width != 200 || smallThumb.Width != 200 {
		t.Fatal("small image was enlarged", small, smallThumb, err)
	}
	logo := image.NewNRGBA(image.Rect(0, 0, 400, 400))
	logo.Set(10, 10, color.NRGBA{255, 0, 0, 255})
	var out bytes.Buffer
	png.Encode(&out, logo)
	display, thumbnail, err = artistRenditions(out.Bytes())
	if err != nil || display.ContentType != "image/png" || thumbnail.ContentType != "image/png" || thumbnail.Width != ArtistThumbnailWidth {
		t.Fatal("transparent logo was flattened", display, thumbnail, err)
	}
	for _, bad := range [][]byte{nil, []byte("<svg xmlns='http://www.w3.org/2000/svg'/>"), testJPEG(t, 8, 8)} {
		if _, _, err = artistRenditions(bad); err == nil {
			t.Fatal("accepted unsupported image")
		}
	}
}

func TestArchiveCollectionArtistStoresMedia(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	raw := []byte(strings.Replace(string(fixture("show", "Room")), `"collection":["etree"]`, `"collection":["WeenCollection","etree"]`, 1))
	if err := s.Import(ctx, "show", raw, 200, "", ""); err != nil {
		t.Fatal(err)
	}
	meta := func(id string) ArtistSource {
		body := `{"metadata":{"identifier":"` + id + `","mediatype":"collection","title":"Ween (collection)","collection":["etree"]}}`
		return ArtistSource{URL: "https://archive.org/metadata/" + id, Raw: base64.StdEncoding.EncodeToString([]byte(body)), FetchedAt: time.Now()}
	}
	image := base64.StdEncoding.EncodeToString(testJPEG(t, 2000, 1000))
	photo := func(id string) *ArtistPhoto {
		url := "https://archive.org/download/" + id + "/" + id + "_itemimage.jpg"
		return &ArtistPhoto{URL: url, DataURL: url, Data: image, Source: "https://archive.org/details/" + id, Credit: "Ween collection, Live Music Archive", License: ArchiveImageLicense}
	}
	bundle := ArtistBundle{ArchiveCollection: "WeenCollection", Sources: []ArtistSource{meta("WeenCollection")}, Photo: photo("WeenCollection"),
		Biography: &ArtistBiography{Text: "Ween is a band.", Source: "https://archive.org/details/WeenCollection", License: ArchiveDescriptionLicense}}
	other := bundle
	other.ArchiveCollection, other.Sources, other.Photo, other.Biography = "OtherCollection", []ArtistSource{meta("OtherCollection")}, photo("OtherCollection"), nil
	if err := s.ImportArtist(ctx, "Ween", other, "Collection without the artist's recordings"); err == nil {
		t.Fatal("bound a collection that holds none of the artist's recordings")
	}
	for name, mutate := range map[string]func(*ArtistBundle){
		"photo without data":       func(b *ArtistBundle) { p := *b.Photo; p.Data, p.DataURL = "", ""; b.Photo = &p },
		"foreign photo collection": func(b *ArtistBundle) { b.Photo = photo("OtherCollection") },
		"licensed as Wikipedia":    func(b *ArtistBundle) { bio := *b.Biography; bio.License = "CC BY-SA 4.0"; b.Biography = &bio },
		"no identity":              func(b *ArtistBundle) { b.ArchiveCollection = "" },
	} {
		candidate := bundle
		mutate(&candidate)
		if err := s.ImportArtist(ctx, "Ween", candidate, "Invalid fixture"); err == nil {
			t.Fatal("accepted", name)
		}
	}
	if err := s.ImportArtist(ctx, "Ween", bundle, "Collection holds the artist's recordings; test data only"); err != nil {
		t.Fatal(err)
	}
	artist, err := s.Artist(ctx, "Ween")
	if err != nil {
		t.Fatal(err)
	}
	var profile struct {
		Name              string  `json:"name"`
		ArchiveCollection string  `json:"archive_collection"`
		MusicBrainzID     *string `json:"musicbrainz_id"`
		Photo             struct {
			Data      string      `json:"data_base64"`
			Image     ArtistMedia `json:"image"`
			Thumbnail ArtistMedia `json:"thumbnail"`
		} `json:"photo"`
	}
	if err = json.Unmarshal(artist["profile"].(json.RawMessage), &profile); err != nil || profile.Name != "Ween (collection)" || profile.ArchiveCollection != "WeenCollection" || profile.MusicBrainzID != nil || profile.Photo.Data != "" {
		t.Fatal("profile", profile, err)
	}
	contentType, data, err := s.ArtistMedia(ctx, profile.Photo.Thumbnail.SHA256)
	if err != nil || contentType != "image/jpeg" || len(data) == 0 {
		t.Fatal("thumbnail unavailable", err)
	}
	if _, _, err = s.ArtistMedia(ctx, strings.Repeat("0", 64)); err == nil {
		t.Fatal("served unreferenced media")
	}
	listed := false
	if err = s.SetListing(ctx, ListingDecision{Identifiers: []string{"show"}, Listed: &listed}, "Exclude only recording"); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.ArtistMedia(ctx, profile.Photo.Image.SHA256); err == nil {
		t.Fatal("excluded-only artist media served")
	}
}
