package community

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

type Message struct {
	Seq        int64  `json:"seq"`
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	Alias      string `json:"alias"`
	Body       string `json:"body"`
	Collection string `json:"collection"`
	Program
	CreatedAt time.Time `json:"created_at"`
}

func (s *Server) room(w http.ResponseWriter, r *http.Request) {
	collection := r.PathValue("collection")
	if !s.Allowed(collection) {
		fail(w, 404, "Room unavailable.")
		return
	}
	program, err := s.Current(r.Context(), collection)
	if err != nil {
		fail(w, 503, "Channel unavailable.")
		return
	}
	reply(w, 200, map[string]any{"collection": collection, "current": program, "read": s.ChatRead, "post": s.ChatRead && s.ChatPost, "retention_days": 30, "poll_seconds": 10, "moderation_contact": s.ModerationContact})
}

const messageColumns = `seq,id,user_id,alias,body,collection,revision,program_start,episode,original_date,created_at`

func scanMessage(row interface{ Scan(...any) error }) (Message, error) {
	var m Message
	err := row.Scan(&m.Seq, &m.ID, &m.UserID, &m.Alias, &m.Body, &m.Collection, &m.Revision, &m.Start, &m.Episode, &m.OriginalDate, &m.CreatedAt)
	return m, err
}
func (s *Server) messages(w http.ResponseWriter, r *http.Request) {
	collection := r.PathValue("collection")
	if !s.ChatRead {
		fail(w, 403, "Chat is disabled.")
		return
	}
	if !s.Allowed(collection) {
		fail(w, 404, "Room unavailable.")
		return
	}
	cursor, err := parseCursor(r)
	if err != nil {
		fail(w, 400, "Invalid cursor.")
		return
	}
	before := int64(0)
	if value := r.URL.Query().Get("before"); value != "" {
		before, err = strconv.ParseInt(value, 10, 64)
		if err != nil || before <= 0 || cursor != 0 {
			fail(w, 400, "Invalid history cursor.")
			return
		}
	}
	user := ""
	if r.Header.Get("Authorization") != "" {
		_, u := s.identity(w, r, "chat")
		if u == nil {
			return
		}
		user = u.ID
	}
	order := "DESC"
	if cursor > 0 {
		order = "ASC"
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT `+messageColumns+` FROM chat_messages WHERE deployment=$1 AND collection=$2 AND seq>$3 AND ($6::bigint=0 OR seq<$6) AND NOT hidden AND created_at>$4 AND NOT EXISTS(SELECT 1 FROM chat_blocks b WHERE b.user_id=$5 AND b.blocked_id=chat_messages.user_id) ORDER BY seq `+order+` LIMIT 50`, s.Trust.Deployment, collection, cursor, s.now().Add(-30*24*time.Hour), user, before)
	if err != nil {
		fail(w, 503, "Chat unavailable.")
		return
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			fail(w, 503, "Chat unavailable.")
			return
		}
		if m.Seq > cursor {
			cursor = m.Seq
		}
		out = append(out, m)
	}
	if rows.Err() != nil {
		fail(w, 503, "Chat unavailable.")
		return
	}
	if order == "DESC" {
		slices.Reverse(out)
	}
	oldest := int64(0)
	if len(out) > 0 {
		oldest = out[0].Seq
	}
	reply(w, 200, map[string]any{"messages": out, "after": cursor, "before": oldest, "replace": r.URL.Query().Get("after") == ""})
}

type Submission struct {
	Body string `json:"body"`
	Key  string `json:"idempotency_key"`
	Program
	Mode   string `json:"mode"`
	Paused bool   `json:"paused"`
}

func (s *Server) programMatches(ctx context.Context, collection string, v Submission) bool {
	p, err := s.Current(ctx, collection)
	return err == nil && p != nil && v.Mode == "channel" && !v.Paused && v.Revision == p.Revision && v.Start.Equal(p.Start) && v.Episode == p.Episode
}
func rate(ctx context.Context, tx pgx.Tx, deployment, key string, now time.Time, window time.Duration, limit int) error {
	bucket := now.Unix() / int64(window.Seconds())
	var count int
	err := tx.QueryRow(ctx, `INSERT INTO chat_rate_windows(deployment,key,bucket,count,expires_at) VALUES($1,$2,$3,1,$4) ON CONFLICT(deployment,key,bucket) DO UPDATE SET count=chat_rate_windows.count+1 RETURNING count`, deployment, key, bucket, now.Add(2*window)).Scan(&count)
	if err != nil {
		return err
	}
	if count > limit {
		return fmt.Errorf("rate exceeded")
	}
	return nil
}
func (s *Server) post(w http.ResponseWriter, r *http.Request) {
	_, u := s.identity(w, r, "chat")
	if u == nil {
		return
	}
	collection := r.PathValue("collection")
	if !s.ChatRead || !s.ChatPost {
		fail(w, 403, "Posting is disabled.")
		return
	}
	if !s.Allowed(collection) {
		fail(w, 404, "Room unavailable.")
		return
	}
	var v Submission
	if !decode(w, r, &v) {
		return
	}
	if !utf8.ValidString(v.Body) || utf8.RuneCountInString(v.Body) > 1000 || len(v.Body) > 4096 || strings.TrimSpace(v.Body) == "" || strings.ContainsRune(v.Body, '\x00') || !validName(v.Key) || len(v.Key) > 128 {
		fail(w, 400, "Use 1–1,000 characters, at most 4 KiB, and a valid retry key.")
		return
	}
	encoded, _ := json.Marshal(struct {
		Body, Collection, Revision, Episode string
		Start                               time.Time
	}{v.Body, collection, v.Revision, v.Episode, v.Start})
	digest := Hash(string(encoded))
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		fail(w, 503, "Chat unavailable.")
		return
	}
	defer tx.Rollback(r.Context()) // Serialize retries and user deletion with the same account lock.
	var exists bool
	err = tx.QueryRow(r.Context(), `SELECT true FROM personal_users WHERE id=$1 FOR UPDATE`, u.ID).Scan(&exists)
	if err != nil {
		fail(w, 401, "Account unavailable.")
		return
	}
	var previousID, previousDigest string
	err = tx.QueryRow(r.Context(), `SELECT message_id,digest FROM chat_requests WHERE deployment=$1 AND user_id=$2 AND key=$3`, s.Trust.Deployment, u.ID, v.Key).Scan(&previousID, &previousDigest)
	if err == nil {
		if previousDigest != digest {
			fail(w, 409, "Retry key already used for another message.")
			return
		}
		m, err := scanMessage(tx.QueryRow(r.Context(), `SELECT `+messageColumns+` FROM chat_messages WHERE id=$1 AND NOT hidden AND created_at>$2`, previousID, s.now().Add(-30*24*time.Hour)))
		if err != nil {
			fail(w, 410, "The earlier message is no longer available.")
			return
		}
		reply(w, 200, m)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		fail(w, 503, "Chat unavailable.")
		return
	}
	if !s.programMatches(r.Context(), collection, v) {
		fail(w, 409, "The channel changed. Rejoin and review your message before sending.")
		return
	}
	var restricted bool
	err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM chat_restrictions WHERE deployment=$1 AND user_id=$2 AND until_at>$3)`, s.Trust.Deployment, u.ID, s.now()).Scan(&restricted)
	if err != nil {
		fail(w, 503, "Chat unavailable.")
		return
	}
	if restricted {
		fail(w, 403, "Posting is restricted.")
		return
	}
	if rate(r.Context(), tx, s.Trust.Deployment, "post:"+u.ID, s.now(), time.Minute, 5) != nil || rate(r.Context(), tx, s.Trust.Deployment, "room:"+collection, s.now(), time.Minute, 100) != nil {
		fail(w, 429, "Posting limit reached. Try again later.")
		return
	}
	p, _ := s.Current(r.Context(), collection)
	if p == nil || !s.programMatches(r.Context(), collection, v) {
		fail(w, 409, "The channel changed. Rejoin before sending.")
		return
	}
	id := RandomID()
	m, err := scanMessage(tx.QueryRow(r.Context(), `INSERT INTO chat_messages(id,deployment,collection,user_id,alias,body,revision,program_start,episode,original_date,idempotency_key,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING `+messageColumns, id, s.Trust.Deployment, collection, u.ID, u.Alias, v.Body, p.Revision, p.Start, p.Episode, p.OriginalDate, v.Key, s.now()))
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO chat_requests(deployment,user_id,key,message_id,digest,created_at) VALUES($1,$2,$3,$4,$5,$6)`, s.Trust.Deployment, u.ID, v.Key, id, digest, s.now())
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		fail(w, 503, "Chat unavailable.")
		return
	}
	reply(w, 201, m)
}
func (s *Server) userMessages(ctx context.Context, id string) ([]Message, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+messageColumns+` FROM chat_messages WHERE deployment=$1 AND user_id=$2 AND created_at>$3 ORDER BY seq`, s.Trust.Deployment, id, s.now().Add(-30*24*time.Hour))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		m, e := scanMessage(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (s *Server) deleteMessage(w http.ResponseWriter, r *http.Request) {
	_, u := s.identity(w, r, "chat")
	if u == nil {
		return
	}
	id := r.PathValue("id")
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		fail(w, 503, "Deletion unavailable.")
		return
	}
	defer tx.Rollback(r.Context())
	var owner string
	err = tx.QueryRow(r.Context(), `SELECT user_id FROM chat_messages WHERE deployment=$1 AND id=$2 FOR UPDATE`, s.Trust.Deployment, id).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		reply(w, 200, map[string]bool{"deleted": true})
		return
	}
	if err != nil {
		fail(w, 503, "Deletion unavailable.")
		return
	}
	if owner != u.ID {
		fail(w, 404, "Message unavailable.")
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO personal_deletions(deployment,kind,opaque_id,deleted_at) VALUES($1,'message',$2,$3) ON CONFLICT DO NOTHING`, s.Trust.Deployment, id, s.now())
	if err == nil {
		_, err = tx.Exec(r.Context(), `DELETE FROM chat_messages WHERE id=$1`, id)
	}
	if err == nil {
		err = s.journal("message", id, time.Time{})
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		fail(w, 503, "Deletion unavailable.")
		return
	}
	reply(w, 200, map[string]bool{"deleted": true})
}
func (s *Server) report(w http.ResponseWriter, r *http.Request) {
	_, u := s.identity(w, r, "chat")
	if u == nil {
		return
	}
	var v struct {
		Reason string `json:"reason"`
	}
	if !decode(w, r, &v) {
		return
	}
	if strings.TrimSpace(v.Reason) == "" || len(v.Reason) > 4096 {
		fail(w, 400, "A bounded report reason is required.")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		fail(w, 503, "Reports unavailable.")
		return
	}
	defer tx.Rollback(r.Context())
	var collection string
	err = tx.QueryRow(r.Context(), `SELECT collection FROM chat_messages WHERE deployment=$1 AND id=$2 AND created_at>$3`, s.Trust.Deployment, r.PathValue("id"), s.now().Add(-30*24*time.Hour)).Scan(&collection)
	if err != nil || !s.Allowed(collection) {
		fail(w, 404, "Message unavailable.")
		return
	}
	var duplicate bool
	err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM chat_reports WHERE deployment=$1 AND message_id=$2 AND user_id=$3)`, s.Trust.Deployment, r.PathValue("id"), u.ID).Scan(&duplicate)
	if err != nil {
		fail(w, 503, "Reports unavailable.")
		return
	}
	if !duplicate {
		if rate(r.Context(), tx, s.Trust.Deployment, "report:"+u.ID, s.now(), time.Hour, 3) != nil {
			fail(w, 429, "Report limit reached.")
			return
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO chat_reports(id,deployment,message_id,user_id,reason,created_at) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`, RandomID(), s.Trust.Deployment, r.PathValue("id"), u.ID, v.Reason, s.now())
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		fail(w, 503, "Reports unavailable.")
		return
	}
	reply(w, 200, map[string]bool{"reported": true})
}
func (s *Server) blocks(w http.ResponseWriter, r *http.Request) {
	_, u := s.identity(w, r, "chat")
	if u == nil {
		return
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT blocked_id FROM chat_blocks WHERE user_id=$1 ORDER BY blocked_id LIMIT 1000`, u.ID)
	if err != nil {
		fail(w, 503, "Blocks unavailable.")
		return
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil {
			fail(w, 503, "Blocks unavailable.")
			return
		}
		out = append(out, id)
	}
	if rows.Err() != nil {
		fail(w, 503, "Blocks unavailable.")
		return
	}
	reply(w, 200, map[string]any{"blocked": out})
}
func (s *Server) block(w http.ResponseWriter, r *http.Request) {
	_, u := s.identity(w, r, "chat")
	if u == nil {
		return
	}
	id := r.PathValue("id")
	if len(id) != 64 || id == u.ID {
		fail(w, 400, "Invalid blocked identity.")
		return
	}
	var err error
	if r.Method == http.MethodDelete {
		_, err = s.Pool.Exec(r.Context(), `DELETE FROM chat_blocks WHERE user_id=$1 AND blocked_id=$2`, u.ID, id)
	} else {
		_, err = s.Pool.Exec(r.Context(), `INSERT INTO chat_blocks(user_id,blocked_id) SELECT $1,$2 WHERE EXISTS(SELECT 1 FROM personal_users WHERE id=$2 AND deployment=$3) AND (SELECT count(*) FROM chat_blocks WHERE user_id=$1)<1000 ON CONFLICT DO NOTHING`, u.ID, id, s.Trust.Deployment)
	}
	if err != nil {
		fail(w, 503, "Blocks unavailable.")
		return
	}
	reply(w, 200, map[string]bool{"blocked": r.Method != http.MethodDelete})
}
