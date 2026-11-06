package source

import "testing"

func TestCatalogSnapshotIdentityAndCompleteInventory(t *testing.T) {
	for _, media := range []string{"texts", "audio", "etree", "movies", "unknown-future-media"} {
		raw := []byte(`{"metadata":{"identifier":"source-item","mediatype":"` + media + `","unknown":{"keep":true}},"files":[{"name":"original","private":"true","unknown":[1,2]}],"extra":42}`)
		if err := CatalogSnapshot("source-item", raw); err != nil {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{`null`, `[]`, `{"metadata":{"identifier":"other"},"files":[]}`, `{"metadata":{"identifier":["source-item"]},"files":[]}`, `{"metadata":{"identifier":"source-item"}}`, `{"metadata":{"identifier":"source-item"},"files":[null]}`} {
		if CatalogSnapshot("source-item", []byte(raw)) == nil {
			t.Fatalf("accepted invalid metadata: %s", raw)
		}
	}
}

func TestCatalogSnapshotGenericProvider(t *testing.T) {
	valid := `{"source":{"provider":"marxists.org","id":"issue","url":"https://www.marxists.org/history/usa/pubs/issue.pdf"},"metadata":{"publication":"Proletarec"},"files":[{"name":"issue.pdf","sha256":"x"}]}`
	if err := CatalogSnapshot("issue", []byte(valid)); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		`{"source":{"provider":"marxists.org","id":"issue","url":"https://www.marxists.org/"}}`,
		`{"source":{"provider":"marxists.org","id":"issue","url":"https://www.marxists.org/"},"files":[null]}`,
		`{"source":{"provider":"marxists.org","id":"issue","url":"https://www.marxists.org/"},"metadata":{"identifier":"issue"},"files":[]}`,
		`{"source":{"provider":"marxists.org","id":"other","url":"https://www.marxists.org/"},"files":[]}`,
		`{"source":{"provider":"archive.org","id":"issue","url":"https://archive.org/details/issue"},"files":[]}`,
	} {
		if CatalogSnapshot("issue", []byte(raw)) == nil {
			t.Fatalf("accepted invalid generic snapshot: %s", raw)
		}
	}
}
