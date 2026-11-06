# Chicago Labor Archive assessment — November 6, 2026

This report assesses the Chicago Labor Archive frontend and its plan
(`docs/plans/0011-chicago-periodical-archive.md` in the source worktree).
At the time of the assessment the work was still uncommitted on
`feat/organize-periodical-library` in a separate T3 worktree (base `316c21f`). A
snapshot was copied into this worktree for testing and then removed, so the
source worktree was never modified. Snapshot hashes: tracked diff
`c64ba5e9…dcaa0`, untracked files tar `e38cc344…fe975`.

## Method

- **Required gate.** `make verify` passed on the snapshot (exit 0): Go
  vet/race/build, 51 viewer tests, packed consumer checks, operator script tests,
  web typecheck/lint, 10 web tests (3 new Chicago appearance tests and a Chicago
  prerender test) and the production build. The build emits both
  `dist/index.html` and `dist/chicago/index.html`.
- **Browser.** The actual production `dist` was served by an assessment-only mock
  of the catalog/print API. The mock returned the **4,742 real MIA PDFs** of the
  20 selected publications (SDS excluded) with labels and dates parsed from the
  acquisition manifest, served the original files, and generated page text with
  `pdftotext`. Headless Chromium covered 1440, 390 and 320 px, light and dark
  mode, keyboard focus, deep links and failure states. axe-core checked WCAG
  2.0/2.1/2.2 A and AA rules.
- **Corpus.** `pdftotext` was run on the first two pages of 8 randomly sampled
  PDFs per publication (160 files) to see which titles actually have a text
  layer.

The building agent's own receipt (`_planning/chicago-design/frontend-applied/`)
reports 208 browser checks against two synthetic records and a 1 KB PDF. This
assessment does not repeat that work. It uses the real corpus and catalog size.

## What works

- The catalog renders 24 rows per page from 4,742 items with no horizontal
  overflow at 1440, 390 or 320 px. On narrow screens, table rows become labeled
  cards.
- Title search (54 Proletarec issues), date range (407 issues in 1909) and the
  empty-result message behave as labeled.
- Real newspaper scans open in the PDF reader. An 8-page Chicago Daily Socialist
  issue rendered in about 0.8 s on loopback. The download link carries
  `action=download` and the original filename.
- For a PDF with text (International Socialist Review), in-document search
  returned four page hits. Choosing the last hit moved the reader to page 4.
- Reloading a deep link (`?item=…&representation=…`) reopens the same document.
- Dark mode follows the system setting before React loads and swaps to the
  light-on-dark mark. The meta `theme-color` follows the theme. Presets, custom
  hex values (`#ffff00` is adjusted to `#707000` in light mode), invalid-input
  alerts and storage persistence work.
- The catalog's 503 state shows a retry button. An engine that lacks the
  collection gets an explicit "Archive unavailable" page.
- axe-core found **no violations and no incomplete results** in light/1440 or
  dark/390, with the reader and appearance panel open.
- The reference site at `/` is unaffected and does not load the Chicago
  stylesheet.

## Findings

### 1. Merging to `main` would publish `/chicago/` on arkhiv.subcult.tv (blocker)

`verify.yml` (added in `316c21f`) builds `web/dist` on every push to main and
deploys it as the `arkhiv-seo-web` image for `https://arkhiv.subcult.tv/`. The
Chicago entry and its brand assets and fonts are part of that `dist`, and the
reference application image copies the same directory. A merge would therefore
expose `/chicago/` publicly. That engine is not scoped to
`chicago-labor-archive`, so the page would show "Archive unavailable". Plan 0011
and AGENTS.md both require a separate owner decision for public deployment,
and the plan places the Chicago interface in a separate consumer repository.
Keep the branch unmerged, or move the frontend into its own consumer repository
before any merge, as the plan intends.

### 2. The engine contract assumes every source is Internet Archive (blocker for real data)

`backend/internal/app/transport.go:56` and `catalog/catalog.go:648` build
`source_url` as `https://archive.org/details/<source_item_id>`.
`representationURL()` in the viewer SDK rejects any other source host. The
reader always shows "Source item at Internet Archive". The Chicago corpus comes
from marxists.org, so its records cannot be represented without inventing an IA
identity. Phase 2 needs a small core change that adds a generic, reviewed
source identity. Plan 0011's information model already asks for "Provider and
exact URL".

### 3. Two-thirds of the pilot scans have no searchable text

In 54 of 160 sampled PDFs, the first two pages had more than 50 words of text.
Chicago Daily Socialist, Workers' Call/Chicago Socialist, Hope, Agitator, Social
Democratic Herald, Appeal to Reason and others were 0/8. Proletarec,
International Socialist Review, Proletario and Proletarian News were 8/8. For
the image-only issues, the reader honestly says "Document search is
unavailable". Until OCR runs, however, the archive's core Chicago dailies cannot
be searched. This confirms the plan's OCR phase. Use these counts to size the
pilot rather than treating OCR as optional.

The scans are also large: Chicago Daily Socialist has a median of 15.3 MB per
issue (maximum 40.8 MB), and Workers' Call 14.7 MB. Measure range loading and
mobile transfer for these files before deciding on a reading-derivative budget.

### 4. Viewer page, catalog page and filters are not in the URL

Reloading a document opened at page 4 returns it to page 1. Catalog pages exist
only in component state: there is no way to link to page 40 of 198, and only
Previous/Next buttons are available. The plan requires "Keep viewer page state
in the URL" and stable citation routes, so these gaps belong to phase 3.

### 5. Text bug at 700 px and below: "suburbsand"

`facsimile.css` hides the `<br>` in the scope aside at 700 px and below, which
leaves "Chicago, its suburbsand the Calumet region." Replace the `<br>` with a
space and a CSS line break, or add a space before it.

### 6. Closing a document drops keyboard focus

After "Close item", focus returns to `<body>` and the page stays scrolled to the
former reader position (y≈2550). Return focus to the Read button that opened the
document. This comes from shared `MixedArchive` code, so it also affects the
reference catalog.

### 7. Chicago-specific copy lives in the shared reference component

`MixedArchive.tsx` hardcodes the scope aside and the "A regional record, in its
own words." section. `Header.tsx` hardcodes "Chicago / Labor Archive" instead of
`brand.title`, although the plan leaves the final name open. If this code stays
in the engine repository, move the copy into `site.json`. If it moves to a
consumer repository, the problem goes away.

### 8. Smaller issues

- The reader eyebrow shows the raw collection slug (`print · chicago-labor-archive`).
- The item count has no thousands separator ("4742 items").
- An inverted date range (from 1912 to 1909) returns zero results with no
  explanation. The UI sends the range unchecked; engine behavior was not tested.
- The "Share item" button sits flush against the "Share on Bluesky" link.
- Fonts ship as TTF (~200 KB for three faces) with no preload. WOFF2 would be
  smaller.
- Both mark images are requested on every load; one is always hidden.
- `/api/account/v1/session` returns 404 when personal features are off, which
  logs a console error on every load.
- `/chicago/` has no prerendered HTML, canonical URL or sitemap entry. Crawlers
  see an empty root.
- MIA labels for International Socialist Review are article titles ("Executive
  Committee Rule"), not issues. This is the issue/article distinction that the
  plan's information model is meant to handle.

## Not covered

No real engine, database, import, access review or print indexing ran; the API
was mocked. The mock implements only the endpoints the frontend calls, so the
following were not tested: engine search ranking, access filtering, catalog
ordering, Range responses and the actual `text_status` values. There was no
physical mobile device, screen reader, hosted CI or public URL check. The plan's
phase 1 inventory, phase 2 processing pilot and later phases have no
implementation to test yet.
