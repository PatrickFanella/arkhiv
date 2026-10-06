package community

import (
	"crypto/ed25519"
	"crypto/rand"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAssertionBoundaries(t *testing.T) {
	public, key, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	trust := Trust{Key: public, Issuer: "consumer", Deployment: "brand-a"}
	for _, test := range []struct {
		name, issuer, deployment, scope string
		at                              time.Time
		want                            bool
	}{{"valid", "consumer", "brand-a", "personal", now, true}, {"foreign brand", "consumer", "brand-b", "personal", now, false}, {"foreign issuer", "other", "brand-a", "personal", now, false}, {"expired", "consumer", "brand-a", "personal", now.Add(-2 * time.Minute), false}, {"future", "consumer", "brand-a", "personal", now.Add(time.Minute), false}, {"wrong scope", "consumer", "brand-a", "chat", now, false}} {
		t.Run(test.name, func(t *testing.T) {
			raw, err := Sign(key, test.issuer, test.deployment, "provider", "subject", Hash("session"), []string{"personal"}, test.at)
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("Authorization", "Bearer "+raw)
			_, err = trust.Verify(r, test.scope, now)
			if (err == nil) != test.want {
				t.Fatalf("authorization = %v", err)
			}
		})
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-User-ID", "owner")
	if _, err := trust.Verify(r, "personal", now); err == nil {
		t.Fatal("unsigned forwarded identity accepted")
	}
}
