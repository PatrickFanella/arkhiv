package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"github.com/jackc/pgx/v5"
)

type RetainedExport struct {
	Collection string                       `json:"collection"`
	Module     string                       `json:"module"`
	Recovery   string                       `json:"independent_recovery_sha256"`
	Tables     map[string][]json.RawMessage `json:"tables"`
	Counts     map[string]int               `json:"counts"`
}

// RetainLegacy keeps the complete independently recovered donor export and
// projects its audio inventory. It never imports serving approvals or schedules.
func (d *DB) RetainLegacy(ctx context.Context, raw []byte, collection, notes string) error {
	var e RetainedExport
	if len(raw) > 256<<20 || json.Unmarshal(raw, &e) != nil || e.Collection != collection || !domain.ValidID(collection) || (e.Module != "audio" && e.Module != "print") || len(e.Recovery) != 64 || strings.TrimSpace(notes) == "" || len(e.Tables) == 0 || len(e.Tables) != len(e.Counts) {
		return fmt.Errorf("invalid retained export")
	}
	for _, c := range e.Recovery {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return fmt.Errorf("invalid recovery digest")
		}
	}
	for table, rows := range e.Tables {
		count, ok := e.Counts[table]
		if !ok || len(rows) != count {
			return fmt.Errorf("retained table counts differ: %s", table)
		}
	}
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var previous string
	previousErr := tx.QueryRow(ctx, `SELECT sha256 FROM retained_legacy_exports WHERE collection_id=$1 AND module=$2 ORDER BY created_at DESC LIMIT 1`, collection, e.Module).Scan(&previous)
	if previousErr != nil && previousErr != pgx.ErrNoRows {
		return previousErr
	}
	if previousErr == nil && previous != domain.Digest(raw) {
		return fmt.Errorf("retained export differs; review a later recovery import explicitly")
	}
	if _, err = tx.Exec(ctx, `INSERT INTO retained_legacy_exports(collection_id,module,sha256,recovery_sha256,payload,notes) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`, collection, e.Module, domain.Digest(raw), e.Recovery, raw, notes); err != nil {
		return err
	}
	if e.Module == "audio" {
		rows, ok := e.Tables["source_items"]
		if !ok {
			return fmt.Errorf("retained audio inventory required")
		}
		inventory, _ := json.Marshal(rows)
		// The recovery record stays immutable. This derived index can be rebuilt from
		// it; current core decisions independently govern public visibility/playback.
		var existing int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM retained_audio_items WHERE collection_id=$1`, collection).Scan(&existing); err != nil {
			return err
		}
		if existing != 0 && existing != len(rows) {
			return fmt.Errorf("retained inventory differs; review its replacement explicitly")
		}
		_, err = tx.Exec(ctx, `INSERT INTO retained_audio_items(collection_id,id,show_date,date_text,artist,venue,location,title,kind,source,lineage,taper,transferer,downloads,avg_rating,num_reviews,item_size,added_at,tracks,songs,duration,warnings,summary,snapshot_sha256,observed_at,updated_at)
  SELECT $1,id,show_date,date_text,artist,venue,location,title,kind,source,lineage,taper,transferer,downloads,avg_rating,num_reviews,item_size,added_at,tracks,songs,duration,warnings,summary,snapshot_sha256,observed_at,updated_at FROM jsonb_to_recordset($2::jsonb) AS x(id text,show_date date,date_text text,artist text,venue text,location text,title text,kind text,source text,lineage text,taper text,transferer text,downloads int,avg_rating real,num_reviews int,item_size bigint,added_at timestamptz,tracks jsonb,songs text,duration float8,warnings jsonb,summary jsonb,snapshot_sha256 text,observed_at timestamptz,updated_at timestamptz) ON CONFLICT(collection_id,id) DO NOTHING`, collection, inventory)
		if err != nil {
			return err
		}
	}
	if e.Module == "print" {
		// Materialize once. Scanning and decompressing a complete donor export for
		// every edition makes large independently recovered imports impractical.
		_, err = tx.Exec(ctx, `WITH recovered AS MATERIALIZED (
 SELECT payload->'tables' AS tables FROM retained_legacy_exports WHERE collection_id=$1 AND module='print' AND recovery_sha256=$2
), texts AS MATERIALIZED (
 SELECT row->>'asset_sha' AS asset,jsonb_agg(row) AS sections FROM recovered,jsonb_array_elements(COALESCE(tables->'page_text','[]'::jsonb)) row GROUP BY row->>'asset_sha'
), variants AS MATERIALIZED (
 SELECT row->>'asset_sha' AS asset,jsonb_agg(row) AS variants FROM recovered,jsonb_array_elements(COALESCE(tables->'variants','[]'::jsonb)) row GROUP BY row->>'asset_sha'
)
 INSERT INTO retained_print_indexes(collection_id,recovery_sha256,asset_sha256,derivative,sections,variants)
 SELECT $1,$2,row->>'asset_sha',row,COALESCE(texts.sections,'[]'::jsonb),COALESCE(variants.variants,'[]'::jsonb)
 FROM recovered CROSS JOIN LATERAL jsonb_array_elements(COALESCE(tables->'derivatives','[]'::jsonb)) row
 LEFT JOIN texts ON texts.asset=row->>'asset_sha' LEFT JOIN variants ON variants.asset=row->>'asset_sha'
 ON CONFLICT DO NOTHING`, collection, e.Recovery)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
