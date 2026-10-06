package source

import (
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"testing"
)

func TestConfiguredMultiFileSource(t *testing.T) {
	raw := []byte(`{"metadata":{"identifier":"series","mediatype":"movies","collection":["television"],"title":"Series","date":"1993-09-10"},"files":[{"name":"01 Flight.mp4","size":"30","length":"12","md5":"abc","sha1":"def"},{"name":"02 Earth.mp4","size":"31","length":"13"},{"name":"metadata.xml","size":"1"}]}`)
	c := domain.Collection{ID: "bill-nye", Policy: domain.Local, Budget: 1 << 20, Reserve: 100, SourceItem: "series"}
	first, err := ItemFile(c, "series", "01 Flight.mp4", raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ItemFile(c, "series", "02 Earth.mp4", raw)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID || first.SourceItemID != "series" || len(first.Files) != 3 || first.Date != "" || first.Files[0].SHA1 != "def" {
		t.Fatal("source lineage or full inventory", first)
	}
	if _, err = ItemFile(c, "other", "01 Flight.mp4", raw); err == nil {
		t.Fatal("unconfigured source accepted")
	}
	if _, err = ItemFile(c, "series", "private.mp4", raw); err == nil {
		t.Fatal("nonexistent asset accepted")
	}
	entry := domain.Entry{Episode: first, Policy: domain.Local}
	if entry.ValidateDecision(domain.Decision{Playback: true, Published: true, File: "02 Earth.mp4", Notes: "Manipulated file"}) == nil {
		t.Fatal("file alias switched to another episode")
	}
	c.SourceItem = "child"
	raw = []byte(`{"metadata":{"identifier":"child","mediatype":"movies","collection":["democracy_now_vid"],"title":"Democracy child"},"files":[{"name":"a.mp4","size":"1","length":"10"}]}`)
	if _, err = ItemFile(c, "child", "a.mp4", raw); err == nil {
		t.Fatal("relabeled Democracy child accepted")
	}
	c.SourceItem = domain.DemocracyNow
	if _, err = ItemFile(c, domain.DemocracyNow, "01 Flight.mp4", raw); err == nil {
		t.Fatal("Democracy item bypass accepted")
	}
}

func TestAVIImportPreservesSourceWithoutPlaybackApproval(t *testing.T) {
	raw := []byte(`{"metadata":{"identifier":"series","mediatype":"movies","title":"Series","date":"1993-09-10"},"files":[{"name":"01 Flight.avi","source":"original","size":"234848256","length":"1380.75","md5":"91b3b5e772b76c3a03b5238103ff114a"},{"name":"source.xml","size":"2"}]}`)
	c := domain.Collection{ID: "private-processing", Policy: domain.Local, Budget: 768 << 20, Reserve: 16 << 30, SourceItem: "series"}
	e, err := ItemFile(c, "series", "01 Flight.avi", raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Files) != 2 || e.Date != "" || e.SourceFile != "01 Flight.avi" {
		t.Fatal("source lineage changed", e)
	}
	entry := domain.Entry{Episode: e, Policy: domain.Local}
	if entry.ValidateDecision(domain.Decision{Acquisition: true, File: e.SourceFile, Notes: "Implicit"}) == nil {
		t.Fatal("AVI implicitly acquired")
	}
	if entry.ValidateDecision(domain.Decision{Acquisition: true, ProcessingOriginal: true, File: e.SourceFile, Notes: "Explicit private pilot"}) != nil {
		t.Fatal("approved original refused")
	}
}
