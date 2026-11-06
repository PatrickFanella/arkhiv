package print_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/app"
	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	printmodule "git.subcult.tv/subculture-collective/vhness/internal/print"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// A public scan gains a paged OCR companion through a rebind, becomes
// searchable across the archive, and is labeled only after review.
func TestOCRRebindTextSearchAndLabelsIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("disposable database absent; OCR rebind and labels unverified")
	}
	ctx := context.Background()
	admin, err := catalog.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Pool.Close()
	schema := fmt.Sprintf("rebind_test_%x", time.Now().UnixNano())
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Pool.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	defer admin.Pool.Exec(ctx, "DROP SCHEMA "+quoted+" CASCADE")
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	db := &catalog.DB{Pool: pool}
	if err = db.Migrate(ctx, "../../../migrations"); err != nil {
		t.Fatal(err)
	}
	c := domain.Collection{ID: "regional", Policy: domain.Local, Budget: 1 << 20, Reserve: 1}
	if err = db.Collection(ctx, c); err != nil {
		t.Fatal(err)
	}
	media, derivatives := t.TempDir(), t.TempDir()
	scan := pdfFixture("", "")
	ocr := []byte("Front page alpha\fThe strikeword appears on page two\f")
	for name, data := range map[string][]byte{"issue.pdf": scan, "issue.ocr.txt": ocr} {
		if err = os.WriteFile(filepath.Join(media, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	source := `"source":{"provider":"marxists.org","id":"issue-source","url":"https://www.marxists.org/history/usa/pubs/issue.pdf"}`
	first := []byte(`{` + source + `,"files":[{"name":"issue.pdf"}]}`)
	second := []byte(`{` + source + `,"files":[{"name":"issue.pdf"},{"name":"issue.ocr.txt","derived":{"from":"issue.pdf","tool":"synthetic"}}]}`)
	for _, raw := range [][]byte{first, second} {
		if err = db.Snapshot(ctx, "issue-source", raw, 200, "", "", nil); err != nil {
			t.Fatal(err)
		}
	}
	rep := domain.Representation{ID: "pdf", Media: domain.PrintMedia, SourceItemID: "issue-source", Snapshot: domain.Digest(first), File: "issue.pdf", Collections: []string{c.ID}, Extension: domain.MediaExtension{Print: &domain.PrintExtension{DocumentID: "issue", Variant: "Original PDF"}}}
	if err = db.InsertCatalog(ctx, domain.CatalogItem{ID: "issue", Title: "A regional issue", Collections: []string{c.ID}, Representations: []domain.Representation{rep}}, "Synthetic regional issue"); err != nil {
		t.Fatal(err)
	}
	recovery := domain.Digest([]byte("synthetic recovery"))
	original := domain.LocalOriginal{File: "issue.pdf", Bytes: int64(len(scan)), SHA256: domain.Digest(scan), Path: "issue.pdf"}
	review := domain.TransportReview{Snapshot: rep.Snapshot, Files: []string{"issue.pdf"}, Read: true, Public: true, Notes: "Synthetic public reading"}
	if err = db.ReviewPublicTransport(ctx, "issue", "pdf", c.ID, review, domain.PreservationApproval{Snapshot: rep.Snapshot, IndependentRecoverySHA256: recovery, Manifest: []domain.LocalOriginal{original}}); err != nil {
		t.Fatal(err)
	}
	if err = printmodule.Index(ctx, db, media, derivatives, "issue", "pdf", c.ID); err != nil {
		t.Fatal(err)
	}
	if index, err := db.PrintIndex(ctx, "issue", "pdf", "issue.pdf"); err != nil || index.TextStatus != "no-text" {
		t.Fatal("scan without text", err, index)
	}

	rebound := rep
	rebound.Snapshot = domain.Digest(second)
	rebound.Extension = domain.MediaExtension{Print: &domain.PrintExtension{DocumentID: "issue", Variant: "Original PDF", Companions: []domain.PrintCompanion{{File: "issue.ocr.txt", Role: "ocr"}}}}
	review = domain.TransportReview{Snapshot: rebound.Snapshot, Files: []string{"issue.pdf", "issue.ocr.txt"}, Read: true, Public: true, Notes: "Synthetic public reading with OCR"}
	approval := domain.PreservationApproval{Snapshot: rebound.Snapshot, IndependentRecoverySHA256: recovery, Manifest: []domain.LocalOriginal{original, {File: "issue.ocr.txt", Bytes: int64(len(ocr)), SHA256: domain.Digest(ocr), Path: "issue.ocr.txt"}}}
	wrongFile := rebound
	wrongFile.File = "issue.ocr.txt"
	if db.RebindRepresentation(ctx, "issue", wrongFile, c.ID, review, approval, true, "Must keep the primary file") == nil {
		t.Fatal("rebind changed the primary file")
	}
	if db.RebindRepresentation(ctx, "issue", rebound, c.ID, review, approval, true, "") == nil {
		t.Fatal("rebind without notes")
	}
	stale := review
	stale.Snapshot = rep.Snapshot
	if db.RebindRepresentation(ctx, "issue", rebound, c.ID, stale, approval, true, "Review must bind the new snapshot") == nil {
		t.Fatal("rebind accepted a stale review")
	}
	if err = db.RebindRepresentation(ctx, "issue", rebound, c.ID, review, approval, true, "Synthetic OCR companion"); err != nil {
		t.Fatal(err)
	}
	dropped := rebound
	dropped.Snapshot = domain.Digest(first)
	dropped.Extension = rep.Extension
	if db.RebindRepresentation(ctx, "issue", dropped, c.ID, review, approval, true, "Must not drop the companion") == nil {
		t.Fatal("rebind dropped a companion")
	}
	if err = printmodule.Index(ctx, db, media, derivatives, "issue", "pdf", c.ID); err != nil {
		t.Fatal(err)
	}
	if index, err := db.PrintIndex(ctx, "issue", "pdf", "issue.pdf"); err != nil || index.TextStatus != "page-text" || index.TextFile != "issue.ocr.txt" {
		t.Fatal("paged OCR companion", err, index)
	}

	public := &catalog.DB{Pool: pool, PublicOnly: true}
	handler := (&app.Server{DB: public, Media: media, Derivatives: derivatives, Collection: c.ID}).Handler()
	get := func(path string, v any) int {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if v != nil && w.Code == 200 {
			if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
				t.Fatal(path, err)
			}
		}
		return w.Code
	}
	var hits []domain.PrintHit
	if code := get("/api/catalog/v1/items/issue/representations/pdf/search?q=strikeword", &hits); code != 200 || len(hits) != 1 || hits[0].Page != 2 || !strings.Contains(hits[0].Snippet, "strikeword") || strings.Contains(hits[0].Snippet, "Sel=") {
		t.Fatal("OCR search page", code, hits)
	}
	var page catalog.CatalogPage
	if code := get("/api/catalog/v1/items?text=strikeword", &page); code != 200 || page.Total != 1 || len(page.Items[0].TextHits) != 1 || page.Items[0].TextHits[0].Page != 2 || page.SearchCoverage[1] != "text" || strings.Contains(page.Items[0].TextHits[0].Snippet, "Sel=") || !strings.Contains(page.Items[0].TextHits[0].Snippet, "strikeword") {
		t.Fatal("archive text search", code, page)
	}
	if code := get("/api/catalog/v1/items?text=absentword", &page); code != 200 || page.Total != 0 {
		t.Fatal("archive text search matched absent word", code, page)
	}

	if err = db.ImportLabels(ctx, []domain.Label{{ID: "strikes", Kind: "topic", Label: "Strikes", Aliases: []string{"walkout"}}}); err != nil {
		t.Fatal(err)
	}
	proposal := domain.LabelProposal{Item: "issue", Label: "strikes", Method: "lexicon-v1", Score: 3, Evidence: json.RawMessage(`{"pages":[2]}`)}
	if n, err := db.ProposeLabels(ctx, []domain.LabelProposal{proposal}, "Synthetic proposal"); err != nil || n != 1 {
		t.Fatal("propose", n, err)
	}
	var labels []catalog.CatalogLabelSummary
	if code := get("/api/catalog/v1/items?label=strikes", &page); code != 200 || page.Total != 0 {
		t.Fatal("proposed label filtered publicly", code, page)
	}
	if code := get("/api/catalog/v1/labels", &labels); code != 200 || len(labels) != 0 {
		t.Fatal("proposed label listed publicly", code, labels)
	}
	if _, err = db.ReviewLabels(ctx, []domain.LabelDecision{{Item: "issue", Label: "strikes", State: "accepted"}}, "Synthetic acceptance"); err != nil {
		t.Fatal(err)
	}
	if n, err := db.ProposeLabels(ctx, []domain.LabelProposal{proposal}, "A later run"); err != nil || n != 0 {
		t.Fatal("rerun overwrote a reviewed label", n, err)
	}
	if code := get("/api/catalog/v1/items?label=strikes", &page); code != 200 || page.Total != 1 || len(page.Items[0].Labels) != 1 || page.Items[0].Labels[0].Label != "Strikes" {
		t.Fatal("accepted label filter", code, page)
	}
	if code := get("/api/catalog/v1/labels?kind=topic", &labels); code != 200 || len(labels) != 1 || labels[0].Items != 1 {
		t.Fatal("accepted label list", code, labels)
	}
	if code := get("/api/catalog/v1/labels?collection=foreign", nil); code != 404 {
		t.Fatal("foreign label scope", code)
	}
	if _, err = db.ReviewLabels(ctx, []domain.LabelDecision{{Item: "issue", Label: "strikes", State: "rejected"}}, "Synthetic rejection"); err != nil {
		t.Fatal(err)
	}
	if code := get("/api/catalog/v1/items?label=strikes", &page); code != 200 || page.Total != 0 {
		t.Fatal("rejected label still filtered", code, page)
	}
	if _, err = db.ProposeLabels(ctx, []domain.LabelProposal{{Item: "missing", Label: "strikes", Method: "lexicon-v1"}}, "Unknown item"); err == nil {
		t.Fatal("proposal for unknown item accepted")
	}
}
