# Milestone 1: a preserved, readable Anarchist Zine Archive

Status: private pilot qualified; implementation and qualification merged in PRs #19/#20. The owner-approved first collection is now public at https://arkhiv.subcult.tv with unknown rights, attribution/contact and no asserted reuse license.
Delivery closeout: Gitea #21 is complete; the private runtime and original issues/milestones are reconciled.
Next phases: [managed operations, public release, and second archive](0002-managed-archive-and-public-release.md).
Evidence: [October 2 qualification report](../reports/pilot-qualification-20261002.md).
Date: 2026-10-02. Source: https://archive.org/details/ZineArchive.

## Outcome

A visitor can browse 25–50 reviewed zines, open a document in the site's reader,
search catalog metadata and available extracted text, and download a locally
preserved copy. After acquisition, those actions work with Archive.org blocked.
An operator can resume an interrupted import, verify stored bytes, correct the
catalog, and restore the archive from backup.

Arkhiv is the reusable framework; the Anarchist Zine Archive is its first
configured collection. Retain the selected Go/React/PostgreSQL stack.

## Source evidence and selection

Live metadata inspection during the October 2 discovery found `ZineArchive` is
one IA `texts` item containing 700 PDF files and 470 `_djvu.txt` files. Those are
file counts, not unique works, issues, or a claim that every file is readable.
The inventory includes screen/print variants; filename order is not curation.
Counts and source bytes may change before ingestion. Refresh the metadata first.

Primary references:

- [Source item](https://archive.org/details/ZineArchive)
- [Source metadata](https://archive.org/metadata/ZineArchive)
- [Metadata API](https://archive.org/developers/md-read.html)
- [Automated-access guidance](https://archive.org/developers/bots.html)

Start with a small inspection sample, then select 25–50 distinct documents in a
reviewable manifest. Include a screen/print pair, an OCR companion, a scan without
usable text, a duplicate, and incomplete metadata. Synthetic fixtures cover
corruption and interrupted transfer. Set document, file, byte, and concurrency
limits before the real batch. Do not treat all 700 PDFs as one book or automatically
publish every discovered file. Initial suggested cap: 50 documents, 2 GiB total,
250 MiB per file, two concurrent transfers; exceeding a cap produces a report.

## Decisions

- Store originals locally; the reader and downloads must serve local assets.
- Use a managed filesystem archive root first, backed by an independent backup.
  Keep storage access behind a small Go interface so object storage can follow.
- Keep original bytes immutable, addressed by a local SHA-256 digest. Maintain
  source item/file identity and the original filename separately.
- Save raw IA metadata snapshots and acquisition manifests before catalog edits.
- Separate source records, documents, editions/variants, files, and optional
  publication/series membership. Grouping suggestions require review.
- Keep raw metadata, normalized fields, editorial overrides, and generated text
  distinct. Re-ingestion never overwrites editorial corrections.
- Use PostgreSQL for catalog state, job checkpoints, and initial full-text search.
  Page text can have its own index. No separate search cluster is required for
  this pilot; evaluate multilingual behavior with the selected documents.
- Use a locally bundled PDF.js reader with local fonts/workers/assets. Preserve
  the original PDF and generate a reader derivative only when needed.
- Use existing text layers/OCR first. A missing-text state is valid; full OCR for
  every document is not a prerequisite. Search coverage must be visible.
- Acquisition does not imply publication: record access and reuse information,
  review status, and the publication decision independently. Unknown rights stay
  unknown. Source attribution remains available on each record.
- One deployable site and one collection configuration first. Source adapters
  accept explicit IA item IDs and collection/query discovery without conflating
  an IA collection with an Arkhiv editorial collection.
- Operator actions start as a CLI. Public routes are read-only. A curator web
  application, user accounts, and automated public publishing are later work.

## Ordered implementation units

### 1. Collection configuration and inventory

Add `collections/anarchist-zines.yaml`, `backend/cmd/arkhiv/`, and
`backend/internal/source/internetarchive/`. Define the collection identity,
branding, source item, inclusion manifest, limits, and publication defaults.
A `plan` command fetches metadata and reports candidates, formats, byte estimates,
missing fields, alternate versions, and exclusions without downloading payloads.
Use a descriptive User-Agent, timeouts, cached metadata, bounded retries, and
Retry-After handling. Filenames and descriptions are untrusted inputs.

Check: fixture-backed adapter tests cover changing metadata, absent checksums,
missing items, 429 responses, malformed records, and multiple PDFs in one item.
Review the real inventory and exact planned bytes before running the pilot batch.

### 2. Preservation store and resumable acquisition

Add `backend/internal/storage/`, `backend/internal/ingest/`, and an `acquire`
command. Stream to temporary files, enforce limits during transfer, validate
source checksums when supplied, compute SHA-256, then atomically promote verified
bytes. Keep incomplete/failed work distinct from verified files. Resume only when
the server range response and source identity support it; otherwise restart that
file safely. Do not overwrite earlier bytes if a source filename changes content.
Constrain paths and redirects; never trust source filenames as filesystem paths.

Check: local HTTP fixtures exercise interruptions, ignored Range headers, changed
ETags, checksum mismatch, path traversal names, oversize responses, and retries.
A repeated import must not duplicate stored bytes or lose acquisition history.
A real acquisition receipt lists exact source IDs, files, sizes, hashes, and errors.

### 3. Catalog persistence and editorial review

Add versioned migrations after `0001_init.sql` and `backend/internal/catalog/`.
Introduce a migration runner with a schema-version table and locking. Model
collections, source snapshots, documents, variants, assets, acquisition attempts,
text derivatives, and review/publication state. Keep uncertain dates explicit.
Add CLI commands to inspect, correct, group, and publish selected catalog records.
Publication requires a verified local readable asset and a recorded review.

Check: disposable PostgreSQL migration/integration tests prove repeatability,
relationships, rejection of incomplete publication, variant grouping, and
preservation of editorial changes across a source refresh. Back up before schema
changes; destructive rollback must never remove preserved document files.

### 4. Text, covers, and search

Add `backend/internal/index/` and resource-limited extraction jobs. Produce covers,
page counts, and text from verified files. Track derivative tool/version and input
hash. Retain page mapping when available; plain OCR must not invent page numbers.
Expose read-only catalog/detail/search endpoints, with collection scoping,
pagination, selected facets, source links, and explicit text-coverage status.

Check: use a small checked-in synthetic PDF/text fixture with known terms on known
pages. Assert metadata search, page search where supported, empty results, stable
pagination, and no leakage of unpublished documents. Deleting and rebuilding the
index must preserve originals and editorial records.

### 5. Catalog and local reader

Add catalog, document, and reader routes under `web/src/`; bundle PDF.js locally.
Serve published files through opaque asset IDs with correct Content-Type, length,
ETag, byte-range, and download disposition behavior. Display publication/variant
relationships, provenance, missing metadata, and text coverage. Provide reader
page navigation, zoom, search when available, and original-file download.
Apply the collection's name, palette, typography, and editorial introduction from
configuration; avoid a bespoke fork for this collection.

Check: browser verification covers desktop and narrow screens, keyboard navigation,
loading/empty/error states, search-to-document navigation, page links, and actual
PDF rendering. Block Archive.org requests and verify both reading and downloads.
An unpublished or invalid asset ID must never expose a filesystem path or file.

### 6. Pilot import, recovery, and operating instructions

Run the bounded batch with outbound Archive.org access in a dedicated acquisition
workflow. The normal T3 preview/test network remains isolated; do not disable its
isolation to make ingestion pass. Transfer only reviewed manifests and verified
payloads into the development/pilot store through an explicit import step.

Document import/resume, integrity audit, derivative rebuild, backup, and restore
commands in `docs/runbooks/`. Back up the catalog, raw metadata, manifests, original
files, and collection config to an independent destination. Restore to a fresh
location/database and compare hashes and record counts. Do not call a single disk
copy a completed preservation strategy.

Check: after restore, browse/search/read/download the selected corpus with IA
unreachable. Record counts of discovered, selected, verified, published, searchable,
and failed documents separately. A source outage should affect new acquisition,
not existing verified and published documents.

## Acceptance and evidence

| Requirement | Required evidence |
| --- | --- |
| 25–50 distinct reviewed documents | Selection manifest and catalog report; variants counted separately |
| Local originals preserved | Acquisition receipt, SHA-256 verification, raw metadata snapshots |
| Retry without duplication | Interrupted-transfer and repeated-import tests plus a pilot rerun |
| Searchable catalog | Metadata queries over the corpus; known-term checks for documents with text |
| On-site reading and downloads | Rendered browser check and matching downloaded file hash |
| No runtime dependence on IA | Reading/search/download checks while source requests are blocked |
| Editorial changes retained | Refresh integration test preserving overrides and publication state |
| Branded reusable configuration | Same application renders a second synthetic collection configuration |
| Recovery | Restore into fresh storage/database, compare hashes, repeat reader checks |
| Repository quality | `make verify`, database integration suite, browser evidence; hosted CI reported separately |

As units land, add the integration and browser commands to `make verify` or an
explicit documented companion gate. Planned command names are not current tools.

## Deferred

Whole-collection mirroring, generalized crawling, multi-tenant hosting, curator
accounts, annotations, semantic/vector search, automatic AI publication, exhaustive
OCR, audio/video, and public deployment. None is needed to demonstrate the first
complete document-preservation and reading path.

## Remaining operational decisions

The reviewed manifest selects 25 documents, 26 PDF variants, and 17 OCR companions
(31,039,265 planned bytes). Originals live on Almaz at
`/mnt/spektr/server/data/arkhiv/pilot`; independent backups live on NUC at
`/home/onnwee/backups/arkhiv`. The current pilot is private. Item-level rights and
a public release decision remain open. All 26 PDF variants passed extraction; 24 documents
have page text and one has document-level OCR without page mapping. See the
[operator runbook](../runbooks/pilot-operations.md). The independent NUC restore, rendered reader checks with Archive.org blocked,
and hosted CI passed; evidence is in the qualification report. Public ingress and
scheduled preservation operations remain follow-up work.

## Catalog design follow-up

The owner selected **A — Front Cover** on October 3. The [selected interface](../design/catalog-previews.md) uses actual catalog metadata and local cover scans. Search, pagination, document reading, downloads, provenance and rights/contact remain available. Comparison prototypes and unused layouts are removed. Required repository verification and desktop/narrow browser checks qualify the implementation; deployment remains separate.

## Full collection follow-up

The owner authorized automatic archival access for the complete Anarchist source. [The full-collection plan](0004-full-anarchist-collection.md) supersedes the original pilot scope and individual approval policy for this collection only.
