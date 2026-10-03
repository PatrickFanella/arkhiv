# Metadata foundation evidence — 2026-10-03

Host: Kvant. Source: `/home/onnwee/.t3/worktrees/vhness/t3code-8feb0f6c`.
Issues: #3–#7 and the source resolution contract for #8.

The first `t3-dev --profile vhness verify` run found a timestamp assertion that
compared nanoseconds against PostgreSQL microsecond precision. The controlled
clock now uses microsecond precision. The second run passed formatting, vet,
Go race tests including disposable PostgreSQL catalog/worker integration, binary
build, frozen dependencies, TypeScript, lint, and web build. There are currently
no frontend unit tests; the web runner explicitly permits that empty suite.

Integration evidence covers exact raw response bytes, retained unknown file
fields, source history, editorial separation, changed-migration rejection,
Democracy Now! policy protection at both domain and SQL boundaries, queue
deduplication, schedule persistence, and a two-hour source backoff retained
across worker restart. Source-client tests reject media endpoints, unsafe
redirects, and oversized responses. They do not prove a complete live inventory.

The isolated preview started with three labeled synthetic metadata records and
passed database readiness. Desktop collaborative-browser inspection rendered
the catalog; searching for episode 02 returned exactly one fixture. Browser
resize timed out and subsequent native preview navigation displayed a browser
error despite successful loopback readiness checks. Narrow-viewport proof remains
unverified; this is not an accepted browser check.

The shared launcher gained a VHNESS profile through its authoring repository,
in an isolated worktree. Eleven launcher tests passed. Before deploying the
profile and guide, installed runtime files and existing profile entries matched
the source baseline. Only the profile and guide were replaced, with before-state
backups. Ten VHNESS project actions were read back from the installer result.
Other active preview resources were preserved.

Live metadata acquisition, complete source coverage, actual video playback,
shared-channel browser timing, sustained pilot qualification, independent backup,
and optional local preservation remain open. No Democracy Now! video has been
acquired or stored.
