package source

import (
	"encoding/json"
	"fmt"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
)

// CatalogSnapshot validates source identity without treating documents or
// recordings as episodes. The caller persists the complete original bytes.
func CatalogSnapshot(id string, raw []byte) error {
	if !domain.ValidID(id) || len(raw) == 0 || len(raw) > 32<<20 {
		return fmt.Errorf("invalid or oversized source snapshot")
	}
	var payload struct {
		Metadata map[string]json.RawMessage `json:"metadata"`
		Files    []json.RawMessage          `json:"files"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	var identity string
	if err := json.Unmarshal(payload.Metadata["identifier"], &identity); err != nil || identity != id || payload.Files == nil {
		return fmt.Errorf("source identity or file inventory mismatch")
	}
	for _, file := range payload.Files {
		var entry map[string]json.RawMessage
		if err := json.Unmarshal(file, &entry); err != nil || entry == nil {
			return fmt.Errorf("source file inventory contains a non-object entry")
		}
	}
	return nil
}
