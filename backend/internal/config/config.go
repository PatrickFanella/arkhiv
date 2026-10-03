// Package config validates collection and acquisition contracts.
package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Limits struct {
	Documents   int   `json:"documents" yaml:"documents"`
	TotalBytes  int64 `json:"total_bytes" yaml:"total_bytes"`
	FileBytes   int64 `json:"file_bytes" yaml:"file_bytes"`
	Concurrency int   `json:"concurrency" yaml:"concurrency"`
}
type Selection struct {
	File     string   `json:"file" yaml:"file"`
	Key      string   `json:"key" yaml:"key"`
	Title    string   `json:"title" yaml:"title"`
	Creator  string   `json:"creator" yaml:"creator"`
	Date     string   `json:"date" yaml:"date"`
	Language string   `json:"language" yaml:"language"`
	Subjects []string `json:"subjects" yaml:"subjects"`
	Rights   string   `json:"rights" yaml:"rights"`
}
type Updates struct {
	Enabled          bool   `json:"enabled" yaml:"enabled"`
	Policy           string `json:"policy" yaml:"policy"`
	CadenceHours     int    `json:"cadence_hours" yaml:"cadence_hours"`
	MinimumFreeBytes int64  `json:"minimum_free_bytes" yaml:"minimum_free_bytes"`
}
type Collection struct {
	Updates      Updates     `json:"updates" yaml:"updates"`
	ID           string      `json:"id" yaml:"id"`
	Name         string      `json:"name" yaml:"name"`
	Introduction string      `json:"introduction" yaml:"introduction"`
	Accent       string      `json:"accent" yaml:"accent"`
	Item         string      `json:"item" yaml:"item"`
	Limits       Limits      `json:"limits" yaml:"limits"`
	Selection    []Selection `json:"selection" yaml:"selection"`
}

var safeID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,100}$`)

func ValidID(s string) bool { return safeID.MatchString(s) }
func Load(path string) (Collection, error) {
	f, e := os.Open(path)
	if e != nil {
		return Collection{}, e
	}
	defer f.Close()
	var c Collection
	d := yaml.NewDecoder(io.LimitReader(f, 2<<20))
	d.KnownFields(true)
	if e = d.Decode(&c); e != nil {
		return c, e
	}
	return c, c.Validate()
}
func (c Collection) Validate() error {
	if !ValidID(c.ID) || !ValidID(c.Item) || strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("valid collection id, item and name required")
	}
	if !regexp.MustCompile(`^#[a-fA-F0-9]{6}$`).MatchString(c.Accent) {
		return fmt.Errorf("accent must be a six-digit hex color")
	}
	l := c.Limits
	if l.Documents < 1 || l.Documents > 50 || l.FileBytes < 1 || l.FileBytes > 250<<20 || l.TotalBytes < 1 || l.TotalBytes > 2<<30 || l.Concurrency < 1 || l.Concurrency > 2 {
		return fmt.Errorf("limits exceed pilot bounds")
	}
	if c.Updates.Policy != "" && c.Updates.Policy != "review" && c.Updates.Policy != "acquire" {
		return fmt.Errorf("updates policy must be review or acquire")
	}
	if c.Updates.CadenceHours < 0 || c.Updates.CadenceHours > 8760 || c.Updates.MinimumFreeBytes < 0 {
		return fmt.Errorf("invalid update limits")
	}
	seen := map[string]bool{}
	keys := map[string]bool{}
	for _, s := range c.Selection {
		if s.File == "" || strings.HasPrefix(s.File, "/") || strings.Contains(s.File, "\\") || strings.Contains("/"+s.File+"/", "/../") || s.Key == "" || strings.TrimSpace(s.Title) == "" || seen[s.File] {
			return fmt.Errorf("selection requires unique files, keys and titles")
		}
		seen[s.File] = true
		keys[s.Key] = true
	}
	if len(keys) > l.Documents {
		return fmt.Errorf("too many documents")
	}
	return nil
}
func SaveJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0600)
}
