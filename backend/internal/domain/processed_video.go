package domain

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
)

// ProcessedVideoReview records private derivative access independently of the
// source inventory and the original's acquisition-only decision.
type ProcessedVideoReview struct {
	Captions       []LocalVideoCaption `json:"captions,omitempty"`
	Snapshot       string              `json:"snapshot_sha256"`
	OriginalFile   string              `json:"original_file"`
	OriginalSHA256 string              `json:"original_sha256"`
	OriginalBytes  int64               `json:"original_bytes"`
	Path           string              `json:"path"`
	SHA256         string              `json:"sha256"`
	Bytes          int64               `json:"bytes"`
	Duration       float64             `json:"duration"`
	Width          int                 `json:"width"`
	Height         int                 `json:"height"`
	ReceiptPath    string              `json:"processing_receipt_path"`
	ReceiptSHA256  string              `json:"processing_receipt_sha256"`
	ReceiptBytes   int64               `json:"processing_receipt_bytes"`
	RecoveryPath   string              `json:"recovery_receipt_path"`
	RecoverySHA256 string              `json:"recovery_receipt_sha256"`
	RecoveryBytes  int64               `json:"recovery_receipt_bytes"`
	Published      bool                `json:"published"`
	Playback       bool                `json:"playback"`
	Channel        bool                `json:"channel"`
	Public         bool                `json:"public"`
	Notes          string              `json:"notes"`
}

// LocalVideoCaption binds an explicitly preserved source WebVTT sidecar. A
// generated or converted caption requires another provenance/processing design.
type LocalVideoCaption struct {
	CaptionReview
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

func (e Entry) ValidateLocalVideoCaptions(r ProcessedVideoReview) error {
	if len(r.Captions) > 16 {
		return fmt.Errorf("too many local video captions")
	}
	seen := map[string]bool{}
	var total int64
	for _, caption := range r.Captions {
		if !SafeFile(caption.File) || !strings.HasSuffix(strings.ToLower(caption.File), ".vtt") || !localVideoPath(caption.Path) || !validDigest(caption.SHA256) || caption.Bytes <= 0 || caption.Bytes > 1<<20 || strings.TrimSpace(caption.Label) == "" || len(caption.Label) > 200 || !languagePattern.MatchString(caption.Language) || seen[caption.File] {
			return fmt.Errorf("invalid preserved source WebVTT review")
		}
		seen[caption.File] = true
		found := false
		for _, file := range e.Episode.Files {
			if file.Name == caption.File && !file.Private && file.Bytes == caption.Bytes && (validSourceChecksum(file.MD5, 32) || validSourceChecksum(file.SHA1, 40)) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("local caption absent, restricted or not pinned to source checksum")
		}
		total += caption.Bytes
	}
	if r.OriginalBytes+r.Bytes+total > e.Budget || r.OriginalBytes+r.Bytes+total > 768<<20 {
		return fmt.Errorf("local captions exceed separate processing working allowance")
	}
	return nil
}

func localVideoPath(path string) bool {
	return filepath.IsLocal(path) && filepath.Clean(path) == path && path != "." && !strings.ContainsAny(path, "\\\x00")
}

func (e Entry) ValidateProcessedVideo(r ProcessedVideoReview) error {
	f, ok := e.SelectedOriginal()
	if !ok || !e.Decision.ProcessingOriginal || e.Policy != Local || e.Episode.DemocracySource() || e.Decision.Public || e.Decision.Published || e.Decision.Playback || e.Decision.Channel {
		return fmt.Errorf("processed video requires an acquisition-only preserved original")
	}
	if r.Public || r.Published && !r.Playback || r.Channel && !r.Playback || strings.TrimSpace(r.Notes) == "" || len(r.Notes) > 4096 {
		return fmt.Errorf("invalid private processed-video access review")
	}
	if r.Snapshot != e.SourceHash || r.Snapshot != e.LocalSnapshot || !validDigest(r.Snapshot) || r.OriginalFile != f.Name || r.OriginalFile != e.LocalFile || r.OriginalSHA256 != e.LocalSHA256 || !validDigest(r.OriginalSHA256) || r.OriginalBytes != f.Bytes || e.LocalPath == "" {
		return fmt.Errorf("processed video is not bound to the current preserved original")
	}
	if !localVideoPath(r.Path) || strings.ToLower(filepath.Ext(r.Path)) != ".mp4" || !validDigest(r.SHA256) || r.Bytes <= 0 || r.Bytes > 384<<20 || r.OriginalBytes > (768<<20)-r.Bytes || e.Budget < r.OriginalBytes+r.Bytes {
		return fmt.Errorf("invalid processed-video path, digest or storage allowance")
	}
	if !Duration(r.Duration) || math.Abs(r.Duration-f.Duration) > .15 || r.Width <= 0 || r.Height <= 0 || r.Width > 8192 || r.Height > 8192 {
		return fmt.Errorf("invalid processed-video dimensions or duration")
	}
	for _, receipt := range []struct {
		path, hash string
		bytes      int64
	}{{r.ReceiptPath, r.ReceiptSHA256, r.ReceiptBytes}, {r.RecoveryPath, r.RecoverySHA256, r.RecoveryBytes}} {
		if !localVideoPath(receipt.path) || !validDigest(receipt.hash) || receipt.bytes <= 0 || receipt.bytes > 4<<20 {
			return fmt.Errorf("processing and independent recovery receipts required")
		}
	}
	return e.ValidateLocalVideoCaptions(r)
}

func (e Entry) Published() bool {
	if e.Decision.ProcessedVideo != nil {
		return e.Decision.ProcessedVideo.Published
	}
	return e.Decision.Published
}

func (e Entry) ChannelApproved() bool {
	if e.Decision.ProcessedVideo != nil {
		return e.ValidateProcessedVideo(*e.Decision.ProcessedVideo) == nil && e.Decision.ProcessedVideo.Channel
	}
	return e.Decision.Channel
}

func (r ProcessedVideoReview) File() File {
	return File{Name: "processed-" + r.SHA256 + ".mp4", Format: "Processed H.264/AAC MP4", Source: "local_processing", Original: r.OriginalFile, Bytes: r.Bytes, Duration: r.Duration, Width: r.Width, Height: r.Height}
}
