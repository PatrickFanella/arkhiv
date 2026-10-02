# Arkhiv

A framework for preserving document collections and publishing branded archives
with an on-site reader. The first planned collection is the Anarchist Zine Archive.

**Current status: bootstrapped, not an operational document archive.** The Go API
serves process health; the React frontend shows the collection preparation page.
Acquisition, catalog persistence, search, and document reading are milestone 1 work.

## Development

Stack: Go 1.26+, Node 24, pnpm 10.33.0, React/Vite, PostgreSQL 17, Docker Compose.
Install the pinned pnpm version rather than regenerating its lockfile with a
new major version.

On a configured T3 host, from this checkout:

```sh
t3-dev --profile arkhiv start
t3-dev --profile arkhiv verify
t3-dev --profile arkhiv sync
t3-dev --profile arkhiv stop
```

The launcher copies the source into an isolated environment. Run `sync` after
edits, or use its Watch Changes action. See the installed
[agent guide](/home/onnwee/.local/share/t3-dev-environments/AGENT-GUIDE.md).

For manual development with the toolchains installed:

```sh
make deps
make dev-api # terminal 1, port 8080
make dev-web # terminal 2, port 5173; proxies /api to Go
make verify
```

For the containerized build, `docker compose up --build` serves the site on
`http://127.0.0.1:5173`. PostgreSQL and the API also bind to loopback. The sample
password is for disposable local development. PostgreSQL exists in Compose but
is not used by the application yet. No migrations run automatically.

`GET /api/health` confirms the process responds; it does not assert database or
archive readiness. The Vite and Nginx configurations both proxy this route.

## Project documents

- [Milestone 1: Anarchist Zine Archive](docs/plans/0001-anarchist-zine-archive.md)
- [Stack and development boundaries](docs/reference/stack.md)
- [Documentation index](docs/README.md)
- [Open Pilot](docs/open-pilot.md)

The inherited Open Pilot issue and pull-request templates are retained. The
canonical remote is `https://git.subcult.tv/subculture-collective/arkhiv.git`.
Source code is GPL-3.0-only; this does not license acquired archive documents.
