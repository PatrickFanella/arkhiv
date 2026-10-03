# VHNESS roadmap

Status: implementation and private qualification in progress. Metadata, source
streaming, shared schedules, and manifest-bounded local preservation are
implemented in the review stack. Complete collection coverage and sustained
qualification remain open. No public deployment is authorized. Updated 2026-10-03.

The owner subsequently requested three deployable products: the configurable
white-label app plus both branded apps. Bill Nye preservation now covers all
100 source MP4s, 13,763,456,996 bytes, with an explicit 16 GiB reviewed budget,
the existing 1 GiB reserve, and independent NUC recovery. The original two-file
pilot described below remains historical evidence. Source-only Democracy Now!
policy is unchanged. See [deployment operations](docs/DEPLOYMENT.md).

The owner selected separate Democracy Now! and Bill Nye app repositories with
a shared engine. Each app owns its design system and deployment; VHNESS naming
and visual design remain deferred. The [standalone product plan](docs/plans/0002-standalone-products.md)
records runtime isolation, asset intake, visual directions, and delivery gates.
This extends the pilot roadmap without expanding either delivery policy.

VHNESS starts with Democracy Now! on Internet Archive. The first private version
includes a browsable episode library, on-demand viewing, and a shared scheduled
replay channel. It stores complete metadata and streams video directly from the
source. Democracy Now! videos are never acquired or cached by VHNESS.

The owner activated the optional preservation pilot for two Bill Nye source MP4s:
01 Flight and 02 Earth's Crust, totaling 274,780,857 bytes. The approved limit is
512 MiB with a 1 GiB free-space reserve, stored below
~/.local/state/vhness/private-pilot/media/bill-nye and backed up independently
on NUC below /home/onnwee/Backups/vhness-private-pilot. The selection does not
change Democracy Now!'s streaming-only policy. See the dated preservation report
for measured evidence and remaining qualification limits.

[Roadmap tracking issue #2](https://git.subcult.tv/subculture-collective/VHNESS/issues/2) · [All issues](https://git.subcult.tv/subculture-collective/VHNESS/issues) · [Milestones](https://git.subcult.tv/subculture-collective/VHNESS/milestones)

The [design plan](docs/plans/0001-video-archive-and-channel.md) owns product and
architecture scope. Gitea issues own executable task state, acceptance criteria,
and native dependencies. This roadmap preserves milestone ordering and exit
gates; it does not duplicate mutable issue checklists or claim implementation.

## Milestones

| Milestone | Result and exit gate |
| --- | --- |
| [M1 — Metadata foundation](https://git.subcult.tv/subculture-collective/VHNESS/milestone/101) | Store complete source metadata and history; discover and refresh Democracy Now! resumably without requesting media payloads. Exit: fixture-backed source identity, response fidelity, resume, editorial separation, and backoff checks. |
| [M2 — Source-streaming episode library](https://git.subcult.tv/subculture-collective/VHNESS/milestone/102) | Browse Democracy Now! episodes and stream directly from Internet Archive. Exit: real browser playback and seeking, visible source identity, failure states, and proof that no video is stored by VHNESS. |
| [M3 — Shared scheduled replay channel](https://git.subcult.tv/subculture-collective/VHNESS/milestone/103) | Persist a shared timeline alongside independent on-demand playback. Exit: measured mid-program joins, program transitions, safe switching, and explicit replay/unavailable states. |
| [M4 — Democracy Now! private qualification](https://git.subcult.tv/subculture-collective/VHNESS/milestone/104) | Complete collection-wide metadata discovery, independent recovery, operational visibility, and a private streaming/channel run. Exit: full-episode viewing, program transition, metadata restore, and no-local-video evidence. |
| [M5 — Optional local preservation](https://git.subcult.tv/subculture-collective/VHNESS/milestone/105) | Add a local-delivery adapter for another explicitly selected collection. Activation requires a reviewed sample, destination, byte budget, and free-space reserve. Exit: bounded acquisition, local playback through both modes, and independent restore. Does not block the Democracy Now! streaming pilot. |

No release dates are committed. M1–M4 deliver and qualify the Democracy Now!
streaming pilot. M5 is a separately activated local-preservation path. Native
issue prerequisites control task ordering; tasks with no mutual prerequisites
can proceed independently.

## M1 — Metadata foundation

| Issue | Prerequisites |
| --- | --- |
| [#3 — Define metadata, delivery-policy, and playback contracts](https://git.subcult.tv/subculture-collective/VHNESS/issues/3) | None |
| [#4 — Bootstrap Go/React/PostgreSQL and isolated development checks](https://git.subcult.tv/subculture-collective/VHNESS/issues/4) | [#3](https://git.subcult.tv/subculture-collective/VHNESS/issues/3) |
| [#5 — Persist complete metadata snapshots, file inventories, and review history](https://git.subcult.tv/subculture-collective/VHNESS/issues/5) | [#3](https://git.subcult.tv/subculture-collective/VHNESS/issues/3), [#4](https://git.subcult.tv/subculture-collective/VHNESS/issues/4) |
| [#6 — Discover Democracy Now! episodes in bounded resumable batches](https://git.subcult.tv/subculture-collective/VHNESS/issues/6) | [#5](https://git.subcult.tv/subculture-collective/VHNESS/issues/5) |
| [#7 — Refresh metadata with source history and persistent backoff](https://git.subcult.tv/subculture-collective/VHNESS/issues/7) | [#6](https://git.subcult.tv/subculture-collective/VHNESS/issues/6) |

## M2 — Source-streaming episode library

| Issue | Prerequisites |
| --- | --- |
| [#8 — Resolve reviewed source-hosted playback URLs without proxying video](https://git.subcult.tv/subculture-collective/VHNESS/issues/8) | [#5](https://git.subcult.tv/subculture-collective/VHNESS/issues/5) |
| [#9 — Prove direct source playback, seeking, and browser compatibility](https://git.subcult.tv/subculture-collective/VHNESS/issues/9) | [#8](https://git.subcult.tv/subculture-collective/VHNESS/issues/8), [#4](https://git.subcult.tv/subculture-collective/VHNESS/issues/4) |
| [#10 — Build the dated episode library and on-demand player](https://git.subcult.tv/subculture-collective/VHNESS/issues/10) | [#8](https://git.subcult.tv/subculture-collective/VHNESS/issues/8), [#4](https://git.subcult.tv/subculture-collective/VHNESS/issues/4) |

## M3 — Shared scheduled replay channel

| Issue | Prerequisites |
| --- | --- |
| [#11 — Persist channel schedules and expose server-time now/next](https://git.subcult.tv/subculture-collective/VHNESS/issues/11) | [#3](https://git.subcult.tv/subculture-collective/VHNESS/issues/3), [#8](https://git.subcult.tv/subculture-collective/VHNESS/issues/8) |
| [#12 — Implement Tune in and safe switching between viewing modes](https://git.subcult.tv/subculture-collective/VHNESS/issues/12) | [#11](https://git.subcult.tv/subculture-collective/VHNESS/issues/11), [#10](https://git.subcult.tv/subculture-collective/VHNESS/issues/10), [#9](https://git.subcult.tv/subculture-collective/VHNESS/issues/9) |
| [#13 — Qualify synchronized joins, transitions, and channel failures](https://git.subcult.tv/subculture-collective/VHNESS/issues/13) | [#12](https://git.subcult.tv/subculture-collective/VHNESS/issues/12) |

## M4 — Democracy Now! private qualification

| Issue | Prerequisites |
| --- | --- |
| [#14 — Complete collection-wide metadata inventory and reconciliation](https://git.subcult.tv/subculture-collective/VHNESS/issues/14) | [#7](https://git.subcult.tv/subculture-collective/VHNESS/issues/7) |
| [#15 — Back up and independently restore metadata, decisions, and schedules](https://git.subcult.tv/subculture-collective/VHNESS/issues/15) | [#5](https://git.subcult.tv/subculture-collective/VHNESS/issues/5), [#11](https://git.subcult.tv/subculture-collective/VHNESS/issues/11) |
| [#16 — Add metadata-worker and streaming-channel operational visibility](https://git.subcult.tv/subculture-collective/VHNESS/issues/16) | [#7](https://git.subcult.tv/subculture-collective/VHNESS/issues/7), [#11](https://git.subcult.tv/subculture-collective/VHNESS/issues/11) |
| [#17 — Qualify the Democracy Now! private streaming and channel pilot](https://git.subcult.tv/subculture-collective/VHNESS/issues/17) | [#14](https://git.subcult.tv/subculture-collective/VHNESS/issues/14), [#15](https://git.subcult.tv/subculture-collective/VHNESS/issues/15), [#16](https://git.subcult.tv/subculture-collective/VHNESS/issues/16), [#10](https://git.subcult.tv/subculture-collective/VHNESS/issues/10), [#9](https://git.subcult.tv/subculture-collective/VHNESS/issues/9), [#13](https://git.subcult.tv/subculture-collective/VHNESS/issues/13) |

## M5 — Optional local preservation

| Issue | Prerequisites |
| --- | --- |
| [#18 — Select a local-preservation sample, destination, and storage budget](https://git.subcult.tv/subculture-collective/VHNESS/issues/18) | [#17](https://git.subcult.tv/subculture-collective/VHNESS/issues/17) |
| [#19 — Implement manifest-bounded acquisition with quotas and integrity](https://git.subcult.tv/subculture-collective/VHNESS/issues/19) | [#18](https://git.subcult.tv/subculture-collective/VHNESS/issues/18), [#5](https://git.subcult.tv/subculture-collective/VHNESS/issues/5) |
| [#20 — Serve verified local video through the common playback adapter](https://git.subcult.tv/subculture-collective/VHNESS/issues/20) | [#19](https://git.subcult.tv/subculture-collective/VHNESS/issues/19), [#8](https://git.subcult.tv/subculture-collective/VHNESS/issues/8), [#11](https://git.subcult.tv/subculture-collective/VHNESS/issues/11) |
| [#21 — Qualify local playback, integrity, and independent media recovery](https://git.subcult.tv/subculture-collective/VHNESS/issues/21) | [#20](https://git.subcult.tv/subculture-collective/VHNESS/issues/20), [#15](https://git.subcult.tv/subculture-collective/VHNESS/issues/15) |

## After private qualification

[#22](https://git.subcult.tv/subculture-collective/VHNESS/issues/22) records the public-release decision: approved source use, audience,
hosting, operational limits, and outstanding rollout work. It does not authorize
or perform a deployment. Optional local preservation is not a prerequisite for
deciding whether to release the streaming product.

## Confirmed boundaries

- Both on-demand viewing and a shared scheduled channel are required.
- Delivery policy is per collection and independent of viewing mode.
- Democracy Now! is source-streaming only; browser buffering does not authorize
  an application-managed persistent media cache or download queue.
- Complete source responses, every file entry, unknown fields, history, editorial
  decisions, refresh state, and schedule revisions are stored locally.
- Metadata discovery covers the full selected collection without a permanent
  date/topic filter. Initial playback verification uses a bounded sample.
- Source disappearance preserves the stored catalog but can make streaming
  unavailable. Metadata recovery cannot recover missing source video.
- Original broadcast dates, source attribution, and replay labels remain visible.
- Viewer controls never change the shared timeline. Gaps, missing programs,
  autoplay restrictions, and exhausted schedules must be explicit.
- Local acquisition requires an exact manifest, quotas, free-space reserve,
  verified integrity, and independent recovery. It is prohibited for Democracy Now!
- Accounts, uploads, personal playlists, multiple channels, offline playback, and
  continuous encoded broadcast output are deferred.

## Evidence and maintenance

Keep issues open until their required acceptance criteria have receipts. Local
checks, hosted CI, rendered UI, real audiovisual playback, measured channel drift,
sustained viewing, and independent recovery are distinct proof boundaries.

Update this roadmap and the design plan when milestone scope changes. Update
the tracking issue when work is qualified; do not close tasks merely because
their code merged. No live discovery job, scheduler, acquisition, or public
service is activated by creating these planning records.
