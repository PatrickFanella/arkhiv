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
variants and orders filenames naturally. Nested source paths are retained in
metadata but remain unsupported by the existing representation file contract.
Playback uses the existing representation resolver and a browser-to-source URL.

Changed source snapshots or recorded withdrawals invalidate those mappings until
an operator re-reviews them. Re-review preserves source and decision history.
Local preservation continues to use the existing manifest, quota, integrity and
independent recovery workflow; the metadata worker never performs acquisition.

Migration 0014 adds collection-scoped indexes, corrections and cursor worker state.
Existing source snapshots, catalog decisions, media approvals and schedules are
retained. Old consumers use their existing APIs. This module is prepared for core
0.4.0; the published 0.3.0 artifact does not contain it. Package updates and public
rollout remain separate release decisions.
