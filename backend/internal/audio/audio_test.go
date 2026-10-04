package audio_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/app"
	audiomodule "git.subcult.tv/subculture-collective/vhness/internal/audio"
	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"git.subcult.tv/subculture-collective/vhness/internal/music"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Synthetic PCM audio verifies bytes, duration and scheduling, not music fidelity.
func tone(seconds int) []byte {
	count := seconds * 8000
	var b bytes.Buffer
	b.WriteString("RIFF")
	binary.Write(&b, binary.LittleEndian, uint32(36+count*2))
	b.WriteString("WAVEfmt ")
	binary.Write(&b, binary.LittleEndian, uint32(16))
	binary.Write(&b, binary.LittleEndian, uint16(1))
	binary.Write(&b, binary.LittleEndian, uint16(1))
	binary.Write(&b, binary.LittleEndian, uint32(8000))
	binary.Write(&b, binary.LittleEndian, uint32(16000))
	binary.Write(&b, binary.LittleEndian, uint16(2))
	binary.Write(&b, binary.LittleEndian, uint16(16))
	b.WriteString("data")
	binary.Write(&b, binary.LittleEndian, uint32(count*2))
	for i := 0; i < count; i++ {
		binary.Write(&b, binary.LittleEndian, int16(4000*math.Sin(float64(i)*2*math.Pi*440/8000)))
	}
	return b.Bytes()
}
func TestContinuousAudioRightsAndStationIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("disposable database absent; audio integration unverified")
	}
	ctx := context.Background()
	admin, err := catalog.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Pool.Close()
	schema := fmt.Sprintf("audio_test_%x", time.Now().UnixNano())
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Pool.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	defer admin.Pool.Exec(ctx, "DROP SCHEMA "+quoted+" CASCADE")
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	db := &catalog.DB{Pool: pool}
	if err = db.Migrate(ctx, "../../../migrations"); err != nil {
		t.Fatal(err)
	}
	c := domain.Collection{ID: "audio-fixture", Policy: domain.Local, Budget: 1 << 20, Reserve: 1}
	if err = db.Collection(ctx, c); err != nil {
		t.Fatal(err)
	}
	media, derived := t.TempDir(), t.TempDir()
	one, continuous := tone(1), tone(2)
	for _, name := range []string{"one.wav", "two.wav"} {
		if err = os.WriteFile(filepath.Join(media, name), one, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err = os.WriteFile(filepath.Join(derived, "continuous.wav"), continuous, 0600); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"metadata":{"identifier":"audio-source"},"files":[{"name":"one.wav"},{"name":"two.wav"}],"retained_unknown":"preserved"}`)
	snapshot := domain.Digest(raw)
	if err = db.Snapshot(ctx, "audio-source", raw, 200, "", "", nil); err != nil {
		t.Fatal(err)
	}
	r := music.Recording{ID: "recording", ConcertID: "concert", Title: "Synthetic continuous recording", Source: "Generated tone", Taper: "Fixture generator", Duration: 2, Synthetic: true, Warnings: []string{"Unreviewed set boundary retained"}, Tracks: []music.Track{{Title: "First", Position: 0, Start: 0, Duration: 1, File: music.File{Name: "one.wav", Bytes: int64(len(one))}}, {Title: "Second", Position: 1, Start: 1, Duration: 1, File: music.File{Name: "two.wav", Bytes: int64(len(one))}}}, Sets: []music.Set{{Title: "Reviewed second track", First: 1, Last: 1, Reviewed: true}}}
	rep := domain.Representation{ID: "recording", Media: domain.AudioMedia, SourceItemID: "audio-source", Snapshot: snapshot, File: "one.wav", Collections: []string{c.ID}, Extension: domain.MediaExtension{Audio: &domain.AudioExtension{RecordingID: r.ID, ConcertID: r.ConcertID, Recording: &r, Tracks: []domain.AudioTrack{{ID: "one", File: "one.wav"}, {ID: "two", File: "two.wav"}}, Credits: map[string]string{"taper": r.Taper}}}}
	approval := domain.PreservationApproval{Snapshot: snapshot, IndependentRecoverySHA256: domain.Digest([]byte("synthetic original recovery")), Manifest: []domain.LocalOriginal{{File: "one.wav", Bytes: int64(len(one)), SHA256: domain.Digest(one), Path: "one.wav"}, {File: "two.wav", Bytes: int64(len(one)), SHA256: domain.Digest(one), Path: "two.wav"}}}
	review := domain.TransportReview{Snapshot: snapshot, Files: []string{"one.wav", "two.wav"}, Playback: true, Schedule: true, Notes: "Synthetic private audio review"}
	asset := domain.AudioAsset{Snapshot: snapshot, ManifestSHA256: domain.ManifestDigest(approval), Path: "continuous.wav", SHA256: domain.Digest(continuous), Bytes: int64(len(continuous)), Duration: 2, RecoverySHA256: domain.Digest([]byte("synthetic derived recovery")), Tool: "Synthetic PCM fixture"}
	for _, id := range []string{"first", "future"} {
		item := domain.CatalogItem{ID: id, Title: r.Title, Collections: []string{c.ID}, Representations: []domain.Representation{rep}}
		if err = db.InsertCatalog(ctx, item, "Synthetic recording mapping"); err != nil {
			t.Fatal(err)
		}
		if err = db.ReviewTransport(ctx, id, rep.ID, c.ID, review, approval); err != nil {
			t.Fatal(err)
		}
		bad := asset
		bad.ManifestSHA256 = domain.Digest([]byte("foreign input inventory"))
		if err = audiomodule.Register(ctx, db, media, derived, id, rep.ID, c.ID, bad, "Reject stale input receipt"); err == nil {
			t.Fatal("foreign input receipt accepted")
		}
		if err = audiomodule.Register(ctx, db, media, derived, id, rep.ID, c.ID, asset, "Verify synthetic continuous asset"); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	schedule := domain.AudioSchedule{Collection: c.ID, Revision: "fixture-station", Programs: []domain.AudioProgram{{Item: "first", Representation: rep.ID, Start: now, Duration: 2}, {Item: "future", Representation: rep.ID, Start: now.Add(2 * time.Second), Duration: 2}}}
	if err = db.StoreAudioSchedule(ctx, schedule, "Synthetic station"); err != nil {
		t.Fatal(err)
	}
	server := &app.Server{DB: db, Media: media, Derivatives: derived, Collection: c.ID, Clock: func() time.Time { return now.Add(1500 * time.Millisecond) }}
	handler := server.Handler()
	get := func(path string, headers map[string]string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", path, nil)
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		handler.ServeHTTP(w, r)
		return w
	}
	base := "/api/catalog/v1/items/first/representations/recording/audio"
	w := get(base+"/resolve", nil)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte("Fixture generator")) || bytes.Contains(w.Body.Bytes(), []byte("continuous.wav")) {
		t.Fatal("audio contract or private path exposure", w.Code, w.Body.String())
	}
	w = get(base+"/media", map[string]string{"Range": "bytes=0-43"})
	if w.Code != 206 || !bytes.HasPrefix(w.Body.Bytes(), []byte("RIFF")) {
		t.Fatal("continuous range playback", w.Code)
	}
	w = get("/api/audio/channel", nil)
	var station struct {
		State    string  `json:"state"`
		Offset   float64 `json:"offset"`
		Revision string  `json:"revision"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &station); err != nil || station.State != "playing" || station.Offset != 1.5 {
		t.Fatal("server station clock", err, w.Code, w.Body.String())
	}
	if w = get(base+"/resolve?collection=foreign", nil); w.Code != 404 {
		t.Fatal("foreign audio exposed", w.Code)
	}
	server.Collection = ""
	server.Collections = domain.CollectionScope{c.ID, "another-allowed"}
	if w = get(base+"/resolve?collection="+c.ID, nil); w.Code != 200 {
		t.Fatal("allowed multi scope audio unavailable", w.Code)
	}
	if w = get(base+"/resolve", nil); w.Code != 404 {
		t.Fatal("ambiguous audio collection accepted", w.Code)
	}
	if w = get(base+"/media?collection=foreign", nil); w.Code != 404 {
		t.Fatal("audio bytes crossed multi allowlist", w.Code)
	}
	server.Collection = c.ID
	server.Collections = nil
	review.Schedule = false
	review.Notes = "Withdraw future station eligibility"
	if err = db.ReviewTransport(ctx, "future", rep.ID, c.ID, review, approval); err != nil {
		t.Fatal(err)
	}
	w = get("/api/audio/channel", nil)
	if !bytes.Contains(w.Body.Bytes(), []byte(`"state":"unavailable"`)) {
		t.Fatal("withdrawn future program exposed station", w.Body.String())
	}
	if w = get(base+"/resolve", nil); w.Code != 200 {
		t.Fatal("station withdrawal removed independent on-demand playback", w.Code)
	}
	review.Playback = false
	review.Notes = "Withdraw first recording playback"
	if err = db.ReviewTransport(ctx, "first", rep.ID, c.ID, review, approval); err != nil {
		t.Fatal(err)
	}
	if w = get(base+"/media", map[string]string{"If-None-Match": `"sha256-` + asset.SHA256 + `"`}); w.Code != 404 {
		t.Fatal("revoked conditional audio exposed", w.Code)
	}
	review.Playback = true
	review.Notes = "Restore synthetic playback"
	if err = db.ReviewTransport(ctx, "first", rep.ID, c.ID, review, approval); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(media, "two.wav"), []byte("corrupt input"), 0600); err != nil {
		t.Fatal(err)
	}
	if w = get(base+"/media", nil); w.Code != 404 {
		t.Fatal("continuous derivative survived corrupt source input", w.Code)
	}
	if err = os.WriteFile(filepath.Join(media, "two.wav"), one, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(derived, "continuous.wav"), []byte("corrupt derivative"), 0600); err != nil {
		t.Fatal(err)
	}
	if w = get(base+"/media", nil); w.Code != 404 {
		t.Fatal("corrupt continuous derivative exposed", w.Code)
	}
}
