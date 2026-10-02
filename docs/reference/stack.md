# Stack reference

Arkhiv uses the template's Go, React, Vite, TypeScript, Tailwind, PostgreSQL, and
Docker stack. Bootstrap changes do not implement the archive milestone.

| Boundary | Current behavior |
| --- | --- |
| Go service | Standard-library HTTP server, configurable `API_ADDR` (default `:8080`), graceful shutdown |
| Health | `GET /api/health`, JSON identifying Arkhiv and bootstrap stage; process health only |
| Web | Collection preparation page, connection/loading/failure status |
| API origin | Same-origin `/api`; Vite forwards to `127.0.0.1:8080`, Nginx to Compose `api:8080` |
| PostgreSQL | Version 17, disposable T3 database; not connected to application code |
| Migrations | Inherited `0001_init.sql` only; runner and catalog schema are milestone work |
| Storage | No document storage or acquired payloads yet |
| Verification | `make verify`; no live integrations required |
| CI | Gitea workflow installs Go, Node 24, pnpm 10.33.0 and runs `make verify` |

Go builds write to ignored `backend/bin/`. Frontend installs use the committed
lockfile. `make fmt` edits Go formatting; `make verify` only checks it. `make deps`
installs web dependencies; the current Go service has no external dependencies.

The T3 profile is authored in `~/Work/Tools/t3-dev-environments/profiles.json` and
installed on Kvant. It uses an isolated source snapshot, caches, loopback proxy,
and PostgreSQL volume. Test has separate storage and no live-provider network.
Its Check Code action executes the complete repository gate, including Compose
syntax validation without requiring a mounted Docker socket. No archive download
or database-backed request is implied by a green health response.
