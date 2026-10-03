package domain

import "testing"

func TestCatalogTypedIdentity(t *testing.T) {
	r := Representation{ID: "pdf", Media: PrintMedia, SourceItemID: "source", Snapshot: Digest([]byte("raw")), File: "a.pdf", Collections: []string{"mixed"}, Extension: MediaExtension{Print: &PrintExtension{DocumentID: "zine", Variant: "original", Pages: 2}}}
	i := CatalogItem{ID: "talk", Title: "Talk", Collections: []string{"mixed"}, Representations: []Representation{r}}
	if err := i.Validate(); err != nil {
		t.Fatal(err)
	}
	r.Extension.Audio = &AudioExtension{RecordingID: "tape"}
	if r.Validate() == nil {
		t.Fatal("untyped mixed extension accepted")
	}
	r.Extension.Audio = nil
	r.Media = AudioMedia
	if r.Validate() == nil {
		t.Fatal("print payload accepted as audio")
	}
	r.Media = PrintMedia
	r.Collections = []string{"foreign"}
	i.Representations = []Representation{r}
	if i.Validate() == nil {
		t.Fatal("representation granted foreign membership")
	}
	r.Collections = []string{"mixed"}
	r.Snapshot = "../"
	if r.Validate() == nil {
		t.Fatal("invalid source digest accepted")
	}
	a := Representation{ID: "audio", Media: AudioMedia, SourceItemID: "source", Snapshot: Digest([]byte("raw")), File: "a.flac", Collections: []string{"mixed"}, Extension: MediaExtension{Audio: &AudioExtension{RecordingID: "tape", ConcertID: "concert", Tracks: []AudioTrack{{ID: "track", File: "a.flac", Set: "1"}}}}}
	if a.Validate() == nil {
		t.Fatal("uncertain set silently classified")
	}
	a.Extension.Audio.Tracks[0].Set = ""
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
}
