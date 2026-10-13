package sourcecatalog

import (
	"testing"
)

func TestNormalizeKeepsUncertainty(t *testing.T) {
	raw := []byte(`{"metadata":{"identifier":"show","collection":["etree"],"title":"  Ween   Live at Marquee on 2003-09-23 ","creator":["Ween"],"date":"2003-09-23T00:00:00Z","description":"<p>Encore &amp; notes</p><script>bad()</script>"},"files":[{"name":"song.flac","title":"Ocean Man","unknown":"kept"}]}`)
	fields, rules, err := Normalize("etree", "show", raw)
	if err != nil {
		t.Fatal(err)
	}
	if fields["venue"] != "Marquee" || fields["date"] != "2003-09-23" || fields["description"] != "Encore & notes" || fields["songs"] != "Ocean Man" {
		t.Fatal(fields)
	}
	if rules.Derived["venue"] == "" {
		t.Fatal("inferred venue lacks provenance")
	}
}
func TestNormalizeRejectsIdentityAndScope(t *testing.T) {
	for _, raw := range []string{`{}`, `{"metadata":{"identifier":"other","collection":"etree"},"files":[]}`, `{"metadata":{"identifier":"show","collection":"other"},"files":[]}`, `{"metadata":{"identifier":"show","collection":"etree"},"files":[null]}`} {
		if _, _, err := Normalize("etree", "show", []byte(raw)); err == nil {
			t.Fatal("accepted", raw)
		}
	}
}
func TestEditsCannotChangeInventoryOrDelivery(t *testing.T) {
	for _, patch := range []map[string]any{{"files": []any{}}, {"identifier": "new"}, {"playback": true}, {"venue": true}, {"date": "2003-02-30"}, {"title": " "}} {
		if err := ValidatePatch(patch, "reviewed correction"); err == nil {
			t.Fatal("accepted", patch)
		}
	}
	if err := ValidatePatch(map[string]any{"venue": nil}, "restore source venue"); err != nil {
		t.Fatal(err)
	}
}
