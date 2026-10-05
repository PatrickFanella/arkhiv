package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
)

// CurationManifest records editorial intent, never delivery or public authority.
type CurationManifest struct {
	Schema     string            `json:"schema"`
	Intent     string            `json:"intent"`
	Target     string            `json:"target_collection"`
	Access     map[string]bool   `json:"access"`
	Mappings   []CurationMapping `json:"mappings"`
	Unresolved []json.RawMessage `json:"unresolved_items"`
}
type CurationMapping struct {
	Item     domain.CatalogItem `json:"catalog_item"`
	Evidence string             `json:"evidence_sha256"`
	Notes    string             `json:"curation_notes"`
	Access   map[string]bool    `json:"access"`
}

func noCurationAccess(a map[string]bool) bool {
	keys := []string{"read", "playback", "download", "acquisition", "relay", "schedule", "public"}
	if len(a) != len(keys) {
		return false
	}
	for _, key := range keys {
		v, ok := a[key]
		if !ok || v {
			return false
		}
	}
	return true
}
func (m CurationManifest) Validate() error {
	if m.Schema != "arkhiv.curation.v1" || m.Intent != "reviewed-curation" || !domain.ValidID(m.Target) || !noCurationAccess(m.Access) || len(m.Mappings) > 1000 || len(m.Unresolved) > 1000 {
		return fmt.Errorf("invalid curation manifest or attempted access grant")
	}
	seen := map[string]bool{}
	for _, mapping := range m.Mappings {
		if seen[mapping.Item.ID] || strings.TrimSpace(mapping.Notes) == "" || !noCurationAccess(mapping.Access) || len(mapping.Evidence) != 64 || len(mapping.Item.Representations) == 0 {
			return fmt.Errorf("invalid curation mapping")
		}
		for _, c := range mapping.Evidence {
			if !strings.ContainsRune("0123456789abcdef", c) {
				return fmt.Errorf("invalid curation evidence digest")
			}
		}
		seen[mapping.Item.ID] = true
	}
	return nil
}

// PrepareCuration retains a reviewable draft against the current catalog. It
// changes no catalog membership, source response, schedule or transport review.
// Source inventories missing in this instance require a separate source import.
func (d *DB) PrepareCuration(ctx context.Context, raw []byte, scope domain.CollectionScope, notes string) (string, error) {
	if len(raw) > 16<<20 || strings.TrimSpace(notes) == "" {
		return "", fmt.Errorf("bounded manifest and review notes required")
	}
	var manifest CurationManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return "", err
	}
	if err := manifest.Validate(); err != nil {
		return "", err
	}
	if !scope.Allows(manifest.Target) {
		return "", fmt.Errorf("curation target outside instance scope")
	}
	if _, err := d.GetCollection(ctx, manifest.Target); err != nil {
		return "", err
	}
	for _, mapping := range manifest.Mappings {
		current, err := d.Catalog(ctx, CatalogQuery{Scope: []string(scope), ID: mapping.Item.ID, Page: 1})
		if err != nil || len(current.Items) != 1 {
			return "", fmt.Errorf("curation item unavailable in this instance: %s", mapping.Item.ID)
		}
		item := current.Items[0]
		if item.Title != mapping.Item.Title || item.Description != mapping.Item.Description || item.Date != mapping.Item.Date || item.Creator != mapping.Item.Creator {
			return "", fmt.Errorf("curation metadata changed: %s", item.ID)
		}
		seen := map[string]bool{}
		for _, rep := range mapping.Item.Representations {
			if seen[rep.ID] {
				return "", fmt.Errorf("duplicate curation representation")
			}
			seen[rep.ID] = true
			found := false
			for _, actual := range item.Representations {
				if actual.ID != rep.ID {
					continue
				}
				a, _ := json.Marshal(actual.Extension)
				b, _ := json.Marshal(rep.Extension)
				if actual.Media == rep.Media && actual.SourceItemID == rep.SourceItemID && actual.Snapshot == rep.Snapshot && actual.File == rep.File && actual.LegacyEpisodeID == rep.LegacyEpisodeID && string(a) == string(b) {
					found = true
				}
			}
			if !found {
				return "", fmt.Errorf("curation source, file or credits changed: %s", rep.ID)
			}
		}
	}
	hash := domain.Digest(raw)
	_, err := d.Pool.Exec(ctx, `INSERT INTO curation_drafts(sha256,collection_id,manifest,notes) VALUES($1,$2,$3,$4) ON CONFLICT(sha256) DO NOTHING`, hash, manifest.Target, raw, notes)
	return hash, err
}
