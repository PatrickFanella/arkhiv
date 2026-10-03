# Public archive release qualification — 2026-10-03

The owner separately authorized public activation. The Anarchist Zine Archive
is live at **https://arkhiv.subcult.tv** with **25 documents and 26 exact reviewed
PDF variants**. Originals are downloaded, preserved locally and independently
backed up; the site reads/downloads those preserved bytes. Rights remain unknown,
source attribution and `info@subcult.tv` are visible, and no reuse license is asserted.
The 15 A-Info issues remain privately reviewed with no public approvals.

## Release identity and scope

Caddy activated only the Arkhiv site/import at **01:49:08 UTC**, with a successful
configuration validation and scoped reload. The existing wildcard tunnel/DNS,
other site routes, private pilot and primary PostgreSQL service were preserved.
The retained pre-change Caddyfile hash is
`0849b154e5645241ffc464116de95f4591b851d6726d684a3832305dcbe3529f`;
the activated configuration hash is
`c71115d0c5cf07e62169e5c99bba679f0670a9c7c6497e59403aa5fb2176bbdb`.
Rollback removes only the Arkhiv import, validates and reloads; it must not overwrite
subsequent unrelated edits with an old whole Caddyfile.

| Artifact | Exact source | Actual image identity |
| --- | --- | --- |
| Public API | `85f100f20b3f2260f852108aa47863908fb8814e` | `sha256:09c261dd3f3d9ced6891bf9a6cd95fd203d6404af9d1f0aa20704d2c3e1a3203` |
| Public web | `5280c4cb8ac44fee2d41cc75199356564428acc7` | `sha256:e427de6afa55e4dcb414208452ca27f768c78aa7c9475941673c0a86150612a3` |
| Acquisition operator | `03c3306c013f517dd014e9294aa04af6b127df74` | `sha256:b6c35e0e93396719b02fbbc1c67aa6fce1ef9be4456ed2d2adad38b917c2bfb0` |

The private reader remains at merged source `6470472`. Public readiness reports
public mode and full API revision `85f100f`; the operations helper now identifies
source `7425fee6f5a97e6528f904ecdec37cbb52bdbca8`. Exact web CI runs 12094/12095 passed;
operator runs 12083/12084 passed; monitor runs 12097/12098 passed. The first 12097
attempt failed in dependency installation with a pnpm worker exit. Its failed-job
retry passed; that infrastructure failure is retained separately from code checks.
All hosted verification includes the disposable PostgreSQL 17 browser smoke.

Public serving uses a SELECT-only database role, read-only originals/root filesystem,
private backend networking and bounded memory/CPU/PID settings. The API cannot read
raw source snapshots. Nginx serves same-origin local PDF.js worker/fonts/assets and
enforces the reviewed variant allowlist across catalog and asset routes.
[Staging and rollback](public-release-staging-20261003.md) records the concrete
configuration and separately retained pre-activation qualification.

## Actual HTTPS and access proof

DNS resolved the public domain through the existing Cloudflare route. A verified
TLS 1.3 connection on October 3 validated the hostname/certificate, issued by
Google Trust Services WE1 and valid through December 15, 2026. Requests returned
HTTP 200, `Cache-Control: no-store` and edge status `DYNAMIC`; no public/CDN byte
cache was observed for tested routes.

The actual-domain browser probe verified **25 records, 26 original PDF SHA-256/size
matches and 26 painted PDF canvases**, with Archive.org blocked and zero source
requests. All **75 second-archive denied routes** passed for GET and HEAD with
conditional/range headers. A 390×844 reader/search check passed page navigation,
125% zoom and scoped search without horizontal overflow.

The native T3 browser separately painted page two at **1120×865**, with no reader
alerts and a direct Archive.org fetch blocked. It displayed unknown rights,
archival access approval, no reuse license, source attribution and the correction/
removal mail link containing the public document URL. Public navigation excludes
A-Info. Canonical links/title and entry-page social metadata point to the approved
site. Deterministic desktop/narrow checks cover keyboard focus, visible outline,
labels, loading, empty/error states, page/zoom/search and downloads.

A bounded actual-domain revocation test ran **01:52:50–01:52:53 UTC**. Ten requests
covered document/asset/download/cover/page-search GET and HEAD with range and
conditional headers. All returned **404**, `no-store` and `DYNAMIC`; the original
bytes stayed preserved. Reapplying the exact owner manifest restored all 26
approvals immediately, with audited events and no asserted reuse license. This
proves current access withdrawal, not retraction of copies already downloaded.

## Monitoring and observation

The initial Python-default readiness request returned 403, while requests with an
explicit Arkhiv monitor user agent and JSON Accept header returned 200. The runner
now uses that identified request and still checks public mode/exact revision. The
failed receipt remains; NUC's existing ntfy receiver cached the failure and recovery
as rows 19055/19056. Receiver cache evidence does not prove a user saw the messages.
The corrected monitor returned no problems at 01:53:59 UTC.

The defined **10-minute window**, 2026-10-03T01:53:31.502714+00:00 through 2026-10-03T02:03:31.502748+00:00, completed with **21 healthy samples and zero failures**. Every sample checked exact public readiness, the 25-document/26-variant allowlist and a denied private record through HTTPS with no-store responses. The natural 02:00 UTC monitor trigger completed successfully with no problems. Earlier unqualified default-request samples and the monitor 403 are retained separately. This is initial public qualification, not a long-term availability soak.

Scoped rollback triggers are private-byte exposure, an approved-original hash mismatch, widespread reader failure or persistent readiness failure after bounded diagnosis. Restrict an affected variant immediately when its access/integrity is wrong; remove only the Arkhiv ingress import for a release-wide fault. Retain originals, backup/restore evidence and the failed receipt. The initial monitor-only 403 was diagnosed and corrected while corpus/domain checks stayed healthy.

## Preservation and recovery

Pre-release NUC snapshot `auto-20261003T013906Z` independently verified 323 archive
files and a 2,873,431-byte dump, SHA-256
`9ca795c7de6135d12d9468e104d275b57382a55642aaf6c7e5b5342d7ec8fec9`.
After release/revocation/reapproval, `auto-20261003T015516Z` independently verified
323 files and a 2,873,703-byte dump, SHA-256
`378fcb67944ff5a77e797b0a2071ad41953955c85003d50e818915a0820d4bc2`.
Qualified restore/pre-release snapshots are explicitly pinned; retention planning
keeps them and the newest good snapshot. Pruning remains dry-run and deleted no bytes.

The earlier actual timer-triggered fresh restore from NUC snapshot
`auto-20261003T012109Z` verified both collections: 40 documents/41 variants/73 assets,
all 41 hashes/downloads/rendered PDFs with source access blocked, and exact document,
variant, asset, editorial and access-history rows. Its separate
[second-archive report](second-archive-qualification-20261003.md) names that exact
restore input; this is not a claim that the later post-release snapshot was restored.

## Evidence and closeout

Safe code/qualification reports are committed; originals, raw metadata, database
dumps, credentials and generated evidence remain outside Git. Primary release
receipts/screenshots are on Almaz under
`/mnt/spektr/server/data/arkhiv/qualification/public-release-20261003`; independent
backup verification is on NUC; native screenshots and TLS/observation receipts are
on Kvant under `~/.local/state/arkhiv/execution-20261003`.

Canonical Almaz/NUC deployment facts are maintained in Machinum's Arkhiv notes.
The required full isolated `make verify`, race-enabled updates/recovery and four
isolated desktop/narrow browser checks passed. These deterministic gates, hosted
CI, actual domain, source-blocked reading, independent receiving-host backup and
fresh restore are separate proof boundaries. Weekly/monthly unattended cycles,
long-term public soak and an off-site copy are not claimed.
