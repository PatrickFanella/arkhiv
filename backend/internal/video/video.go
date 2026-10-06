// Package video verifies private processed video without altering source metadata.
package video

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"git.subcult.tv/subculture-collective/vhness/internal/preservation"
)

func separateRoots(roots ...string) error {
	var resolved []string
	for _, root := range roots {
		if !filepath.IsAbs(root) {
			return fmt.Errorf("absolute separate original, derivative and receipt roots required")
		}
		path, err := filepath.EvalSymlinks(root)
		if err != nil {
			return err
		}
		for _, prior := range resolved {
			if path == prior || strings.HasPrefix(path, prior+string(filepath.Separator)) || strings.HasPrefix(prior, path+string(filepath.Separator)) {
				return fmt.Errorf("processed-video roots overlap")
			}
		}
		resolved = append(resolved, path)
	}
	return nil
}

type evidenceFile struct {
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}
type processingEvidence struct {
	Original       evidenceFile `json:"original"`
	OriginalSHA256 string       `json:"original_sha256"`
	Manifest       struct {
		OriginalBytes int64  `json:"original_bytes"`
		Snapshot      string `json:"snapshot_sha256"`
	} `json:"manifest"`
	Derivative     evidenceFile `json:"derivative"`
	FullOriginal   bool         `json:"full_decode_original"`
	FullDerivative bool         `json:"full_decode_derivative"`
	Tool           string       `json:"tool"`
	ToolImage      string       `json:"tool_image"`
	Public         bool         `json:"public"`
}

func verifyReceipts(root string, r domain.ProcessedVideoReview) error {
	f, err := preservation.Verify(root, r.ReceiptPath, r.ReceiptSHA256, r.ReceiptBytes)
	if err != nil {
		return err
	}
	defer f.Close()
	var processing processingEvidence
	if err = json.NewDecoder(io.LimitReader(f, 4<<20)).Decode(&processing); err != nil {
		return err
	}
	if processing.OriginalSHA256 != "" {
		processing.Original = evidenceFile{processing.OriginalSHA256, processing.Manifest.OriginalBytes}
		if processing.Manifest.Snapshot != r.Snapshot {
			return fmt.Errorf("processing receipt source snapshot differs")
		}
	}
	if processing.Public || !processing.FullOriginal || !processing.FullDerivative || processing.Original.SHA256 != r.OriginalSHA256 || processing.Original.Bytes != r.OriginalBytes || processing.Derivative.SHA256 != r.SHA256 || processing.Derivative.Bytes != r.Bytes || strings.TrimSpace(processing.Tool) == "" || len(strings.TrimPrefix(processing.ToolImage, "sha256:")) != 64 {
		return fmt.Errorf("processing receipt differs from reviewed original or derivative")
	}
	fresh, err := preservation.Verify(root, r.RecoveryPath, r.RecoverySHA256, r.RecoveryBytes)
	if err != nil {
		return err
	}
	defer fresh.Close()
	var recovery struct {
		Host           string   `json:"independent_host"`
		Root           string   `json:"fresh_restore_root"`
		Database       string   `json:"fresh_database"`
		AllHashes      bool     `json:"all_hashes_match"`
		Immutable      bool     `json:"original_immutable"`
		Restored       bool     `json:"derivative_and_processing_receipts_restored"`
		Public         bool     `json:"public"`
		OriginalHash   string   `json:"original_sha256"`
		DerivativeHash string   `json:"derivative_sha256"`
		ProcessingHash string   `json:"processing_receipt_sha256"`
		CaptionHashes  []string `json:"caption_sha256"`
	}
	if err = json.NewDecoder(io.LimitReader(fresh, 4<<20)).Decode(&recovery); err != nil {
		return err
	}
	if recovery.Host == "" || recovery.Root == "" || recovery.Database == "" || !recovery.AllHashes || !recovery.Immutable || !recovery.Restored || recovery.Public || recovery.OriginalHash != r.OriginalSHA256 || recovery.DerivativeHash != r.SHA256 || recovery.ProcessingHash != r.ReceiptSHA256 {
		return fmt.Errorf("independent private original/derivative/receipt restore evidence required")
	}
	if len(recovery.CaptionHashes) != len(r.Captions) {
		return fmt.Errorf("caption restore inventory differs")
	}
	for i, caption := range r.Captions {
		if recovery.CaptionHashes[i] != caption.SHA256 {
			return fmt.Errorf("caption independent restore binding differs")
		}
	}
	return nil
}

// Verify hashes the original, processing/recovery receipts and derivative before
// returning the same open descriptor used for Range or conditional responses.
func Verify(mediaRoot, derivativeRoot, receiptRoot string, e domain.Entry) (*os.File, error) {
	if e.Decision.ProcessedVideo == nil {
		return nil, fmt.Errorf("processed-video review absent")
	}
	r := *e.Decision.ProcessedVideo
	if err := e.ValidateProcessedVideo(r); err != nil {
		return nil, err
	}
	if err := separateRoots(mediaRoot, derivativeRoot, receiptRoot); err != nil {
		return nil, err
	}
	original, err := preservation.Verify(mediaRoot, e.LocalPath, r.OriginalSHA256, r.OriginalBytes)
	if err != nil {
		return nil, err
	}
	original.Close()
	if err = verifyReceipts(receiptRoot, r); err != nil {
		return nil, err
	}
	for i := range r.Captions {
		caption, err := VerifyCaption(mediaRoot, e, i)
		if err != nil {
			return nil, err
		}
		caption.Close()
	}
	return preservation.Verify(derivativeRoot, r.Path, r.SHA256, r.Bytes)
}

func VerifyCaption(mediaRoot string, e domain.Entry, index int) (*os.File, error) {
	if e.Decision.ProcessedVideo == nil || e.ValidateProcessedVideo(*e.Decision.ProcessedVideo) != nil || index < 0 || index >= len(e.Decision.ProcessedVideo.Captions) {
		return nil, fmt.Errorf("caption is not reviewed for this source")
	}
	c := e.Decision.ProcessedVideo.Captions[index]
	f, err := preservation.Verify(mediaRoot, c.Path, c.SHA256, c.Bytes)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*os.File, error) { f.Close(); return nil, err }
	_, md5, sha1, _, err := preservation.HashAll(f)
	if err != nil {
		return fail(err)
	}
	for _, source := range e.Episode.Files {
		if source.Name == c.File && ((source.MD5 != "" && source.MD5 != md5) || (source.SHA1 != "" && source.SHA1 != sha1)) {
			return fail(fmt.Errorf("caption differs from pinned source checksums"))
		}
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return fail(err)
	}
	data, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil {
		return fail(err)
	}
	text := strings.TrimPrefix(string(data), "\ufeff")
	if !utf8.Valid(data) || strings.ContainsRune(text, '\x00') || !(strings.HasPrefix(text, "WEBVTT\n") || strings.HasPrefix(text, "WEBVTT\r\n") || strings.HasPrefix(text, "WEBVTT ")) {
		return fail(fmt.Errorf("caption is not source WebVTT"))
	}
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return fail(err)
	}
	return f, nil
}

type boundedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.limit {
		return 0, fmt.Errorf("video probe exceeds bound")
	}
	return b.Buffer.Write(p)
}

type stream struct {
	Kind       string `json:"codec_type"`
	Codec      string `json:"codec_name"`
	Pixels     string `json:"pix_fmt"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	SAR        string `json:"sample_aspect_ratio"`
	DAR        string `json:"display_aspect_ratio"`
	Rate       string `json:"avg_frame_rate"`
	Duration   string `json:"duration"`
	Start      string `json:"start_time"`
	SampleRate string `json:"sample_rate"`
	Channels   int    `json:"channels"`
	Layout     string `json:"channel_layout"`
}
type probe struct {
	Streams []stream `json:"streams"`
	Format  struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func inspect(ctx context.Context, f *os.File) (probe, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var p probe
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return p, err
	}
	cmd := exec.CommandContext(ctx, "prlimit", "--as=536870912", "--cpu=20", "--", "ffprobe", "-v", "error", "-show_streams", "-show_format", "-of", "json", "/proc/self/fd/3")
	cmd.ExtraFiles = []*os.File{f}
	out, stderr := &boundedBuffer{limit: 1 << 20}, &boundedBuffer{limit: 8192}
	cmd.Stdout, cmd.Stderr = out, stderr
	if err := cmd.Run(); err != nil {
		return p, fmt.Errorf("bounded video probe failed: %w", err)
	}
	err := json.Unmarshal(out.Bytes(), &p)
	return p, err
}
func pair(p probe) (stream, stream, error) {
	var v, a stream
	if len(p.Streams) != 2 {
		return v, a, fmt.Errorf("unpreserved caption or additional streams")
	}
	for _, s := range p.Streams {
		switch s.Kind {
		case "video":
			v = s
		case "audio":
			a = s
		default:
			return v, a, fmt.Errorf("unsupported stream")
		}
	}
	if v.Kind == "" || a.Kind == "" {
		return v, a, fmt.Errorf("one video and one audio stream required")
	}
	return v, a, nil
}
func number(s string) (float64, error) {
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, fmt.Errorf("finite video timing required")
	}
	return n, nil
}
func ratio(s string) (float64, error) {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '/' || r == ':' })
	if len(parts) != 2 {
		return 0, fmt.Errorf("measured video proportions/rate required")
	}
	a, err := number(parts[0])
	if err != nil {
		return 0, err
	}
	b, err := number(parts[1])
	if err != nil || b <= 0 || a <= 0 {
		return 0, fmt.Errorf("invalid video ratio")
	}
	return a / b, nil
}
func compare(before, after probe, r domain.ProcessedVideoReview) error {
	v, a, err := pair(before)
	if err != nil {
		return err
	}
	dv, da, err := pair(after)
	if err != nil {
		return err
	}
	if dv.Codec != "h264" || da.Codec != "aac" || dv.Pixels != "yuv420p" || v.Width != dv.Width || v.Height != dv.Height || dv.Width != r.Width || dv.Height != r.Height || a.SampleRate != da.SampleRate || a.Channels != da.Channels || a.Layout != da.Layout {
		return fmt.Errorf("processed video codecs, dimensions or audio layout differ")
	}
	for _, values := range [][2]string{{v.SAR, dv.SAR}, {v.DAR, dv.DAR}, {v.Rate, dv.Rate}} {
		left, err := ratio(values[0])
		if err != nil {
			return err
		}
		right, err := ratio(values[1])
		if err != nil || math.Abs(left-right) > .001 {
			return fmt.Errorf("processed video proportions or frame rate differ")
		}
	}
	for _, values := range [][2]string{{before.Format.Duration, after.Format.Duration}, {v.Duration, dv.Duration}, {a.Duration, da.Duration}} {
		left, err := number(values[0])
		if err != nil {
			return err
		}
		right, err := number(values[1])
		if err != nil || left <= 0 || math.Abs(left-right) > .1 {
			return fmt.Errorf("processed video duration differs")
		}
	}
	duration, err := number(after.Format.Duration)
	if err != nil || math.Abs(duration-r.Duration) > .1 {
		return fmt.Errorf("reviewed processed-video duration differs")
	}
	vs, err := number(v.Start)
	if err != nil {
		return err
	}
	as, err := number(a.Start)
	if err != nil {
		return err
	}
	dvs, err := number(dv.Start)
	if err != nil {
		return err
	}
	das, err := number(da.Start)
	if err != nil || math.Abs((vs-as)-(dvs-das)) > .1 {
		return fmt.Errorf("audio/video start timing differs")
	}
	return nil
}

func Register(ctx context.Context, db *catalog.DB, mediaRoot, derivativeRoot, receiptRoot, id string, r domain.ProcessedVideoReview) error {
	e, err := db.Entry(ctx, id)
	if err != nil {
		return err
	}
	e.Decision.ProcessedVideo = &r
	c, err := db.GetCollection(ctx, e.Episode.CollectionID)
	if err != nil {
		return err
	}
	var used int64
	for _, root := range []string{mediaRoot, derivativeRoot, receiptRoot} {
		n, err := preservation.Usage(root)
		if err != nil {
			return err
		}
		if n > c.Budget-used {
			return fmt.Errorf("original, derivative and receipt stores exceed reviewed working allowance")
		}
		used += n
		free, err := preservation.FreeBytes(root)
		if err != nil || free < c.Reserve {
			return fmt.Errorf("processed-video free-space reserve not met")
		}
	}
	f, err := Verify(mediaRoot, derivativeRoot, receiptRoot, e)
	if err != nil {
		return err
	}
	defer f.Close()
	original, err := preservation.Verify(mediaRoot, e.LocalPath, r.OriginalSHA256, r.OriginalBytes)
	if err != nil {
		return err
	}
	defer original.Close()
	before, err := inspect(ctx, original)
	if err != nil {
		return err
	}
	after, err := inspect(ctx, f)
	if err != nil {
		return err
	}
	if err = compare(before, after, r); err != nil {
		return err
	}
	return db.ReviewProcessedVideo(ctx, id, r)
}
