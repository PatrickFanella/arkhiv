package community

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/testsupport"
	"github.com/jackc/pgx/v5"
)

func TestPersonalChatIsolationAndRecoveryIntegration(t *testing.T) {
	pool, owner, sourceSchema := testsupport.Personal(t)
	ctx := context.Background()
	public, key, _ := ed25519.GenerateKey(rand.Reader)
	clock := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	program := &Program{Revision: "schedule-1", Start: clock.Add(-time.Minute), Episode: "episode-1", OriginalDate: "1989-10-05"}
	s := &Server{Pool: pool, Trust: Trust{Key: public, Issuer: "consumer", Deployment: "brand-a"}, Clock: func() time.Time { return clock }, Allowed: func(c string) bool { return c == "collection-a" }, Available: func(_ context.Context, c, id string) bool { return c == "collection-a" && id == "episode-1" }, Current: func(context.Context, string) (*Program, error) { return program, nil }, ChatRead: true, ChatPost: true, Moderators: map[string]bool{Hash("provider\x00moderator"): true}}
	mux := http.NewServeMux()
	s.RecoveryDirectory = t.TempDir()
	if err := os.Chmod(s.RecoveryDirectory, 0700); err != nil {
		t.Fatal(err)
	}
	s.Register(mux)
	tokens := map[string]string{}
	for _, subject := range []string{"alice", "bob", "moderator"} {
		hash := Hash(subject + "-session")
		if _, err := owner.Exec(ctx, `INSERT INTO personal_sessions(hash,deployment,issuer,subject,authenticated_at,expires_at) VALUES($1,'brand-a','provider',$2,$3,$4)`, hash, subject, clock, clock.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		raw, err := Sign(key, "consumer", "brand-a", "provider", subject, hash, []string{"personal", "chat", "moderate"}, clock)
		if err != nil {
			t.Fatal(err)
		}
		tokens[subject] = raw
	}
	request := func(subject, method, path string, body any, status int) *httptest.ResponseRecorder {
		t.Helper()
		raw := ""
		if body != nil {
			b, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			raw = string(b)
		}
		r := httptest.NewRequest(method, path, strings.NewReader(raw))
		if subject != "" {
			r.Header.Set("Authorization", "Bearer "+tokens[subject])
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
		}
		return w
	}
	ids := map[string]string{}
	for _, subject := range []string{"alice", "bob", "moderator"} {
		var result struct {
			User User `json:"user"`
		}
		if err := json.Unmarshal(request(subject, "GET", "/api/personal/v1/me", nil, 200).Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		ids[subject] = result.User.ID
	}
	request("", "GET", "/api/personal/v1/me", nil, 401)
	request("alice", "PUT", "/api/personal/v1/favorites/foreign/episode-1", nil, 404)
	first := request("alice", "PUT", "/api/personal/v1/favorites/collection-a/episode-1", nil, 200).Body.String()
	second := request("alice", "PUT", "/api/personal/v1/favorites/collection-a/episode-1", nil, 200).Body.String()
	if first != second {
		t.Fatal("favorite retry changed revision")
	}
	request("alice", "DELETE", "/api/personal/v1/favorites/collection-a/episode-1", nil, 200)
	if !strings.Contains(request("alice", "GET", "/api/personal/v1/favorites", nil, 200).Body.String(), `"removed":true`) {
		t.Fatal("removal tombstone absent")
	}
	if strings.Contains(request("bob", "GET", "/api/personal/v1/favorites", nil, 200).Body.String(), "episode-1") {
		t.Fatal("foreign favorites leaked")
	}
	request("alice", "PUT", "/api/personal/v1/favorites/collection-a/episode-1", nil, 200)
	post := Submission{Body: "<script>plain text, not HTML</script>", Key: "retry-1", Program: *program, Mode: "channel"}
	stale := post
	stale.Revision = "old"
	request("alice", "POST", "/api/chat/v1/rooms/collection-a/messages", stale, 409)
	paused := post
	paused.Paused = true
	request("alice", "POST", "/api/chat/v1/rooms/collection-a/messages", paused, 409)
	onDemand := post
	onDemand.Mode = "ondemand"
	request("alice", "POST", "/api/chat/v1/rooms/collection-a/messages", onDemand, 409)
	var message Message
	if err := json.Unmarshal(request("alice", "POST", "/api/chat/v1/rooms/collection-a/messages", post, 201).Body.Bytes(), &message); err != nil {
		t.Fatal(err)
	}
	retry := request("alice", "POST", "/api/chat/v1/rooms/collection-a/messages", post, 200)
	if !strings.Contains(retry.Body.String(), message.ID) {
		t.Fatal("retry lost identity")
	}
	different := post
	different.Body = "different"
	request("alice", "POST", "/api/chat/v1/rooms/collection-a/messages", different, 409)
	request("bob", "DELETE", "/api/chat/v1/messages/"+message.ID, nil, 404)
	request("bob", "PUT", "/api/chat/v1/blocks/"+ids["alice"], nil, 200)
	if strings.Contains(request("bob", "GET", "/api/chat/v1/rooms/collection-a/messages", nil, 200).Body.String(), message.ID) {
		t.Fatal("blocked message visible")
	}
	request("bob", "DELETE", "/api/chat/v1/blocks/"+ids["alice"], nil, 200)
	request("bob", "POST", "/api/chat/v1/messages/"+message.ID+"/reports", map[string]string{"reason": "test report"}, 200)
	request("bob", "POST", "/api/chat/v1/messages/"+message.ID+"/reports", map[string]string{"reason": "retry"}, 200)
	if strings.Contains(request("", "GET", "/api/chat/v1/rooms/collection-a/messages", nil, 200).Body.String(), "test report") {
		t.Fatal("private report leaked")
	}
	request("alice", "GET", "/api/chat/v1/moderation", nil, 403)
	request("moderator", "POST", "/api/chat/v1/moderation", map[string]any{"action": "hide", "target": message.ID, "reason": "reviewed"}, 200)
	if strings.Contains(request("", "GET", "/api/chat/v1/rooms/collection-a/messages", nil, 200).Body.String(), message.ID) {
		t.Fatal("withdrawn message visible")
	}
	request("alice", "POST", "/api/chat/v1/rooms/collection-a/messages", post, 410)
	request("moderator", "POST", "/api/chat/v1/moderation", map[string]any{"action": "ban", "target": ids["bob"], "reason": "reviewed", "hours": 24}, 200)
	bobPost := post
	bobPost.Key = "bob-post"
	request("bob", "POST", "/api/chat/v1/rooms/collection-a/messages", bobPost, 403)
	// Save an older personal snapshot to an independently provisioned schema.
	restored, restoreOwner, restoreSchema := testsupport.Personal(t)
	for _, table := range []string{"personal_users", "personal_sessions", "personal_favorites", "chat_messages", "chat_reports", "chat_requests"} {
		sql := fmt.Sprintf("INSERT INTO %s.%s OVERRIDING SYSTEM VALUE SELECT * FROM %s.%s", pgx.Identifier{restoreSchema}.Sanitize(), pgx.Identifier{table}.Sanitize(), pgx.Identifier{sourceSchema}.Sanitize(), pgx.Identifier{table}.Sanitize())
		if _, err := restoreOwner.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	request("alice", "DELETE", "/api/chat/v1/messages/"+message.ID, nil, 200)
	request("alice", "POST", "/api/chat/v1/rooms/collection-a/messages", post, 410)
	request("alice", "DELETE", "/api/personal/v1/me", nil, 200)
	request("alice", "GET", "/api/personal/v1/me", nil, 401)
	checkpoint, err := RecoveryCheckpoint(ctx, pool, "brand-a", clock)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err = MergeRecoveryJournal(checkpoint, s.RecoveryDirectory, clock)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(checkpoint)
	if strings.Contains(string(encoded), "script") || strings.Contains(string(encoded), "provider") || strings.Contains(string(encoded), "episode-1") {
		t.Fatal("checkpoint contains bodies or personal identity")
	}
	if err = ReapplyRecovery(ctx, restored, checkpoint, "brand-a", clock); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{`SELECT count(*) FROM personal_sessions`, `SELECT count(*) FROM personal_users WHERE id='` + ids["alice"] + `'`, `SELECT count(*) FROM personal_favorites WHERE user_id='` + ids["alice"] + `'`, `SELECT count(*) FROM chat_messages WHERE id='` + message.ID + `'`} {
		var n int
		if err = restoreOwner.QueryRow(ctx, query).Scan(&n); err != nil || n != 0 {
			t.Fatalf("resurrected private data: %s: %d %v", query, n, err)
		}
	}
	var restricted bool
	if err = restoreOwner.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM chat_restrictions WHERE user_id=$1 AND until_at>$2)`, ids["bob"], clock).Scan(&restricted); err != nil || !restricted {
		t.Fatal("ban not restored")
	}
	// A personal serving role cannot create schema or write an unrelated archive table.
	if _, err = owner.Exec(ctx, `CREATE TABLE archive_canary(value text)`); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO archive_canary VALUES('forbidden')`); err == nil {
		t.Fatal("personal role wrote archive table")
	}
	if _, err = pool.Exec(ctx, `CREATE TABLE forbidden(value text)`); err == nil {
		t.Fatal("personal role changed schema")
	}
}

func TestChatLimitsAndRetentionIntegration(t *testing.T) {
	pool, owner, _ := testsupport.Personal(t)
	ctx := context.Background()
	public, key, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Now().UTC()
	session := Hash("session")
	_, err := owner.Exec(ctx, `INSERT INTO personal_sessions VALUES($1,'brand','provider','subject',$2,$3)`, session, now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	token, _ := Sign(key, "consumer", "brand", "provider", "subject", session, []string{"personal", "chat"}, now)
	program := &Program{Revision: "revision", Start: now, Episode: "episode", OriginalDate: "2026"}
	s := &Server{Pool: pool, Trust: Trust{Key: public, Issuer: "consumer", Deployment: "brand"}, Clock: func() time.Time { return now }, Allowed: func(c string) bool { return c == "room" }, Available: func(context.Context, string, string) bool { return true }, Current: func(context.Context, string) (*Program, error) { return program, nil }, ChatRead: true, ChatPost: true}
	mux := http.NewServeMux()
	s.Register(mux)
	for i := 0; i < 6; i++ {
		body, _ := json.Marshal(Submission{Body: "hello", Key: fmt.Sprint(i), Program: *program, Mode: "channel"})
		r := httptest.NewRequest("POST", "/api/chat/v1/rooms/room/messages", strings.NewReader(string(body)))
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		want := 201
		if i == 5 {
			want = 429
		}
		if w.Code != want {
			t.Fatalf("rate %d: %d %s", i, w.Code, w.Body.String())
		}
	}
	old := now.Add(-31 * 24 * time.Hour)
	if _, err = owner.Exec(ctx, `UPDATE chat_messages SET created_at=$1`, old); err != nil {
		t.Fatal(err)
	}
	if _, err = owner.Exec(ctx, `INSERT INTO personal_deletions VALUES('brand','message',$1,$2)`, RandomID(), now.Add(-91*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = Prune(ctx, pool, "brand", now); err != nil {
		t.Fatal(err)
	}
	var messages, deletions int
	if err = owner.QueryRow(ctx, `SELECT (SELECT count(*) FROM chat_messages),(SELECT count(*) FROM personal_deletions)`).Scan(&messages, &deletions); err != nil || messages != 0 || deletions != 0 {
		t.Fatalf("retention: %d %d %v", messages, deletions, err)
	}
}
