# Streaming player and shared channel — October 3, 2026

The private player resolves approved source-hosted MP4s directly to Internet
Archive. One video element owns the active on-demand or scheduled session.
Scheduled replay uses server time plus measured request latency; polling refreshes
Now/Next, replaces ended programs, and corrects drift. Pause is local; Rejoin seeks
to the current shared position. Dates and replay labels remain visible.

The isolated `t3-dev --profile vhness verify` gate passed Go formatting, vet,
race tests with disposable PostgreSQL integration, binary build, frozen pnpm
installation, TypeScript, ESLint, four frontend model tests, and production build.
The frontend tests cover elapsed clock offset, duration clamps, asset identity,
and permitted source/local URL boundaries.

The loopback private pilot on Kvant stores two reviewed real episodes,
`dn2026-1002_vid` and `dn2026-1001_vid`, with complete source responses and file
inventories. Both are source-streaming only, with no acquisition or public release
permission. The first bounded discovery run found 6,626 source items, queued
1,000, imported 98 and retained two failed/rejected outcomes. This is partial
inventory evidence; it does not prove complete collection coverage.

API readiness passed. Native browser navigation now returns a blank
`chrome-error://chromewebdata/` page despite HTTP readiness. Repeated corrected
navigation to the preview and private pilot did not recover it. Actual remote
video rendering, seeking, audiovisual fidelity, mobile layout, and two-session
channel synchronization remain unqualified. No real video has been downloaded.

Runtime state and credentials live outside Git under
`~/.local/state/vhness/private-pilot`. `scripts/pilot.py status` identifies the
owned API, and `stop` terminates only its recorded executable while preserving
the PostgreSQL volume. Public deployment is outside this change.
