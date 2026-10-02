package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPromotionIntegrityAndConfinement(t *testing.T) {
	s, e := Open(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	payload := []byte("preserved original")
	h, n, _ := Digest(bytes.NewReader(payload))
	if e = os.WriteFile(filepath.Join(s.Path, "part"), []byte("wrong"), 0600); e != nil {
		t.Fatal(e)
	}
	if e = s.Promote("part", h, n); e == nil {
		t.Fatal("promoted mismatched bytes")
	}
	os.WriteFile(filepath.Join(s.Path, "part"), payload, 0600)
	if e = s.Promote("part", h, n); e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(s.Path, "again"), payload, 0600)
	if e = s.Promote("again", h, n); e != nil {
		t.Fatal(e)
	}
	if e = s.Verify(h, n); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Read("../../etc/passwd"); e == nil {
		t.Fatal("accepted unsafe hash")
	}
	outside := t.TempDir()
	os.Symlink(outside, filepath.Join(s.Path, "escape"))
	if e = s.WriteJSON("escape/leaked.json", map[string]string{"test": "no"}); e == nil {
		t.Fatal("escaped root")
	}
	if _, e = os.Stat(filepath.Join(outside, "leaked.json")); !os.IsNotExist(e) {
		t.Fatal("wrote outside root")
	}
}
