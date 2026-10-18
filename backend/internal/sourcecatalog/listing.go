package sourcecatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ListingDecision affects discovery in the source catalog, not source retention
// or independent transport permissions. Listed is required to avoid an omitted
// boolean silently excluding records.
type ListingDecision struct {
	Identifiers []string `json:"identifiers"`
	Listed      *bool    `json:"listed"`
}

func (s *Store) IsListed(ctx context.Context, id string) (bool, error) {
	var listed bool
	err := s.Core.Pool.QueryRow(ctx, `SELECT NOT excluded FROM source_catalog_items WHERE collection_id=$1 AND identifier=$2`, s.Collection, id).Scan(&listed)
	return listed, err
}

// SetListing applies an audited, bounded operator decision atomically. Import
// never writes excluded, so subsequent source refreshes preserve this decision.
func (s *Store) SetListing(ctx context.Context, decision ListingDecision, notes string) error {
	if decision.Listed == nil || len(decision.Identifiers) < 1 || len(decision.Identifiers) > 1000 || strings.TrimSpace(notes) == "" || len(notes) > 4000 {
		return fmt.Errorf("1–1000 source identifiers, explicit listed decision and bounded notes required")
	}
	ids := append([]string(nil), decision.Identifiers...)
	sort.Strings(ids)
	for i, id := range ids {
		if !ItemID.MatchString(id) || (i > 0 && id == ids[i-1]) {
			return fmt.Errorf("invalid or duplicate source identifier")
		}
	}
	tx, err := s.Core.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT identifier FROM source_catalog_items WHERE collection_id=$1 AND identifier=ANY($2) ORDER BY identifier FOR UPDATE`, s.Collection, ids)
	if err != nil {
		return err
	}
	found, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	if len(found) != len(ids) {
		return fmt.Errorf("one or more source identifiers are not indexed in this collection")
	}
	patch, _ := json.Marshal(map[string]bool{"listed": *decision.Listed})
	if _, err = tx.Exec(ctx, `INSERT INTO source_catalog_history(collection_id,item_id,patch,notes)
 SELECT collection_id,identifier,$3,$4 FROM source_catalog_items
 WHERE collection_id=$1 AND identifier=ANY($2) AND excluded=$5`, s.Collection, ids, patch, notes, *decision.Listed); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE source_catalog_items SET excluded=$3,updated_at=now()
 WHERE collection_id=$1 AND identifier=ANY($2) AND excluded<>$3`, s.Collection, ids, !*decision.Listed); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
