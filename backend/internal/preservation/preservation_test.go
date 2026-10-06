package preservation

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
)

func fixture(t *testing.T, handler http.HandlerFunc) (*Acquirer, domain.Collection, domain.Entry, Manifest) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	content := []byte("test-original-video-content")
	c := domain.Collection{ID: "fixture", Policy: domain.Local, Budget: 1 << 20, Reserve: 100}
	e := domain.Entry{Policy: domain.Local, SourceHash: domain.Digest([]byte("snapshot")), Episode: domain.Episode{ID: "fixture-episode", CollectionID: c.ID, Title: "Generated fixture", Files: []domain.File{{Name: "original.mp4", Bytes: int64(len(content)), Duration: 10}}}, Decision: domain.Decision{Acquisition: true, Published: true, Playback: true, Channel: true, File: "original.mp4", Notes: "Generated local fixture"}}
	m := Manifest{Collection: c.ID, ID: e.Episode.ID, Snapshot: e.SourceHash, File: e.Decision.File, Bytes: int64(len(content)), SHA256: domain.Digest(content), Notes: "Reviewed generated sample"}
	a := New(t.TempDir())
	a.Base = server.URL
	a.HTTP = server.Client()
	a.Free = func(string) (int64, error) { return 1 << 30, nil }
	return a, c, e, m
}
func complete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("ETag", `"stable"`)
	fmt.Fprint(w, "test-original-video-content")
}
func TestAcquisitionVerificationAndImmutableOriginal(t *testing.T) {
	calls := 0
	a, c, e, m := fixture(t, func(w http.ResponseWriter, r *http.Request) { calls++; complete(w, r) })
	receipt, err := a.Acquire(context.Background(), c, e, m)
	if err != nil {
		t.Fatal(err)
	}
	f, err := Verify(a.Root, receipt.Path, m.SHA256, m.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err = a.Acquire(context.Background(), c, e, m); err != nil || calls != 1 {
		t.Fatal("idempotence", err, calls)
	}
	if err = os.WriteFile(filepath.Join(a.Root, receipt.Path), []byte(strings.Repeat("x", int(m.Bytes))), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = a.Acquire(context.Background(), c, e, m); err == nil {
		t.Fatal("corrupt immutable original accepted")
	}
}
func TestPolicyManifestBudgetAndReserveBeforeRequest(t *testing.T) {
	calls := 0
	a, c, e, m := fixture(t, func(w http.ResponseWriter, r *http.Request) { calls++; complete(w, r) })
	stream := c
	stream.ID = domain.DemocracyNow
	stream.Policy = domain.Stream
	if _, err := a.Acquire(context.Background(), stream, e, m); err == nil {
		t.Fatal("stream acquisition accepted")
	}
	bad := m
	bad.Snapshot = "changed"
	if _, err := a.Acquire(context.Background(), c, e, bad); err == nil {
		t.Fatal("changed snapshot accepted")
	}
	small := c
	small.Budget = 10
	if _, err := a.Acquire(context.Background(), small, e, m); err == nil {
		t.Fatal("budget exceeded")
	}
	a.Free = func(string) (int64, error) { return c.Reserve, nil }
	if _, err := a.Acquire(context.Background(), c, e, m); err == nil {
		t.Fatal("reserve exceeded")
	}
	if calls != 0 {
		t.Fatal("unsafe request before checks", calls)
	}
}
func TestResumeRequiresStableIdentityAndExactRange(t *testing.T) {
	first := true
	badRange := true
	a, c, e, m := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"stable"`)
		if first {
			first = false
			w.Header().Set("Content-Length", "27")
			w.WriteHeader(200)
			fmt.Fprint(w, "test-")
			return
		}
		if r.Header.Get("Range") != "bytes=5-" || r.Header.Get("If-Range") != `"stable"` {
			t.Error("missing resume validation")
		}
		w.Header().Set("Content-Length", "22")
		cr := "bytes 5-26/27"
		if badRange {
			cr = "bytes 0-21/27"
		}
		w.Header().Set("Content-Range", cr)
		w.WriteHeader(206)
		fmt.Fprint(w, "original-video-content")
	})
	if _, err := a.Acquire(context.Background(), c, e, m); err == nil {
		t.Fatal("interruption ignored")
	}
	if _, err := a.Acquire(context.Background(), c, e, m); err == nil {
		t.Fatal("broken range accepted")
	}
	changed := m
	changed.Notes = "changed"
	if _, err := a.Acquire(context.Background(), c, e, changed); err == nil {
		t.Fatal("changed manifest resumed")
	}
	badRange = false
	if _, err := a.Acquire(context.Background(), c, e, m); err != nil {
		t.Fatal("valid resume failed", err)
	}
}
func TestChecksumFailureAndDuringTransferReserve(t *testing.T) {
	a, c, e, m := fixture(t, complete)
	m.SHA256 = domain.Digest([]byte("wrong"))
	if _, err := a.Acquire(context.Background(), c, e, m); err == nil {
		t.Fatal("checksum mismatch accepted")
	}
	paths, _ := filepath.Glob(filepath.Join(a.Root, "*.part"))
	if len(paths) != 1 {
		t.Fatal("recoverable partial lost")
	}
	a, c, e, m = fixture(t, complete)
	calls := 0
	a.Free = func(string) (int64, error) {
		calls++
		if calls > 1 {
			return c.Reserve, nil
		}
		return 1 << 30, nil
	}
	if _, err := a.Acquire(context.Background(), c, e, m); err == nil {
		t.Fatal("during-transfer reserve violated")
	}
}
func TestSourceChecksumAndEscape(t *testing.T) {
	a, c, e, m := fixture(t, complete)
	e.Episode.Files[0].MD5 = strings.Repeat("0", 32)
	if _, err := a.Acquire(context.Background(), c, e, m); err == nil {
		t.Fatal("source MD5 ignored")
	}
	outside := filepath.Join(t.TempDir(), "outside")
	os.WriteFile(outside, []byte("secret"), 0600)
	os.Symlink(outside, filepath.Join(a.Root, "escape"))
	if _, err := Verify(a.Root, "escape", domain.Digest([]byte("secret")), 6); err == nil {
		t.Fatal("symlink escape accepted")
	}
}

func TestPinnedSourceChecksumsWithoutPriorSHA256(t *testing.T) {
	a, c, e, m := fixture(t, complete)
	_, md, one, _, err := HashAll(strings.NewReader("test-original-video-content"))
	if err != nil {
		t.Fatal(err)
	}
	e.Episode.Files[0].MD5 = md
	e.Episode.Files[0].SHA1 = one
	m.SHA256 = ""
	receipt, err := a.Acquire(context.Background(), c, e, m)
	if err != nil || receipt.SHA256 != domain.Digest([]byte("test-original-video-content")) {
		t.Fatal("source checksum acquisition", err, receipt)
	}
	a, c, e, m = fixture(t, complete)
	m.SHA256 = ""
	e.Episode.Files[0].MD5 = md
	e.Episode.Files[0].SHA1 = strings.Repeat("0", 40)
	if _, err = a.Acquire(context.Background(), c, e, m); err == nil {
		t.Fatal("source SHA1 mismatch accepted")
	}
	a, c, e, m = fixture(t, complete)
	m.SHA256 = ""
	if _, err = a.Acquire(context.Background(), c, e, m); err == nil {
		t.Fatal("unpinned checksum accepted")
	}
}

func TestPrivateProcessingOriginalUsesPinnedAcquisitionGuards(t *testing.T) {
	calls := 0
	a, c, e, m := fixture(t, func(w http.ResponseWriter, r *http.Request) { calls++; complete(w, r) })
	e.Episode.Files[0].Name = "original.avi"
	e.Episode.Files[0].Source = "original"
	e.Episode.Files[0].MD5 = "91b3b5e772b76c3a03b5238103ff114a"
	e.Decision = domain.Decision{Acquisition: true, ProcessingOriginal: true, File: "original.avi", Notes: "Approved bounded private original"}
	m.File = e.Decision.File
	// The generated bytes differ from the real source digest; pinned SHA256 wins
	// only when the supplied source checksum agrees too. Use the fixture checksum.
	_, md, _, _, err := HashAll(strings.NewReader("test-original-video-content"))
	if err != nil {
		t.Fatal(err)
	}
	e.Episode.Files[0].MD5 = md
	unapproved := e
	unapproved.Decision.ProcessingOriginal = false
	if _, err := a.Acquire(context.Background(), c, unapproved, m); err == nil || calls != 0 {
		t.Fatal("unapproved request", err, calls)
	}
	receipt, err := a.Acquire(context.Background(), c, e, m)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || receipt.SHA256 != m.SHA256 {
		t.Fatal("verified original receipt", receipt, calls)
	}
	if _, err = a.Acquire(context.Background(), c, e, m); err != nil || calls != 1 {
		t.Fatal("idempotence", err, calls)
	}
	if e.Watchable() {
		t.Fatal("preservation enabled playback")
	}
}
