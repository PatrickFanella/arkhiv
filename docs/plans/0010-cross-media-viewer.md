# Viewer feedback across media

Issues #115 and #116 extend #105 to audio and the mixed print/audio/video
reference interface. The brands keep their own frontends and release adoption.
No acquisition, artwork extraction or public activation is added.

## Audio

Audio and native mixed-media playback share the validated volume/mute helper
with video. The existing browser storage key remains compatible. Preferences
restore on mount and observe native changes; storage failure uses memory.
Restoring preferences does not initiate playback or bypass autoplay rules.

On-demand audio keeps position and reviewed track/set navigation. A station
has read-only position, disabled track jumps, original dates, pause, mute and
fresh resume/rejoin. Active, ready, non-seeking playback must be within three
seconds of the station clock before it is labeled synchronized. Correction
uses the existing buffered-position guard. Program changes still come from
schedule reads, never viewer schedule writes. Station transport rejects a
response for another collection.

## Print and the mixed catalog

The reference offers grid/table browsing, creator substring filters and bounded
original-date filters. Combined filters apply before pagination and retain the
existing public view and collection scope. Date bounds require complete ISO
query dates. Source date comparisons use complete YYYY-MM-DD strings; unknown
and partial dates are excluded only while a date bound is active. Source values
are displayed unchanged. Private notes and source responses stay out of the
projection. Print page counts/variants and audio credits remain visible when
available. Covers are not synthesized or acquired for this work.

Shared links bind collection, logical item and representation. Opening a link
fetches current scoped metadata and resolves current representation access.
Copy has a manual fallback; social links open compose intents. Focus mode exits
with Escape and preserves the document reader, text and search controls.

Favorites use `catalog:<item-id>` references, distinct from existing episode
references. The personal service checks the current scoped catalog before adding
one. Deletion still allows removal of unavailable references. Browser favorites
work without account services. Optional sync uses the accepted explicit import,
revision-conflict review, export and account deletion controls. Unavailable
references are retained in the favorite list rather than silently deleted.

Chat currently binds a reviewed video-channel program. Print has no scheduled
program identity. Adding audio discussion requires distinct audio/video room
identity in mixed collections, rather than binding an audio listener to the
video program. This is a separate extension to the current chat contract.

## Evidence and release

The isolated shared `make verify && make verify-curation` passed with disposable
public-boundary, canonical-favorite and combined-filter integration checks.
The SDK checks include station controls, measured playback states and encoded
mixed-media filter/share identities. New package/core versions are 0.3.1/0.2.1.
Publication and consumer pin adoption are recorded in the qualification log.
Hosted CI is temporarily owner-waived while Kvant is down.

The collaborative preview still returns an automation execution error on
snapshot, including after reopening the inline preview. Browser evidence for
these additions is incomplete. Issues remain open until that evidence exists.
