package internetarchive

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLongRetryAfterReturnsDurableBackoff(t *testing.T) {
	requests := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(429)
	}))
	defer s.Close()
	c := &Client{Base: s.URL, HTTP: s.Client()}
	_, err := c.Request(context.Background(), "/metadata/fixture", nil)
	var backoff *RetryError
	if !errors.As(err, &backoff) || time.Until(backoff.At) < 59*time.Minute || requests != 1 {
		t.Fatal("did not honor source backoff", err, requests)
	}
}
