package domain

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

type Action string

const (
	ReadAction     Action = "read"
	PlayAction     Action = "play"
	DownloadAction Action = "download"
	AcquireAction  Action = "acquire"
	RelayAction    Action = "relay"
	ScheduleAction Action = "schedule"
)

// TransportReview is independent of catalog listing. A brand config never
// supplies it. Its exact snapshot/file binding invalidates stale approvals.
type TransportReview struct {
	Snapshot    string   `json:"snapshot_sha256"`
	Files       []string `json:"files"`
	Read        bool     `json:"read"`
	Playback    bool     `json:"playback"`
	Download    bool     `json:"download"`
	Acquisition bool     `json:"acquisition"`
	Relay       bool     `json:"relay"`
	Schedule    bool     `json:"schedule"`
	Public      bool     `json:"public"`
	Notes       string   `json:"notes"`
}
type LocalOriginal struct {
	File   string `json:"file"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
	Path   string `json:"path"`
}
type PreservationApproval struct {
	Snapshot                  string          `json:"snapshot_sha256"`
	Manifest                  []LocalOriginal `json:"manifest"`
	IndependentRecoverySHA256 string          `json:"independent_recovery_sha256"`
}

func validDigest(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}

// DemocracyIdentity reads retained source provenance, including membership in a
// mixed/curated collection. Changing the catalog collection cannot bypass it.
func DemocracyIdentity(collection Collection, rep Representation, raw []byte) (bool, error) {
	if collection.ID == DemocracyNow || collection.SourceItem == DemocracyNow || rep.SourceItemID == DemocracyNow {
		return true, nil
	}
	var envelope struct {
		Metadata struct {
			Collection json.RawMessage `json:"collection"`
		} `json:"metadata"`
		SourceCollections []string `json:"source_collections"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return false, fmt.Errorf("invalid source provenance")
	}
	ids := envelope.SourceCollections
	if len(envelope.Metadata.Collection) > 0 && string(envelope.Metadata.Collection) != "null" {
		var list []string
		if err := json.Unmarshal(envelope.Metadata.Collection, &list); err != nil {
			var one string
			if err = json.Unmarshal(envelope.Metadata.Collection, &one); err != nil {
				return false, fmt.Errorf("invalid source collection identity")
			}
			list = []string{one}
		}
		ids = append(ids, list...)
	}
	for _, id := range ids {
		if id == DemocracyNow {
			return true, nil
		}
	}
	return false, nil
}

func AuthorizeTransport(collection Collection, rep Representation, raw []byte, review TransportReview, approval PreservationApproval, action Action, file string) error {
	if err := collection.Validate(); err != nil {
		return err
	}
	if err := rep.Validate(); err != nil {
		return err
	}
	if Digest(raw) != rep.Snapshot {
		return fmt.Errorf("source inventory does not match bound snapshot")
	}
	if review.Snapshot != rep.Snapshot || !validDigest(review.Snapshot) || strings.TrimSpace(review.Notes) == "" {
		return fmt.Errorf("transport review unavailable or stale")
	}
	member := false
	for _, id := range rep.Collections {
		if id == collection.ID {
			member = true
		}
	}
	if !member {
		return fmt.Errorf("representation outside reviewed collection")
	}
	selected := false
	seen := map[string]bool{}
	validFile := SafeFile
	if collection.Policy == Stream && rep.Media == AudioMedia {
		validFile = SourceAudioFile
	}
	for _, name := range review.Files {
		if !validFile(name) || seen[name] {
			return fmt.Errorf("invalid reviewed files")
		}
		seen[name] = true
		if name == file {
			selected = true
		}
	}
	if !selected {
		return fmt.Errorf("file outside exact transport allowlist")
	}
	belongs := file == rep.File
	if rep.Extension.Audio != nil {
		for _, track := range rep.Extension.Audio.Tracks {
			if track.File == file {
				belongs = true
			}
		}
	}
	if rep.Extension.Print != nil {
		for _, companion := range rep.Extension.Print.Companions {
			if companion.File == file {
				belongs = true
			}
		}
	}
	if !belongs {
		return fmt.Errorf("file outside representation edition")
	}
	// Even a reviewed spelling must exist in the bound complete source snapshot.
	var source struct {
		Files []struct {
			Name    string          `json:"name"`
			Private json.RawMessage `json:"private"`
		} `json:"files"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		return fmt.Errorf("source file inventory unavailable")
	}
	present := false
	for _, f := range source.Files {
		if f.Name == file && (len(f.Private) == 0 || string(f.Private) == "false" || string(f.Private) == `"false"`) {
			present = true
		}
	}
	if !present {
		return fmt.Errorf("source file absent or restricted")
	}
	democracy, err := DemocracyIdentity(collection, rep, raw)
	if err != nil {
		return err
	}
	if democracy && collection.Policy != Stream {
		return fmt.Errorf("Democracy source must remain source-streaming")
	}
	if (collection.Policy == Stream || democracy) && (review.Acquisition || review.Relay || review.Download) {
		return fmt.Errorf("streaming-only source cannot acquire, relay or enable downloads")
	}
	allowed := false
	switch action {
	case ReadAction:
		allowed = review.Read && rep.Media == PrintMedia
	case PlayAction:
		allowed = review.Playback && (rep.Media == AudioMedia || rep.Media == VideoMedia)
	case DownloadAction:
		allowed = review.Download
	case AcquireAction:
		allowed = review.Acquisition
	case RelayAction:
		allowed = review.Relay
	case ScheduleAction:
		allowed = review.Schedule && review.Playback && (rep.Media == AudioMedia || rep.Media == VideoMedia)
	default:
		return fmt.Errorf("unknown transport action")
	}
	if !allowed {
		return fmt.Errorf("transport action not reviewed")
	}
	if collection.Policy == Local {
		if approval.Snapshot != rep.Snapshot || !validDigest(approval.IndependentRecoverySHA256) || len(approval.Manifest) == 0 || len(approval.Manifest) > 1000 {
			return fmt.Errorf("local delivery requires bound manifest and independent recovery evidence")
		}
		var total int64
		found := false
		files := map[string]bool{}
		for _, original := range approval.Manifest {
			if !SafeFile(original.File) || files[original.File] || original.Bytes <= 0 || original.Bytes > collection.Budget-total || !validDigest(original.SHA256) || !filepath.IsLocal(original.Path) || filepath.Clean(original.Path) != original.Path || original.Path == "." || strings.ContainsRune(original.Path, '\x00') {
				return fmt.Errorf("invalid original or manifest exceeds reviewed budget")
			}
			files[original.File] = true
			total += original.Bytes
			if original.File == file {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("file outside preservation manifest")
		}
	}
	return nil
}
