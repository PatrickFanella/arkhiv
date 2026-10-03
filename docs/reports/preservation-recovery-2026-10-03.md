# Preservation and recovery evidence — October 3, 2026

The optional adapter now has manifest-bounded acquisition, quota/reserve checks,
recoverable partials, strict source identity/range validation, SHA-256/source-MD5
verification, immutable original promotion, catalog lineage, and integrity audits.
Source metadata workers remain unable to request video, and Democracy Now! is
rejected by acquisition policy. The existing playback and schedule contracts also
resolve reviewed local originals. HTTP serving verifies access before Range or
conditional responses and rejects missing/corrupt originals and path escape.

Controlled tests passed for completed transfer/idempotence, interruption and
resume, changed manifest, invalid range, prohibited streaming policy, byte budget,
free-space reserve before/during transfer, checksum mismatch, source-MD5 mismatch,
and root escape. Disposable PostgreSQL/HTTP integration passed local byte ranges,
ETags, review withdrawal, shared clock resolution, and corruption rejection.
The full isolated gate also passed Go formatting/vet/race/build, TypeScript/lint,
frozen pnpm dependencies, five frontend model tests, and production build.

The generated fixture collection is `vhness-local-fixture`. It holds two
12-second 320×180 H.264/AAC originals of 316,925 and 316,834 bytes, under a 10 MiB
budget with 64 MiB free-space reserve. Their SHA-256 values are:

- `424a96e10ce8ea66b894c343f95b3d87178f14b8bdf61816d3c8938abb00eb26`
- `c954d2a087e793b5c5771728e6eb3d76695383899fa2052088e7f5d1f9c42e9e`

A live local HTTP request returned 206 with exactly the requested 16 bytes. These
are generated adapter fixtures. This initial receipt preceded activation of the
owner-approved Bill Nye sample described below.

Independent recovery receipt `20261003T041215Z` on NUC restored a 4,697,101-byte
PostgreSQL dump with SHA-256
`eb24e85ab60ecc0fc8b5a075436a79bb07ed73ca258686bc1b39430cdb37859c`.
All 11 table fingerprints matched the isolated local baseline; 196 raw snapshots
passed exact byte/hash validation. The preserved media archive had SHA-256
`7175578edc7ccd4a55a3d47b0f3076cfc2b1f199d9c4c173b8336d63c9989160`.
Both extracted independent originals matched their recorded SHA-256 values.
Evidence is under `~/.local/state/vhness/private-pilot/backups/20261003T041215Z`
and `/home/onnwee/Backups/vhness-private-pilot/20261003T041215Z` on NUC.
This receipt captures partial item metadata with the full 6,626-identity queue;
it does not prove complete item metadata coverage.

The complete identifier inventory is discovered. At 04:17 UTC, 241 queue items
were imported, two explicitly rejected, and 6,383 remained pending. A bounded
metadata-only job is active and retains checkpoints/backoff. Coverage remains open
until every discovered item has a retained outcome and the completion restore
receipt passes. Worker interruption/resume retained its queue and metadata.
Search response bytes and 304 observations are now retained as well.

At the initial native-browser check, local navigation failed with `net::ERR_CONNECTION_REFUSED` while
API readiness and local HTTP checks pass. External Archive pages render. The owner authorized a separate local browser afterward. The Bill Nye
qualification below records actual playback, seeking, synchronized transitions,
source-blocked rendering, and independently restored application playback.
Continuous full-episode source viewing remains a separate check. Optional remux/transcode and
caption-sidecar acquisition are not implemented; compatible originals need none
for the generated sample. The owner subsequently selected the Bill Nye archive and approved two source MP4s, a 512 MiB budget, 1 GiB reserve, the pilot media directory, and independent NUC backup. See [the Bill Nye qualification report](bill-nye-private-2026-10-03.md) for
acquisition and measured recovery evidence.

VHNESS foundation CI runs 12116/12117 and streaming-player CI runs 12120/12121
passed on their exact PR heads. The preservation head requires its own hosted CI
receipt. Public release remains outside this private qualification.
