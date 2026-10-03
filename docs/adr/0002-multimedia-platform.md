# ADR 0002: Shared media platform with independently designed consumers

Accepted for implementation, 2026-10-03. Tracks #36 and #37.

## Decision

VHNESS is canonical for the common Go engine and frontend packages. Keep the
existing stack and reference client working during extraction. Arkhiv and Dedhed
are donor engines, not schema authorities for the whole platform. The owner
selected distinct branded interfaces built from shared components.

The dated [consumer inventory](../reference/platform-consumers.json) records
source revisions, runtime images and unavailable evidence. It is an observation;
the future upgrade registry must validate live pins before proposing updates.
No separately named generic print/audio repositories were found in the authenticated
organization list. Arkhiv/Dedhed supply those engines; the combined neutral app
belongs here. Existing donor work and running previews remain untouched.

## Domain

Collections have stable IDs and explicit server scope. An item can belong to
several collections. A catalog item holds common descriptive metadata and source
references; its representations hold typed print/audio/video metadata. Asset roles
are distinct from representations. OCR, covers, captions and alternate encodings
do not create unrelated catalog items. Source identity is provider/item/file;
editorial identity is independently reviewed.

Print retains document variants and pages. Audio retains generic recordings and
optional concert identity, alternate tapes, ordered tracks, reviewed/unknown sets
and taper/transfer credits. Video retains episode/file-based-series identity,
renditions and captions. Preserve raw snapshots/unknown fields separately from
editorial corrections. Catalog search reports its metadata/page/caption coverage.

Example identity mappings:

| Source | Catalog mapping |
| --- | --- |
| One IA item with three zine PDFs | Three print items, each with its reviewed PDF representation and OCR companion |
| Two tapes of one concert | Two audio items with one reviewed concert grouping; independent credits and access decisions |
| One talk with MP4, audio and transcript PDF | One item with three reviewed representations and explicit search coverage |
| Mixed collection | Membership references print/audio/video items; no collection-wide forced media type |

## Packages and interface freedom

Start with `packages/viewer` (`@subcult/archive-viewer`) for the typed video API,
headless viewer hook and composed player/library/schedule components. Extract
additional common/media packages when actual boundaries require them. Publish
compiled JavaScript and declarations; React remains a peer dependency. Consumer
builds must not require private sibling source checkouts. The reference app imports
the same package entry points as brands.

Components preserve semantic/native media controls and expose composition/style
hooks. Branding controls pages, navigation, typography, assets and copy. Headless
hooks allow custom controls; consumers must verify their own labels/focus/keyboard
behavior. Shared package updates do not rewrite custom page layouts.

## Versions, scope and policy

Keep legacy video API version `1` supported while introducing a separately
versioned catalog contract. Published packages use semantic versions: patch for
compatible fixes, minor for additive capabilities, major for breaking contracts.
Declare supported API versions and reject incompatible configurations explicitly.
At least the current and previous release remain supported during staged consumer
adoption. Deprecation must identify consumers and a migration path before removal.

Use immutable engine digests and exact package pins/locks. Each release states
API, schema, configuration and minimum-consumer compatibility. Additive migrations
precede clients that need them; destructive migration waits for consumer adoption
and recovery proof. Rollback uses a schema-compatible artifact or restored checkpoint,
never an old binary against an assumed-compatible new database.

Separate publication, playback, acquisition and relay decisions, default denied
unless existing reviewed state explicitly permits them. Brand configuration cannot
grant those decisions. Streaming-only collections never gain media acquisition,
proxy/cache/transcode or offline fallback. Preserve Democracy Now! permanent
source-only enforcement and Bill Nye's reviewed manifest/budget.

One deployment may allow several collections but owns its database, credentials,
decisions, schedules, workers, media and backups. Every server operation enforces
that scope; a frontend selector is not enforcement. Listener-local controls never
mutate shared timelines.

## Implementation order and evidence

First prove versioned video package consumption and automated upgrade PRs in
neutral and distinctive consumers. Then introduce catalog/policy and reviewed
print/audio modules. Keep existing APIs and URLs compatible while rehearsing
per-product migration in disposable databases. Independent restore, browser and
real media qualification remain separate from unit/build checks. Public deployment
and new content approval retain their concrete release decisions.

Inventory limitations are explicit inputs for later migration/qualification,
not justification for resetting services or claiming historical receipts are fresh.
