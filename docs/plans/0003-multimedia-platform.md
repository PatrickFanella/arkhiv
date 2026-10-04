# Shared multi-media archive platform

Decision recorded 2026-10-03. Status: implementation underway. The owner selected distinct branded
interfaces built from shared components and requested a repository base, issues
and roadmap. This document owns the architecture; dated reports and Gitea issues record qualification.

## Base and rationale

Use the existing **VHNESS** repository as the canonical platform base. Retain its
repository slug and history. The owner selected Arkhiv as the product name. Start implementation
from current reviewed `main`, rather than the stale primary local checkout.
Planning baseline: `5df4fc6a038b2066fec7df061c53d39642d7a52e`.

VHNESS already has a collection-scoped Go engine, React reference client,
PostgreSQL migrations, source/local delivery, persisted replay schedules,
independent recovery tools, API compatibility checks and separate standalone
frontend consumers. These are the closest existing implementation of the required
upstream/consumer boundary. The initial video adapters were exported source copies pinned by hash manifests.
Versioned package releases and hosted downstream upgrade proposals are now
qualified in the shared video report; consumer deployment remains independent.

Arkhiv contributes document ingestion, variants, local PDF reading, OCR/page
search, allowlist enforcement and recovery. Dedhed contributes concert/recording
identity, tracks, sets, credits, audio continuity and shared station behavior.
Reuse reviewed modules and regression evidence from both, with source revisions
and GPL provenance. Do not copy either entire application or assume its database
model is the common catalog. Their current checkouts and active design work remain
owned by their existing projects.

The [execution roadmap](0003-multimedia-roadmap.md) links Gitea milestones,
acceptance criteria, native prerequisites and verification boundaries. The
original [video roadmap](../../ROADMAP.md) remains historical/current video task
authority. Existing checked-but-open issues are reconciled during inventory;
this planning change does not close them.

## Repository and interface ownership

| Owner | Responsibility |
| --- | --- |
| VHNESS platform repository | Common engine, migrations, source/provenance utilities, policy enforcement, media modules, typed API/client, shared frontend hooks/components, releases and consumer registry |
| Neutral reference frontend in the platform | A complete white-label interface for print, audio, video and mixed collections; neutral identity and validated configuration |
| Branded frontend repositories | Page layouts, navigation, typography, assets, copy, editorial pages, custom interactions and dependency/deployment pins |
| Each deployment | Its own database, media, credentials, worker state, access decisions, schedules, logs, quotas and backups |

Keep Go under `backend/` and SQL under `migrations/`. Introduce a pnpm workspace
for shared frontend packages while retaining a working `web/` reference client
through extraction. Final package paths and names belong to the contracts issue;
do not change import boundaries across every consumer before that contract is
reviewable. Retain the existing Go/React/PostgreSQL stack and pinned toolchain.

Share API calls, cancellation/errors, search state, reader/player behavior and
schedule synchronization. Provide reusable components with documented styling
hooks and composition points, plus headless hooks for bespoke controls. A branded
site can have a substantially different interface without copying this behavior.
Custom markup owns its keyboard, focus, labeling and accessibility checks.

An upstream playback fix applies to every consumer of the changed playback
package after upgrade. A default-player accessibility fix applies to consumers
of that component. A reference-homepage redesign does not replace branded page
layouts. New API capabilities may require brand-specific interface adoption.
Record these distinctions in release notes and consumer verification.

## Catalog and media boundaries

Use collections with many-to-many item membership, shared catalog items, and
typed media representations. An item may offer print, audio and video together;
a collection may contain all three. Keep files and their roles explicit:
originals, renditions, OCR, covers and captions are not all independent items.

Archive.org source identity does not define catalog granularity. One source item
may contain multiple documents or episodes. A concert may have several source
recordings. Preserve raw source snapshots, every file entry, unknown fields,
fetch/refresh history and source-to-catalog mapping independently of editorial
corrections. Avoid collapsing alternate recordings or source variants by title
or date alone.

| Module | Specialized model and capabilities |
| --- | --- |
| Print | Documents, PDF variants, pages, OCR, page search, covers, local reader and permitted downloads |
| Audio | Generic recordings plus concert extensions, alternate tapes, ordered tracks, reviewed/unknown sets, credits, continuous playback and shared stations |
| Video | Episodes, file-based series, renditions, captions, source/local playback and scheduled replay |

Common browse/search includes media and collection filters. Expose the actual
search coverage of each representation: metadata search must not imply OCR,
caption or transcript search exists for every item. Preserve public URLs through
compatible endpoints or explicit redirects during migration.

Shared scheduling may reuse timeline mechanics, but module eligibility and media
transport remain explicit. Listener controls never change the server schedule.
Exhausted schedules, missing media, source failure and autoplay restrictions stay
visible; no silent substitutions or unbounded retries.

## Policy and operational isolation

Keep publication, playback, acquisition and relay as independent reviewed
decisions. Collections provide defaults and bounded reviewed exceptions;
frontend configuration cannot grant permissions. Enforce allowlists on the server
across metadata, search, covers, reading, playback, downloads and range requests.
Migration preserves existing decisions and audit history instead of approving
everything imported into the new platform.

Democracy Now! remains permanently `source_stream`: no application-managed video
download, proxy/cache, transcode or offline fallback. Metadata workers cannot
fetch video. Bill Nye retains its reviewed 100-MP4 manifest, 16 GiB cap, 1 GiB
reserve and independent backup. New local-preservation collections require their
own exact manifest, capacity review and recovery evidence. Sharing an engine
does not extend these permissions to another collection.

Retain isolated deployments initially. A site may select several collections,
but its allowed collection set is enforced in API, playback, scheduling and
operator paths. Replace the current one-collection compatibility assertion
deliberately when mixed collections are implemented. Do not combine branded
databases, credentials or schedules to achieve shared code reuse.

## Releases and downstream updates

Publish versioned frontend packages and immutable engine image digests from
reviewed commits. Use a registry selected during release implementation and the
existing secret provider for scoped CI credentials. Consumer builds must work
without sibling private source checkouts. Record source SHA, license, checksums,
API/schema/config compatibility, changelog and reproducible dependency locks.

Define compatibility windows, deprecation, package semantic versioning and
coordinated upgrade order for engine, migrations and frontend. Do not deploy a
new client against an incompatible API. Retain previous verified artifacts and
compatible recovery checkpoints.

Maintain a canonical consumer registry. A platform release automatically proposes
Gitea upgrade PRs that update packages, lockfiles, engine digests and compatibility
metadata where needed. Repeated runs are idempotent. Consumer checks exercise
enabled media and reject incompatible versions. Proposed upgrades, passing CI,
merged pins and deployed revisions are separate states.

Use reviewed upgrades initially. Per-product automatic deployment policies can
be chosen later; this roadmap does not enable unattended public rollout. Branded
repositories own their independent release timing and layout changes.

## Delivery sequence and proof

1. Inventory exact canonical sources and deployed consumers; define domain/API,
   package and branding contracts.
2. Extract video components/hooks, publish releases and automate consumer upgrade
   PRs. Prove one observable shared fix reaches the neutral video app and a
   distinctive branded frontend before expanding media scope.
3. Add common catalog and policy enforcement; integrate reviewed print/audio
   modules and build the neutral mixed-media reference frontend.
4. Adopt the packages in each video, print and audio consumer through separate
   PRs, preserving interface ownership and operational isolation.
5. Rehearse per-product migration, independent restore and rollback. Qualify a
   second shared release across every registered consumer and record real media
   behavior, browser/device limits and natural operations.
6. Review exact public rollout/cutover scope per product, linking existing release
   gates. Perform deployment only after its concrete plan has separate authority.

Required software gate remains `make verify`, extended deliberately for workspace
packages. Database checks use explicitly disposable `TEST_DATABASE_URL` targets.
Never reset live preview/production databases or replace active services for
verification. Documentation-only changes use diff/link and hosted-record checks;
they do not establish application correctness.

Preserve local checks, hosted CI, rendered interfaces, real PDF reading/audio/video
playback, listening review, schedule timing, device coverage, independent restore,
natural jobs/soak and public deployment as distinct evidence. Synthetic fixtures
prove contract behavior, not real media fidelity. Source-streamed media recovery
restores metadata/decisions; it cannot recover video that disappeared upstream.

## Scope limits and unresolved inventory

Accounts, uploads, a shared tenant control plane, arbitrary plugins, universal
theme editing, mobile apps and new continuous encoded broadcasts are outside
this plan. Repository renaming remains deferred. No donor repository is retired
before consumer adoption, recoverable originals and remaining owners are known.

The separate generic audio/print frontend repository names and deployed revisions
were not confirmed in the initial local inspection. Inventory must identify them,
along with active source owners, before writing their migration PRs. The registry
target, package names and compatibility support window are implementation
decisions owned by the relevant contract/release issues. No release dates are
committed.
