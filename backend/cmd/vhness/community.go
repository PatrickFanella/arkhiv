package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/community"
	"git.subcult.tv/subculture-collective/vhness/internal/sessionservice"
)

func serveSession(ctx context.Context) error {
	var c sessionservice.Config
	if err := community.ReadConfig(os.Getenv("ARKHIV_SESSION_CONFIG_FILE"), &c); err != nil {
		return err
	}
	pool, err := community.OpenPersonal(ctx, c.DatabaseURLFile)
	if err != nil {
		return err
	}
	defer pool.Close()
	key, err := community.ReadPrivateKey(c.PrivateKeyFile)
	if err != nil {
		return err
	}
	secret := ""
	if c.ClientSecretFile != "" {
		raw, err := os.ReadFile(c.ClientSecretFile)
		if err != nil {
			return fmt.Errorf("OIDC client secret unavailable")
		}
		secret = strings.TrimSpace(string(raw))
	}
	service, err := sessionservice.New(c, pool, key, secret)
	if err != nil {
		return err
	}
	handler := service.Handler()
	server := &http.Server{Addr: env("SESSION_ADDR", "127.0.0.1:8081"), Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		handler.ServeHTTP(w, r.WithContext(request))
	}), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	err = server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
func communityCommand(ctx context.Context, command, file string) error {
	var c community.Config
	if err := community.ReadConfig(os.Getenv("ARKHIV_COMMUNITY_CONFIG_FILE"), &c); err != nil {
		return err
	}
	if c.Deployment == "" {
		return fmt.Errorf("deployment identity required")
	}
	raw, err := os.ReadFile(c.DatabaseURLFile)
	if err != nil {
		return fmt.Errorf("personal connection file unavailable")
	}
	d, err := catalog.Open(ctx, strings.TrimSpace(string(raw)))
	if err != nil {
		return err
	}
	defer d.Pool.Close()
	var archive bool
	if err = d.Pool.QueryRow(ctx, `SELECT to_regclass('episodes') IS NOT NULL OR to_regclass('snapshots') IS NOT NULL`).Scan(&archive); err != nil || archive {
		return fmt.Errorf("community commands require a separate personal database without archive tables")
	}
	switch command {
	case "community-migrate":
		return d.Migrate(ctx, env("ARKHIV_COMMUNITY_MIGRATIONS", "../migrations/community"))
	case "community-prune":
		count, err := community.Prune(ctx, d.Pool, c.Deployment, time.Now().UTC())
		if err != nil {
			return err
		}
		if err = community.PruneRecoveryJournal(c.RecoveryDirectory, c.Deployment, time.Now().UTC()); err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"expired_unresolved_reports": count, "chat_days": 30, "personal_backup_days": 35, "opaque_deletion_days": 90})
	case "community-checkpoint":
		now := time.Now().UTC()
		checkpoint, err := community.RecoveryCheckpoint(ctx, d.Pool, c.Deployment, now)
		if err != nil {
			return err
		}
		checkpoint, err = community.MergeRecoveryJournal(checkpoint, c.RecoveryDirectory, now)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(checkpoint)
	case "community-recover":
		checkpoint, readErr := community.ReadCheckpoint(file)
		if readErr != nil {
			return readErr
		}
		checkpoint, err = community.MergeRecoveryJournal(checkpoint, c.RecoveryDirectory, time.Now().UTC())
		if err != nil {
			return err
		}
		return community.ReapplyRecovery(ctx, d.Pool, checkpoint, c.Deployment, time.Now().UTC())
	}
	return fmt.Errorf("unknown community command")
}
