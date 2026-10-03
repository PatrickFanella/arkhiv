package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRejectsTrailingConfiguration(t *testing.T) {
	c := Collection{ID: "zines", Item: "ZineArchive", Name: "Zines", Accent: "#123456", Limits: Limits{1, 100, 100, 1}}
	raw, _ := json.Marshal(c)
	path := filepath.Join(t.TempDir(), "collection.yaml")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"\nupdates:\n  enabled: true\n", "\n---\nupdates:\n  enabled: true\n"} {
		if err := os.WriteFile(path, append(append([]byte{}, raw...), []byte(suffix)...), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatal("ignored trailing configuration")
		}
	}
}

func TestLimitsAndDistinctDocuments(t *testing.T) {
	c := Collection{ID: "zines", Item: "ZineArchive", Name: "Zines", Accent: "#123456", Limits: Limits{1, 100, 100, 1}, Selection: []Selection{{File: "screen.pdf", Key: "one", Title: "One"}, {File: "print.pdf", Key: "one", Title: "One"}}}
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	c.Selection[1].Key = "two"
	if c.Validate() == nil {
		t.Fatal("ignored distinct-document limit")
	}
	c.Selection = c.Selection[:1]
	c.Selection[0].File = "../escape.pdf"
	if c.Validate() == nil {
		t.Fatal("accepted unsafe source name")
	}
	c.Selection = nil
	c.Limits.TotalBytes = 9 << 30
	if c.Validate() == nil {
		t.Fatal("accepted excessive batch")
	}
}

func TestAutomaticPublicPolicyRequiresExactOwnerScope(t *testing.T) {
	c := Collection{ID: "zines", Item: "ZineArchive", Name: "Zines", Accent: "#123456", Limits: Limits{1000, 8 << 30, 250 << 20, 2}}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.PublicAccess = PublicAccess{Policy: "verified-automatic", Item: "ZineArchive", Review: "Owner decision", PublicURL: "https://archive.example", Contact: "info@example.org"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*PublicAccess){func(p *PublicAccess) { p.Item = "other" }, func(p *PublicAccess) { p.Review = "" }, func(p *PublicAccess) { p.Contact = "" }, func(p *PublicAccess) { p.Policy = "approve-everything" }} {
		broken := c
		change(&broken.PublicAccess)
		if broken.Validate() == nil {
			t.Fatal("accepted incomplete automatic approval scope")
		}
	}
}
