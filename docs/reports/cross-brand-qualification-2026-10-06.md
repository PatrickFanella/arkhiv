# Cross-brand qualification — October 6, 2026

This supplements the retained October 3 inventory with the actual published viewer 0.4.1, core 0.3.0 and reference engine 0.4.0. The SDK-only patch does not imply another engine image. Machine-readable current source heads and qualification boundaries are in `docs/reference/platform-consumers.json` under `current_release_qualification`; prior source/license/runtime evidence remains intact.

| Consumer | Media | Ready adoption PR | SDK / core | Source gate |
|---|---|---|---|---|
| bill-nye-tv | video | [PR](https://git.subcult.tv/subculture-collective/bill-nye-tv/pulls/14) | 0.4.1 / 0.3.0 | Passed |
| democracy-now-tv | video | [PR](https://git.subcult.tv/subculture-collective/democracy-now-tv/pulls/14) | 0.4.1 / 0.3.0 | Passed |
| video-archive-template | video | [PR](https://git.subcult.tv/subculture-collective/video-archive-template/pulls/13) | 0.4.1 / 0.3.0 | Passed |
| dedhed | audio | [PR](https://git.subcult.tv/subculture-collective/dedhed/pulls/35) | 0.4.1 / 0.3.0 | Passed |
| arkhiv-anarchist | print | [PR](https://git.subcult.tv/subculture-collective/arkhiv-anarchist/pulls/64) | 0.4.1 / 0.3.0 | Passed |
| arkhiv-riot-grrrl | print | [PR](https://git.subcult.tv/subculture-collective/arkhiv-riot-grrrl/pulls/15) | 0.4.1 / 0.3.0 | Passed |

The published registry tarball checksum matched an independent download. Six frozen-lock consumer gates passed with independent SDK/core pins, brand-owned markup/styles and disposable database integration where required. Root make verify and make verify-curation passed, including 42 SDK tests, packed ESM composition, backend race/database checks and 19 frozen curation tests. The new upstream changes include separate exact-program audio chat, preserved native metadata filters, and temporary-favorite storage warnings; consumer source adoption is reviewable in the listed PRs. Existing default-branch merges and this newer unmerged adoption are distinct.

Actual browser evidence covers generated video playback, metadata/library controls, grid/table persistence, local favorite recovery warnings, media-separated audio chat and moderation, real source-image loading/fallback, volume/mute remount/boundary/reload behavior and curation export/draft nonpublication. Reports are `issue-audit-2026-10-06.md`, `volume-qualification-2026-10-06.md`, `artwork-qualification-2026-10-06.md` and `curation-qualification-2026-10-06.md`. Product-specific receipts live in their own repositories.

Real retained print/audio conversion and independent restores are documented in the October 5 release qualification and retained-backend migration report; that proof does not activate the candidate backend. Almaz's natural October 5 native public replay/backup jobs recorded success (Arkhiv replay 19:10→19:10:25, backup 19:30→19:33:48; Dedhed replay 19:25→19:25:07, backup 19:45→19:45:07). Legacy print backup/audit/retention also ran successfully. Shared-core natural maintenance, a sustained collection-wide soak and human complete-concert segue listening remain separate outstanding evidence.

Hosted CI is explicitly owner-waived while Kvant is down. Device discovery found no iOS platform (macOS/Xcode unavailable) or Android platform (SDK unavailable), so physical/device qualification remains open. Synthetic fixture decoding cannot establish real source audiovisual fidelity. Browser pausing when muted media leaves view keeps suspension and two-viewer continuous drift qualification open under #108. No viewer control writes a shared schedule. Democracy Now! remains metadata-only/source-streaming.

Issue #51 remains open for those unavailable/unfinished proof tiers; #98 owns the separately reviewed native backend cutover. Anarchist #56 owns measured backup capacity, deduplicated store/restore qualification and separate activation. The updated release support notes distinguish SDK-only updates from platform/core releases and retain current/previous immutable receipts.

## Later SDK 0.4.2 and merged-adoption update

All earlier adoption and bounded-backup PRs merged. SDK 0.4.2 was published from
1619d86a615bf4bd69c7a601bf585ff5b3a6ef89 and its registry download matched
6eaf356a37bfcde823227f61f0fe42a50385dde4502efdd35eab7ecd675bb04d.
Root gates passed with 45 SDK tests. Six frozen-lock consumer gates passed;
incoming native favorite-warning and bounded-backup changes were preserved
and the affected native gates repeated successfully. Video SDK adoption reached
the merged PRs; audio/print use ready follow-ups Dedhed #36, Anarchist #66 and
Riot Grrrl #17. Library-proof follow-ups are Bill #15, Democracy Now #15 and
template #14. Reports library-qualification-2026-10-06.md and
channel-continuity-2026-10-06.md distinguish completed library acceptance from
unqualified continuous two-viewer playback. The earlier source-head inventory
is retained under release_qualification_history.
