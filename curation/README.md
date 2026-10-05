# Arkhiv curation

Arkhiv owns this private discovery and review workbench. Its implementation was
copied from Kollektor at the revision recorded in [SOURCE.json](SOURCE.json),
including the original workflow tests, locked dependencies, and
[AGPL-3.0 license](LICENSE). It runs from this repository and requires no Kollektor
application, checkout, or service. The shared Go engine and branded frontend
repositories keep their existing ownership.

The workbench stores local collections, saved searches, query-bound discovery
checkpoints, matching-source provenance, review decisions, and notes. Discovery
is bounded to three 100-item pages per run. It preserves decisions across
rediscovery and restarts. Accepted selections export as JSON, CSV, or an inquiry
draft. The combined-catalog view retains selected print, audio, and video
representations and exports reviewed `arkhiv.curation.v1` drafts.

## Development and verification

Use Python 3.12 or newer and uv 0.12.22. From the repository root:

```sh
make verify-curation
cd curation
uv run --frozen python -m arkhiv_curation
```

Open `http://127.0.0.1:18764`. State defaults to `curation/data/curation.sqlite`
when launched from the `curation` directory. `--no-discovery` disables the worker
for an isolated UI preview. Tests use temporary SQLite databases and controlled
Archive/core responses. They do not fetch live media.

`ARKHIV_CURATION_DB` selects a workspace database. Set
`ARKHIV_CURATION_CORE_URL` to the Arkhiv API base URL for the instance being
curated. An optional `ARKHIV_CURATION_CORE_TOKEN` authenticates to a protected
gateway. Catalog requests are read-only; a local review decision does not grant
core publication, playback, download, or acquisition permission.

## Container and private access

```sh
docker build --target verify -f curation/Dockerfile curation
docker build --target runtime -t arkhiv-curation:REVIEWED_REVISION -f curation/Dockerfile curation
ARKHIV_CURATION_IMAGE=arkhiv-curation:REVIEWED_REVISION \
  docker compose -f deploy/curation-compose.yml up -d
```

Select an immutable image for release deployments. The Compose configuration
binds to loopback, uses a dedicated data volume, and runs as an unprivileged user
with a read-only image. Remote access belongs behind the existing authenticated
proxy, preserving the original Host header. This workbench has no public login
flow and is not part of the public catalog listener.

The headless core image contains no curation UI or Python runtime. Engine-only
consumers can continue to upgrade the Go engine independently; the curation
artifact is built and verified separately from the same maintained repository.

## Backup, restore, and existing Kollektor workspaces

The recovery command uses SQLite's online backup API, verifies database integrity
and foreign keys, and publishes a new file without overwriting an existing path.
It retains all tables, including accepted/rejected items, notes, search progress,
run history, and combined-catalog attachments.

```sh
cd curation
uv run --frozen python -m arkhiv_curation.recovery data/curation.sqlite backups/reviewed-checkpoint.sqlite
uv run --frozen python -m arkhiv_curation.recovery backups/reviewed-checkpoint.sqlite data/restored.sqlite
ARKHIV_CURATION_DB=data/restored.sqlite uv run --frozen python -m arkhiv_curation
```

Use the same command to copy an existing Kollektor SQLite workspace into a new
Arkhiv-owned path. Preserve the original workspace and its service. The schema
is retained, and the command includes committed WAL data from a live database.
Inspect the returned checksum and table counts, then verify the recovered
collections, decisions, exports, and catalog connection before selecting that
workspace. A database copy does not start discovery or change public approval.

For an image rollback, retain the prior image and select it with a compatible
checkpoint in a separate database path. Do not replace newer review decisions
with an older checkpoint.
