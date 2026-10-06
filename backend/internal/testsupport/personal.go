// Package testsupport provisions task-owned schemas only on TEST_DATABASE_URL.
package testsupport

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Personal(t *testing.T) (*pgxpool.Pool, *pgxpool.Pool, string) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL absent; personal database integration unverified")
	}
	ctx := context.Background()
	admin, err := catalog.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("personal_test_%d", time.Now().UnixNano())
	role := schema + "_serve"
	schemaSQL, roleSQL := pgx.Identifier{schema}.Sanitize(), pgx.Identifier{role}.Sanitize()
	if _, err = admin.Pool.Exec(ctx, "CREATE SCHEMA "+schemaSQL+"; CREATE ROLE "+roleSQL+" NOLOGIN"); err != nil {
		admin.Pool.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Pool.Exec(ctx, "DROP SCHEMA "+schemaSQL+" CASCADE; DROP ROLE "+roleSQL)
		admin.Pool.Close()
	})
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	owner, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(owner.Close)
	_, current, _, _ := runtime.Caller(0)
	if err = (&catalog.DB{Pool: owner}).Migrate(ctx, filepath.Join(filepath.Dir(current), "../../../migrations/community")); err != nil {
		t.Fatal(err)
	}
	if _, err = admin.Pool.Exec(ctx, "GRANT USAGE ON SCHEMA "+schemaSQL+" TO "+roleSQL+"; GRANT SELECT,INSERT,UPDATE,DELETE ON ALL TABLES IN SCHEMA "+schemaSQL+" TO "+roleSQL+"; GRANT USAGE,SELECT ON ALL SEQUENCES IN SCHEMA "+schemaSQL+" TO "+roleSQL); err != nil {
		t.Fatal(err)
	}
	if _, err = admin.Pool.Exec(ctx, "REVOKE ALL ON "+schemaSQL+".schema_versions FROM "+roleSQL); err != nil {
		t.Fatal(err)
	}
	config.AfterConnect = func(ctx context.Context, c *pgx.Conn) error { _, err := c.Exec(ctx, "SET ROLE "+roleSQL); return err }
	serving, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(serving.Close)
	return serving, owner, schema
}
