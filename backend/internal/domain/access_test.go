package domain

import "testing"

func TestTransportIndependenceAndSourcePolicy(t *testing.T) {
	raw := []byte(`{"metadata":{"collection":["democracy_now_vid"]},"files":[{"name":"talk.mp4"},{"name":"private.mp4","private":true}]}`)
	rep := Representation{ID: "video", Media: VideoMedia, SourceItemID: "talk", Snapshot: Digest(raw), File: "talk.mp4", Collections: []string{"mixed"}, Extension: MediaExtension{Video: &VideoExtension{EpisodeID: "talk"}}}
	c := Collection{ID: "mixed", Policy: Stream}
	review := TransportReview{Snapshot: rep.Snapshot, Files: []string{"talk.mp4"}, Playback: true, Notes: "Reviewed source playback"}
	if err := AuthorizeTransport(c, rep, raw, review, PreservationApproval{}, PlayAction, "talk.mp4"); err != nil {
		t.Fatal(err)
	}
	for _, action := range []Action{ReadAction, AcquireAction, DownloadAction, RelayAction, ScheduleAction} {
		if AuthorizeTransport(c, rep, raw, review, PreservationApproval{}, action, "talk.mp4") == nil {
			t.Fatal("playback inherited permission", action)
		}
	}
	c.Policy = Local
	c.Budget = 100
	c.Reserve = 1
	review.Acquisition = true
	if AuthorizeTransport(c, rep, raw, review, PreservationApproval{}, AcquireAction, "talk.mp4") == nil {
		t.Fatal("curated membership bypassed Democracy provenance")
	}
	review.Acquisition = false
	local := PreservationApproval{Snapshot: rep.Snapshot, IndependentRecoverySHA256: Digest([]byte("synthetic restore")), Manifest: []LocalOriginal{{File: "talk.mp4", Bytes: 1, SHA256: Digest([]byte("fixture")), Path: "original.mp4"}}}
	if AuthorizeTransport(c, rep, raw, review, local, PlayAction, "talk.mp4") == nil {
		t.Fatal("existing local file bypassed permanent source delivery")
	}
	review.Acquisition = false
	c.Policy = Stream
	if AuthorizeTransport(c, rep, raw, review, PreservationApproval{}, PlayAction, "private.mp4") == nil {
		t.Fatal("unlisted private file exposed")
	}
	review.Files = append(review.Files, "private.mp4")
	if AuthorizeTransport(c, rep, raw, review, PreservationApproval{}, PlayAction, "private.mp4") == nil {
		t.Fatal("review bypassed source restriction")
	}
	review.Public = true
	if err := AuthorizeTransport(c, rep, raw, review, PreservationApproval{}, PlayAction, "talk.mp4"); err != nil {
		t.Fatal("explicit public review lost source playback permission", err)
	}
	review.Public = false
	if AuthorizeTransport(c, rep, append(raw, ' '), review, PreservationApproval{}, PlayAction, "talk.mp4") == nil {
		t.Fatal("mismatched raw snapshot trusted")
	}
}

func TestLocalManifestAndStaleReview(t *testing.T) {
	raw := []byte(`{"files":[{"name":"talk.pdf"}]}`)
	rep := Representation{ID: "pdf", Media: PrintMedia, SourceItemID: "talk", Snapshot: Digest(raw), File: "talk.pdf", Collections: []string{"zines"}, Extension: MediaExtension{Print: &PrintExtension{DocumentID: "doc", Variant: "original", Pages: 1}}}
	c := Collection{ID: "zines", Policy: Local, Budget: 100, Reserve: 1}
	review := TransportReview{Snapshot: rep.Snapshot, Files: []string{"talk.pdf"}, Read: true, Notes: "Reviewed original"}
	approval := PreservationApproval{Snapshot: rep.Snapshot, IndependentRecoverySHA256: Digest([]byte("restore")), Manifest: []LocalOriginal{{File: "talk.pdf", Bytes: 100, SHA256: Digest([]byte("original")), Path: "original.pdf"}}}
	if err := AuthorizeTransport(c, rep, raw, review, approval, ReadAction, "talk.pdf"); err != nil {
		t.Fatal(err)
	}
	approval.Manifest[0].Bytes = 101
	if AuthorizeTransport(c, rep, raw, review, approval, ReadAction, "talk.pdf") == nil {
		t.Fatal("manifest exceeded collection budget")
	}
	approval.Manifest[0].Bytes = 100
	approval.IndependentRecoverySHA256 = ""
	if AuthorizeTransport(c, rep, raw, review, approval, ReadAction, "talk.pdf") == nil {
		t.Fatal("unrecoverable local original enabled")
	}
	approval.IndependentRecoverySHA256 = Digest([]byte("restore"))
	review.Snapshot = Digest([]byte("old"))
	if AuthorizeTransport(c, rep, raw, review, approval, ReadAction, "talk.pdf") == nil {
		t.Fatal("stale source review enabled")
	}
}
