package config

import "testing"

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
	c.Limits.TotalBytes = 3 << 30
	if c.Validate() == nil {
		t.Fatal("accepted excessive batch")
	}
}
