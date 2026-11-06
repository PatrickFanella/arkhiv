package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// InternetArchive is the provider of snapshots in the metadata API format.
const InternetArchive = "archive.org"

var providerPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$`)

// SourceIdentity names where a retained snapshot came from and the exact page
// an operator reviewed. Internet Archive snapshots keep their metadata API
// shape. Other providers use a generic envelope:
//
//	{"source":{"provider":"marxists.org","id":"<source id>","url":"https://..."},"files":[{"name":"..."}]}
//
// A generic source has no provider download contract in the engine, so it can
// only be delivered from reviewed local originals.
type SourceIdentity struct {
	Provider string `json:"provider"`
	URL      string `json:"url"`
}

// Generic reports whether delivery depends on a provider other than the
// Internet Archive.
func (s SourceIdentity) Generic() bool { return s.Provider != InternetArchive }

// ParseSourceIdentity derives the provider identity of a snapshot retained
// under the given source identifier. It never trusts a URL outside the named
// provider's host.
func ParseSourceIdentity(id string, raw []byte) (SourceIdentity, error) {
	if !ValidID(id) {
		return SourceIdentity{}, fmt.Errorf("invalid source identifier")
	}
	var envelope struct {
		Source json.RawMessage `json:"source"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return SourceIdentity{}, fmt.Errorf("invalid source snapshot")
	}
	if len(envelope.Source) == 0 || string(envelope.Source) == "null" {
		return SourceIdentity{Provider: InternetArchive, URL: "https://archive.org/details/" + url.PathEscape(id)}, nil
	}
	var source struct {
		Provider string `json:"provider"`
		ID       string `json:"id"`
		URL      string `json:"url"`
	}
	decoder := json.NewDecoder(bytes.NewReader(envelope.Source))
	if err := decoder.Decode(&source); err != nil {
		return SourceIdentity{}, fmt.Errorf("invalid source provider identity")
	}
	if source.ID != id {
		return SourceIdentity{}, fmt.Errorf("source provider identity mismatch")
	}
	if !providerPattern.MatchString(source.Provider) || len(source.Provider) > 253 || source.Provider == InternetArchive || strings.HasSuffix(source.Provider, "."+InternetArchive) {
		return SourceIdentity{}, fmt.Errorf("invalid source provider")
	}
	u, err := url.Parse(source.URL)
	if err != nil || len(source.URL) > 2048 || u.Scheme != "https" || u.User != nil || u.Port() != "" || u.Fragment != "" || u.Opaque != "" {
		return SourceIdentity{}, fmt.Errorf("invalid source URL")
	}
	host := u.Hostname()
	if host != source.Provider && !strings.HasSuffix(host, "."+source.Provider) {
		return SourceIdentity{}, fmt.Errorf("source URL outside provider")
	}
	return SourceIdentity{Provider: source.Provider, URL: u.String()}, nil
}
