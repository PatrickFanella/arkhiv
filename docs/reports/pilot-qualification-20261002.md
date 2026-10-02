# Private pilot qualification — 2026-10-02

The Anarchist Zine Archive passed its private preservation, catalog, reader, and
independent restore qualification. Public release remains a separate decision.
The initial implementation merged in [PR #19](https://git.subcult.tv/subculture-collective/arkhiv/pulls/19);
hardening and qualification are in [PR #20](https://git.subcult.tv/subculture-collective/arkhiv/pulls/20).
[issue #14](https://git.subcult.tv/subculture-collective/arkhiv/issues/14) holds
qualification receipts. Issues remain open for review and merge.

## Corpus and preservation

The live `ZineArchive` metadata listed 700 PDFs and 470 OCR text files. These are
file counts, not distinct works. The explicit configuration selected the following:

| Observation | Result |
| --- | ---: |
| Distinct selected, verified, reviewed documents | 25 |
| Documents published within the private pilot | 25 |
| PDF variants | 26 |
| OCR companions | 17 |
| Preserved unique assets | 43 |
| Original bytes | 31,039,265 |
| Documents with page text | 24 |
| Documents with document-only OCR | 1 |
| Failed acquisitions or PDF extractions | 0 |
| Searchable text rows | 447 |

The Le Guin document retains separate print/read variants. Queering Protest Sites
uses its OCR companion without invented page numbers. The corpus spans mutual aid,
prisons and abolition, gender, ecology, work, international solidarity, and theory.
The committed selection manifest records filenames, grouping, and inclusion rationale.
Source snapshots, checksums, and acquisition receipts remain outside Git.

The final acquisition rerun reused all 43 assets, with no new transfers or failures.
Catalog totals and publication remained stable; reviewed creator/title corrections
survived. Integration tests separately prove that changed source metadata cannot
overwrite editorial corrections. Unknown creators, dates, and item-level rights
remain explicit. Private review does not establish permission for a public release.

## Homelab destinations

- Primary: **Almaz**, `/mnt/spektr/server/data/arkhiv/pilot`, on the Spektr SSD
  (`/dev/sde1`; approximately 183 GiB available when selected). The acquisition
  operator owns its files as UID/GID 1000. Promoted originals have mode 0400;
  application serving uses a read-only mount.
- Independent backup: **NUC**, `/home/onnwee/backups/arkhiv`, on `/dev/sdb2`,
  approximately 103 GiB available when selected. Directory mode is 0700.
- Qualified snapshot: `pilot-qualified-20261002` on NUC; **148 archive files**
  and an **827,040-byte PostgreSQL custom dump**. The manifest verifies every
  file and the database dump. Both hosts are in the same homelab; this is an
  independent host/disk copy, with no off-site copy or scheduled retention claim.

The private Compose project on Almaz runs PostgreSQL 17 with a matching backup
client. Web ingress binds only `127.0.0.1:54610`. Catalog/API networking is internal;
a separate operator profile permits bounded acquisition. Browser connections are
restricted to the same origin by CSP. Credentials are in a private `.env` and do
not appear in source, issues, or this report.

Runtime API image: `sha256:62fafaf069dda465bf3379aff563b338fe5acb89e4906013afc784afc1a5b716`.
Runtime web image: `sha256:16184a6766719937b9edd6c03a861dbcc6c3c4f8079e2ebf35f93c08279a3e7f`.
Both image labels identify code revision `5ad25731a9963acc3131a705cb78c8c24e46194c`.
Later PR commits add inventory reporting and verification assertions; they are not
claimed as deployed runtime images.

## Independent restore proof

The final snapshot was transferred to NUC, checked there, then fetched back from
NUC into a separate staging location. Restore used a new archive root
`/mnt/spektr/server/data/arkhiv/qualification/restored-final-20261002` and fresh
PostgreSQL database `arkhiv_restore_final_20261002`.

The restored catalog exactly matched the final primary API response: 25 documents,
26 variants, 43 assets, one source snapshot, 156 acquisition attempts, 58 review
events, and 447 text rows. All 26 PDF download SHA-256 hashes and lengths matched
the originals. The query `solidarity` returned 14 documents before and after restore.
Editorial corrections and variant grouping were retained.

The restored reader rendered page two in the native T3 browser. An explicit
Archive.org fetch was rejected by the browser CSP, while the local PDF remained
rendered; the restored API also could not reach the source. Actual reading,
search, and downloads used locally preserved assets. This is private pilot proof,
not a public ingress or production release check.

## Verification and evidence

- Required `t3-dev --profile arkhiv verify` passed in disposable isolated snapshots,
  including formatting, TypeScript, lint, Go/web tests, PostgreSQL integration,
  both builds, Compose configuration, and the retained Open Pilot template.
- Separate race-enabled integration/recovery checks and Go race tests passed.
- Four isolated browser tests passed at desktop and 390px widths, covering actual
  PDF rendering, navigation, zoom, search/page links, original download hash,
  keyboard focus, states, collection configuration, and no horizontal overflow.
- Native T3 browser inspection and screenshots show the actual primary and restored
  corpus. Native viewport resize timed out; narrow-width proof comes from the
  automated browser suite, not a native device test.
- [Hosted CI run 11960](https://git.subcult.tv/subculture-collective/arkhiv/actions/runs/11960)
  passed for exact runtime revision `5ad2573`, including the separate browser/PG17
  gate. Later source-head CI is linked from PR #20 and issue #14.
- The companion [T3 environment PR #3](https://git.subcult.tv/PatrickFanella/t3-dev-environments/pulls/3)
  adds isolated Arkhiv fixture/database gates. Eleven launcher regression tests
  passed locally. This does not claim hosted CI for that tools repository.

Bootstrap hosted CI exposed an incorrect pnpm version; the implementation pins
10.33.0. Native reader verification exposed the worker's MIME type; Nginx now serves
`.mjs` and WASM correctly. A PostgreSQL backup client/server version mismatch was
resolved with matching clients and a disposable test-only DB version override;
the retained preview database was preserved.

Attached evidence:

- [restored-reader-page2.png](https://git.subcult.tv/attachments/abe62093-7254-49e7-b5c5-b692e88664ab)
- [acquisition-repeat.json](https://git.subcult.tv/attachments/72db0360-3339-45b0-9954-18ae2978b2cc)
- [restored-catalog-checks.json](https://git.subcult.tv/attachments/ee9a1420-48ee-45c2-9621-1eb4ede374ac)
- [original-download-hashes.json](https://git.subcult.tv/attachments/28bab93d-829e-4ca7-80d9-29a360549e6a)
- [primary-reader.png](https://git.subcult.tv/attachments/0e4243f8-7825-4ff5-8eb7-ab960e770122)

Detailed local acquisition/recovery evidence is under
`~/.local/state/arkhiv/implementation-20261002/` on Kvant. Acquired documents,
generated stores, database dumps, and credentials are excluded from Git.

## Operating boundary and next work

Use the [pilot runbook](../runbooks/pilot-operations.md) for import/resume, review,
integrity checks, extraction, backup, and fresh-destination restore. Public release,
a second real archive/discovery, curator accounts, scheduled backup monitoring,
retention, periodic restore practice, and improved OCR remain follow-up work.
The existing Gitea roadmap keeps those phases separate.

The primary site is inspectable from Kvant through the task's SSH forward at
`http://127.0.0.1:54611/`; the restored copy is at `http://127.0.0.1:54613/`.
Those forwards depend on the current SSH session. To reopen the primary forward,
run `ssh -N -L 127.0.0.1:54611:127.0.0.1:54610 almaz`.
To stop this pilot, run `docker compose --project-name arkhiv-pilot down` from
`/mnt/spektr/server/data/arkhiv` on Almaz. Do not use `--volumes` or delete archive
roots/backups. Keep restored storage as qualification evidence. No other service
needs to be stopped or reconfigured.
