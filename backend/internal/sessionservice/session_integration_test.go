package sessionservice

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/community"
	"git.subcult.tv/subculture-collective/vhness/internal/testsupport"
	"github.com/golang-jwt/jwt/v5"
)

func TestConsumerOIDCSessionIntegration(t *testing.T) {
	pool, _, _ := testsupport.Personal(t)
	public, key, _ := ed25519.GenerateKey(rand.Reader)
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	nonce, challenge := "", ""
	tokenNonce := ""
	issuer := ""
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			write(w, 200, map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/jwks", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/jwks":
			write(w, 200, map[string]any{"keys": []any{map[string]any{"kty": "RSA", "use": "sig", "alg": "RS256", "kid": "fixture", "n": base64.RawURLEncoding.EncodeToString(rsaKey.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(rsaKey.E)).Bytes())}}})
		case "/token":
			if r.ParseForm() != nil {
				w.WriteHeader(400)
				return
			}
			digest := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if base64.RawURLEncoding.EncodeToString(digest[:]) != challenge || r.Form.Get("code") != "fixture-code" {
				w.WriteHeader(400)
				return
			}
			n := nonce
			if tokenNonce != "" {
				n = tokenNonce
			}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"iss": issuer, "sub": "opaque-provider-subject", "aud": "fixture-client", "exp": now.Add(time.Hour).Unix(), "iat": now.Unix(), "auth_time": now.Unix(), "nonce": n})
			token.Header["kid"] = "fixture"
			raw, e := token.SignedString(rsaKey)
			if e != nil {
				t.Error(e)
				w.WriteHeader(500)
				return
			}
			write(w, 200, map[string]any{"access_token": "server-only-fixture", "token_type": "Bearer", "expires_in": 3600, "id_token": raw})
		default:
			w.WriteHeader(404)
		}
	}))
	defer provider.Close()
	issuer = provider.URL
	core := &community.Server{Pool: pool, Trust: community.Trust{Key: public, Issuer: "consumer-bff", Deployment: "product-a"}, Clock: func() time.Time { return now }, Allowed: func(c string) bool { return c == "collection" }, Available: func(context.Context, string, string) bool { return true }, Current: func(context.Context, string) (*community.Program, error) { return nil, nil }}
	coreMux := http.NewServeMux()
	core.Register(coreMux)
	api := httptest.NewServer(coreMux)
	defer api.Close()
	config := Config{Origin: "http://127.0.0.1:12345", Issuer: issuer, ClientID: "fixture-client", CoreURL: api.URL, Deployment: "product-a", AssertionIssuer: "consumer-bff", AllowLoopbackHTTP: true}
	service, err := New(config, pool, key, "")
	if err != nil {
		t.Fatal(err)
	}
	service.Clock = func() time.Time { return now }
	handler := service.Handler()
	request := func(method, path string, cookies []*http.Cookie, csrf string, origin bool) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, config.Origin+path, nil)
		for _, c := range cookies {
			r.AddCookie(c)
		}
		if origin {
			r.Header.Set("Origin", config.Origin)
		}
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		r.Header.Set("Authorization", "Bearer attacker-controlled-browser-token")
		r.Header.Set("X-User-ID", "attacker")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	login := func() (*httptest.ResponseRecorder, string) {
		t.Helper()
		w := request("POST", "/api/account/v1/login", nil, "", true)
		if w.Code != 200 {
			t.Fatalf("login: %d %s", w.Code, w.Body.String())
		}
		var result map[string]string
		if json.Unmarshal(w.Body.Bytes(), &result) != nil {
			t.Fatal("invalid login response")
		}
		auth, err := url.Parse(result["url"])
		if err != nil {
			t.Fatal(err)
		}
		nonce = auth.Query().Get("nonce")
		challenge = auth.Query().Get("code_challenge")
		if nonce == "" || challenge == "" || auth.Query().Get("code_challenge_method") != "S256" || auth.Query().Get("max_age") != "300" {
			t.Fatal("OIDC flow lacks nonce or PKCE")
		}
		return w, auth.Query().Get("state")
	}
	if w := request("POST", "/api/account/v1/login", nil, "", false); w.Code != 403 {
		t.Fatal("cross-origin sign-in accepted")
	}
	start, state := login()
	if w := request("GET", "/api/account/v1/callback?state="+state+"&code=fixture-code", nil, "", false); w.Code != 400 {
		t.Fatal("callback without bound cookie accepted")
	}
	signed := request("GET", "/api/account/v1/callback?state="+state+"&code=fixture-code", start.Result().Cookies(), "", false)
	if signed.Code != 303 {
		t.Fatalf("callback: %d %s", signed.Code, signed.Body.String())
	}
	var session *http.Cookie
	for _, c := range signed.Result().Cookies() {
		if c.Name == service.cookieName(sessionCookie) {
			session = c
		}
	}
	if session == nil || !session.HttpOnly || session.SameSite != http.SameSiteLaxMode || session.Path != "/" || len(session.Value) != 64 {
		t.Fatal("opaque session cookie not hardened")
	}
	if w := request("GET", "/api/account/v1/callback?state="+state+"&code=fixture-code", start.Result().Cookies(), "", false); w.Code != 400 {
		t.Fatal("callback replay accepted")
	}
	cookies := []*http.Cookie{session}
	status := request("GET", "/api/account/v1/session", cookies, "", false)
	var stateResponse struct {
		Authenticated bool   `json:"authenticated"`
		CSRF          string `json:"csrf"`
	}
	if json.Unmarshal(status.Body.Bytes(), &stateResponse) != nil || !stateResponse.Authenticated || stateResponse.CSRF == "" {
		t.Fatal("session absent")
	}
	if strings.Contains(status.Body.String(), "provider") || strings.Contains(status.Body.String(), "opaque-provider-subject") || strings.Contains(status.Body.String(), "server-only-fixture") {
		t.Fatal("provider identity or tokens leaked")
	}
	me := request("GET", "/api/personal/v1/me", cookies, "", false)
	if me.Code != 200 {
		t.Fatalf("signed proxy: %d %s", me.Code, me.Body.String())
	}
	if w := request("PUT", "/api/personal/v1/favorites/collection/episode", cookies, "wrong", true); w.Code != 403 {
		t.Fatal("invalid CSRF accepted")
	}
	if w := request("PUT", "/api/personal/v1/favorites/collection/episode", cookies, stateResponse.CSRF, false); w.Code != 403 {
		t.Fatal("missing origin accepted")
	}
	if w := request("PUT", "/api/personal/v1/favorites/collection/episode", cookies, stateResponse.CSRF, true); w.Code != 200 {
		t.Fatalf("favorite proxy: %d %s", w.Code, w.Body.String())
	}
	if w := request("POST", "/api/account/v1/logout-all", cookies, stateResponse.CSRF, true); w.Code != 200 {
		t.Fatalf("logout: %d %s", w.Code, w.Body.String())
	}
	if w := request("GET", "/api/personal/v1/me", cookies, "", false); w.Code != 401 {
		t.Fatal("revoked session remained usable")
	}
	badStart, badState := login()
	tokenNonce = "wrong"
	if w := request("GET", "/api/account/v1/callback?state="+badState+"&code=fixture-code", badStart.Result().Cookies(), "", false); w.Code != 400 {
		t.Fatal("wrong nonce accepted")
	}
	secure := Server{Config: service.Config}
	secure.Config.Origin = "https://product.example"
	w := httptest.NewRecorder()
	secure.setCookie(w, sessionCookie, community.RandomID(), 300)
	c := w.Result().Cookies()[0]
	if !c.Secure || !c.HttpOnly || c.Name != sessionCookie {
		t.Fatal("public session cookie not Secure/__Host scoped")
	}
}
func TestSessionConfigRejectsPublicHTTPAndIssuerConfusion(t *testing.T) {
	_, key, _ := ed25519.GenerateKey(rand.Reader)
	base := Config{Origin: "https://product.example", Issuer: "https://identity.example/realms/product", CoreURL: "http://core:8080", ClientID: "client", Deployment: "product", AssertionIssuer: "bff"}
	if _, err := New(base, nil, key, ""); err != nil {
		t.Fatalf("issuer realm path rejected: %v", err)
	}
	for _, mutate := range []func(*Config){func(c *Config) { c.Origin = "http://public.example"; c.AllowLoopbackHTTP = true }, func(c *Config) { c.Issuer = "https://user:secret@identity.example" }, func(c *Config) { c.Origin = "https://product.example/path" }, func(c *Config) { c.CoreURL = "https://core.example/?redirect=other" }} {
		c := base
		mutate(&c)
		if _, err := New(c, nil, key, ""); err == nil {
			t.Fatal("invalid trust origin accepted")
		}
	}
}
