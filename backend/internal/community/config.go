package community

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Deployment        string   `json:"deployment"`
	AssertionIssuer   string   `json:"assertion_issuer"`
	PublicKeyFile     string   `json:"public_key_file"`
	DatabaseURLFile   string   `json:"database_url_file"`
	ChatRead          bool     `json:"chat_read"`
	ChatPost          bool     `json:"chat_post"`
	Moderators        []string `json:"moderators"`
	ModerationContact string   `json:"moderation_contact"`
	RecoveryDirectory string   `json:"recovery_directory"`
}

func ReadConfig(path string, value any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("configuration file unavailable")
	}
	if len(raw) > 64<<10 {
		return fmt.Errorf("configuration exceeds limit")
	}
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	return d.Decode(value)
}
func ReadPublicKey(path string) (ed25519.PublicKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("public key unavailable")
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("PEM public key required")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	k, ok := key.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Ed25519 public key required")
	}
	return k, nil
}
func ReadPrivateKey(path string) (ed25519.PrivateKey, error) {
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("private key requires owner-only permissions")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("private key unavailable")
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("PEM private key required")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	k, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("Ed25519 private key required")
	}
	return k, nil
}

// OpenPersonal rejects superusers, table owners and archive writers. Provision a
// dedicated database and role, apply optional migrations with an operator role,
// then grant only the personal/chat tables to the runtime role.
func OpenPersonal(ctx context.Context, path string) (*pgxpool.Pool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("personal database connection file unavailable")
	}
	pool, err := pgxpool.New(ctx, strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("invalid personal database configuration")
	}
	var unsafe bool
	err = pool.QueryRow(ctx, `SELECT (SELECT rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls FROM pg_roles WHERE rolname=current_user) OR has_database_privilege(current_user,current_database(),'CREATE') OR EXISTS(SELECT 1 FROM pg_namespace n WHERE n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname NOT LIKE 'pg_%' AND has_schema_privilege(current_user,n.oid,'CREATE')) OR EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind IN ('r','p') AND n.nspname NOT IN ('pg_catalog','information_schema') AND (pg_has_role(current_user,c.relowner,'USAGE') OR (c.relname NOT LIKE 'personal_%' AND c.relname NOT LIKE 'chat_%' AND has_table_privilege(current_user,c.oid,'INSERT,UPDATE,DELETE,TRUNCATE'))))`).Scan(&unsafe)
	if err != nil || unsafe {
		pool.Close()
		return nil, fmt.Errorf("personal serving role must be a nonowner without archive write privileges")
	}
	return pool, nil
}

func Load(ctx context.Context, path string) (*Server, error) {
	if path == "" {
		return nil, nil
	}
	var c Config
	if err := ReadConfig(path, &c); err != nil {
		return nil, err
	}
	if !validName(c.Deployment) || c.AssertionIssuer == "" || c.ChatPost && (!c.ChatRead || len(c.Moderators) == 0 || strings.TrimSpace(c.ModerationContact) == "") {
		return nil, fmt.Errorf("invalid community configuration")
	}
	key, err := ReadPublicKey(c.PublicKeyFile)
	if err != nil {
		return nil, err
	}
	pool, err := OpenPersonal(ctx, c.DatabaseURLFile)
	if err != nil {
		return nil, err
	}
	moderators := map[string]bool{}
	for _, id := range c.Moderators {
		if len(id) != 64 {
			pool.Close()
			return nil, fmt.Errorf("moderator roster requires opaque issuer/subject hashes")
		}
		moderators[id] = true
	}
	if !filepath.IsAbs(c.RecoveryDirectory) {
		pool.Close()
		return nil, fmt.Errorf("independent recovery journal directory required")
	}
	return &Server{Pool: pool, Trust: Trust{Key: key, Issuer: c.AssertionIssuer, Deployment: c.Deployment}, ChatRead: c.ChatRead, ChatPost: c.ChatPost, Moderators: moderators, ModerationContact: c.ModerationContact, RecoveryDirectory: c.RecoveryDirectory}, nil
}
