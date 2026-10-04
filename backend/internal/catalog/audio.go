package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
)

func (d *DB) AudioAsset(ctx context.Context, item, representation, collection string) (domain.AudioAsset, error) {
	var raw []byte
	var asset domain.AudioAsset
	err := d.Pool.QueryRow(ctx, `SELECT receipt FROM audio_assets WHERE item_id=$1 AND representation_id=$2 AND collection_id=$3`, item, representation, collection).Scan(&raw)
	if err != nil {
		return asset, err
	}
	err = json.Unmarshal(raw, &asset)
	return asset, err
}

// RegisterAudioAsset records an operator-verified derivative; it never acquires
// sources or changes transport rights. Publication/browser gates remain separate.
func (d *DB) RegisterAudioAsset(ctx context.Context, item, representation, collection string, asset domain.AudioAsset, notes string) error {
	if strings.TrimSpace(notes) == "" {
		return fmt.Errorf("audio receipt requires review notes")
	}
	entry, err := d.Transport(ctx, item, representation, collection)
	if err != nil {
		return err
	}
	if err = domain.AuthorizeAudioAsset(entry.Collection, entry.Representation, entry.Raw, entry.Review, entry.Preservation, asset, domain.PlayAction); err != nil {
		return err
	}
	data, err := json.Marshal(asset)
	if err != nil {
		return err
	}
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO audio_assets(item_id,representation_id,collection_id,receipt,notes) VALUES($1,$2,$3,$4,$5) ON CONFLICT(item_id,representation_id,collection_id) DO UPDATE SET receipt=excluded.receipt,notes=excluded.notes,created_at=now()`, item, representation, collection, data, notes); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audio_asset_history(item_id,representation_id,collection_id,receipt,notes) VALUES($1,$2,$3,$4,$5)`, item, representation, collection, data, notes); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (d *DB) StoreAudioSchedule(ctx context.Context, schedule domain.AudioSchedule, notes string) error {
	if err := schedule.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(notes) == "" {
		return fmt.Errorf("schedule requires review notes")
	}
	for _, p := range schedule.Programs {
		e, err := d.Transport(ctx, p.Item, p.Representation, schedule.Collection)
		if err != nil {
			return err
		}
		a, err := d.AudioAsset(ctx, p.Item, p.Representation, schedule.Collection)
		if err != nil {
			return err
		}
		if err = domain.AuthorizeAudioAsset(e.Collection, e.Representation, e.Raw, e.Review, e.Preservation, a, domain.ScheduleAction); err != nil {
			return err
		}
		if p.Duration > a.Duration {
			return fmt.Errorf("program exceeds preserved audio duration")
		}
	}
	data, err := json.Marshal(schedule)
	if err != nil {
		return err
	}
	_, err = d.Pool.Exec(ctx, `INSERT INTO audio_schedules(collection_id,revision,schedule,notes) VALUES($1,$2,$3,$4)`, schedule.Collection, schedule.Revision, data, notes)
	return err
}
func (d *DB) AudioSchedule(ctx context.Context, collection string) (domain.AudioSchedule, error) {
	var s domain.AudioSchedule
	var data []byte
	err := d.Pool.QueryRow(ctx, `SELECT schedule FROM audio_schedules WHERE collection_id=$1 ORDER BY created_at DESC,revision DESC LIMIT 1`, collection).Scan(&data)
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(data, &s)
	return s, err
}
