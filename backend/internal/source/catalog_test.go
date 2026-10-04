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
