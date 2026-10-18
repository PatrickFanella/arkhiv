# Source metadata catalog

The shared engine's `source_catalog` module version 1 imports complete source
metadata before any decision to acquire or play media. It is separate from the
reviewed logical-item/representation catalog. A source can remain searchable
without supported files, known dates, reviewed set boundaries or playback approval.

`discover-catalog --collection COLLECTION --pages 1 --items 25 --delay 1s`
uses cursor discovery, persists discovered IDs and the next cursor atomically,
then fetches a bounded metadata queue. All source responses and complete file
inventories are retained through the existing snapshot/observation store. It uses
the engine's metadata-only client, conditional fetches, deadline and Retry-After
handling. It never requests media. Bounds are 100 pages, 1,000 items, a 30-minute
maximum run deadline and a request interval no greater than one minute. Repeated
runs resume progress and persisted backoff. The private operator environment owns
live discovery; the isolated launcher uses fixtures and cannot reach the source.

`refresh-catalog` restarts discovery and requeues known items without deleting
metadata or editorial history. It does not enumerate by date, artist or popularity.
The cursor endpoint avoids the advanced-search 10,000-result limit. An absent
query-specific metadata field partitions source responses: October 13, 2026 live
checks observed response reuse across different queries/cursors with equal fields.

`import-source-catalog --collection COLLECTION --id ID --file metadata.json`
imports an exact saved response and its search projection. This operation requires
matching source identity, selected collection membership and an object-based file
inventory. It retains unknown fields and private entries. Normalization cleans
whitespace, strips HTML from searchable notes, validates calendar dates and may
extract a missing venue from a conventional source title. Derived fields and
uncertainty remain explicit. It infers neither source classification nor sets.

`edit-source-catalog --collection COLLECTION --id ID --file correction.json
--notes 'Reason and evidence'` applies a bounded string-field patch. A null removes
an override and restores the current source value. Overrides and notes have their
own history; a later import replaces only the normalized source projection. The
original response bytes never change. Identity, file inventories and transport
permissions cannot be edited through this command.

Private read-only routes are `/api/source-catalog/v1/items`, `/facets`, `/status`,
`/items/{id}` and `/items/{id}/source`. Every request selects an allowed collection.
The item response includes effective fields, original source metadata, complete
files, normalization provenance, correction history and exact snapshot identity.
Search covers artist, title, venue, place, notes, source credits, lineage, subjects
and track titles. Filters include artist, venue, year, complete date ranges and
availability. Page cursors are bound to the filter and collection. Source-only
inventory is unavailable in public mode; public consumers retain the existing
reviewed catalog and transport API.

`review-source-audio --collection COLLECTION --id ID --notes 'Playback review'`
is a separate explicit operator action for a source-streaming collection. It maps
supported public files with safe filenames into the existing typed audio catalog,
then creates an exact snapshot/file-bound playback review. It grants no acquisition,
download, relay, scheduling or public access. It prefers MP3, M4A, Ogg, FLAC and WAV
and falls back to the audio in an MPEG4 concert recording when available. Codec
decoding remains a browser capability; approvals do not claim every file decodes.
variants and orders filenames naturally. Core 0.4.1 accepts safe nested audio
source paths and escapes each URL segment. Traversal, absolute paths, backslashes
and control characters are rejected. Local preservation retains its existing
root-filename contract.
Playback uses the existing representation resolver and a browser-to-source URL.

Changed source snapshots or recorded withdrawals invalidate those mappings until
an operator re-reviews them. Re-review preserves source and decision history.
Local preservation continues to use the existing manifest, quota, integrity and
independent recovery workflow; the metadata worker never performs acquisition.

`review-source-audio-batch --collection COLLECTION --items 100 --timeout 10m
--notes 'Owner approved private source playback'` explicitly approves a bounded
batch from retained metadata. It never requests media or metadata. Persistent
outcomes distinguish reviewed, unsupported, held and retryable sources. Successful
and unsupported snapshots are skipped on subsequent runs; failures retry after
an hour. Changed metadata requeues the source. Only the batch's unchanged prior
decision can be renewed automatically; independent approvals or revocations need
an operator decision. Single-item review can resolve a held record.

The private status response includes current playback coverage. Counts describe
snapshot-bound approvals, not source CDN availability or browser codec support.
Withdrawn records and changed snapshots cannot count as currently reviewed.

Migration 0014 adds collection-scoped indexes, corrections and cursor worker state;
0015 adds playback-review outcomes and indexes for corpus-scale review and reporting.
Existing source snapshots, catalog decisions, media approvals and schedules are
retained. Old consumers use their existing APIs. Core 0.4.0 introduced this module;
core 0.4.1 adds batch playback review and nested source audio. Package updates and public
rollout remain separate release decisions.

## Excluding retained recordings

An operator can remove records from source-catalog browse, search, facets and
HTTP detail/source routes while retaining their exact snapshots, corrections,
review attempts and import accounting. Listing decisions survive metadata
re-imports and remain separate from independent transport permissions.

Use the operator command:

```sh
vhness set-source-catalog-listing --collection etree --file listing.json \
  --notes "Owner excluded unsupported recordings"
```

 The JSON input is
`{"identifiers":["SOURCE_ID"],"listed":false}`. A decision accepts 1–1000 unique
existing identifiers, applies atomically, and records each changed listing in
source history. Set `listed` to `true` with new notes to restore them.

Status retains the full `imported` count and reports `excluded` separately.
Search totals, facets and playback coverage omit excluded records. The playback
review worker skips them. Listing is a presentation decision; revoke independent
transport reviews separately when removing previously granted media permissions.
