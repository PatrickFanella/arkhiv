# Second archive qualification — 2026-10-03

A-Info proves the framework can preserve and serve a collection whose documents
live in separate Internet Archive items. The private pilot contains 15 issues
from São Paulo with title dates in 2011–2012. Their source is
[the A-Info collection](https://archive.org/details/a_info_zine); the bounded query
and exact selection are in [the selection report](a-info-selection-20261003.md).
The first archive remains available throughout qualification.

## Counts and preserved state

| Observation | A-Info | First archive | Combined |
| --- | ---: | ---: | ---: |
| Matching discovered/selected items | 15 | 1 bundled item | 16 |
| Selected/verified/reviewed/private-published documents | 15 | 25 | 40 |
| Selected/verified PDF variants | 15 | 26 | 41 |
| Verified OCR companions | 15 | 17 | 32 |
| Unique preserved assets | 30 | 43 | 73 |
| Original asset bytes | 98,015,146 | 31,039,265 | 129,054,411 |
| Documents with page text | 15 | 24 | 39 |
| Documents with document-only OCR | 0 | 1 | 1 |
| Public-approved variants | 0 | 26 | 26 |
| Remaining failed selected acquisitions/extractions | 0 | 0 | 0 |

The parent A-Info collection listed 139 items; qualification selected the 15
matching the explicit query, rather than mirroring that parent. Source inventory
counts include all file formats: A-Info checked 165 files and the first bundled
item checked 5,337. These are not document counts. Initial inventory events remain
in review (180 and 5,338 respectively); an event is not an imported/public work.

All 30 A-Info files have exact source identities, source checksums, verified local
SHA-256 and size receipts. Initial acquisition encountered official regional
Archive.org redirects and a 503 Retry-After for an OCR companion. Regional host
validation now accepts only the authenticated HTTPS Archive.org namespace.
Verified partial work was retained; the final bounded resume reused completed
assets and finished with 30 files, 98,015,146 bytes and no remaining failures.
Historical failed attempts remain evidence, not discarded successful results.

Fourteen issues have four pages and one has twelve. All 15 have page text.
Creator and precise catalog date remain unknown; title months are not silently
converted to exact dates. Private review records readability, attribution and
unknown rights. It does not approve a second public release.

The editorial title of `a_info_1` is `A-Info 1 — outubro de 2011`; its original
source title remains independently preserved. A repeated manual refresh and the
real scheduled check retained this correction, source provenance and first-archive
publication/grouping. Native desktop inspection painted page two of this actual
issue, showed the A-Info collection name and purple controls, and blocked a direct
Archive.org fetch. The reader uses local files, fonts and worker assets.

## Updates and deterministic checks

Multi-item discovery enforces query/item/page/request/time limits. Complete query
baselines survive partial pagination and missing/malformed metadata; source removals
queue review and retain local documents. Shared-byte fixtures retain distinct
collection/item provenance while deduplicating storage. Scoped search/navigation,
facets and public allowlisting retain collection boundaries.

The evolving-source PostgreSQL fixture adds a third item, acquires it under the
explicit policy, leaves publication/public approval false, and repeats without
new acquisition or duplication. Other fixtures cover changed bytes, exclusions,
failed baselines, removals and durable payload backoff. This is synthetic addition
proof; no real new source item is claimed.

The actual Almaz update timer ran **01:35:12–01:36:29 UTC**. Operator source
`03c3306c013f517dd014e9294aa04af6b127df74` checked both sources: 165 and 5,337 files,
zero added/content/metadata/removed/uncertain changes, and zero acquisitions.
This controlled initial timer qualification bypassed cadence explicitly while
retaining enable and source-backoff checks. The temporary setting was removed.
Ordinary cadence remains 24 hours for the first archive, 168 hours for A-Info;
both use review-only acquisition. The overnight/weekly calendar cycle has not
been observed.

## Independent backup and fresh recovery

NUC independently verified snapshot `auto-20261003T012109Z`: **317 archive files**,
**2,444,342-byte catalog dump**, SHA-256
`6615196d1be1cdd51e11d2d64c6c3eb5e13f97447e4b77101783069ed6da0d53`.
The restore timer ran **01:37:21–01:38:27 UTC**, fetched that exact NUC snapshot,
verified it, and created fresh storage/database in
`qualification/arkhiv-drill-20261003t013723295175z` on Almaz.

Audit matched **73 assets, 41 variants and 40 private-published documents**.
The restored reader matched all **41 PDF download hashes/sizes** and painted all
**41 variants**, with Archive.org blocked and zero source requests. Scoped
`solidarity` and `anarquismo` search checks passed. Direct primary/restored row
comparison matched every document, variant, asset, 100 editorial review events
and 53 public-access events, including metadata overrides, grouping and approval
flags. The exact isolated stack was removed afterward; original qualification,
restored bytes, screenshots and receipts remain retained. Public deployment
credentials were not copied; the drill's public role has NOLOGIN.

A later complete backup, `auto-20261003T013906Z`, includes the source-check receipts:
323 files, 2,873,431-byte dump, SHA-256
`9ca795c7de6135d12d9468e104d275b57382a55642aaf6c7e5b5342d7ec8fec9`.
It is independently verified; the actual restore above used the earlier named
snapshot. No off-site copy or long-term monthly recovery soak is claimed.

## Code and evidence boundaries

PR #44 implements query discovery/configuration; PR #46 adds durable payload
backoff and new-item acquisition/replay proof; PR #47 adds public readiness and
fresh-role restore handling. All are merged. Exact runtime source `03c3306` passed
hosted runs 12083/12084; web/operations source `77b500d` passed 12088/12089.
The private web still identifies merged `6470472`; the restored API identifies
`03c3306`, probe `4194aaf`, and operations helper `77b500d`.

Required isolated `make verify`, race-enabled `make test-updates` and
`make test-recovery` passed. Exact-source hosted CI also passed `make test-browser`
on disposable PostgreSQL 17. Actual homelab corpus/restore and native browser
proof are separate from those fixtures. Public staging separately denies all
second-archive routes; its [report](public-release-staging-20261003.md) records that
boundary. Public-domain activation/qualification subsequently passed #33–#34; see [the public report](public-release-qualification-20261003.md).

Private evidence is on Kvant under `~/.local/state/arkhiv/execution-20261003`
and on Almaz under `/mnt/spektr/server/data/arkhiv/qualification`, including the
scheduled refresh, independent restore, catalog comparison and reader screenshots.
Acquired documents, generated stores and database dumps remain outside Git.
Canonical Almaz/NUC facts are committed in Machinum's Arkhiv operations notes;
shared host briefing edits and generated offline files were preserved.
