# Shared viewer changes

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
