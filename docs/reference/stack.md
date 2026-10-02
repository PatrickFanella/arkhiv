# Stack reference

Arkhiv uses Go, React, Vite, TypeScript, Tailwind, PostgreSQL, and Docker.

| Boundary | Current behavior |
| --- | --- |
| Go API | Standard-library HTTP server, graceful shutdown, pgx PostgreSQL pool |
| Readiness | `/api/health` checks process response; `/api/ready` pings the catalog |
| Catalog | Collection-scoped published documents, variants, facets, metadata and text search |
| Web | Configured collection name/introduction/accent, catalog, detail, local PDF.js reader |
| API origin | Same-origin `/api`; Vite forwards to local Go and Nginx to Compose `api:8080` |
| PostgreSQL | Version 17 for Compose/homelab/browser fixtures; 15 for shared-tools integration snapshots |
| Migrations | Sorted transactional SQL, advisory lock, schema-version checksums |
| Preservation | Filesystem originals addressed by SHA-256; raw source snapshots, receipts and checkpoints |
| Extraction | Bounded Poppler jobs, page text or document-level OCR, cover generation |
| Recovery | Original audit, portable catalog dump, complete file manifest, fresh-destination restore |
| Verification | `make verify`, explicit integration/recovery gates, isolated browser suite |
| CI | Pinned Go/Node/pnpm, repository gate, isolated browser/PostgreSQL 17 smoke |

Go builds write to ignored `backend/bin/`. Frontend installs use the committed
pnpm lockfile. PDF.js workers, fonts, CMaps, and WASM assets are copied from the
installed package during development/build and served locally. Generated assets
are excluded from Git and lint; module workers and WASM have explicit MIME types.

The T3 profile is authored in `~/Work/Tools/t3-dev-environments`. It uses an
isolated source snapshot, caches, loopback proxy, and separate test storage.
Preview seeds synthetic fixtures; normal app/test networks cannot acquire live
source files. Its Check Code action runs `make verify` without a host Docker socket.
The separate browser gate builds its own disposable project, with an internal
runtime network and matching PostgreSQL 17 backup tools. Live acquisition and
independent restore are recorded as separate pilot evidence.
