package domain

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"
)

// AudioAsset is a derived continuous recording, never an IA original filename.
// Its receipt binds the complete preserved input inventory and independent recovery.
type AudioAsset struct {
	Snapshot       string  `json:"snapshot_sha256"`
	ManifestSHA256 string  `json:"manifest_sha256"`
	Path           string  `json:"path"`
	SHA256         string  `json:"sha256"`
	Bytes          int64   `json:"bytes"`
	Duration       float64 `json:"duration"`
	RecoverySHA256 string  `json:"independent_recovery_sha256"`
	Tool           string  `json:"tool"`
}

func ManifestDigest(approval PreservationApproval) string {
	data, _ := json.Marshal(approval.Manifest)
	return Digest(data)
}

func AuthorizeAudioAsset(c Collection, rep Representation, raw []byte, review TransportReview, approval PreservationApproval, asset AudioAsset, action Action) error {
	if rep.Media != AudioMedia || rep.Extension.Audio == nil || rep.Extension.Audio.Recording == nil || c.Policy != Local {
		return fmt.Errorf("continuous audio requires a preserved recording")
	}
	if action != PlayAction && action != ScheduleAction {
		return fmt.Errorf("unsupported continuous audio action")
	}
	recording := rep.Extension.Audio.Recording
	if err := rep.Validate(); err != nil {
		return err
	}
	if asset.Snapshot != rep.Snapshot || !validDigest(asset.SHA256) || !validDigest(asset.RecoverySHA256) || asset.ManifestSHA256 != ManifestDigest(approval) || strings.TrimSpace(asset.Tool) == "" || !filepath.IsLocal(asset.Path) || filepath.Clean(asset.Path) != asset.Path || asset.Path == "." || strings.ContainsRune(asset.Path, '\x00') || asset.Bytes <= 0 || asset.Bytes > c.Budget || math.IsNaN(asset.Duration) || math.IsInf(asset.Duration, 0) || math.Abs(asset.Duration-recording.Duration) > 0.15 {
		return fmt.Errorf("invalid or stale continuous audio receipt")
	}
	switch strings.ToLower(filepath.Ext(asset.Path)) {
	case ".wav", ".mp3", ".flac", ".ogg", ".m4a":
	default:
		return fmt.Errorf("unsupported audio asset")
	}
	var originals int64
	for _, f := range approval.Manifest {
		originals += f.Bytes
	}
	if asset.Bytes > c.Budget-originals {
		return fmt.Errorf("originals and continuous derivative exceed reviewed budget")
	}
	for _, track := range recording.Tracks {
		if err := AuthorizeTransport(c, rep, raw, review, approval, action, track.File.Name); err != nil {
			return err
		}
	}
	return nil
}

type AudioProgram struct {
	Item           string    `json:"item_id"`
	Representation string    `json:"representation_id"`
	Start          time.Time `json:"start"`
	Duration       float64   `json:"duration"`
}
type AudioSchedule struct {
	Collection string         `json:"collection"`
	Revision   string         `json:"revision"`
	Programs   []AudioProgram `json:"programs"`
}

func (s AudioSchedule) Validate() error {
	if !ValidID(s.Collection) || !ValidID(s.Revision) || len(s.Programs) < 1 || len(s.Programs) > 1000 {
		return fmt.Errorf("invalid audio schedule")
	}
	var end time.Time
	for _, p := range s.Programs {
		if !ValidID(p.Item) || !ValidID(p.Representation) || p.Start.IsZero() || p.Start.Before(end) || p.Duration <= 0 || p.Duration > 86400 || math.IsNaN(p.Duration) || math.IsInf(p.Duration, 0) {
			return fmt.Errorf("invalid audio program")
		}
		end = p.Start.Add(time.Duration(p.Duration * float64(time.Second)))
	}
	return nil
}
