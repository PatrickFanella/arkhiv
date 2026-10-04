# Anarchist Archive

Anarchist Archive acquires explicitly selected document files, preserves their original bytes,
and serves branded catalogs with local PDF reading and downloads. The first
collection is the Anarchist Zine Archive.

The implementation includes a PostgreSQL catalog, operator CLI, bounded resumable
acquisition, editorial review, text and cover extraction, metadata/page search,
and backup/restore. The initial manifest selects 25 documents, 26 PDF variants,
and 17 matching OCR companions from Internet Archive's `ZineArchive` item.
The [private pilot qualification](docs/reports/pilot-qualification-20261002.md) passed:
all originals were verified, the independent NUC backup was restored, and on-site
reading/search/downloads worked with Archive.org blocked. The initial implementation merged in PR #19; hardening and the qualification
report merged in [PR #20](https://git.subcult.tv/subculture-collective/arkhiv-alpha/pulls/20).
The owner-approved first collection moves from arkhiv.subcult.tv to [anarchist-archive.subcult.tv](https://anarchist-archive.subcult.tv) during the reviewed rollout. Its full selected source now has 699 documents and 700 PDF variants available for public reading/downloads with unknown rights, source attribution, correction/removal contact and no asserted reuse license.

The [next delivery plan](docs/plans/0002-managed-archive-and-public-release.md)
covers merged-revision rollout, scheduled preservation and active-source updates,
public release, and a second real archive. Managed jobs are installed and initially qualified. A-Info adds 15 privately reviewed issues; a fresh independent NUC restore verified all 40 documents/41 PDF variants with source access blocked. Gitea #21–#38 records the separate implementation and live evidence.

## Development

Stack: Go 1.26+, Node 24, pnpm 10.33.0, React/Vite, PostgreSQL 17, Docker Compose.

```sh
t3-dev --profile arkhiv start
t3-dev --profile arkhiv sync
t3-dev --profile arkhiv verify
t3-dev --profile arkhiv test --command 'make test-integration'
t3-dev --profile arkhiv stop
make test-browser
```

The T3 preview seeds two clearly labeled synthetic collections. Its application
and test networks cannot reach Archive.org. The integration snapshot uses
PostgreSQL 15 to match the shared tools image's backup client; the isolated browser
suite and homelab pilot use PostgreSQL 17 with a matching client. Browser tests
run in a separate disposable Compose project and clean up their own containers.
`make test-recovery` runs the recovery assertions in the integration suite.

`make verify` checks formatting, TypeScript, lint, Go/web tests, both Go binaries,
web build, Compose configuration, and the retained Open Pilot template. Database
integration runs when `TEST_DATABASE_URL` points to the disposable launcher DB;
the explicit integration target refuses to run without it. Hosted CI and rendered
pilot/restore evidence remain separate checks.

For a manual container build, `docker compose up --build` serves loopback port
5173. The API applies checksum-locked migrations and requires a managed archive
root when a database is configured. The sample database password is for disposable
local development. `/api/health` checks process health; `/api/ready` checks database
connectivity. Catalog and asset routes are read-only; mutations use the CLI.

The [operator runbook](docs/runbooks/pilot-operations.md) covers planning, acquisition,
review, extraction, integrity checks, backup, and restore. Acquired documents,
credentials, generated data, and binaries stay outside Git. Original source
metadata remains independent of editorial overrides.

## Project documents

- [Anarchist Zine Archive milestone](docs/plans/0001-anarchist-zine-archive.md)
- [Managed operations, public release, and second archive](docs/plans/0002-managed-archive-and-public-release.md)
- [Pilot operations and recovery](docs/runbooks/pilot-operations.md)
- [Stack and development boundaries](docs/reference/stack.md)
- [Documentation index](docs/README.md)
- [Open Pilot](docs/open-pilot.md)

Source code is GPL-3.0-only; that license does not apply to acquired documents.

The public site uses `https://anarchist-archive.subcult.tv`, with correction and removal requests directed to `info@subcult.tv`. The full selected Anarchist source is public approved under its verified automatic policy; A-Info remains private. The [roadmap](docs/plans/0002-managed-archive-and-public-release.md) records access review, activation and qualification separately.
Managed job runners are documented in [the operations runbook](docs/runbooks/managed-operations.md).

## Reviewed public access

Private publication and public approval are separate decisions. `ARKHIV_PUBLIC=1`
starts the API with the variant allowlist enforced on collections, catalog, search,
facets, covers, reading and downloads. It does not run migrations; deploy migrations
with the operator first and use a separate read-only database role and read-only
archive mount. The private pilot API remains available for internal qualification.

The owner approved public archival access to the exact 25-document, 26-variant
manifest in `collections/anarchist-zines-public-review.json`. Rights remain unknown;
Anarchist Archive asserts no reuse license. The public URL is `https://anarchist-archive.subcult.tv` and
correction/removal contact is `info@subcult.tv`. Build-time `VITE_PUBLIC_URL` and
`VITE_CORRECTION_CONTACT` customize those values for another installation.

Apply a reviewed manifest with the operator command `approve-public --manifest PATH
--review "owner decision and evidence"`. It verifies source/document identity,
original filename, SHA-256, size, private publication and readable extraction before
committing the entire approval. `revoke-public --variant ID --review "reason"`
withdraws catalog, search, cover, reader and download access and preserves originals
and decision history. New variants start unapproved, including additions to a
privately published document. The source updater grants public access only when an explicit `verified-automatic` collection policy is configured and integrity/reader checks pass. Other collections retain individual approval; explicit revocations and withdrawals stay in force.

`web/nginx.public.conf` routes only to the dedicated public API and disables proxy
caching. Catalog and document responses use `Cache-Control: no-store`; approval is
checked before conditional or range handling. Copies already downloaded by a reader
cannot be recalled by revocation.

Source update jobs persist payload `Retry-After` deadlines and stop further source requests until that deadline. Expired payload backoff resumes pending acquisition before the next collection cadence. For a bounded operator qualification, `refresh --scheduled --force-source-check` bypasses cadence while retaining disabled-collection and source-backoff checks; ordinary scheduled runs keep each collection's configured cadence.

The public entry page includes Open Graph and Twitter summary metadata. Document canonical links and titles update in the browser; document-specific social previews are not server rendered. Public readiness monitoring checks the qualified revision and mode.

## Catalog design

The owner selected **A — Front Cover** from the [inspiration studies](docs/design/catalog-previews.md). The interface now displays real catalog metadata and local cover scans in the selected layout, with existing search, reading, downloads and rights information. The comparison prototypes are removed. The selected Front Cover frontend is deployed to the existing public service. The complete source qualification and rollback are recorded in the expansion report.

## Full Anarchist collection and automatic access

The owner authorized automatic archival reading and downloads for the Anarchist collection on October 3, 2026. Its configuration now selects all 700 unrestricted PDFs from the measured `ZineArchive` source, preserving the existing pilot identities; two existing variants share one document, for 699 selected document keys. Filename-derived titles for additions are provisional catalog labels, not researched bibliographic titles. Grouping similar filenames requires a separate editorial decision.

The full source PDFs total 3,510,725,119 bytes before OCR companions. Explicit collection limits are 1,000 documents, 8 GiB per acquisition batch, 250 MiB per file, two transfers at most, and a 2 GiB free-space reserve. Framework validation supports these broader bounds; other configured archives retain their own limits.

`approve-automatic --config PATH` applies the recorded policy to the exact collection/source item after original hashes and successful reader manifests are verified. It writes exact variant/hash decision manifests and events, asserts no reuse license, and respects revocations, document withdrawals and source holds/rejections. Repeat runs add no duplicate approval events. Scheduled source acquisition applies the same policy after extraction. A collection without this explicit policy continues to require individual public approval.

All 700 selected PDFs and 468 usable OCR companions are acquired and verified, totaling 3,541,315,927 bytes. All PDFs passed extraction; the catalog contains 12,732 pages. All 700 download hashes and first-page reader checks passed on the public site. The independent NUC snapshot passed file/dump verification and isolated catalog replay. The [full-source qualification report](docs/reports/full-anarchist-qualification-20261003.md) records those proof boundaries, rollback and operating limits. See [issue #52](https://git.subcult.tv/subculture-collective/arkhiv-alpha/issues/52) and the [expansion plan](docs/plans/0004-full-anarchist-collection.md).
