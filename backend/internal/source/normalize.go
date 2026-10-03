// Package source handles source metadata, never video payloads.
package source

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
)

func Text(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	case []any:
		a := []string{}
		for _, x := range t {
			a = append(a, Text(x))
		}
		return strings.Join(a, "; ")
	}
	return ""
}

var tags = regexp.MustCompile(`<[^>]*>`)

func Normalize(id, collection string, raw []byte) (domain.Episode, error) {
	var payload struct {
		Metadata map[string]any    `json:"metadata"`
		Files    []json.RawMessage `json:"files"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return domain.Episode{}, err
	}
	m := payload.Metadata
	if m == nil || Text(m["identifier"]) != id || Text(m["mediatype"]) != "movies" {
		return domain.Episode{}, fmt.Errorf("source identity or media type mismatch")
	}
	member := false
	switch c := m["collection"].(type) {
	case string:
		member = c == collection
	case []any:
		for _, x := range c {
			if Text(x) == collection {
				member = true
			}
		}
	}
	if !member {
		return domain.Episode{}, fmt.Errorf("source outside selected collection")
	}
	date := Text(m["date"])
	if len(date) > 10 {
		date = date[:10]
	}
	desc := strings.ReplaceAll(Text(m["description"]), "</p>", "\n")
	e := domain.Episode{ID: id, CollectionID: collection, Title: Text(m["title"]), Date: date, Description: strings.TrimSpace(html.UnescapeString(tags.ReplaceAllString(desc, ""))), Creator: Text(m["creator"]), Rights: Text(m["rights"]), LicenseURL: Text(m["licenseurl"]), Files: []domain.File{}}
	for _, b := range payload.Files {
		var f map[string]any
		if err := json.Unmarshal(b, &f); err != nil {
			return e, err
		}
		bytes, _ := strconv.ParseInt(Text(f["size"]), 10, 64)
		dur, _ := strconv.ParseFloat(Text(f["length"]), 64)
		w, _ := strconv.Atoi(Text(f["width"]))
		h, _ := strconv.Atoi(Text(f["height"]))
		e.Files = append(e.Files, domain.File{Name: Text(f["name"]), Format: Text(f["format"]), Source: Text(f["source"]), Original: Text(f["original"]), Bytes: bytes, MD5: Text(f["md5"]), Duration: dur, Width: w, Height: h, Private: Text(f["private"]) == "true", Raw: b})
	}
	return e, e.Validate()
}
func Preferred(e domain.Episode) (domain.File, bool) {
	var selected domain.File
	found := false
	for _, f := range e.Files {
		if f.Candidate() && (!found || f.Bytes < selected.Bytes) {
			selected = f
			found = true
		}
	}
	return selected, found
}
