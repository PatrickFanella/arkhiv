package community

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Deletion struct {
	Kind string    `json:"kind"`
	ID   string    `json:"opaque_id"`
	At   time.Time `json:"deleted_at"`
}

func ReadCheckpoint(path string) (Checkpoint, error) {
	var c Checkpoint
	raw, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if len(raw) > 32<<20 {
		return c, fmt.Errorf("recovery checkpoint exceeds limit")
	}
	err = json.Unmarshal(raw, &c)
	return c, err
}

type Restriction struct {
	Media string    `json:"media,omitempty"`
	User  string    `json:"opaque_user_id"`
	Until time.Time `json:"until"`
}
type Checkpoint struct {
	Version      int           `json:"version"`
	Deployment   string        `json:"deployment"`
	At           time.Time     `json:"at"`
	Deletions    []Deletion    `json:"deletions"`
	Restrictions []Restriction `json:"restrictions"`
	Withdrawals  []string      `json:"withdrawals"`
}

func RecoveryCheckpoint(ctx context.Context, pool *pgxpool.Pool, deployment string, now time.Time) (Checkpoint, error) {
	c := Checkpoint{Version: 1, Deployment: deployment, At: now, Deletions: []Deletion{}, Restrictions: []Restriction{}, Withdrawals: []string{}}
	rows, err := pool.Query(ctx, `SELECT kind,opaque_id,deleted_at FROM personal_deletions WHERE deployment=$1 AND deleted_at>$2 ORDER BY deleted_at,opaque_id`, deployment, now.Add(-90*24*time.Hour))
	if err != nil {
		return c, err
	}
	for rows.Next() {
		var d Deletion
		if err = rows.Scan(&d.Kind, &d.ID, &d.At); err != nil {
			rows.Close()
			return c, err
		}
		c.Deletions = append(c.Deletions, d)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return c, err
	}
	rows, err = pool.Query(ctx, `SELECT user_id,until_at,media FROM chat_restrictions WHERE deployment=$1 AND until_at>$2 ORDER BY media,user_id`, deployment, now)
	if err != nil {
		return c, err
	}
	for rows.Next() {
		var r Restriction
		if err = rows.Scan(&r.User, &r.Until, &r.Media); err != nil {
			rows.Close()
			return c, err
		}
		c.Restrictions = append(c.Restrictions, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return c, err
	}
	rows, err = pool.Query(ctx, `SELECT id FROM chat_messages WHERE deployment=$1 AND hidden AND created_at>$2 ORDER BY id`, deployment, now.Add(-30*24*time.Hour))
	if err != nil {
		return c, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return c, err
		}
		c.Withdrawals = append(c.Withdrawals, id)
	}
	return c, rows.Err()
}

// ReapplyRecovery never restores database bytes. An operator restores a personal
// backup offline first, then supplies an independently retained CURRENT checkpoint.
// All sessions/flows are revoked before reads or posting may be reopened.
func ReapplyRecovery(ctx context.Context, pool *pgxpool.Pool, c Checkpoint, deployment string, now time.Time) error {
	if c.Version != 1 || c.Deployment != deployment || c.At.After(now) || now.Sub(c.At) > time.Hour {
		return fmt.Errorf("current deployment-scoped checkpoint required")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `DELETE FROM personal_sessions WHERE deployment=$1`, deployment); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM personal_login_flows WHERE deployment=$1`, deployment); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM chat_restrictions WHERE deployment=$1`, deployment); err != nil {
		return err
	}
	for _, d := range c.Deletions {
		if (d.Kind != "account" && d.Kind != "message") || len(d.ID) != 64 || d.At.After(c.At) || d.At.Before(now.Add(-90*24*time.Hour)) {
			return fmt.Errorf("invalid deletion checkpoint")
		}
		if _, err = tx.Exec(ctx, `INSERT INTO personal_deletions(deployment,kind,opaque_id,deleted_at) VALUES($1,$2,$3,$4) ON CONFLICT(deployment,kind,opaque_id) DO UPDATE SET deleted_at=greatest(personal_deletions.deleted_at,excluded.deleted_at)`, deployment, d.Kind, d.ID, d.At); err != nil {
			return err
		}
		if d.Kind == "message" {
			_, err = tx.Exec(ctx, `DELETE FROM chat_messages WHERE deployment=$1 AND id=$2`, deployment, d.ID)
		} else {
			if _, err = tx.Exec(ctx, `DELETE FROM chat_messages WHERE deployment=$1 AND user_id=$2`, deployment, d.ID); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `DELETE FROM chat_reports WHERE deployment=$1 AND user_id=$2`, deployment, d.ID); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `DELETE FROM chat_blocks WHERE user_id=$1 OR blocked_id=$1`, d.ID); err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `DELETE FROM personal_users WHERE deployment=$1 AND id=$2`, deployment, d.ID)
		}
		if err != nil {
			return err
		}
	}
	for _, r := range c.Restrictions {
		if !validMedia(normalizedMedia(r.Media)) || len(r.User) != 64 || r.Until.After(c.At.Add(30*24*time.Hour)) {
			return fmt.Errorf("invalid restriction checkpoint")
		}
		if _, err = tx.Exec(ctx, `INSERT INTO chat_restrictions(deployment,user_id,until_at,media) VALUES($1,$2,$3,$4) ON CONFLICT(deployment,media,user_id) DO UPDATE SET until_at=greatest(chat_restrictions.until_at,excluded.until_at)`, deployment, r.User, r.Until, normalizedMedia(r.Media)); err != nil {
			return err
		}
	}
	for _, id := range c.Withdrawals {
		if len(id) != 64 {
			return fmt.Errorf("invalid withdrawal checkpoint")
		}
		if _, err = tx.Exec(ctx, `UPDATE chat_messages SET hidden=true WHERE deployment=$1 AND id=$2`, deployment, id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// Prune applies only the accepted personal/chat retention limits. It never
// references an archive table. Reports nearing expiry must be reviewed by the
// deployment's moderators; the returned count makes expired unresolved work explicit.
func Prune(ctx context.Context, pool *pgxpool.Pool, deployment string, now time.Time) (int, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var unresolved int
	err = tx.QueryRow(ctx, `SELECT count(*) FROM chat_reports WHERE deployment=$1 AND NOT resolved AND created_at<=$2`, deployment, now.Add(-30*24*time.Hour)).Scan(&unresolved)
	if err != nil {
		return 0, err
	}
	queries := []struct {
		sql string
		at  time.Time
	}{
		{`DELETE FROM chat_messages WHERE deployment=$1 AND created_at<=$2`, now.Add(-30 * 24 * time.Hour)},
		{`DELETE FROM chat_reports WHERE deployment=$1 AND created_at<=$2`, now.Add(-30 * 24 * time.Hour)},
		{`DELETE FROM chat_requests WHERE deployment=$1 AND created_at<=$2`, now.Add(-30 * 24 * time.Hour)},
		{`DELETE FROM chat_moderation WHERE deployment=$1 AND created_at<=$2`, now.Add(-30 * 24 * time.Hour)},
		{`DELETE FROM chat_restrictions WHERE deployment=$1 AND until_at<=$2`, now},
		{`DELETE FROM chat_rate_windows WHERE deployment=$1 AND expires_at<=$2`, now},
		{`DELETE FROM personal_sessions WHERE deployment=$1 AND expires_at<=$2`, now},
		{`DELETE FROM personal_login_flows WHERE deployment=$1 AND expires_at<=$2`, now},
		{`DELETE FROM personal_deletions WHERE deployment=$1 AND deleted_at<=$2`, now.Add(-90 * 24 * time.Hour)},
	}
	for _, q := range queries {
		if _, err = tx.Exec(ctx, q.sql, deployment, q.at); err != nil {
			return 0, err
		}
	}
	return unresolved, tx.Commit(ctx)
}
