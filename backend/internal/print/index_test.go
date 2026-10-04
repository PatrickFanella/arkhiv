package print_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
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

// Synthetic PDF generator adapted from Arkhiv fixtures at the pinned donor SHA.
func pdfFixture(pages ...string) []byte {
	objects := []string{"", "<< /Type /Catalog /Pages 2 0 R >>", ""}
	kids := []string{}
	for i, text := range pages {
		p := 3 + i*2
		kids = append(kids, fmt.Sprintf("%d 0 R", p))
		text = strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)").Replace(text)
		stream := fmt.Sprintf("BT /F1 18 Tf 50 750 Td (%s) Tj ET", text)
		objects = append(objects, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> >> >> /Contents %d 0 R >>", p+1), fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	}
	objects[2] = fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", len(pages), strings.Join(kids, " "))
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i := 1; i < len(objects); i++ {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i, objects[i])
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects), xref)
	return b.Bytes()
}

func TestPrintIndexAndPrivateAccessIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("disposable database absent; PDF integration unverified")
	}
	for _, tool := range []string{"prlimit", "pdfinfo", "pdftotext", "pdftoppm"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Fatal("required print tool missing", tool)
		}
	}
	ctx := context.Background()
	admin, err := catalog.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Pool.Close()
	schema := fmt.Sprintf("print_test_%x", time.Now().UnixNano())
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
	c := domain.Collection{ID: "fixture-print", Policy: domain.Local, Budget: 1 << 20, Reserve: 1}
	if err = db.Collection(ctx, c); err != nil {
		t.Fatal(err)
	}
	media, derivatives := t.TempDir(), t.TempDir()
	native := pdfFixture("Synthetic first page", "Knownword belongs on page two")
	scanned := pdfFixture("")
	ocr := []byte("Companionword has document coverage without invented page mapping.")
	for name, data := range map[string][]byte{"native.pdf": native, "scanned.pdf": scanned, "companion.txt": ocr} {
		if err = os.WriteFile(filepath.Join(media, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	raw := []byte(`{"metadata":{"identifier":"print-source"},"files":[{"name":"native.pdf"},{"name":"scanned.pdf"},{"name":"companion.txt"}]}`)
	snapshot := domain.Digest(raw)
	if err = db.Snapshot(ctx, "print-source", raw, 200, "", "", nil); err != nil {
		t.Fatal(err)
	}
	item := domain.CatalogItem{ID: "print-fixture", Title: "Synthetic print editions", Collections: []string{c.ID}, Representations: []domain.Representation{
		{ID: "native", Media: domain.PrintMedia, SourceItemID: "print-source", Snapshot: snapshot, File: "native.pdf", Collections: []string{c.ID}, Extension: domain.MediaExtension{Print: &domain.PrintExtension{DocumentID: "doc", Variant: "native"}}},
		{ID: "ocr", Media: domain.PrintMedia, SourceItemID: "print-source", Snapshot: snapshot, File: "scanned.pdf", Collections: []string{c.ID}, Extension: domain.MediaExtension{Print: &domain.PrintExtension{DocumentID: "doc", Variant: "scanned", Companions: []domain.PrintCompanion{{File: "companion.txt", Role: "ocr"}}}}},
	}}
	if err = db.InsertCatalog(ctx, item, "Synthetic print mapping"); err != nil {
		t.Fatal(err)
	}
	nativeReview := domain.TransportReview{Snapshot: snapshot, Files: []string{"native.pdf"}, Read: true, Notes: "Synthetic local reading"}
	nativeApproval := domain.PreservationApproval{Snapshot: snapshot, IndependentRecoverySHA256: domain.Digest([]byte("synthetic recovery receipt")), Manifest: []domain.LocalOriginal{{File: "native.pdf", Bytes: int64(len(native)), SHA256: domain.Digest(native), Path: "native.pdf"}}}
	if err = db.ReviewTransport(ctx, item.ID, "native", c.ID, nativeReview, nativeApproval); err != nil {
		t.Fatal(err)
	}
	if err = printmodule.Index(ctx, db, media, derivatives, item.ID, "native", c.ID); err != nil {
		t.Fatal(err)
	}
	index, err := db.PrintIndex(ctx, item.ID, "native", "native.pdf")
	if err != nil || index.Pages != 2 || index.TextStatus != "page-text" || index.CoverBytes <= 0 {
		t.Fatal("native PDF extraction", err, index)
	}
	server := &app.Server{DB: db, Media: media, Derivatives: derivatives, Collection: c.ID}
	handler := server.Handler()
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}
	base := "/api/catalog/v1/items/print-fixture/representations/native"
	var hits []domain.PrintHit
	w := get(base + "/search?q=knownword")
	if err = json.Unmarshal(w.Body.Bytes(), &hits); err != nil || w.Code != 200 || len(hits) != 1 || hits[0].Page != 2 {
		t.Fatal("native page search", err, w.Code, hits)
	}
	if w = get(base + "/cover"); w.Code != 200 || !bytes.HasPrefix(w.Body.Bytes(), []byte("\x89PNG")) {
		t.Fatal("generated cover", w.Code)
	}
	if w = get(base + "/search?q=knownword&collection=foreign"); w.Code != 404 {
		t.Fatal("foreign print search exposed", w.Code)
	}
	ocrReview := domain.TransportReview{Snapshot: snapshot, Files: []string{"scanned.pdf", "companion.txt"}, Read: true, Notes: "Synthetic OCR companion reading"}
	ocrApproval := domain.PreservationApproval{Snapshot: snapshot, IndependentRecoverySHA256: domain.Digest([]byte("synthetic recovery receipt")), Manifest: []domain.LocalOriginal{{File: "scanned.pdf", Bytes: int64(len(scanned)), SHA256: domain.Digest(scanned), Path: "scanned.pdf"}, {File: "companion.txt", Bytes: int64(len(ocr)), SHA256: domain.Digest(ocr), Path: "companion.txt"}}}
	if err = db.ReviewTransport(ctx, item.ID, "ocr", c.ID, ocrReview, ocrApproval); err != nil {
		t.Fatal(err)
	}
	if err = printmodule.Index(ctx, db, media, derivatives, item.ID, "ocr", c.ID); err != nil {
		t.Fatal(err)
	}
	w = get("/api/catalog/v1/items/print-fixture/representations/ocr/search?q=companionword")
	if err = json.Unmarshal(w.Body.Bytes(), &hits); err != nil || len(hits) != 1 || hits[0].Page != 0 {
		t.Fatal("OCR companion invented page mapping", err, w.Code, hits)
	}
	ocrReview.Files = []string{"scanned.pdf"}
	ocrReview.Notes = "Withdraw companion access while retaining PDF reading"
	if err = db.ReviewTransport(ctx, item.ID, "ocr", c.ID, ocrReview, ocrApproval); err != nil {
		t.Fatal(err)
	}
	if w = get("/api/catalog/v1/items/print-fixture/representations/ocr/search?q=companionword"); w.Code != 404 {
		t.Fatal("cached withdrawn OCR exposed", w.Code)
	}
	ocrReview.Files = []string{"scanned.pdf", "companion.txt"}
	ocrReview.Notes = "Restore companion access"
	if err = db.ReviewTransport(ctx, item.ID, "ocr", c.ID, ocrReview, ocrApproval); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(media, "companion.txt"), []byte("corrupt OCR"), 0600); err != nil {
		t.Fatal(err)
	}
	if w = get("/api/catalog/v1/items/print-fixture/representations/ocr/search?q=companionword"); w.Code != 404 {
		t.Fatal("cached corrupt OCR exposed", w.Code)
	}
	nativeReview.Read = false
	nativeReview.Notes = "Withdraw synthetic reading"
	if err = db.ReviewTransport(ctx, item.ID, "native", c.ID, nativeReview, nativeApproval); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{base + "/cover", base + "/search?q=knownword", base + "/print"} {
		if w = get(path); w.Code != 404 {
			t.Fatal("withdrawn derivative exposed", path, w.Code)
		}
	}
	nativeReview.Read = true
	nativeReview.Notes = "Restore synthetic review"
	if err = db.ReviewTransport(ctx, item.ID, "native", c.ID, nativeReview, nativeApproval); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(media, "native.pdf"), []byte("corrupt original"), 0600); err != nil {
		t.Fatal(err)
	}
	if w = get(base + "/cover"); w.Code != 404 {
		t.Fatal("cover survived corrupt original", w.Code)
	}
}
