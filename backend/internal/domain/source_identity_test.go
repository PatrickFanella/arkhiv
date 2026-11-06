package domain

import "testing"

func TestSourceIdentityProviders(t *testing.T) {
	ia, err := ParseSourceIdentity("paper", []byte(`{"metadata":{"identifier":"paper"},"files":[]}`))
	if err != nil || ia.Generic() || ia.URL != "https://archive.org/details/paper" {
		t.Fatal("internet archive identity", ia, err)
	}
	generic, err := ParseSourceIdentity("mia-proletarec-1910-01", []byte(`{"source":{"provider":"marxists.org","id":"mia-proletarec-1910-01","url":"https://www.marxists.org/history/usa/pubs/proletarec/index.htm"},"files":[]}`))
	if err != nil || !generic.Generic() || generic.Provider != "marxists.org" || generic.URL != "https://www.marxists.org/history/usa/pubs/proletarec/index.htm" {
		t.Fatal("generic identity", generic, err)
	}
	for _, source := range []string{
		`{"provider":"marxists.org","id":"other","url":"https://www.marxists.org/"}`,
		`{"provider":"archive.org","id":"paper","url":"https://archive.org/details/paper"}`,
		`{"provider":"ia.archive.org","id":"paper","url":"https://ia.archive.org/details/paper"}`,
		`{"provider":"Marxists.org","id":"paper","url":"https://www.marxists.org/"}`,
		`{"provider":"localhost","id":"paper","url":"https://localhost/"}`,
		`{"provider":"marxists.org","id":"paper","url":"http://www.marxists.org/"}`,
		`{"provider":"marxists.org","id":"paper","url":"https://evil.invalid/"}`,
		`{"provider":"marxists.org","id":"paper","url":"https://marxists.org.evil.invalid/"}`,
		`{"provider":"marxists.org","id":"paper","url":"https://user@www.marxists.org/"}`,
		`{"provider":"marxists.org","id":"paper","url":"https://www.marxists.org:8443/"}`,
		`{"provider":"marxists.org","id":"paper","url":"https://www.marxists.org/#top"}`,
		`"marxists.org"`,
	} {
		if _, err := ParseSourceIdentity("paper", []byte(`{"source":`+source+`,"files":[]}`)); err == nil {
			t.Fatal("accepted invalid provider identity", source)
		}
	}
}

func TestGenericSourcesRequireLocalOriginals(t *testing.T) {
	raw := []byte(`{"source":{"provider":"marxists.org","id":"issue","url":"https://www.marxists.org/history/usa/pubs/issue.pdf"},"files":[{"name":"issue.pdf"}]}`)
	rep := Representation{ID: "pdf", Media: PrintMedia, SourceItemID: "issue", Snapshot: Digest(raw), File: "issue.pdf", Collections: []string{"regional"}, Extension: MediaExtension{Print: &PrintExtension{DocumentID: "issue", Variant: "Original PDF"}}}
	review := TransportReview{Snapshot: rep.Snapshot, Files: []string{"issue.pdf"}, Read: true, Download: true, Notes: "Reviewed regional original"}
	approval := PreservationApproval{Snapshot: rep.Snapshot, IndependentRecoverySHA256: Digest([]byte("restore")), Manifest: []LocalOriginal{{File: "issue.pdf", Bytes: 10, SHA256: Digest([]byte("original")), Path: "pubs/issue.pdf"}}}
	local := Collection{ID: "regional", Policy: Local, Budget: 100, Reserve: 1}
	for _, action := range []Action{ReadAction, DownloadAction} {
		if err := AuthorizeTransport(local, rep, raw, review, approval, action, "issue.pdf"); err != nil {
			t.Fatal(action, err)
		}
	}
	stream := Collection{ID: "regional", Policy: Stream}
	review.Download = false
	if AuthorizeTransport(stream, rep, raw, review, PreservationApproval{}, ReadAction, "issue.pdf") == nil {
		t.Fatal("generic source streamed without a provider contract")
	}
}
