# Shared viewer changes

## 0.2.0 (unreleased)

- Add the catalog entry for scoped mixed-media metadata browsing and validated
  representation transport URLs.
- Add the `@subcult/archive-viewer/print` entry with PDF page controls, zoom,
  readable text and optional search through a consumer-supplied PDF backend.
- Keep PDF runtime dependencies and worker assets in the consumer. Video
  consumers continue to import the existing root entry.
- Add the audio entry for full recordings, reviewed sets, track navigation,
  source credits and shared station clock calculations.
- Scope video channel/playback requests to the selected collection and hide
  stale sessions, errors and loading state during collection changes.

Print reading requires the engine's reviewed representation access. Search must
report actual indexed coverage; OCR without page mapping remains document text.

## 0.1.1

- Tie catalog results and errors to the active collection, query, year and page.
  A pending or failed search cannot display a previous search's episode cards.
- Ignore catalog responses after their request is aborted.
- Give a branded library the accessible name supplied by its heading, announce
  result counts and expose a busy state while results load.

Compatible with video API 1 and configuration contract 1. Consumers retain their
page layouts, assets, CSS and collection delivery policies. Upgrade the exact
package pin, lockfile, engine digest and release receipt together.

## 0.1.0

Initial package extraction: compiled ESM and declarations, native video player,
source-policy validation, headless viewing/schedule logic and composed screen,
library, compatibility and replay components. React remains a peer dependency.
# 0.2.1

- Export shared audio controls for distinctive consumer interfaces; the neutral
  player uses the same seek bounds, transport state and completion behavior.
