// Package sessionservice is the optional consumer-owned OIDC/session boundary.
// Run it separately from the archive API and configure it per product.
package sessionservice

import (
	"context"
	"crypto/ed25519"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/community"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
)

type Config struct {
	Origin            string `json:"origin"`
	Issuer            string `json:"issuer"`
	ClientID          string `json:"client_id"`
	ClientSecretFile  string `json:"client_secret_file"`
	PrivateKeyFile    string `json:"private_key_file"`
	DatabaseURLFile   string `json:"database_url_file"`
	CoreURL           string `json:"core_url"`
	Deployment        string `json:"deployment"`
	AssertionIssuer   string `json:"assertion_issuer"`
	AllowLoopbackHTTP bool   `json:"allow_loopback_http"`
}
type Server struct {
	Config       Config
	Pool         *pgxpool.Pool
	Key          ed25519.PrivateKey
	ClientSecret string
	Clock        func() time.Time
	client       *http.Client
	mutex        sync.Mutex
	provider     *oidc.Provider
}

const sessionCookie = "__Host-arkhiv-session"
const flowCookie = "__Host-arkhiv-login"

func (s *Server) now() time.Time {
	if s.Clock != nil {
		return s.Clock().UTC()
	}
	return time.Now().UTC()
}
func validateURL(raw string, loopback bool, core bool) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" && u.Path != "/" {
		return errors.New("exact origin URL required")
	}
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme == "http" && (core || loopback && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")) {
		return nil
	}
	return errors.New("HTTPS required")
}
func New(c Config, pool *pgxpool.Pool, key ed25519.PrivateKey, secret string) (*Server, error) {
	issuer, issuerErr := url.Parse(c.Issuer)
	issuerOrigin := ""
	if issuerErr == nil && issuer.User == nil && issuer.RawQuery == "" && issuer.Fragment == "" {
		issuerOrigin = issuer.Scheme + "://" + issuer.Host
	}
	if validateURL(c.Origin, c.AllowLoopbackHTTP, false) != nil || strings.HasSuffix(c.Origin, "/") || validateURL(issuerOrigin, c.AllowLoopbackHTTP, false) != nil || validateURL(c.CoreURL, false, true) != nil || c.ClientID == "" || c.Deployment == "" || c.AssertionIssuer == "" || len(key) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid consumer session configuration")
	}
	return &Server{Config: c, Pool: pool, Key: key, ClientSecret: secret, client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (s *Server) secure() bool { return strings.HasPrefix(s.Config.Origin, "https://") }
func (s *Server) cookieName(base string) string {
	if s.secure() {
		return base
	}
	return strings.TrimPrefix(base, "__Host-")
}
func (s *Server) setCookie(w http.ResponseWriter, name, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: s.cookieName(name), Value: value, Path: "/", Secure: s.secure(), HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: maxAge})
}
func (s *Server) cookie(r *http.Request, name string) string {
	c, err := r.Cookie(s.cookieName(name))
	if err != nil || len(c.Value) != 64 {
		return ""
	}
	return c.Value
}
func (s *Server) origin(r *http.Request) bool { return r.Header.Get("Origin") == s.Config.Origin }
func (s *Server) oidc(ctx context.Context) (*oidc.Provider, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.provider != nil {
		return s.provider, nil
	}
	provider, err := oidc.NewProvider(oidc.ClientContext(ctx, s.client), s.Config.Issuer)
	if err == nil {
		s.provider = provider
	}
	return provider, err
}
func (s *Server) oauth(p *oidc.Provider) oauth2.Config {
	return oauth2.Config{ClientID: s.Config.ClientID, ClientSecret: s.ClientSecret, Endpoint: p.Endpoint(), RedirectURL: s.Config.Origin + "/api/account/v1/callback", Scopes: []string{oidc.ScopeOpenID}}
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/account/v1/session", s.status)
	mux.HandleFunc("POST /api/account/v1/login", s.login)
	mux.HandleFunc("GET /api/account/v1/callback", s.callback)
	mux.HandleFunc("POST /api/account/v1/logout", s.logout)
	mux.HandleFunc("POST /api/account/v1/logout-all", s.logout)
	mux.HandleFunc("/api/personal/v1/", s.proxy)
	mux.HandleFunc("/api/chat/v1/", s.proxy)
	return mux
}

type identity struct {
	Hash, Issuer, Subject  string
	Authenticated, Expires time.Time
}

func (s *Server) identity(r *http.Request) (*identity, error) {
	raw := s.cookie(r, sessionCookie)
	if raw == "" {
		return nil, errors.New("sign-in required")
	}
	v := &identity{Hash: community.Hash(raw)}
	err := s.Pool.QueryRow(r.Context(), `SELECT issuer,subject,authenticated_at,expires_at FROM personal_sessions WHERE hash=$1 AND deployment=$2 AND expires_at>$3`, v.Hash, s.Config.Deployment, s.now()).Scan(&v.Issuer, &v.Subject, &v.Authenticated, &v.Expires)
	if err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Server) csrf(r *http.Request) bool {
	if !s.origin(r) {
		return false
	}
	raw := s.cookie(r, sessionCookie)
	if raw == "" {
		return false
	}
	want := community.Hash("csrf:" + raw)
	return subtle.ConstantTimeCompare([]byte(want), []byte(r.Header.Get("X-CSRF-Token"))) == 1
}
func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	v, err := s.identity(r)
	if err != nil {
		if s.cookie(r, sessionCookie) != "" && !errors.Is(err, pgx.ErrNoRows) {
			write(w, 503, map[string]string{"error": "Session service unavailable."})
			return
		}
		write(w, 200, map[string]any{"authenticated": false, "deployment": s.Config.Deployment})
		return
	}
	write(w, 200, map[string]any{"authenticated": true, "deployment": s.Config.Deployment, "csrf": community.Hash("csrf:" + s.cookie(r, sessionCookie)), "expires_at": v.Expires, "reauthentication_required": s.now().Sub(v.Authenticated) > 5*time.Minute})
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !s.origin(r) {
		write(w, 403, map[string]string{"error": "Same-origin sign-in required."})
		return
	}
	p, err := s.oidc(r.Context())
	if err != nil {
		write(w, 503, map[string]string{"error": "Sign-in provider unavailable. Anonymous viewing remains available."})
		return
	}
	state, cookie, nonce := community.RandomID(), community.RandomID(), community.RandomID()
	verifier := oauth2.GenerateVerifier()
	_, err = s.Pool.Exec(r.Context(), `DELETE FROM personal_login_flows WHERE expires_at<=$1`, s.now())
	if err == nil {
		var count int
		err = s.Pool.QueryRow(r.Context(), `SELECT count(*) FROM personal_login_flows WHERE deployment=$1`, s.Config.Deployment).Scan(&count)
		if err == nil && count >= 1000 {
			write(w, 429, map[string]string{"error": "Sign-in temporarily busy."})
			return
		}
	}
	if err == nil {
		_, err = s.Pool.Exec(r.Context(), `INSERT INTO personal_login_flows(state_hash,cookie_hash,deployment,nonce,verifier,expires_at) VALUES($1,$2,$3,$4,$5,$6)`, community.Hash(state), community.Hash(cookie), s.Config.Deployment, nonce, verifier, s.now().Add(5*time.Minute))
	}
	if err != nil {
		write(w, 503, map[string]string{"error": "Sign-in unavailable."})
		return
	}
	s.setCookie(w, flowCookie, cookie, 300)
	config := s.oauth(p)
	write(w, 200, map[string]string{"url": config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oidc.Nonce(nonce), oauth2.SetAuthURLParam("max_age", "300"))})
}
func (s *Server) callback(w http.ResponseWriter, r *http.Request) {
	state, cookie, code := r.URL.Query().Get("state"), s.cookie(r, flowCookie), r.URL.Query().Get("code")
	if len(state) != 64 || cookie == "" || code == "" || len(code) > 4096 {
		write(w, 400, map[string]string{"error": "Invalid sign-in response."})
		return
	}
	var nonce, verifier string
	err := s.Pool.QueryRow(r.Context(), `DELETE FROM personal_login_flows WHERE state_hash=$1 AND cookie_hash=$2 AND deployment=$3 AND expires_at>$4 RETURNING nonce,verifier`, community.Hash(state), community.Hash(cookie), s.Config.Deployment, s.now()).Scan(&nonce, &verifier)
	s.setCookie(w, flowCookie, "", -1)
	if err != nil {
		write(w, 400, map[string]string{"error": "Sign-in response expired or already used."})
		return
	}
	p, err := s.oidc(r.Context())
	if err != nil {
		write(w, 503, map[string]string{"error": "Sign-in provider unavailable."})
		return
	}
	config := s.oauth(p)
	token, err := config.Exchange(oidc.ClientContext(r.Context(), s.client), code, oauth2.VerifierOption(verifier))
	if err != nil {
		write(w, 400, map[string]string{"error": "Sign-in exchange failed."})
		return
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		write(w, 400, map[string]string{"error": "Verified identity required."})
		return
	}
	verified, err := p.Verifier(&oidc.Config{ClientID: s.Config.ClientID}).Verify(oidc.ClientContext(r.Context(), s.client), raw)
	if err != nil || subtle.ConstantTimeCompare([]byte(verifiedNonce(verified)), []byte(nonce)) != 1 {
		write(w, 400, map[string]string{"error": "Identity validation failed."})
		return
	}
	var claims struct {
		AuthTime int64 `json:"auth_time"`
	}
	if verified.Claims(&claims) != nil || claims.AuthTime <= 0 || time.Unix(claims.AuthTime, 0).Before(s.now().Add(-5*time.Minute)) || time.Unix(claims.AuthTime, 0).After(s.now()) || verified.Subject == "" || len(verified.Subject) > 512 {
		write(w, 400, map[string]string{"error": "Recent provider authentication required."})
		return
	}
	session := community.RandomID()
	expiry := s.now().Add(8 * time.Hour)
	if verified.Expiry.Before(expiry) {
		expiry = verified.Expiry
	}
	_, err = s.Pool.Exec(r.Context(), `INSERT INTO personal_sessions(hash,deployment,issuer,subject,authenticated_at,expires_at) VALUES($1,$2,$3,$4,$5,$6)`, community.Hash(session), s.Config.Deployment, verified.Issuer, verified.Subject, time.Unix(claims.AuthTime, 0), expiry)
	if err != nil {
		write(w, 503, map[string]string{"error": "Session unavailable."})
		return
	}
	s.setCookie(w, sessionCookie, session, int(expiry.Sub(s.now()).Seconds()))
	http.Redirect(w, r, s.Config.Origin+"/", http.StatusSeeOther)
}
func verifiedNonce(t *oidc.IDToken) string {
	if t == nil {
		return ""
	}
	return t.Nonce
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if !s.csrf(r) {
		write(w, 403, map[string]string{"error": "Same-origin session action required."})
		return
	}
	v, err := s.identity(r)
	if err == nil {
		if strings.HasSuffix(r.URL.Path, "logout-all") {
			_, err = s.Pool.Exec(r.Context(), `DELETE FROM personal_sessions WHERE deployment=$1 AND issuer=$2 AND subject=$3`, s.Config.Deployment, v.Issuer, v.Subject)
		} else {
			_, err = s.Pool.Exec(r.Context(), `DELETE FROM personal_sessions WHERE hash=$1`, v.Hash)
		}
	}
	if err != nil {
		write(w, 503, map[string]string{"error": "Sign-out unavailable."})
		return
	}
	s.setCookie(w, sessionCookie, "", -1)
	write(w, 200, map[string]bool{"signed_out": true})
}
func (s *Server) assertion(v *identity) (string, error) {
	return community.Sign(s.Key, s.Config.AssertionIssuer, s.Config.Deployment, v.Issuer, v.Subject, v.Hash, []string{"personal", "chat", "moderate"}, s.now())
}
func (s *Server) String() string {
	return fmt.Sprintf("consumer session service for %s", s.Config.Deployment)
}
