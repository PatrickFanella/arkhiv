// Package storage preserves immutable originals and rebuildable derivatives.
package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

type Store struct {
	Root *os.Root
	Path string
}

var hashPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("archive path required")
	}
	if e := os.MkdirAll(path, 0700); e != nil {
		return nil, e
	}
	r, e := os.OpenRoot(path)
	return &Store{r, path}, e
}
func (s *Store) Close() error { return s.Root.Close() }
func Original(hash string) (string, error) {
	if !hashPattern.MatchString(hash) {
		return "", fmt.Errorf("invalid hash")
	}
	return "originals/" + hash[:2] + "/" + hash, nil
}
func (s *Store) Read(hash string) (*os.File, error) {
	path, e := Original(hash)
	if e != nil {
		return nil, e
	}
	return s.Root.Open(path)
}
func Digest(r io.Reader) (string, int64, error) {
	h := sha256.New()
	n, e := io.Copy(h, r)
	return hex.EncodeToString(h.Sum(nil)), n, e
}
func (s *Store) Verify(hash string, size int64) error {
	f, e := s.Read(hash)
	if e != nil {
		return e
	}
	defer f.Close()
	actual, n, e := Digest(f)
	if e != nil {
		return e
	}
	if actual != hash || n != size {
		return fmt.Errorf("integrity mismatch for %s", hash)
	}
	return nil
}
func (s *Store) Promote(part, hash string, size int64) error {
	path, e := Original(hash)
	if e != nil {
		return e
	}
	if e = s.Root.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	if _, e = s.Root.Stat(path); e == nil {
		if e = s.Verify(hash, size); e != nil {
			return e
		}
		return s.Root.Remove(part)
	} else if !os.IsNotExist(e) {
		return e
	}
	input, e := s.Root.Open(part)
	if e != nil {
		return e
	}
	actual, n, e := Digest(input)
	input.Close()
	if e != nil {
		return e
	}
	if actual != hash || n != size {
		return fmt.Errorf("promotion integrity mismatch")
	}
	if e = s.Root.Chmod(part, 0400); e != nil {
		return e
	}
	return s.Root.Rename(part, path)
}
func (s *Store) WriteJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return s.WriteBytes(path, append(b, '\n'))
}
func (s *Store) WriteBytes(path string, b []byte) error {
	if len(b) > 16<<20 {
		return fmt.Errorf("metadata exceeds bound")
	}
	if e := s.Root.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	tmp := path + ".tmp"
	f, e := s.Root.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	_, e = f.Write(b)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	return s.Root.Rename(tmp, path)
}
func (s *Store) JSON(path string, v any) error {
	f, e := s.Root.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	return json.NewDecoder(io.LimitReader(f, 4<<20)).Decode(v)
}
