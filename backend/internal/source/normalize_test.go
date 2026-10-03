package source

import (
	"encoding/json"
	"testing"
)

func TestNormalizePreservesAllFilesAndUnknownFields(t *testing.T) {
	raw := []byte(`{"metadata":{"identifier":"sample","mediatype":"movies","collection":["democracy_now_vid"],"title":"Example","date":"2026-10-02","description":"<p>One &amp; two</p>","unknown":{"nested":true}},"files":[{"name":"a.mp4","format":"h.264","size":"123","length":"60","mystery":42},{"name":"source.xml","format":"Metadata","custom":"preserved"}]}`)
	e, err := Normalize("sample", "democracy_now_vid", raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Files) != 2 || e.Description != "One & two" || e.Creator != "" {
		t.Fatalf("normalization %#v", e)
	}
	var f map[string]any
	if err := json.Unmarshal(e.Files[0].Raw, &f); err != nil || f["mystery"] != float64(42) {
		t.Fatal("unknown field lost")
	}
	if _, err := Normalize("wrong", "democracy_now_vid", raw); err == nil {
		t.Fatal("identity mismatch accepted")
	}
	if _, err := Normalize("sample", "other", raw); err == nil {
		t.Fatal("membership mismatch accepted")
	}
	if f, ok := Preferred(e); !ok || f.Name != "a.mp4" {
		t.Fatal("candidate selection")
	}
}

func TestPartialSourceDateKeepsPrecision(t *testing.T) {
	for _, date := range []string{"2006", "2006-08"} {
		raw := []byte(`{"metadata":{"identifier":"partial","mediatype":"movies","collection":["democracy_now_vid"],"title":"Partial date","date":"` + date + `"},"files":[]}`)
		e, err := Normalize("partial", "democracy_now_vid", raw)
		if err != nil || e.Date != date {
			t.Fatal("invented date or discarded metadata", err, e)
		}
	}
}
