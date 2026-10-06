package community

import (
	"net/http"
	"strings"
	"time"
)

func (s *Server) moderationList(w http.ResponseWriter, r *http.Request) {
	_, u := s.identity(w, r, "moderate")
	if u == nil {
		return
	}
	if !u.Moderator {
		fail(w, 403, "Moderator permission required.")
		return
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT r.id,r.message_id,r.reason,r.resolved,r.created_at,m.collection,m.body FROM chat_reports r JOIN chat_messages m ON m.id=r.message_id WHERE r.deployment=$1 AND r.created_at>$2 ORDER BY r.created_at DESC LIMIT 100`, s.Trust.Deployment, s.now().Add(-30*24*time.Hour))
	if err != nil {
		fail(w, 503, "Moderation unavailable.")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, message, reason, collection, body string
		var resolved bool
		var at time.Time
		if rows.Scan(&id, &message, &reason, &resolved, &at, &collection, &body) != nil {
			fail(w, 503, "Moderation unavailable.")
			return
		}
		if s.Allowed(collection) {
			out = append(out, map[string]any{"id": id, "message_id": message, "reason": reason, "resolved": resolved, "created_at": at, "collection": collection, "body": body})
		}
	}
	if rows.Err() != nil {
		fail(w, 503, "Moderation unavailable.")
		return
	}
	reply(w, 200, map[string]any{"reports": out})
}
func (s *Server) moderate(w http.ResponseWriter, r *http.Request) {
	_, u := s.identity(w, r, "moderate")
	if u == nil {
		return
	}
	if !u.Moderator {
		fail(w, 403, "Moderator permission required.")
		return
	}
	var v struct {
		Action string `json:"action"`
		Target string `json:"target"`
		Reason string `json:"reason"`
		Hours  int    `json:"hours"`
	}
	if !decode(w, r, &v) {
		return
	}
	if len(v.Target) != 64 || strings.TrimSpace(v.Reason) == "" || len(v.Reason) > 4096 {
		fail(w, 400, "Target and bounded moderation reason required.")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		fail(w, 503, "Moderation unavailable.")
		return
	}
	defer tx.Rollback(r.Context())
	switch v.Action {
	case "hide":
		var collection string
		err = tx.QueryRow(r.Context(), `SELECT collection FROM chat_messages WHERE deployment=$1 AND id=$2 FOR UPDATE`, s.Trust.Deployment, v.Target).Scan(&collection)
		if err != nil || !s.Allowed(collection) {
			fail(w, 404, "Message unavailable.")
			return
		}
		_, err = tx.Exec(r.Context(), `UPDATE chat_messages SET hidden=true WHERE id=$1`, v.Target)
	case "mute", "ban":
		if v.Hours < 1 || v.Hours > 720 {
			fail(w, 400, "Restriction must be 1–720 hours.")
			return
		}
		var exists bool
		err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM personal_users WHERE deployment=$1 AND id=$2)`, s.Trust.Deployment, v.Target).Scan(&exists)
		if err != nil || !exists {
			fail(w, 404, "User unavailable.")
			return
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO chat_restrictions(deployment,user_id,until_at) VALUES($1,$2,$3) ON CONFLICT(deployment,user_id) DO UPDATE SET until_at=excluded.until_at`, s.Trust.Deployment, v.Target, s.now().Add(time.Duration(v.Hours)*time.Hour))
	case "resolve", "reopen":
		result, e := tx.Exec(r.Context(), `UPDATE chat_reports SET resolved=$3 WHERE deployment=$1 AND id=$2`, s.Trust.Deployment, v.Target, v.Action == "resolve")
		err = e
		if err == nil && result.RowsAffected() != 1 {
			fail(w, 404, "Report unavailable.")
			return
		}
	default:
		fail(w, 400, "Unknown moderation action.")
		return
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO chat_moderation(id,deployment,actor,action,target,reason,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, RandomID(), s.Trust.Deployment, u.ID, v.Action, v.Target, v.Reason, s.now())
	}
	if err == nil && v.Action == "hide" {
		err = s.journal("hide", v.Target, time.Time{})
	}
	if err == nil && (v.Action == "mute" || v.Action == "ban") {
		err = s.journal("restriction", v.Target, s.now().Add(time.Duration(v.Hours)*time.Hour))
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		fail(w, 503, "Moderation unavailable.")
		return
	}
	reply(w, 200, map[string]bool{"applied": true})
}
