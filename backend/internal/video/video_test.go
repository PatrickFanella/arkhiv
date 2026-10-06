package video

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
)

func fixture(t *testing.T) (string, string, string, domain.Entry) {
	t.Helper()
	root := t.TempDir()
	media := filepath.Join(root, "originals")
	derived := filepath.Join(root, "derivatives")
	receipts := filepath.Join(root, "evidence")
	for _, path := range []string{media, derived, receipts} {
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	original, output := []byte("immutable original"), []byte("private derivative")
	if err := os.WriteFile(filepath.Join(media, "source.avi"), original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(derived, "video.mp4"), output, 0600); err != nil {
		t.Fatal(err)
	}
	e := domain.Entry{Episode: domain.Episode{ID: "video", CollectionID: "private", Title: "Source", Files: []domain.File{{Name: "source.avi", Source: "original", Bytes: int64(len(original)), Duration: 10, MD5: strings.Repeat("a", 32)}}}, Policy: domain.Local, Budget: 768 << 20, SourceHash: strings.Repeat("b", 64), LocalSnapshot: strings.Repeat("b", 64), LocalFile: "source.avi", LocalPath: "source.avi", LocalSHA256: domain.Digest(original), Decision: domain.Decision{Acquisition: true, ProcessingOriginal: true, File: "source.avi", Notes: "Original approval"}}
	r := domain.ProcessedVideoReview{Snapshot: e.SourceHash, OriginalFile: e.LocalFile, OriginalSHA256: e.LocalSHA256, OriginalBytes: int64(len(original)), Path: "video.mp4", SHA256: domain.Digest(output), Bytes: int64(len(output)), Duration: 10, Width: 320, Height: 240, ReceiptPath: "processing.json", RecoveryPath: "restore.json", Published: true, Playback: true, Channel: true, Notes: "Private review"}
	p, _ := json.Marshal(processingEvidence{Original: evidenceFile{r.OriginalSHA256, r.OriginalBytes}, Derivative: evidenceFile{r.SHA256, r.Bytes}, FullOriginal: true, FullDerivative: true, Tool: "Reviewed tool", ToolImage: strings.Repeat("c", 64)})
	recovery, _ := json.Marshal(map[string]any{"independent_host": "independent-fixture", "fresh_restore_root": "/fresh-fixture", "fresh_database": "fresh-fixture", "all_hashes_match": true, "original_immutable": true, "derivative_and_processing_receipts_restored": true, "public": false, "original_sha256": r.OriginalSHA256, "derivative_sha256": r.SHA256, "processing_receipt_sha256": domain.Digest(p)})
	for name, data := range map[string][]byte{"processing.json": p, "restore.json": recovery} {
		if err := os.WriteFile(filepath.Join(receipts, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	r.ReceiptSHA256 = domain.Digest(p)
	r.ReceiptBytes = int64(len(p))
	r.RecoverySHA256 = domain.Digest(recovery)
	r.RecoveryBytes = int64(len(recovery))
	e.Decision.ProcessedVideo = &r
	return media, derived, receipts, e
}

func TestVerifyChecksOriginalDerivativeAndBothReceipts(t *testing.T) {
	for _, target := range []string{"original", "derivative", "processing", "recovery"} {
		t.Run(target, func(t *testing.T) {
			media, derived, receipts, e := fixture(t)
			f, err := Verify(media, derived, receipts, e)
			if err != nil {
				t.Fatal(err)
			}
			f.Close()
			paths := map[string]string{"original": filepath.Join(media, e.LocalPath), "derivative": filepath.Join(derived, e.Decision.ProcessedVideo.Path), "processing": filepath.Join(receipts, e.Decision.ProcessedVideo.ReceiptPath), "recovery": filepath.Join(receipts, e.Decision.ProcessedVideo.RecoveryPath)}
			data, err := os.ReadFile(paths[target])
			if err != nil {
				t.Fatal(err)
			}
			data[0] ^= 1
			if err = os.WriteFile(paths[target], data, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err = Verify(media, derived, receipts, e); err == nil {
				t.Fatal("same-size changed bytes accepted")
			}
		})
	}
	media, derived, receipts, e := fixture(t)
	if _, err := Verify(media, media, receipts, e); err == nil {
		t.Fatal("overlapping roots accepted")
	}
	e.SourceHash = strings.Repeat("d", 64)
	if _, err := Verify(media, derived, receipts, e); err == nil {
		t.Fatal("stale source snapshot accepted")
	}
}

func TestRegistrationProbePreservesProportionsTimingAndCaptions(t *testing.T) {
	p := probe{Streams: []stream{{Kind: "video", Codec: "mpeg4", Pixels: "yuv420p", Width: 320, Height: 240, SAR: "1:1", DAR: "4:3", Rate: "25/1", Duration: "10", Start: "0"}, {Kind: "audio", Codec: "mp3", Duration: "10", Start: "0", SampleRate: "48000", Channels: 2, Layout: "stereo"}}}
	p.Format.Duration = "10"
	q := p
	q.Streams = append([]stream(nil), p.Streams...)
	q.Streams[0].Codec = "h264"
	q.Streams[1].Codec = "aac"
	r := domain.ProcessedVideoReview{Duration: 10, Width: 320, Height: 240}
	if err := compare(p, q, r); err != nil {
		t.Fatal(err)
	}
	q.Streams[0].DAR = "16:9"
	if compare(p, q, r) == nil {
		t.Fatal("changed display proportions accepted")
	}
	q.Streams[0].DAR = "4:3"
	q.Streams[1].Start = "2"
	if compare(p, q, r) == nil {
		t.Fatal("changed audiovisual timing accepted")
	}
	q.Streams[1].Start = "0"
	p.Streams = append(p.Streams, stream{Kind: "subtitle"})
	if compare(p, q, r) == nil {
		t.Fatal("silently dropped captions accepted")
	}
}

func TestSourceCaptionIntegrityAndIndependentRestore(t *testing.T) {
	media, derived, receipts, e := fixture(t)
	data := []byte("WEBVTT\n\n00:00.000 --> 00:01.000\nGenerated test caption only.\n")
	md := md5.Sum(data)
	c := domain.LocalVideoCaption{CaptionReview: domain.CaptionReview{File: "fixture.vtt", Label: "Generated fixture", Language: "en"}, Path: "fixture.vtt", SHA256: domain.Digest(data), Bytes: int64(len(data))}
	e.Episode.Files = append(e.Episode.Files, domain.File{Name: c.File, Bytes: c.Bytes, MD5: hex.EncodeToString(md[:])})
	e.Decision.ProcessedVideo.Captions = []domain.LocalVideoCaption{c}
	if err := os.WriteFile(filepath.Join(media, c.Path), data, 0600); err != nil {
		t.Fatal(err)
	}
	f, err := VerifyCaption(media, e, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err = Verify(media, derived, receipts, e); err == nil {
		t.Fatal("caption without independent restore binding accepted")
	}
	r := e.Decision.ProcessedVideo
	path := filepath.Join(receipts, r.RecoveryPath)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var recovery map[string]any
	if err = json.Unmarshal(raw, &recovery); err != nil {
		t.Fatal(err)
	}
	recovery["caption_sha256"] = []string{c.SHA256}
	raw, _ = json.Marshal(recovery)
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	r.RecoverySHA256 = domain.Digest(raw)
	r.RecoveryBytes = int64(len(raw))
	f, err = Verify(media, derived, receipts, e)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	data[len(data)-2] ^= 1
	if err = os.WriteFile(filepath.Join(media, c.Path), data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = VerifyCaption(media, e, 0); err == nil {
		t.Fatal("modified caption accepted")
	}
}
