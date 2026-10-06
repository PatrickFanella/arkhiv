package community

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RecoveryEvent contains only opaque IDs and bounded moderation state. The
// independently retained directory is outside the personal backup rotation.
type RecoveryEvent struct {
	Media      string    `json:"media,omitempty"`
	Version    int       `json:"version"`
	Deployment string    `json:"deployment"`
	At         time.Time `json:"at"`
	Kind       string    `json:"kind"`
	ID         string    `json:"opaque_id"`
	Until      time.Time `json:"until,omitempty"`
}

func WriteRecoveryEvent(directory string, event RecoveryEvent) error {
	if !filepath.IsAbs(directory) || event.Version != 1 || len(event.ID) != 64 {
		return fmt.Errorf("independent recovery journal required")
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("private recovery journal directory required")
	}
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(directory, ".recovery-")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if _, err = file.Write(raw); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(temporary, filepath.Join(directory, "recovery-"+Hash(string(raw))+".json")); err != nil {
		return err
	}
	dir, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
func (s *Server) journal(kind, id string, until time.Time) error {
	return s.journalMedia(kind, id, until, "")
}
func (s *Server) journalMedia(kind, id string, until time.Time, media string) error {
	return WriteRecoveryEvent(s.RecoveryDirectory, RecoveryEvent{Version: 1, Deployment: s.Trust.Deployment, At: s.now(), Kind: kind, ID: id, Until: until, Media: media})
}
func MergeRecoveryJournal(c Checkpoint, directory string, now time.Time) (Checkpoint, error) {
	if !filepath.IsAbs(directory) {
		return c, fmt.Errorf("independent recovery journal required")
	}
	if c.Version != 1 || c.At.After(now) {
		return c, fmt.Errorf("invalid checkpoint identity")
	}
	c.At = now
	entries, err := os.ReadDir(directory)
	if err != nil {
		return c, err
	}
	restrictions := map[string]RecoveryEvent{}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "recovery-") {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		if !entry.Type().IsRegular() {
			return c, fmt.Errorf("invalid journal entry")
		}
		info, err := entry.Info()
		if err != nil || info.Size() > 8192 {
			return c, fmt.Errorf("journal entry exceeds limit")
		}
		raw, err := os.ReadFile(path)
		if err != nil || len(raw) > 8192 {
			return c, fmt.Errorf("journal entry unavailable")
		}
		if entry.Name() != "recovery-"+Hash(string(raw))+".json" {
			return c, fmt.Errorf("journal integrity mismatch")
		}
		var event RecoveryEvent
		if json.Unmarshal(raw, &event) != nil || event.Version != 1 || len(event.ID) != 64 || event.At.After(now) {
			return c, fmt.Errorf("invalid recovery journal")
		}
		if event.Deployment != c.Deployment || !event.At.After(now.Add(-90*24*time.Hour)) {
			continue
		}
		switch event.Kind {
		case "account", "message":
			c.Deletions = append(c.Deletions, Deletion{Kind: event.Kind, ID: event.ID, At: event.At})
		case "hide":
			c.Withdrawals = append(c.Withdrawals, event.ID)
		case "restriction":
			if !validMedia(normalizedMedia(event.Media)) {
				return c, fmt.Errorf("invalid restriction media")
			}
			key := normalizedMedia(event.Media) + ":" + event.ID
			previous, ok := restrictions[key]
			if !ok || previous.At.Before(event.At) {
				restrictions[key] = event
			}
		default:
			return c, fmt.Errorf("unknown recovery event")
		}
	}
	filtered := c.Restrictions[:0]
	for _, restriction := range c.Restrictions {
		if _, superseded := restrictions[normalizedMedia(restriction.Media)+":"+restriction.User]; !superseded {
			filtered = append(filtered, restriction)
		}
	}
	c.Restrictions = filtered
	for _, event := range restrictions {
		if event.Until.After(now) {
			c.Restrictions = append(c.Restrictions, Restriction{User: event.ID, Until: event.Until, Media: normalizedMedia(event.Media)})
		}
	}
	return c, nil
}

func PruneRecoveryJournal(directory, deployment string, now time.Time) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "recovery-") {
			continue
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("invalid journal entry")
		}
		path := filepath.Join(directory, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if entry.Name() != "recovery-"+Hash(string(raw))+".json" {
			return fmt.Errorf("journal integrity mismatch")
		}
		var event RecoveryEvent
		if json.Unmarshal(raw, &event) != nil || event.Version != 1 {
			return fmt.Errorf("invalid journal entry")
		}
		if event.Deployment == deployment && !event.At.After(now.Add(-90*24*time.Hour)) {
			if err = os.Remove(path); err != nil {
				return err
			}
		}
	}
	return nil
}
