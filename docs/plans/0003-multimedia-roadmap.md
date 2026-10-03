# Multi-media platform roadmap

Status: implementation underway, 2026-10-03. Arkhiv is the product name; VHNESS remains the canonical repository. P1 and P2 are qualified, and the initial P3 catalog is delivered in PR #63. Gitea issues own the remaining gates. Software changes do not imply production migration, acquisition or deployment.

[Qualified shared video update](../reports/shared-video-upgrades-2026-10-03.md) · [Current implementation record](../reports/platform-progress-2026-10-03.md)

[Design and ownership](0003-multimedia-platform.md) · [Tracking issue](https://git.subcult.tv/subculture-collective/VHNESS/issues/35) · [Milestones](https://git.subcult.tv/subculture-collective/VHNESS/milestones)

Gitea issues own executable state, acceptance criteria and native prerequisites. This document owns milestone order and exit gates. The original video roadmap remains in [ROADMAP.md](../../ROADMAP.md). No release dates are committed.

## P1 — Platform ownership and contracts

Inventory canonical sources and consumers; settle package, API, configuration and data boundaries without changing running products.

[Gitea milestone](https://git.subcult.tv/subculture-collective/VHNESS/milestone/106)

| Issue | Prerequisites |
| --- | --- |
| [#36 — Inventory canonical archive sources, consumers and deployment revisions](https://git.subcult.tv/subculture-collective/VHNESS/issues/36) | None |
| [#37 — Define multi-media domain, API and frontend ownership contracts](https://git.subcult.tv/subculture-collective/VHNESS/issues/37) | [#36](https://git.subcult.tv/subculture-collective/VHNESS/issues/36) |

## P2 — Shared releases and video update proof

Publish versioned shared components and engine artifacts; prove an upstream behavior fix reaches neutral and distinctive video frontends through automated upgrade PRs.

[Gitea milestone](https://git.subcult.tv/subculture-collective/VHNESS/milestone/107)

| Issue | Prerequisites |
| --- | --- |
| [#38 — Extract shared typed client, playback hooks and video components](https://git.subcult.tv/subculture-collective/VHNESS/issues/38) | [#37](https://git.subcult.tv/subculture-collective/VHNESS/issues/37) |
| [#39 — Publish compatible engine and frontend package releases](https://git.subcult.tv/subculture-collective/VHNESS/issues/39) | [#38](https://git.subcult.tv/subculture-collective/VHNESS/issues/38) |
| [#40 — Automate branded dependency upgrade PRs and consumer checks](https://git.subcult.tv/subculture-collective/VHNESS/issues/40) | [#39](https://git.subcult.tv/subculture-collective/VHNESS/issues/39) |
| [#41 — Prove one shared fix reaches neutral and distinctive video frontends](https://git.subcult.tv/subculture-collective/VHNESS/issues/41) | [#40](https://git.subcult.tv/subculture-collective/VHNESS/issues/40) |

## P3 — Mixed-media platform and reference app

Add common catalog and policies, print and audio modules, and a neutral frontend supporting mixed collections while preserving specialized media models.

[Gitea milestone](https://git.subcult.tv/subculture-collective/VHNESS/milestone/108)

| Issue | Prerequisites |
| --- | --- |
| [#42 — Implement mixed-media catalog identity, membership and search](https://git.subcult.tv/subculture-collective/VHNESS/issues/42) | [#37](https://git.subcult.tv/subculture-collective/VHNESS/issues/37), [#41](https://git.subcult.tv/subculture-collective/VHNESS/issues/41) |
| [#43 — Unify access and delivery policies without changing collection approvals](https://git.subcult.tv/subculture-collective/VHNESS/issues/43) | [#42](https://git.subcult.tv/subculture-collective/VHNESS/issues/42) |
| [#44 — Integrate Arkhiv print module and shared reader components](https://git.subcult.tv/subculture-collective/VHNESS/issues/44) | [#43](https://git.subcult.tv/subculture-collective/VHNESS/issues/43) |
| [#45 — Integrate Dedhed audio module and shared player components](https://git.subcult.tv/subculture-collective/VHNESS/issues/45) | [#43](https://git.subcult.tv/subculture-collective/VHNESS/issues/43) |
| [#46 — Build neutral white-label frontend for mixed-media collections](https://git.subcult.tv/subculture-collective/VHNESS/issues/46) | [#44](https://git.subcult.tv/subculture-collective/VHNESS/issues/44), [#45](https://git.subcult.tv/subculture-collective/VHNESS/issues/45) |

## P4 — Branded frontend adoption

Move video, print and audio brands onto supported shared dependencies while retaining their interface ownership and isolated operational state.

[Gitea milestone](https://git.subcult.tv/subculture-collective/VHNESS/milestone/109)

| Issue | Prerequisites |
| --- | --- |
| [#47 — Adopt supported shared packages in all video consumers](https://git.subcult.tv/subculture-collective/VHNESS/issues/47) | [#46](https://git.subcult.tv/subculture-collective/VHNESS/issues/46) |
| [#48 — Adopt shared print components in Arkhiv and Riot Grrrl](https://git.subcult.tv/subculture-collective/VHNESS/issues/48) | [#46](https://git.subcult.tv/subculture-collective/VHNESS/issues/46) |
| [#49 — Adopt shared audio components in Dedhed and discovered audio consumers](https://git.subcult.tv/subculture-collective/VHNESS/issues/49) | [#46](https://git.subcult.tv/subculture-collective/VHNESS/issues/46) |

## P5 — Migration and private qualification

Prove migration, independent recovery, rollback, representative media behavior and a repeatable cross-brand release; record each proof tier separately.

[Gitea milestone](https://git.subcult.tv/subculture-collective/VHNESS/milestone/110)

| Issue | Prerequisites |
| --- | --- |
| [#50 — Prove archive migration, independent restore and rollback](https://git.subcult.tv/subculture-collective/VHNESS/issues/50) | [#47](https://git.subcult.tv/subculture-collective/VHNESS/issues/47), [#48](https://git.subcult.tv/subculture-collective/VHNESS/issues/48), [#49](https://git.subcult.tv/subculture-collective/VHNESS/issues/49) |
| [#51 — Qualify cross-brand releases and private mixed-media operations](https://git.subcult.tv/subculture-collective/VHNESS/issues/51) | [#50](https://git.subcult.tv/subculture-collective/VHNESS/issues/50) |

## P6 — Public rollout decision

Review exact deployment, collection and source-use scope after private qualification. Planning and software upgrades do not authorize new public access or acquisitions.

[Gitea milestone](https://git.subcult.tv/subculture-collective/VHNESS/milestone/111)

| Issue | Prerequisites |
| --- | --- |
| [#52 — Review exact public rollout and existing deployment cutover scope](https://git.subcult.tv/subculture-collective/VHNESS/issues/52) | [#51](https://git.subcult.tv/subculture-collective/VHNESS/issues/51) |

## Delivery rules

- Prove shared releases on video before extending the catalog and extracting print/audio modules.
- Print/audio extraction can proceed independently after common catalog and policy contracts are implemented. Each brand adoption has its own PRs and verification.
- Retain old consumers/APIs and running deployments through the documented compatibility window; do not retire originals before verified recovery and consumer adoption.
- Treat local checks, hosted CI, rendered UI, real media behavior, independent restore, natural runs, device coverage and public deployment as separate evidence.
- Existing video public-release issue [#22](https://git.subcult.tv/subculture-collective/VHNESS/issues/22) remains separate. Existing #28–#33 work supplies inputs and evidence; this roadmap does not reopen or silently close those tasks.
- New content discovery, software upgrades and platform reuse never grant acquisition, relay or public access.
