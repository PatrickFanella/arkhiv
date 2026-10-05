package print

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"git.subcult.tv/subculture-collective/vhness/internal/preservation"
)

// RecoveryIndex is prepared only from an independently restored donor and its
// verified file inventory. It does not grant reading or publication permission.
type RecoveryIndex struct {
	Recovery   string         `json:"independent_recovery_sha256"`
	Snapshot   string         `json:"snapshot_sha256"`
	Original   string         `json:"original_sha256"`
	Pages      int            `json:"pages"`
	Status     string         `json:"text_status"`
	Tool       string         `json:"tool"`
	Sections   map[int]string `json:"sections"`
	TextFile   string         `json:"text_file"`
	TextSHA    string         `json:"text_sha256"`
	CoverPath  string         `json:"cover_path"`
	CoverSHA   string         `json:"cover_sha256"`
	CoverBytes int64          `json:"cover_bytes"`
}

func RestoreIndex(ctx context.Context, db *catalog.DB, mediaRoot, derivativeRoot, item, representation, collection string, input RecoveryIndex) error {
	entry, err := db.Transport(ctx, item, representation, collection)
	if err != nil {
		return err
	}
	file := entry.Representation.File
	if entry.Representation.Media != domain.PrintMedia || entry.Collection.Policy != domain.Local || input.Recovery != entry.Preservation.IndependentRecoverySHA256 || input.Snapshot != entry.Representation.Snapshot {
		return fmt.Errorf("print recovery does not match current preserved edition")
	}
	if err = domain.AuthorizeTransport(entry.Collection, entry.Representation, entry.Raw, entry.Review, entry.Preservation, domain.ReadAction, file); err != nil {
		return err
	}
	asset, err := original(entry, file)
	if err != nil {
		return err
	}
	if asset.SHA256 != input.Original {
		return fmt.Errorf("recovered original identity differs")
	}
	f, err := preservation.Verify(mediaRoot, asset.Path, asset.SHA256, asset.Bytes)
	if err != nil {
		return err
	}
	f.Close()
	// Bind pages, tool, and every text section to the immutable recovered database,
	// rather than trusting a second operator-supplied transcription.
	var derivative, text []byte
	err = db.Pool.QueryRow(ctx, `SELECT derivative,sections FROM retained_print_indexes WHERE collection_id=$1 AND recovery_sha256=$2 AND asset_sha256=$5
 AND EXISTS(SELECT 1 FROM jsonb_array_elements(variants) row WHERE row->>'id'=$3 AND row->>'document_id'=$4 AND row->>'asset_sha'=$5 AND row->>'filename'=$6)`, collection, input.Recovery, representation, item, input.Original, file).Scan(&derivative, &text)
	if err != nil {
		return fmt.Errorf("recovered edition/index unavailable: %w", err)
	}
	var retained struct {
		Pages  int    `json:"pages"`
		Status string `json:"status"`
		Tool   string `json:"tool"`
	}
	var rows []struct {
		Page int    `json:"page"`
		Text string `json:"text"`
	}
	if json.Unmarshal(derivative, &retained) != nil || json.Unmarshal(text, &rows) != nil || retained.Pages != input.Pages || retained.Status != input.Status || retained.Tool != input.Tool {
		return fmt.Errorf("recovered index metadata differs")
	}
	sections := map[int]string{}
	for _, row := range rows {
		if _, exists := sections[row.Page]; exists {
			return fmt.Errorf("duplicate recovered page")
		}
		sections[row.Page] = row.Text
	}
	if !reflect.DeepEqual(sections, input.Sections) {
		return fmt.Errorf("recovered text differs")
	}
	if input.Status == "document-text" {
		companion, err := original(entry, input.TextFile)
		if err != nil || companion.SHA256 != input.TextSHA {
			return fmt.Errorf("recovered OCR companion differs")
		}
		if err = domain.AuthorizeTransport(entry.Collection, entry.Representation, entry.Raw, entry.Review, entry.Preservation, domain.ReadAction, input.TextFile); err != nil {
			return err
		}
		f, err := preservation.Verify(mediaRoot, companion.Path, companion.SHA256, companion.Bytes)
		if err != nil {
			return err
		}
		f.Close()
	} else if input.TextFile != "" || input.TextSHA != "" {
		return fmt.Errorf("unexpected OCR companion")
	}
	if input.CoverPath != "" {
		if input.CoverBytes <= 0 || input.CoverBytes > 32<<20 {
			return fmt.Errorf("recovered cover exceeds bound")
		}
		f, err := preservation.Verify(derivativeRoot, input.CoverPath, input.CoverSHA, input.CoverBytes)
		if err != nil {
			return err
		}
		f.Close()
	} else if input.CoverSHA != "" || input.CoverBytes != 0 {
		return fmt.Errorf("incomplete recovered cover")
	}
	index := domain.PrintIndex{Snapshot: input.Snapshot, OriginalSHA256: input.Original, Pages: input.Pages, TextStatus: input.Status, Tool: input.Tool, TextFile: input.TextFile, TextSHA256: input.TextSHA, CoverPath: input.CoverPath, CoverSHA256: input.CoverSHA, CoverBytes: input.CoverBytes}
	current, err := db.Transport(ctx, item, representation, collection)
	if err != nil {
		return err
	}
	if current.Representation.Snapshot != input.Snapshot || current.Preservation.IndependentRecoverySHA256 != input.Recovery {
		return fmt.Errorf("source changed during recovery")
	}
	for _, name := range []string{file, input.TextFile} {
		if name != "" {
			if err = domain.AuthorizeTransport(current.Collection, current.Representation, current.Raw, current.Review, current.Preservation, domain.ReadAction, name); err != nil {
				return err
			}
		}
	}
	return db.StorePrintIndex(ctx, item, representation, file, index, sections)
}
