package community

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Program struct {
	Revision     string    `json:"revision"`
	Start        time.Time `json:"start"`
	Episode      string    `json:"episode"`
	OriginalDate string    `json:"original_date"`
}
type User struct {
	ID        string `json:"id"`
	Alias     string `json:"alias"`
	Revision  int64  `json:"revision"`
	Moderator bool   `json:"moderator"`
}
type Server struct {
	Pool               *pgxpool.Pool
	Trust              Trust
	Clock              func() time.Time
	Allowed            func(string) bool
	Available          func(context.Context, string, string) bool
	Current            func(context.Context, string) (*Program, error)
	ChatRead, ChatPost bool
	ModerationContact  string
	RecoveryDirectory  string
	Moderators         map[string]bool // Hash(provider + NUL + subject), explicitly configured.
}

func (s *Server) now() time.Time {
	if s.Clock != nil {
		return s.Clock().UTC()
	}
	return time.Now().UTC()
}
func reply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, message string) {
	reply(w, status, map[string]string{"error": message})
}
func decode(w http.ResponseWriter, r *http.Request, value any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(value) != nil {
		fail(w, 400, "Invalid request.")
		return false
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		fail(w, 400, "One JSON value required.")
		return false
	}
	return true
}
func validName(value string) bool {
	return value != "" && len(value) <= 512 && !strings.ContainsAny(value, "\x00/\\")
}

func (s *Server) identity(w http.ResponseWriter, r *http.Request, scope string) (*Claims, *User) {
	claims, err := s.Trust.Verify(r, scope, s.now())
	if err != nil {
		fail(w, 401, "Sign-in required.")
		return nil, nil
	}
	var active bool
	err = s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM personal_sessions WHERE hash=$1 AND deployment=$2 AND issuer=$3 AND subject=$4 AND expires_at>$5)`, claims.SessionHash, claims.Deployment, claims.Provider, claims.Subject, s.now()).Scan(&active)
	if err != nil {
		fail(w, 503, "Personal data unavailable.")
		return nil, nil
	}
	if !active {
		fail(w, 401, "Session expired or revoked.")
		return nil, nil
	}
	id := RandomID()
	u := new(User)
	err = s.Pool.QueryRow(r.Context(), `INSERT INTO personal_users(id,deployment,issuer,subject,alias) VALUES($1,$2,$3,$4,$5) ON CONFLICT(deployment,issuer,subject) DO UPDATE SET subject=excluded.subject RETURNING id,alias,revision`, id, claims.Deployment, claims.Provider, claims.Subject, "Viewer "+id[:6]).Scan(&u.ID, &u.Alias, &u.Revision)
	if err != nil {
		fail(w, 503, "Personal data unavailable.")
		return nil, nil
	}
	u.Moderator = s.Moderators[Hash(claims.Provider+"\x00"+claims.Subject)]
	return claims, u
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/personal/v1/me", s.me)
	mux.HandleFunc("PATCH /api/personal/v1/me", s.alias)
	mux.HandleFunc("DELETE /api/personal/v1/me", s.deleteAccount)
	mux.HandleFunc("GET /api/personal/v1/favorites", s.favorites)
	mux.HandleFunc("PUT /api/personal/v1/favorites/{collection}/{item}", s.favorite)
	mux.HandleFunc("DELETE /api/personal/v1/favorites/{collection}/{item}", s.favorite)
	mux.HandleFunc("GET /api/personal/v1/export", s.export)
	mux.HandleFunc("GET /api/chat/v1/rooms/{collection}", s.room)
	mux.HandleFunc("GET /api/chat/v1/rooms/{collection}/messages", s.messages)
	mux.HandleFunc("POST /api/chat/v1/rooms/{collection}/messages", s.post)
	mux.HandleFunc("DELETE /api/chat/v1/messages/{id}", s.deleteMessage)
	mux.HandleFunc("POST /api/chat/v1/messages/{id}/reports", s.report)
	mux.HandleFunc("GET /api/chat/v1/blocks", s.blocks)
	mux.HandleFunc("PUT /api/chat/v1/blocks/{id}", s.block)
	mux.HandleFunc("DELETE /api/chat/v1/blocks/{id}", s.block)
	mux.HandleFunc("GET /api/chat/v1/moderation", s.moderationList)
	mux.HandleFunc("POST /api/chat/v1/moderation", s.moderate)
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	_, u := s.identity(w, r, "personal")
	if u != nil {
		reply(w, 200, map[string]any{"user": u, "deployment": s.Trust.Deployment, "chat_read": s.ChatRead, "chat_post": s.ChatPost, "retention": map[string]int{"chat_days": 30, "personal_backup_days": 35, "opaque_deletion_days": 90}})
	}
}
func (s *Server) alias(w http.ResponseWriter, r *http.Request) {
	_, u := s.identity(w, r, "personal")
	if u == nil {
		return
	}
	var v struct {
		Alias string `json:"alias"`
	}
	if !decode(w, r, &v) {
		return
	}
	v.Alias = strings.TrimSpace(v.Alias)
	low := strings.ToLower(v.Alias)
	if len(v.Alias) > 80 || utf8.RuneCountInString(v.Alias) < 2 || strings.ContainsAny(v.Alias, "\r\n\x00") || strings.Contains(low, "moderator") || strings.Contains(low, "admin") || strings.Contains(low, "official") || strings.Contains(low, "staff") {
		fail(w, 400, "Choose a plain public alias of 2–80 bytes without a staff title.")
		return
	}
	_, err := s.Pool.Exec(r.Context(), `UPDATE personal_users SET alias=$2 WHERE id=$1`, u.ID, v.Alias)
	if err != nil {
		fail(w, 503, "Personal data unavailable.")
		return
	}
	reply(w, 200, map[string]string{"alias": v.Alias})
}

type Favorite struct {
	Collection string `json:"collection"`
	Item       string `json:"item"`
	Removed    bool   `json:"removed"`
	Revision   int64  `json:"revision"`
	Available  bool   `json:"available"`
}

func (s *Server) listFavorites(ctx context.Context, u *User) ([]Favorite, error) {
	rows, err := s.Pool.Query(ctx, `SELECT collection,item,removed,revision FROM personal_favorites WHERE user_id=$1 ORDER BY revision,collection,item LIMIT 10001`, u.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Favorite{}
	for rows.Next() {
		var v Favorite
		if err = rows.Scan(&v.Collection, &v.Item, &v.Removed, &v.Revision); err != nil {
			return nil, err
		}
		v.Available = s.Allowed(v.Collection) && s.Available(ctx, v.Collection, v.Item)
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Server) favorites(w http.ResponseWriter, r *http.Request) {
	_, u := s.identity(w, r, "personal")
	if u == nil {
		return
	}
	values, err := s.listFavorites(r.Context(), u)
	if err != nil {
		fail(w, 503, "Favorites unavailable.")
		return
	}
	reply(w, 200, map[string]any{"favorites": values, "revision": u.Revision})
}
func (s *Server) favorite(w http.ResponseWriter, r *http.Request) {
	_, u := s.identity(w, r, "personal")
	if u == nil {
		return
	}
	collection, item := r.PathValue("collection"), r.PathValue("item")
	if !validName(collection) || !validName(item) || !s.Allowed(collection) {
		fail(w, 404, "Favorite unavailable.")
		return
	}
	removed := r.Method == http.MethodDelete
	if !removed && !s.Available(r.Context(), collection, item) {
		fail(w, 404, "Favorite unavailable.")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		fail(w, 503, "Favorites unavailable.")
		return
	}
	defer tx.Rollback(r.Context())
	var revision int64
	err = tx.QueryRow(r.Context(), `SELECT revision FROM personal_users WHERE id=$1 FOR UPDATE`, u.ID).Scan(&revision)
	var previous bool
	lookup := tx.QueryRow(r.Context(), `SELECT removed FROM personal_favorites WHERE user_id=$1 AND collection=$2 AND item=$3`, u.ID, collection, item).Scan(&previous)
	if err == nil && lookup != nil && !errors.Is(lookup, pgx.ErrNoRows) {
		err = lookup
	}
	if err == nil && (lookup == pgx.ErrNoRows || previous != removed) {
		if expected := r.Header.Get("If-Match"); expected != "" {
			value, parseErr := strconv.ParseInt(expected, 10, 64)
			if parseErr != nil || value < 0 {
				fail(w, 400, "Invalid favorites revision.")
				return
			}
			if value != revision {
				fail(w, 409, "Favorites changed on another device. Review before applying pending changes.")
				return
			}
		}
		var count int
		if !removed {
			err = tx.QueryRow(r.Context(), `SELECT count(*) FROM personal_favorites WHERE user_id=$1`, u.ID).Scan(&count)
			if err == nil && count >= 10000 {
				fail(w, 409, "Favorite limit reached.")
				return
			}
		}
		if err == nil {
			revision++
			_, err = tx.Exec(r.Context(), `UPDATE personal_users SET revision=$2 WHERE id=$1`, u.ID, revision)
		}
		if err == nil {
			_, err = tx.Exec(r.Context(), `INSERT INTO personal_favorites(user_id,collection,item,removed,revision) VALUES($1,$2,$3,$4,$5) ON CONFLICT(user_id,collection,item) DO UPDATE SET removed=excluded.removed,revision=excluded.revision,updated_at=now()`, u.ID, collection, item, removed, revision)
		}
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		fail(w, 503, "Favorites unavailable.")
		return
	}
	reply(w, 200, map[string]any{"revision": revision, "removed": removed})
}

func (s *Server) export(w http.ResponseWriter, r *http.Request) {
	_, u := s.identity(w, r, "personal")
	if u == nil {
		return
	}
	favorites, err := s.listFavorites(r.Context(), u)
	if err != nil {
		fail(w, 503, "Export unavailable.")
		return
	}
	messages, err := s.userMessages(r.Context(), u.ID)
	if err != nil {
		fail(w, 503, "Export unavailable.")
		return
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT message_id,reason,resolved,created_at FROM chat_reports WHERE deployment=$1 AND user_id=$2 ORDER BY created_at`, s.Trust.Deployment, u.ID)
	if err != nil {
		fail(w, 503, "Export unavailable.")
		return
	}
	defer rows.Close()
	reports := []map[string]any{}
	for rows.Next() {
		var id, reason string
		var resolved bool
		var at time.Time
		if rows.Scan(&id, &reason, &resolved, &at) != nil {
			fail(w, 503, "Export unavailable.")
			return
		}
		reports = append(reports, map[string]any{"message_id": id, "reason": reason, "resolved": resolved, "created_at": at})
	}
	if rows.Err() != nil {
		fail(w, 503, "Export unavailable.")
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="archive-personal-export.json"`)
	reply(w, 200, map[string]any{"version": 1, "deployment": s.Trust.Deployment, "user": u, "favorites": favorites, "messages": messages, "reports": reports, "exported_at": s.now()})
}
func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	claims, u := s.identity(w, r, "personal")
	if u == nil {
		return
	}
	var fresh bool
	err := s.Pool.QueryRow(r.Context(), `SELECT authenticated_at BETWEEN $2 AND $3 FROM personal_sessions WHERE hash=$1`, claims.SessionHash, s.now().Add(-5*time.Minute), s.now()).Scan(&fresh)
	if err != nil || !fresh {
		fail(w, 403, "Sign in again before deleting your account.")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		fail(w, 503, "Deletion unavailable.")
		return
	}
	defer tx.Rollback(r.Context())
	_, err = tx.Exec(r.Context(), `INSERT INTO personal_deletions(deployment,kind,opaque_id,deleted_at) VALUES($1,'account',$2,$3) ON CONFLICT DO NOTHING`, s.Trust.Deployment, u.ID, s.now())
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO personal_deletions(deployment,kind,opaque_id,deleted_at) SELECT deployment,'message',id,$3 FROM chat_messages WHERE deployment=$1 AND user_id=$2 ON CONFLICT DO NOTHING`, s.Trust.Deployment, u.ID, s.now())
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `DELETE FROM chat_messages WHERE deployment=$1 AND user_id=$2`, s.Trust.Deployment, u.ID)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `DELETE FROM chat_reports WHERE deployment=$1 AND user_id=$2`, s.Trust.Deployment, u.ID)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `DELETE FROM chat_blocks WHERE user_id=$1 OR blocked_id=$1`, u.ID)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `DELETE FROM personal_sessions WHERE deployment=$1 AND issuer=$2 AND subject=$3`, claims.Deployment, claims.Provider, claims.Subject)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `DELETE FROM personal_users WHERE id=$1`, u.ID)
	}
	if err == nil {
		err = s.journal("account", u.ID, time.Time{})
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
func parseCursor(r *http.Request) (int64, error) {
	v := r.URL.Query().Get("after")
	if v == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if n < 0 {
		return 0, errors.New("invalid cursor")
	}
	return n, err
}
