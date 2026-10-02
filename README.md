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
reading/search/downloads worked with Archive.org blocked. Implementation is in
[PR #19](https://git.subcult.tv/subculture-collective/arkhiv/pulls/19), pending review
and merge. Public release is a separate decision.

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
- [Pilot operations and recovery](docs/runbooks/pilot-operations.md)
- [Stack and development boundaries](docs/reference/stack.md)
- [Documentation index](docs/README.md)
- [Open Pilot](docs/open-pilot.md)

Source code is GPL-3.0-only; that license does not apply to acquired documents.
