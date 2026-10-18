package sourcecatalog

import (
	"context"
	"encoding/json"
	"testing"
)

func TestListingExclusionRetentionRefreshAndRestore(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	for _, id := range []string{"keep", "exclude"} {
		if err := s.Import(ctx, id, fixture(id, id), 200, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	listed := false
	d := ListingDecision{Identifiers: []string{"exclude"}, Listed: &listed}
	if err := s.SetListing(ctx, d, "Owner excluded unsupported recording"); err != nil {
		t.Fatal(err)
	}
	if err := s.Import(ctx, "exclude", fixture("exclude", "Changed source"), 200, "", ""); err != nil {
		t.Fatal(err)
	}
	page, err := s.Search(ctx, Query{})
	if err != nil || page.Total != 1 || page.Items[0]["identifier"] != "keep" {
		t.Fatal(page, err)
	}
	page, err = s.Search(ctx, Query{Search: "Changed"})
	if err != nil || page.Total != 0 {
		t.Fatal(page, err)
	}
	facets, err := s.Facets(ctx)
	if err != nil || len(facets["venues"]) != 1 || facets["venues"][0].Value != "keep" {
		t.Fatal(facets, err)
	}
	if visible, err := s.IsListed(ctx, "exclude"); err != nil || visible {
		t.Fatal(visible, err)
	}
	if raw, err := s.Raw(ctx, "exclude"); err != nil || !json.Valid(raw) {
		t.Fatal("retained bytes lost", err)
	}
	if err := s.ReviewAudio(ctx, "exclude", "Should not approve excluded source"); err == nil {
		t.Fatal("excluded source approved")
	}
	status, err := s.Status(ctx)
	if err != nil || status.Imported != 2 || status.Excluded != 1 || status.Playback.Pending != 1 {
		t.Fatal(status, err)
	}
	// Unknown IDs fail the entire decision, preserving existing listing state.
	listed = true
	d.Identifiers = []string{"exclude", "missing"}
	if err := s.SetListing(ctx, d, "Invalid mixed restore"); err == nil {
		t.Fatal("partial restore accepted")
	}
	if visible, _ := s.IsListed(ctx, "exclude"); visible {
		t.Fatal("failed transaction restored source")
	}
	d.Identifiers = []string{"exclude"}
	if err := s.SetListing(ctx, d, "Owner restored recording"); err != nil {
		t.Fatal(err)
	}
	page, err = s.Search(ctx, Query{})
	if err != nil || page.Total != 2 {
		t.Fatal(page, err)
	}
	var changes int
	if err := s.Core.Pool.QueryRow(ctx, `SELECT count(*) FROM source_catalog_history WHERE item_id='exclude'`).Scan(&changes); err != nil || changes != 2 {
		t.Fatal(changes, err)
	}
}
