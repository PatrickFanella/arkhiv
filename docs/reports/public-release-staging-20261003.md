# Public release staging — 2026-10-03

The owner approved public reading and downloads for the exact 25 pilot documents
and 26 PDF variants. Rights remain unknown, source attribution and the contact
`info@subcult.tv` are visible, and Arkhiv asserts no reuse license. The reviewed
manifest is [anarchist-zines-public-review.json](../../collections/anarchist-zines-public-review.json).
A-Info is preserved and privately reviewed; none of its variants is public approved.

## Artifacts and qualification

| Boundary | Verified result |
| --- | --- |
| Public API | `arkhiv-api:85f100f`, source `85f100f20b3f2260f852108aa47863908fb8814e`; PR #45 merged and exact source hosted CI passed |
| Public web | `arkhiv-web:77b500d`, source `77b500d6c17a5520549c0a7997061e161d7d3a4a`; hosted runs 12088/12089 passed |
| Private pilot | Original private API/web source `6470472` retained; no primary DB replacement |
| Actual staged corpus | 25 documents, 26 SHA-256/size-matched downloads, 26 painted PDF canvases, Archive.org blocked and zero source requests |
| Denied routes | 75 second-archive document/asset/download/cover/search routes denied for GET and HEAD with conditional and range headers |
| Narrow layout | 390×844 reader page navigation, 125% zoom and scoped search passed without horizontal overflow |
| Revocation | Temporary first-variant restriction denied conditional/range document and asset routes; originals remained and the exact reviewed manifest was reapproved |
| Public readiness | HTTP 200, public mode and exact API revision checked by the existing preservation monitor; no problems |
| Caddy candidate | Full current configuration plus the proposed Arkhiv import validated; live config hash remained `0849b154e5645241ffc464116de95f4591b851d6726d684a3832305dcbe3529f` |
| Pre-release NUC backup | `auto-20261003T012109Z`, 317 archive files, 2,444,342-byte catalog dump, SHA-256 `6615196d1be1cdd51e11d2d64c6c3eb5e13f97447e4b77101783069ed6da0d53` |

The public API has read-only archive storage, a read-only root filesystem,
bounded resources, private database networking and a dedicated database role
with only SELECT grants. It cannot select raw source snapshots. Same-origin
Nginx serves the local reader, worker, fonts and PDF assets with no-store responses.
The public web binds only `127.0.0.1:54614` for staging and joins the existing
`projects` network as `arkhiv-public-web` for the proposed Caddy route.

The concrete route is [deploy/arkhiv.Caddyfile](../../deploy/arkhiv.Caddyfile).
The isolated composition and role grants are in
[public-compose.example.yml](../../deploy/public-compose.example.yml) and
[public-role.sql](../../deploy/public-role.sql). Real credentials remain outside Git.

## Activation and rollback

Issue #33 explicitly requires a separately authorized release action. The staged
site is qualified; public ingress has not been activated. DNS and HTTPS already
reach the homelab's existing wildcard route, but `https://arkhiv.subcult.tv` still
returns 404. Staging proof does not establish public-domain qualification (#34).

The proposed action installs only the Arkhiv site file, appends its explicit
import to the current Caddyfile after backing it up, validates it, and reloads
Caddy. Keep the wildcard tunnel and all other routes unchanged. Public readiness
then checks the actual HTTPS domain. Record public hashes, rendered reading,
negative/revoked routes and a defined initial observation window before closing #34.

Ingress rollback removes only that import and reloads validated configuration.
Do not restore an old whole Caddyfile over unrelated subsequent edits. Image rollback
uses the retained public images/composition. Publication restriction uses
`revoke-public` for the exact variant and records a review event; it retains originals
and does not claim to retract copies already downloaded. Private services,
primary catalog, independent backups and restored evidence remain recoverable.

Detailed receipts and screenshots are outside Git on Almaz under
`/mnt/spektr/server/data/arkhiv/qualification/public-staged-20261003` and on Kvant
under `~/.local/state/arkhiv/execution-20261003`. This is initial qualification,
with no public observation window or long-term schedule soak claimed yet.

Desktop native keyboard inspection focused the Arkhiv home link. The native tab was not foreground, so this observation did not establish its focus-visible appearance. A missing global accent fallback was then fixed; all four desktop/narrow deterministic browser tests passed with a visible-outline assertion. The public palette has contrast ratios 7.07:1 for accent links on the page background, 8.56:1 for white button text on the accent, and 11.29:1 for body text on panel backgrounds. Hosted desktop/narrow fixture checks cover loading, empty/error states, labels, keyboard focus and reader controls; the actual corpus probe separately proves real PDF rendering and narrow layout.

## Subsequent activation

The owner separately authorized public activation. Caddy installed only the Arkhiv route/import and reloaded successfully at 2026-10-03 01:49:08 UTC. The web now uses CI-qualified source `5280c4c` with the header focus correction. Actual-domain checks passed for all 26 PDFs and denied routes; the [public qualification report](public-release-qualification-20261003.md) records final observation and monitoring. The pending/404 statements above describe staging before this later action.
