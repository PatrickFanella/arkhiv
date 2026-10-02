package index

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestExtractionToolCancellationAndFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	if _, err := command(ctx, "sleep", "10"); err == nil {
		t.Fatal("cancelled tool unexpectedly succeeded")
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("tool did not stop within the cancellation bound")
	}
	if _, err := command(context.Background(), "false"); err == nil {
		t.Fatal("failed tool unexpectedly succeeded")
	}
}

func TestExtractionOutputBound(t *testing.T) {
	b := bounded{limit: 10}
	if _, err := b.Write([]byte("1234567890")); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Write([]byte(strings.Repeat("x", 100))); err == nil || b.Len() != 10 {
		t.Fatal("overflow was retained or accepted")
	}
}
