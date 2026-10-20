package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"git.subcult.tv/subculture-collective/vhness/internal/sourcecatalog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestArtistMediaIsServedFromTheEngine(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL absent; HTTP/database integration unverified")
	}
	ctx := context.Background()
	admin, err := catalog.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Pool.Close()
	schema := "vhness_artist_media_" + time.Now().Format("20060102150405")
	name := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Pool.Exec(ctx, "CREATE SCHEMA "+name); err != nil {
		t.Fatal(err)
	}
	defer admin.Pool.Exec(ctx, "DROP SCHEMA "+name+" CASCADE")
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	d := &catalog.DB{Pool: pool}
	if err = d.Migrate(ctx, "../../../migrations"); err != nil {
		t.Fatal(err)
	}
	if err = d.Collection(ctx, domain.Collection{ID: "etree", Policy: domain.Stream}); err != nil {
		t.Fatal(err)
	}
	store := &sourcecatalog.Store{Core: d, Collection: "etree"}
	if err = store.Init(ctx); err != nil {
		t.Fatal(err)
	}
	item := `{"metadata":{"identifier":"show","collection":["Band","etree"],"creator":"Band","title":"Show","date":"2003-09-23"},"files":[{"name":"song.mp3"}]}`
	if err = store.Import(ctx, "show", []byte(item), 200, "", ""); err != nil {
		t.Fatal(err)
	}
	var picture bytes.Buffer
	jpeg.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 64, 64)), nil)
	meta := `{"metadata":{"identifier":"Band","mediatype":"collection","title":"Band","collection":["etree"]}}`
	file := "https://archive.org/download/Band/Band_itemimage.jpg"
	bundle := sourcecatalog.ArtistBundle{ArchiveCollection: "Band",
		Sources: []sourcecatalog.ArtistSource{{URL: "https://archive.org/metadata/Band", Raw: base64.StdEncoding.EncodeToString([]byte(meta)), FetchedAt: time.Now()}},
		Photo:   &sourcecatalog.ArtistPhoto{URL: file, DataURL: file, Data: base64.StdEncoding.EncodeToString(picture.Bytes()), Source: "https://archive.org/details/Band", Credit: "Band collection", License: sourcecatalog.ArchiveImageLicense}}
	if err = store.ImportArtist(ctx, "Band", bundle, "Fixture collection binding"); err != nil {
		t.Fatal(err)
	}
	handler := (&Server{DB: d, Collection: "etree"}).Handler()
	get := func(path, etag string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		if etag != "" {
			r.Header.Set("If-None-Match", etag)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	var list struct {
		Artists []struct {
			Photo struct {
				Thumbnail struct{ SHA256 string } `json:"thumbnail"`
			} `json:"photo"`
		} `json:"artists"`
	}
	w := get("/api/source-catalog/v1/artist-photos?collection=etree", "")
	if err = json.Unmarshal(w.Body.Bytes(), &list); err != nil || len(list.Artists) != 1 || list.Artists[0].Photo.Thumbnail.SHA256 == "" {
		t.Fatal(w.Code, w.Body.String())
	}
	id := list.Artists[0].Photo.Thumbnail.SHA256
	w = get("/api/source-catalog/v1/artist-media/"+id+"?collection=etree", "")
	h := w.Header()
	if w.Code != 200 || h.Get("Content-Type") != "image/jpeg" || !strings.Contains(h.Get("Cache-Control"), "immutable") || h.Get("X-Content-Type-Options") != "nosniff" || !strings.Contains(h.Get("Content-Security-Policy"), "sandbox") || w.Body.Len() == 0 {
		t.Fatal(w.Code, h)
	}
	if w = get("/api/source-catalog/v1/artist-media/"+id, `"`+id+`"`); w.Code != 304 || w.Body.Len() != 0 {
		t.Fatal("conditional request", w.Code)
	}
	for _, path := range []string{"/api/source-catalog/v1/artist-media/" + strings.Repeat("0", 64), "/api/source-catalog/v1/artist-media/not-a-hash"} {
		if w = get(path, ""); w.Code != 404 {
			t.Fatal(path, w.Code)
		}
	}
}
