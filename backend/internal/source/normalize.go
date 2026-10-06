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
	return normalize(id, collection, raw, false)
}
func normalize(id, collection string, raw []byte, ownedItem bool) (domain.Episode, error) {
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
	member := ownedItem && id == collection
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
	switch values := m["collection"].(type) {
	case string:
		e.SourceCollections = []string{values}
	case []any:
		for _, v := range values {
			e.SourceCollections = append(e.SourceCollections, Text(v))
		}
	}
	for _, b := range payload.Files {
		var f map[string]any
		if err := json.Unmarshal(b, &f); err != nil {
			return e, err
		}
		bytes, _ := strconv.ParseInt(Text(f["size"]), 10, 64)
		dur, _ := strconv.ParseFloat(Text(f["length"]), 64)
		w, _ := strconv.Atoi(Text(f["width"]))
		h, _ := strconv.Atoi(Text(f["height"]))
		e.Files = append(e.Files, domain.File{Name: Text(f["name"]), Format: Text(f["format"]), Source: Text(f["source"]), Original: Text(f["original"]), Bytes: bytes, MD5: Text(f["md5"]), SHA1: Text(f["sha1"]), Duration: dur, Width: w, Height: h, Private: Text(f["private"]) == "true", Raw: b})
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

// ItemFile materializes one operator-selected episode from a configured multi-file movie item.
func ItemFile(c domain.Collection, item, file string, raw []byte) (domain.Episode, error) {
	if err := c.Validate(); err != nil {
		return domain.Episode{}, err
	}
	if c.Policy != domain.Local || c.SourceItem != item || item == domain.DemocracyNow {
		return domain.Episode{}, fmt.Errorf("unconfigured source item")
	}
	e, err := normalize(item, item, raw, true)
	if err != nil {
		return e, err
	}
	if e.DemocracySource() {
		return e, fmt.Errorf("Democracy source items must remain streaming only")
	}
	found := false
	for _, f := range e.Files {
		if f.Name == file && f.PreservationCandidate() {
			found = true
			break
		}
	}
	if !found {
		return e, fmt.Errorf("selected source file is not eligible")
	}
	e.ID = "asset-" + domain.Digest([]byte(item + "/" + file))[:24]
	e.CollectionID = c.ID
	e.SourceItemID = item
	e.SourceFile = file
	e.Title = e.Title + " · " + strings.TrimSuffix(strings.TrimSuffix(file, ".mp4"), ".avi")
	e.Date = "" // Series debut date is not an individual episode broadcast date.
	return e, e.Validate()
}
