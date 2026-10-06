package community

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.subcult.tv/subculture-collective/vhness/internal/testsupport"
	"github.com/jackc/pgx/v5"
)

func TestOpenPersonalRejectsOwnersAndInheritedSchemaPowers(t *testing.T) {
	_, owner, schema := testsupport.Personal(t)
	value, err := url.Parse(os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "database-url")
	if err = os.WriteFile(path, []byte(value.String()), 0600); err != nil {
		t.Fatal(err)
	}
	if pool, err := OpenPersonal(context.Background(), path); err == nil {
		pool.Close()
		t.Fatal("operator role accepted for serving")
	}
	query := value.Query()
	query.Set("options", "-c role="+schema+"_serve -c search_path="+schema)
	value.RawQuery = strings.ReplaceAll(query.Encode(), "+", "%20")
	if err = os.WriteFile(path, []byte(value.String()), 0600); err != nil {
		t.Fatal(err)
	}
	pool, err := OpenPersonal(context.Background(), path)
	if err != nil {
		connection, connectionError := pgx.Connect(context.Background(), value.String())
		if connectionError != nil {
			t.Fatal(connectionError)
		}
		defer connection.Close(context.Background())
		var role string
		var createDB bool
		var schemas string
		diagnostic := connection.QueryRow(context.Background(), `SELECT current_user,has_database_privilege(current_user,current_database(),'CREATE'),coalesce((SELECT string_agg(nspname,',') FROM pg_namespace WHERE nspname NOT LIKE 'pg_%' AND nspname!='information_schema' AND has_schema_privilege(current_user,oid,'CREATE')),'')`).Scan(&role, &createDB, &schemas)
		t.Fatalf("%v: role=%s database_create=%v schema_create=%s diagnostic=%v", err, role, createDB, schemas, diagnostic)
	}
	pool.Close()
	if _, err = owner.Exec(context.Background(), "GRANT CREATE ON SCHEMA "+pgx.Identifier{schema}.Sanitize()+" TO "+pgx.Identifier{schema + "_serve"}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	if pool, err = OpenPersonal(context.Background(), path); err == nil {
		pool.Close()
		t.Fatal("schema creator accepted for serving")
	}
}
