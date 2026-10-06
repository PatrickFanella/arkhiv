// Package community supplies optional personal data and chat, independently of
// archival storage. Browsers authenticate at the consumer-owned session service.
package community

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const Audience = "arkhiv-community-v1"

type Claims struct {
	jwt.RegisteredClaims
	Deployment  string   `json:"deployment"`
	Provider    string   `json:"provider"`
	SessionHash string   `json:"session_hash"`
	Scopes      []string `json:"scopes"`
}

func RandomID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func Hash(value string) string { b := sha256.Sum256([]byte(value)); return hex.EncodeToString(b[:]) }

func Sign(key ed25519.PrivateKey, issuer, deployment, provider, subject, sessionHash string, scopes []string, now time.Time) (string, error) {
	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{Issuer: issuer, Subject: subject, Audience: jwt.ClaimStrings{Audience}, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute))}, Deployment: deployment, Provider: provider, SessionHash: sessionHash, Scopes: scopes}
	return jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(key)
}

type Trust struct {
	Key                ed25519.PublicKey
	Issuer, Deployment string
}

func (t Trust) Verify(r *http.Request, scope string, now time.Time) (*Claims, error) {
	raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if raw == "" || raw == r.Header.Get("Authorization") || len(raw) > 8192 {
		return nil, errors.New("sign-in required")
	}
	claims := new(Claims)
	_, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) { return t.Key, nil }, jwt.WithValidMethods([]string{"EdDSA"}), jwt.WithIssuer(t.Issuer), jwt.WithAudience(Audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithTimeFunc(func() time.Time { return now }))
	if err != nil || claims.IssuedAt == nil || claims.ExpiresAt == nil || claims.ExpiresAt.Sub(claims.IssuedAt.Time) > time.Minute || claims.Deployment != t.Deployment || claims.Subject == "" || len(claims.Subject) > 512 || claims.Provider == "" || len(claims.Provider) > 2048 || len(claims.SessionHash) != 64 {
		return nil, errors.New("invalid identity")
	}
	for _, allowed := range claims.Scopes {
		if allowed == scope {
			return claims, nil
		}
	}
	return nil, errors.New("insufficient scope")
}
