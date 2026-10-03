# Managed archive, public release, and second-archive plan

Status: implementation in progress. The merged private pilot was deployed on
2026-10-02. Audit, backup, monitoring and dry-run retention runners are implemented;
the four timers have completed initial scheduled runs, and NUC recorded
notification qualification messages. The recurring restore runner and browser probe are implemented; the corrected initial scheduled NUC restore and actual source refresh passed.
The owner approved archival access to the reviewed 25 documents and 26 variants, with unknown rights and no asserted reuse license. Public allowlist implementation and second-source qualification remain in progress. Public ingress remains
unconfigured.

The first private pilot is qualified. PRs #18, #19 and #20 are merged; the
homelab runtime identifies merged revision `6470472`, verified on October 2. The working
archive remains on Almaz at `/mnt/spektr/server/data/arkhiv/pilot`, with independent
backups on NUC at `/home/onnwee/backups/arkhiv`.

[Gitea roadmap #17](https://git.subcult.tv/subculture-collective/arkhiv/issues/17)
is the delivery index. Individual issues own detailed acceptance criteria and
verification commands. This plan connects the four requested phases; satisfied
implementation dependencies can proceed in parallel with operator decisions.

## Ordered delivery

### 1. Pilot delivery closeout

| Issue | Outcome | Dependencies |
| --- | --- | --- |
| [#21](https://git.subcult.tv/subculture-collective/arkhiv/issues/21) | Deploy the merged pilot revision and close the qualified delivery milestone | #14 |

### 2. Managed private archive

| Issue | Outcome | Dependencies |
| --- | --- | --- |
| [#22](https://git.subcult.tv/subculture-collective/arkhiv/issues/22) | Schedule bounded archive integrity audits on Almaz | #21 |
| [#23](https://git.subcult.tv/subculture-collective/arkhiv/issues/23) | Schedule verified catalog and file backups from Almaz to NUC | #21, #22 |
| [#24](https://git.subcult.tv/subculture-collective/arkhiv/issues/24) | Add backup retention, capacity checks and a documented off-site option | #23 |
| [#25](https://git.subcult.tv/subculture-collective/arkhiv/issues/25) | Monitor archive jobs and notify on failed or stale preservation runs | #22, #23 |
| [#26](https://git.subcult.tv/subculture-collective/arkhiv/issues/26) | Run recurring isolated restore drills from the independent NUC backup | #23, #25, #24 |
| [#27](https://git.subcult.tv/subculture-collective/arkhiv/issues/27) | Persist source inventories and detect added, changed and removed archive files | #21 |
| [#28](https://git.subcult.tv/subculture-collective/arkhiv/issues/28) | Acquire eligible source changes and queue additions for editorial review | #27 |
| [#29](https://git.subcult.tv/subculture-collective/arkhiv/issues/29) | Schedule incremental refresh jobs for archives that are still growing | #28, #23, #25 |

### 3. Public archive release

| Issue | Outcome | Dependencies |
| --- | --- | --- |
| [#30](https://git.subcult.tv/subculture-collective/arkhiv/issues/30) | Review public access and reuse decisions for the pilot corpus | #21 |
| [#31](https://git.subcult.tv/subculture-collective/arkhiv/issues/31) | Enforce the reviewed public release allowlist in catalog and asset serving | #30 |
| [#32](https://git.subcult.tv/subculture-collective/arkhiv/issues/32) | Finish public catalog branding, metadata presentation and contact flows | #30, #31 |
| [#33](https://git.subcult.tv/subculture-collective/arkhiv/issues/33) | Prepare a versioned public homelab release and domain ingress | #31, #32, #23, #25 |
| [#34](https://git.subcult.tv/subculture-collective/arkhiv/issues/34) | Qualify the public archive release through the real domain | #33, #26, #29 |

### 4. Second real archive

| Issue | Outcome | Dependencies |
| --- | --- | --- |
| [#35](https://git.subcult.tv/subculture-collective/arkhiv/issues/35) | Select and qualify a second archive source with a different structure | #21, #27 |
| [#36](https://git.subcult.tv/subculture-collective/arkhiv/issues/36) | Implement bounded IA collection/query discovery for multiple source items | #35, #27 |
| [#37](https://git.subcult.tv/subculture-collective/arkhiv/issues/37) | Isolate second-archive branding, taxonomies and acquisition policies | #35, #36, #28 |
| [#38](https://git.subcult.tv/subculture-collective/arkhiv/issues/38) | Import, update and independently restore the second real archive | #37, #29, #26 |

## Active-archive update workflow

The first archive is a bundled IA item; later collections can contain multiple
items. The same workflow must detect new files inside existing items and new
items discovered through a configured collection/query.

1. **#27 — Complete source inventories and change detection.** Persist raw
   snapshots and a durable successful baseline. Distinguish additions, changed
   bytes, metadata-only changes, no change, removals, and uncertainty. Incomplete
   listings or failed requests cannot advance the baseline or infer mass removal.
2. **#28 — Bounded acquisition and review.** Default discoveries to review-only.
   A collection can explicitly enable acquisition within source/format/byte/file/
   document/free-space limits. Preserve earlier original versions and editorial
   decisions. New records and changed-content variants remain unpublished pending
   review; deliberate exclusions survive repeated refresh.
3. **#29 — Scheduled job.** Proposed initial cadence is daily at 04:00
   America/Chicago, with per-collection enable/disable, cadence and policy. Serialize
   writers, resume interrupted phases, respect source Retry-After, and report last
   attempt/success plus checked/acquired/pending/failed counts. Register failures
   and stale runs with #25 monitoring. #36 supplies multi-item discovery.

A source removal creates a review event and retains local preservation history.
A source outage affects new acquisition, while existing local reading and downloads
remain available. Update jobs run through the dedicated source-access operator
workflow; the reader/API and T3 test networks remain isolated.

## Preservation and release boundaries

Daily integrity and backup schedules are proposed in #22/#23; #26 proposes monthly
fresh-destination restore drills and drills after storage/schema changes. #24
introduces dry-run retention with pinned qualification snapshots and capacity
checks. The initial retention proposal is seven daily, four weekly and six monthly
snapshots, subject to review before pruning activation. Never prune originals,
partial evidence, unrelated paths or the last known-good backup. Off-site storage
is an option to evaluate, not a selected or activated third copy.

Public-serving decisions are separate from private publication. #30 reviews the
record/variant subset. The owner selected `https://arkhiv.subcult.tv` and
`info@subcult.tv` for correction/removal requests.
#31 conservatively backfills public approval and tests every catalog/asset route
against private, unknown, revoked and unapproved records. #33 prepares the exact
release and scoped ingress; public activation requires the separate release action.
#34 verifies the actual public domain and defines an initial observation window.

The second-source gate #35 selects a bounded real archive with a different structure,
access/reuse evidence and explicit update policy. The owner requested another zine collection; the selected source is 15 A-Info zines from São Paulo, discovered through a bounded title query. #38 requires actual ingestion, source-unreachable reading, and independent
fresh restoration while the first archive continues working. Epic #15 owns that
outcome. Immediate operations are in the managed phase; epic #16 retains later
curator UI/accounts and evidence-driven OCR/search improvements.

## Verification and delivery

Existing gates are `make verify`, `make test-integration`, `make test-browser`,
and `make test-recovery`. `make test-operations` runs job, backup, notification and retention checks
and is included in `make verify`; `make test-updates` verifies durable inventory and acquisition behavior against
disposable PostgreSQL; bounded multi-item discovery and complete-query failure checks are implemented and verified.

Fixture checks cover job interruption, overlap, disk/transport/checksum failure,
complete-inventory boundaries, evolving sources, publication exclusion, and safe
restore/pruning. Live proof records source/runtime/destination identity, natural
timer invocation, verified NUC snapshots, receiver-observed notifications, native
reader rendering and fresh restore. Synthetic additions prove change behavior;
an actual unchanged-source check does not prove that a real new document appeared.

Issues have native dependencies, explicit test commands, separate operator evidence
and unchecked queue checklists. No `ready-for-agent` or `agent:queued` label is
added. #30 and #35 are operator decision gates. This plan creates no deployment,
notification, pruning, acquisition or public publication action by itself.

The update runner persists acquisition backoff across processes. A controlled source qualification may set `force_source_checks` to bypass collection cadence for that run; disabled collections and source backoff remain enforced. Remove that qualification setting afterward. Fixture qualification includes discovery and acquisition of a new multi-item source record, private review defaults, and duplicate-free replay.
