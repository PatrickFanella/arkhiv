package app

import (
	"context"
	"fmt"
	"testing"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
)

func TestLegacyStatusSeparatesInputFromFailures(t *testing.T) {
	if status, message := legacyStatus(fmt.Errorf("%w: invalid date", catalog.ErrInvalidInput), "bad date"); status != 400 || message != "bad date" {
		t.Fatalf("rejected input must keep the compatibility wording: %d %q", status, message)
	}
	if status, message := legacyStatus(context.DeadlineExceeded, "bad date"); status != 503 || message == "bad date" {
		t.Fatalf("a database or deadline failure must not be reported as caller input: %d %q", status, message)
	}
}
