package sourcecatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ReviewAudio is a distinct operator decision. Metadata import never grants
// playback, downloads, acquisition, scheduling, relay or public visibility.
func (s *Store) ReviewAudio(ctx context.Context, id, notes string) error {
	if !domain.ValidID(id) || strings.TrimSpace(notes) == "" || len(notes) > 4000 {
		return fmt.Errorf("source identity and bounded playback review notes required")
	}
	collection, err := s.Core.GetCollection(ctx, s.Collection)
	if err != nil {
		return err
	}
	if collection.Policy != domain.Stream {
		return fmt.Errorf("source audio review requires a source-streaming collection; local preservation uses the existing manifest workflow")
	}
	detail, err := s.Detail(ctx, id)
	if err != nil {
		return err
	}
	if detail["unavailable"] == true {
		return fmt.Errorf("source is unavailable")
	}
	raw, err := s.Raw(ctx, id)
	if err != nil {
		return err
	}
	var snapshot Snapshot
	if err = json.Unmarshal(raw, &snapshot); err != nil {
		return err
	}
	preferred := map[string]map[string]any{}
	ranks := map[string]int{".mp3": 0, ".m4a": 1, ".ogg": 2, ".flac": 3, ".wav": 4}
	byName := map[string]map[string]any{}
	for _, file := range snapshot.Files {
		byName[sourceFilename(file["name"])] = file
	}
	for _, file := range snapshot.Files {
		name := sourceFilename(file["name"])
		rank, known := ranks[strings.ToLower(filepath.Ext(name))]
		flag, hasPrivate := file["private"]
		if hasPrivate && flag != false && flag != "false" {
			continue
		}
		if !known || !domain.SafeFile(name) {
			continue
		}
		original := sourceFilename(file["original"])
		if original == "" {
			original = name
		}
		if parent := byName[original]; parent != nil && sourceFilename(parent["original"]) != "" {
			original = sourceFilename(parent["original"])
		}
		key := strings.TrimSuffix(original, filepath.Ext(original))
		existing := preferred[key]
		if existing == nil || rank < ranks[strings.ToLower(filepath.Ext(sourceFilename(existing["name"])))] {
			preferred[key] = file
		}
	}
	files := []map[string]any{}
	for _, file := range preferred {
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool {
		return naturalLess(sourceFilename(files[i]["name"]), sourceFilename(files[j]["name"]))
	})
	if len(files) == 0 || len(files) > 1000 {
		return fmt.Errorf("no bounded, supported public audio files with safe source filenames")
	}
	var fields map[string]string
	if err = json.Unmarshal(detail["metadata"].(json.RawMessage), &fields); err != nil {
		return err
	}
	tracks := []domain.AudioTrack{}
	names := []string{}
	for index, file := range files {
		name := sourceFilename(file["name"])
		title := Text(file["title"])
		if title == "" {
			title = strings.TrimSuffix(name, filepath.Ext(name))
		}
		tracks = append(tracks, domain.AudioTrack{ID: fmt.Sprintf("track-%04d", index), File: name, Title: title})
		names = append(names, name)
	}
	itemID := "audio-" + domain.Digest([]byte(s.Collection + "\x00" + id))[:24]
	hash := detail["snapshot_sha256"].(string)
	ownership, _ := json.Marshal(map[string]string{"source_catalog_item": id})
	item := domain.CatalogItem{ID: itemID, Title: fields["title"], Description: fields["description"], Date: fields["date"], Creator: fields["creator"], Editorial: ownership, Collections: []string{s.Collection}, Representations: []domain.Representation{{ID: "audio", Media: domain.AudioMedia, SourceItemID: id, Snapshot: hash, File: names[0], Collections: []string{s.Collection}, Extension: domain.MediaExtension{Audio: &domain.AudioExtension{RecordingID: id, Tracks: tracks, Credits: map[string]string{"source": fields["source"], "taper": fields["taper"], "transferer": fields["transferer"], "venue": fields["venue"]}}}}}}
	if err = item.Validate(); err != nil {
		return err
	}
	var owner string
	err = s.Core.Pool.QueryRow(ctx, `SELECT editorial->>'source_catalog_item' FROM catalog_items WHERE id=$1`, itemID).Scan(&owner)
	if err == pgx.ErrNoRows {
		if err = s.Core.InsertCatalog(ctx, item, notes); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		if owner != id {
			return fmt.Errorf("existing catalog mapping has another owner")
		}
		tx, err := s.Core.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		var sourceID string
		if err = tx.QueryRow(ctx, `SELECT source_item_id FROM catalog_representations WHERE item_id=$1 AND id='audio' FOR UPDATE`, itemID).Scan(&sourceID); err != nil {
			return err
		}
		if sourceID != id {
			return fmt.Errorf("source mapping changed")
		}
		extension, _ := json.Marshal(item.Representations[0].Extension)
		payload, _ := json.Marshal(item)
		if _, err = tx.Exec(ctx, `UPDATE catalog_items SET title=$2,description=$3,date=$4,creator=$5,updated_at=now() WHERE id=$1`, itemID, item.Title, item.Description, item.Date, item.Creator); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE catalog_representations SET source_hash=$2,file=$3,extension=$4 WHERE item_id=$1 AND id='audio'`, itemID, hash, names[0], extension); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO catalog_review_history(item_id,payload,notes) VALUES($1,$2,$3)`, itemID, payload, notes); err != nil {
			return err
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
	}
	return s.Core.ReviewTransport(ctx, itemID, "audio", s.Collection, domain.TransportReview{Snapshot: hash, Files: names, Playback: true, Notes: notes}, domain.PreservationApproval{})
}

func sourceFilename(value any) string { text, _ := value.(string); return text }
