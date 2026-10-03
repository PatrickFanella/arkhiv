# Development and verification

Use profile `vhness` from the current worktree after reading the shared
`~/.local/share/t3-dev-environments/AGENT-GUIDE.md`.

```sh
t3-dev --profile vhness start
t3-dev --profile vhness sync
t3-dev --profile vhness verify
t3-dev --profile vhness test --command 'make test-integration'
t3-dev --profile vhness stop
```

The isolated preview seeds three explicitly synthetic metadata records and stores
no video. Application/test networks cannot reach Internet Archive. The preview
database is retained; tests use a separate disposable PostgreSQL database and
temporary schemas. Do not point `TEST_DATABASE_URL` at preview or production.

`make verify` checks Go formatting, vet, race tests, binary build, frozen web
dependencies, TypeScript, lint, web tests, and production build. The initial
catalog UI has no frontend unit tests; browser proof is a separate gate. Database
tests report a skip if `TEST_DATABASE_URL` is absent. Explicit integration refuses
an absent URL. Actual playback, live source discovery, source failure, independent
restore, hosted CI, and complete pilot qualification require separate receipts.

The CLI requires `DATABASE_URL`. Run it under `backend/`; migration paths default
to `../migrations`. `VHNESS_MIGRATIONS` can select the explicit migration directory.

```sh
go run ./cmd/vhness migrate
go run ./cmd/vhness collection --collection democracy_now_vid --policy source_stream
go run ./cmd/vhness discover --collection democracy_now_vid --pages 1 --items 20 --timeout 10m
go run ./cmd/vhness report
```

Live discovery runs only in a separately configured private operator environment,
outside the isolated preview. Bounds limit one invocation; repeat invocations
resume persisted progress and backoff. `refresh` resets discovery for reconciliation
and queues a bounded known-item batch, retaining source and decision history.
`import`, `review`, and `schedule` accept exact JSON input files. Operator mutations
are CLI-only; web APIs are read-only. `API_ADDR` defaults to loopback port 8080.

Raw source responses, observations, every file inventory, decisions, and schedules
remain in PostgreSQL. Metadata workers cannot request download endpoints, and
streaming playback resolves a browser-to-source URL. No media proxy is provided.
