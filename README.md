# Arkhiv

Arkhiv acquires explicitly selected document files, preserves their original bytes,
and serves branded catalogs with local PDF reading and downloads. The first
collection is the Anarchist Zine Archive.

The implementation includes a PostgreSQL catalog, operator CLI, bounded resumable
acquisition, editorial review, text and cover extraction, metadata/page search,
and backup/restore. The initial manifest selects 25 documents, 26 PDF variants,
and 17 matching OCR companions from Internet Archive's `ZineArchive` item.
The [private pilot qualification](docs/reports/pilot-qualification-20261002.md) passed:
all originals were verified, the independent NUC backup was restored, and on-site
reading/search/downloads worked with Archive.org blocked. The initial implementation merged in PR #19; hardening and the qualification
report merged in [PR #20](https://git.subcult.tv/subculture-collective/arkhiv/pulls/20).
Public release is a separate decision.

The [next delivery plan](docs/plans/0002-managed-archive-and-public-release.md)
covers merged-revision rollout, scheduled preservation and active-source updates,
public release, and a second real archive. Those jobs and release steps are planned
in Gitea issues #21–#38; they are not activated by the plan.

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

The public release will use `https://arkhiv.subcult.tv`, with correction and removal
requests directed to `info@subcult.tv`. Public access remains gated by the reviewed
allowlist work in [the roadmap](docs/plans/0002-managed-archive-and-public-release.md).
Managed job runners are documented in [the operations runbook](docs/runbooks/managed-operations.md).

## Reviewed public access

Private publication and public approval are separate decisions. `ARKHIV_PUBLIC=1`
starts the API with the variant allowlist enforced on collections, catalog, search,
facets, covers, reading and downloads. It does not run migrations; deploy migrations
with the operator first and use a separate read-only database role and read-only
archive mount. The private pilot API remains available for internal qualification.

The owner approved public archival access to the exact 25-document, 26-variant
manifest in `collections/anarchist-zines-public-review.json`. Rights remain unknown;
Arkhiv asserts no reuse license. The public URL is `https://arkhiv.subcult.tv` and
correction/removal contact is `info@subcult.tv`. Build-time `VITE_PUBLIC_URL` and
`VITE_CORRECTION_CONTACT` customize those values for another installation.

Apply a reviewed manifest with the operator command `approve-public --manifest PATH
--review "owner decision and evidence"`. It verifies source/document identity,
original filename, SHA-256, size, private publication and readable extraction before
committing the entire approval. `revoke-public --variant ID --review "reason"`
withdraws catalog, search, cover, reader and download access and preserves originals
and decision history. New variants start unapproved, including additions to a
privately published document. The source updater never grants public access.

`web/nginx.public.conf` routes only to the dedicated public API and disables proxy
caching. Catalog and document responses use `Cache-Control: no-store`; approval is
checked before conditional or range handling. Copies already downloaded by a reader
cannot be recalled by revocation.

Source update jobs persist payload `Retry-After` deadlines and stop further source requests until that deadline. Expired payload backoff resumes pending acquisition before the next collection cadence. For a bounded operator qualification, `refresh --scheduled --force-source-check` bypasses cadence while retaining disabled-collection and source-backoff checks; ordinary scheduled runs keep each collection's configured cadence.

The public entry page includes Open Graph and Twitter summary metadata. Document canonical links and titles update in the browser; document-specific social previews are not server rendered. Public readiness monitoring checks the qualified revision and mode.
