package catalog

import (
	"context"
	"fmt"
	"strings"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
)

func (d *DB) PrintIndex(ctx context.Context, item, representation, file string) (domain.PrintIndex, error) {
	var index domain.PrintIndex
	err := d.Pool.QueryRow(ctx, `SELECT snapshot_sha256,original_sha256,pages,text_status,tool,text_file,text_sha256,cover_path,cover_sha256,cover_bytes FROM print_indexes WHERE item_id=$1 AND representation_id=$2 AND file=$3`, item, representation, file).Scan(&index.Snapshot, &index.OriginalSHA256, &index.Pages, &index.TextStatus, &index.Tool, &index.TextFile, &index.TextSHA256, &index.CoverPath, &index.CoverSHA256, &index.CoverBytes)
	return index, err
}

func (d *DB) StorePrintIndex(ctx context.Context, item, representation, file string, index domain.PrintIndex, sections map[int]string) error {
	if index.Pages < 1 || index.Pages > 10000 || (index.TextStatus != "page-text" && index.TextStatus != "document-text" && index.TextStatus != "no-text") {
		return fmt.Errorf("invalid print index")
	}
	var bytes int
	for page, text := range sections {
		if page < 0 || page > index.Pages {
			return fmt.Errorf("invalid indexed page")
		}
		bytes += len(text)
		if bytes > 16<<20 {
			return fmt.Errorf("indexed text exceeds bound")
		}
	}
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO print_indexes(item_id,representation_id,file,snapshot_sha256,original_sha256,pages,text_status,tool,text_file,text_sha256,cover_path,cover_sha256,cover_bytes) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT(item_id,representation_id,file) DO UPDATE SET snapshot_sha256=excluded.snapshot_sha256,original_sha256=excluded.original_sha256,pages=excluded.pages,text_status=excluded.text_status,tool=excluded.tool,text_file=excluded.text_file,text_sha256=excluded.text_sha256,cover_path=excluded.cover_path,cover_sha256=excluded.cover_sha256,cover_bytes=excluded.cover_bytes,updated_at=now()`, item, representation, file, index.Snapshot, index.OriginalSHA256, index.Pages, index.TextStatus, index.Tool, index.TextFile, index.TextSHA256, index.CoverPath, index.CoverSHA256, index.CoverBytes); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM print_page_text WHERE item_id=$1 AND representation_id=$2 AND file=$3`, item, representation, file); err != nil {
		return err
	}
	for page, text := range sections {
		if strings.TrimSpace(text) != "" {
			if _, err = tx.Exec(ctx, `INSERT INTO print_page_text(item_id,representation_id,file,page,text) VALUES($1,$2,$3,$4,$5)`, item, representation, file, page, text); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func (d *DB) PrintSearch(ctx context.Context, item, representation, file, query string) ([]domain.PrintHit, error) {
	if strings.TrimSpace(query) == "" || len(query) > 200 {
		return nil, fmt.Errorf("invalid print search")
	}
	rows, err := d.Pool.Query(ctx, `SELECT page,left(text,350) FROM print_page_text WHERE item_id=$1 AND representation_id=$2 AND file=$3 AND search @@ plainto_tsquery('simple',$4) ORDER BY page LIMIT 50`, item, representation, file, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hits := []domain.PrintHit{}
	for rows.Next() {
		var hit domain.PrintHit
		if err = rows.Scan(&hit.Page, &hit.Snippet); err != nil {
			return nil, err
		}
		hits = append(hits, hit)
	}
	return hits, rows.Err()
}
