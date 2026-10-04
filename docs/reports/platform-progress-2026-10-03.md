# Arkhiv platform implementation progress

Observed 2026-10-03. VHNESS remains the canonical repository for Arkhiv. Brands
own distinct interfaces built from shared components. This is an implementation
record; it does not claim that print/audio integration or public cutover is done.

Merged PRs #54–#59 establish ownership contracts, pinned donor provenance,
compiled shared video packages, registry release tooling, consumer adoption and
checked upgrade proposals. PR #61 supplies the shared search feedback fix in
0.1.1. PR #62 adds hosted proposal automation and qualified release receipts.
Both video consumers adopted the package through merged PR #2 and received
0.1.1 upgrade PR #3 from the hosted workflow.

The [shared video qualification report](shared-video-upgrades-2026-10-03.md)
records successful hosted publication, actual registry downloads, clean installs,
consumer image builds, repeat-run proposal evidence and browser observations.
The [dated inventory](../reference/platform-consumers.json) preserves the initial
snapshot and a live print-runtime follow-up. Current Almaz labels identify the
public/private API revisions, archive mounts and active operator. System timers
and the independent NUC backup directory were read without changing services.
Public HTTP access and backup-byte qualification remain separate limits.

The broad device credential was not copied to CI. Separate package publish,
package read and repository proposal credentials are stored privately outside
Git. Existing public apps, previews, workers, timers, original archives and backup
owners remain active. The temporary video proof apps are isolated on Kuznya with
separate metadata databases and no acquisition workers or media mounts.

Continue in dependency order from the [platform roadmap](../plans/0003-multimedia-roadmap.md).
The bounded shared video propagation gate is qualified. PR #63 adds the scoped
metadata catalog and live video projection; full isolated verification and final
source-file checks passed. PR #64 adds exact representation transport reviews
and history, local hash/range checks and provenance-based source-streaming rules.
The print extraction adds typed companions, bounded PDF indexing and the shared
reader entry; disposable tests deny revoked or corrupt PDF/OCR derivatives.
Print PR #65 (`7ad16f4`) passed hosted push run 12560/job 22326 and pull-request
run 12561/job 22327, as well as full isolated verification. Real document/browser
qualification and release publication remain open. The audio extraction retains
the pinned Dedhed recording/track/credit contracts, validates continuous assets
separately from originals, and adds independent station revisions and shared
controls/hooks. Full isolated checks passed with generated tones, including
withdrawn future eligibility and corrupt-input/derivative guards. Real music and
browser qualification remain open. Next are the neutral mixed frontend,
An explicit multi-collection allowlist now covers both legacy and typed routes,
operator actions and complete future video schedules; mixed collection summaries
include print/audio membership without changing the legacy video summary shape.
Full isolated verification passed, including empty-scope and excluded/unknown
future-program regressions. The neutral mixed frontend remains next, followed by
all-brand adoption, migration/recovery and private cross-brand qualification.
Milestone 6 records a concrete public rollout decision after those gates; it does
not implicitly authorize deployment or additional media acquisition.
