# VHNESS: video library and scheduled channel

Updated 2026-10-02. Status: design; no application or public deployment exists.

## Confirmed requirements

- Start with Internet Archive's
  [Democracy Now! video collection](https://archive.org/details/democracy_now_vid),
  identifier `democracy_now_vid`.
- Include both an on-demand library and a shared scheduled TV channel.
- Plan two delivery policies: source-hosted streaming and local preservation.
  Choose delivery per collection, independently from viewing mode.
- Democracy Now! must stream from the source. Store its complete metadata;
  never download or persist its video files. The owner has no space for them.
- Preserve original broadcast dates and source attribution. Label archived
  broadcasts as replays rather than implying live Democracy Now! programming.

No year, topic, or episode restriction has been requested. Metadata discovery
should eventually cover the full collection. A small sample is a verification
stage, not a permanent catalog filter.

## Source observations

Read-only Internet Archive metadata and search requests on 2026-10-02 found:

- `democracy_now_vid` is a collection. Its own files are collection assets;
  discover child movie items through collection search.
- `collection:democracy_now_vid AND mediatype:movies` returned 6,626 items at
  inspection time. That count can change.
- The three newest returned identifiers were `dn2026-1002_vid`,
  `dn2026-1001_vid`, and `dn2026-0930_vid`. The first two metadata responses were
  inspected.
- `dn2026-1002_vid` lists a 1920×1080 original MP4 of 1,584,516,847 bytes and an
  854×480 derivative MP4 of 623,803,681 bytes, both about 59 minutes long.
- `dn2026-1001_vid` similarly lists 1080p and 480p MP4 files. Both inspected
  episodes have dates, descriptions, file sizes, source/derivative relationships,
  checksums, and reported durations.
- Neither inspected episode supplied `rights` or `licenseurl`. Retain those
  fields as absent rather than assigning a reuse license. Collection publisher
  metadata must remain distinct from episode creator or rights metadata.

Sources: [collection metadata](https://archive.org/metadata/democracy_now_vid),
[October 2 episode metadata](https://archive.org/metadata/dn2026-1002_vid), and
[October 1 episode metadata](https://archive.org/metadata/dn2026-1001_vid).
Format labels and reported durations do not prove browser compatibility,
availability, seeking, or synchronization. No video was downloaded or played.

## Reference projects and reuse

The initial review used local Arkhiv and Dedhed checkouts, including README
files, agent instructions, Dedhed's roadmap and reuse decision, domain contracts,
and playback implementation. Their remotes match the repositories provided by
the owner. Arkhiv contains work in progress and was left untouched. Hosted state
was not verified: the web reader could not open either repository, and
unauthenticated Gitea API reads returned 404.

Arkhiv supplies source identity, metadata snapshots, editorial separation,
bounded discovery, integrity, and recovery patterns. Its local preservation
patterns apply to collections explicitly configured for local storage.

Dedhed supplies independent personal playback and a shared persisted timeline.
Its concert/set domain and continuous audio processing need video-specific
replacements. Any copied source client must have a reviewed pinned revision,
license provenance, and file hashes. Inspect retry and response-size behavior
before adopting it for collection-wide metadata refresh.

## Viewing and delivery are independent

| Collection policy | On demand | Shared channel | Stored locally |
| --- | --- | --- | --- |
| Source-hosted streaming | Browser plays the selected source file | Browser joins the scheduled source file at the current offset | Complete metadata, decisions, and schedules |
| Local preservation | Browser plays a reviewed local asset | Browser joins the scheduled local asset at the current offset | Complete metadata plus explicitly selected originals and approved derivatives |

Democracy Now! uses the first row. Local preservation for another collection
requires an explicit policy and storage budget. There is no automatic fallback
from source streaming to local acquisition.

## Viewer experience

**Library:** dated episode shelves with title/description search, date/year
filters, and source attribution. Preserve complete descriptions while treating
their HTML as untrusted presentation input. Use source-supplied topics when
available. Description paragraphs do not imply chapter timestamps.

**Episode:** show original broadcast date, description, available credits, source
link, reported duration, and eligible viewing variants. Support starting from
the beginning, seeking, volume, fullscreen, and verified caption tracks. Show
loading, unsupported-format, unavailable-source, and playback-failure states.

**Channel:** show now playing, original broadcast date, upcoming programs, and
Tune in. Join at the current scheduled position. Pausing or choosing an episode
on demand affects only that viewer. Returning to Tune in rejoins the current
program. Clearly label archived content as replay.

The suggested visual direction is a video shelf and television guide, with a
prominent replay screen, readable dates, and conventional player controls.
Final visual design remains open.

## Shared metadata architecture

Use Go, React/TypeScript with Vite, and PostgreSQL, consistent with the reference
projects. Proposed paths are `backend/`, `web/`, `migrations/`, and `scripts/`.

Store complete source responses as well as normalized fields. A normalized
episode record alone does not satisfy the requirement to store all metadata.

| Entity | Stored content |
| --- | --- |
| Collection | Identifier, complete source metadata, delivery policy, discovery configuration, checkpoints, and any local storage budget |
| Source snapshot | Exact complete response bytes, parsed JSON, SHA-256, source identity, fetch time, and available response validators |
| Title/episode | Source ID, title, broadcast date, description, subjects, language, creator, and source attribution where present |
| File inventory | Every returned file entry and field, including originals/derivatives, formats, sizes, checksums, reported durations, dimensions, and private flags |
| Editorial decision | Catalog visibility, permitted viewing modes, preferred source file, corrections, reason, and decision history |
| Schedule | Versioned absolute UTC program entries, chosen playback asset, current revision, and prior schedules |
| Refresh state | Discovery checkpoints, last success, failures, retry deadline, missing-item observations, and snapshot changes |
| Local asset, where enabled | Exact source identity, managed storage path, verified hashes, byte size, processing receipt, and publication state |

Preserve unknown fields and non-video file entries. Normalized search fields are
derived indexes that can be rebuilt from snapshots. Keep editorial overrides
separate from source values. For streaming collections, captions, thumbnails,
and other assets remain source references initially; their binary payloads are
not acquired by metadata workers.

## Discovery and refresh

1. Fetch collection metadata and page through child movie identifiers. Bound
   batch size, deduplicate identifiers, and retain progress across interruption.
2. Fetch complete item metadata, verify identity and collection membership,
   store a hashed snapshot, and normalize title and file records.
3. Retain earlier snapshots when source metadata changes. Refresh source fields
   without overwriting editorial decisions or schedule history.
4. Reconcile periodically for new and backfilled episodes. Recheck known items
   in bounded batches rather than fetching the full collection on every run.
5. Respect source backoff and complete `Retry-After` deadlines. Persist retry
   state and report partial failure without aborting the entire catalog.
6. Mark missing/inaccessible items while preserving metadata and history.
   One failed fetch does not authorize deletion.

Metadata workers request search and metadata endpoints only, under either
delivery policy. Local video acquisition is a separate operator workflow.
Discovery does not automatically grant public catalog or playback access.

## Source-hosted streaming: Democracy Now!

The API selects an eligible file from the stored source inventory and returns
its source-hosted URL. The browser streams directly from Internet Archive.
The backend does not fetch, relay, cache, or transcode video bytes.

The inspected 480p MP4 derivatives are initial lower-bandwidth playback
candidates. Another source-provided variant can be offered after verification.
Verify actual playback, source redirects, Range seeking, aspect ratio,
picture/audio synchronization, and reported versus browser-observed duration.
Use source-hosted HLS only where a usable rendition exists; local HLS generation
is outside this collection's policy.

Browser buffering during playback is expected. VHNESS must not create an
application-managed persistent video cache, download queue, or offline copy.
The metadata catalog must render from local storage when Archive.org is down;
video will show source unavailability rather than attempt local acquisition.

Provide clear errors, the original source link, and bounded retry for source
removal, rate limits, network interruptions, or incompatible files. Viewing
decisions control VHNESS's interface; hiding a title does not revoke access to
the independently hosted source URL.

## Local preservation: other explicitly selected collections

Plan this adapter alongside streaming, behind an explicit collection policy.
The same library and scheduling interfaces should use either delivery adapter.
No local-preservation collection has been selected yet.

Acquisition requires an exact reviewed manifest: item/file identity, selected
originals, maximum file and batch bytes, destination, and access decisions.
Enforce a collection budget and a free-space reserve before transfers. Stop
before exceeding either limit. Do not acquire an entire collection merely
because it has been discovered.

Use bounded resumable downloads, validate resume responses, verify source
checksums where available, and compute local SHA-256. Publish an asset only after
verification. Keep immutable originals and record any playback derivative's
processing parameters, input/output identities, dimensions, frame rate, codecs,
and duration. Remux compatible sources where possible; transcode only within
the approved storage and processing budget.

Serve local eligible assets with Range support and access checks. Preserve
aspect ratio and verified captions. Test actual browser playback; probing alone
is insufficient. Acquired media, generated derivatives, databases, credentials,
and raw snapshots stay outside Git.

Record acquisition, on-demand viewing, channel use, and public release decisions
independently. Define derivative regeneration and retention policies. Never
delete originals or expand a budget through an automatic cleanup mechanism.

## Shared scheduled channel

Persist explicit programs with title/episode ID, chosen asset, delivery policy,
absolute UTC start, and duration. The server returns its time, schedule revision,
current program, offset, and next entry. Either adapter supplies the resolved
playback URL; the scheduling contract stays the same.

Seek to the current offset after media metadata loads, accounting for elapsed
request/load time. If the program changes while loading, request current state
again. Rejoin after a suspended tab, interrupted connection, or explicit Tune in.
Measure drift between two browser sessions and set a practical tolerance; this
design does not promise frame-accurate synchronization.

At a boundary, load the next scheduled asset and join its current position. Do
not depend exclusively on `ended`: buffering or pausing must not stop the clock.
Represent browser autoplay restrictions explicitly when user interaction is
required.

An unavailable program retains its scheduled slot. Show the failure and proceed
according to the clock at the next boundary. Gaps and schedule exhaustion remain
explicit until the operator extends the schedule. There is no silent
substitution, unbounded retry, or streaming-to-local-storage fallback.

The first channel is a scheduled Democracy Now! replay experience. A continuous
encoded broadcast output is a separate future transport choice.

## Backup and recovery

Both delivery policies require independent backup of metadata, snapshots,
editorial history, schedules, and worker checkpoints, plus a disposable restore
drill. A metadata restore must rebuild the catalog and shared timeline.

For streaming collections, recovery cannot restore source video removed from
Internet Archive. Verify that the restored catalog renders while the source is
blocked and the player reports unavailability.

For local-preservation collections, also back up originals and processing
receipts to an independent destination. Decide whether derivatives are backed
up or regenerated, budget that storage, and demonstrate restored playback with
the upstream source unavailable.

## Delivery order and verification

1. **Metadata foundation:** bootstrap the isolated Go/React/PostgreSQL workflow;
   implement snapshots, file inventory, normalization, discovery/checkpoints,
   decisions, and explicit per-collection delivery policy. Verify response
   fidelity, membership checks, duplicates, resume, changes, and backoff.
2. **Democracy Now! streaming library:** import a small real metadata sample,
   build library and episode pages, and resolve source-hosted viewing URLs.
   Verify loading, seeking, picture/audio synchronization, failure states, and
   browser-to-source requests. Assert no video bytes reach application storage.
3. **Shared replay channel:** implement persisted now/next, Tune in, mode
   switching, boundaries, and tab-resume behavior. Measure two viewers joining
   mid-program and crossing a boundary; on-demand controls must not mutate the
   schedule.
4. **Full metadata coverage and streaming qualification:** complete resumable
   discovery, exercise new/changed/missing items, restore the metadata store,
   and verify a full-episode viewing session plus a scheduled transition.
5. **Local preservation adapter:** implement the separately planned manifest,
   budget, resumable transfer, integrity, optional derivatives, local serving,
   and recovery contracts. Qualify with a small selected collection only after
   its source, storage destination, and budget are specified. Test both viewing
   modes through the same delivery interface.

Local checks, hosted CI, rendered UI, real audiovisual playback, synchronization,
sustained viewing, and recovery are distinct evidence. None have passed merely
because this plan exists. Local-preservation readiness does not delay the
streaming-only Democracy Now! pilot.

## Initial boundaries and remaining choices

The proposed first operational scope is a private pilot. Public hosting remains
an owner decision. Accounts, uploads, personal playlists, multiple channels,
offline playback, and continuous encoded broadcasting are deferred.

Choose target browsers/devices and a channel-drift tolerance during playback
qualification. A reasonable initial sample is the three latest discovered
episodes; this does not limit eventual full-collection metadata coverage.
Choose another collection, destination, and explicit storage budget before
activating local preservation. Democracy Now!'s no-local-video policy remains
in force regardless of that later choice.
