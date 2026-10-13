# Source playback coverage release, October 13, 2026

Merged [PR 129](https://git.subcult.tv/subculture-collective/arkhiv/pulls/129)
adds bounded batch source playback review, persistent outcomes, current coverage
counts, safe nested audio paths and MPEG4 concert audio fallback. Metadata workers
still fetch metadata only. Batch approval grants private playback of supported
public source files, preserves independent decisions and denies other actions.

Core 0.4.1 and viewer 0.4.3 were published independently from exact merged source
`66ee6be43c37445ce23eb7198ae20852932e7ef6`.
The [core receipt](../releases/core-release-0.4.1.json) binds the immutable image
and additive migration 0015; all previous migration checksums remain unchanged.
The [viewer receipt](../releases/viewer-release-0.4.3.json) binds the actual
registry tarball checksum. The kit remains compatible with the version-1 API
of core 0.3.0, so sibling SDK upgrades can remain independent of core adoption.

The isolated shared T3 launcher passed `make verify`, `make test-integration`
and all 46 SDK tests. `make verify-curation` passed 19 tests. Both hosted feature
checks passed before merge. Integration coverage includes resumability, nested
source authorization, unsupported files, snapshot changes, independent revocation
protection and denial of acquisition, download, relay and scheduling permissions.

Publication on Dozor used the official Dockerfile `core` target, matching
source/version/artifact labels, a clean merged checkout and temporary agent
authentication removed after publication. The image was pulled by registry
digest and contains no frontend, Node, npm or `VHNESS_WEB`. The SDK tarball from
the isolated successful build was compared against every tracked package source
file in that clean checkout, published, downloaded with npm and checked byte for
byte against the tested artifact. No release receipt was inferred from a local
image ID or mutable tag.

The Live Music importer, reviewer and API now run the published immutable core
against their retained database, without resetting data. Consumer verification
passed with both published receipts, and its web image and Compose configurations
built and validated. A real MPEG4 concert source played in Chromium. A real nested
MP3 subsequently played through published core 0.4.1 and viewer 0.4.3 after the
preview's older Vite dependency cache was refreshed. Both checks reached ready
state 4 with advancing playback time and no player error, then were stopped.

Full etree ingestion remains an ongoing private operation. Approval counts are
snapshot-bound permissions, not whole-corpus decoding or CDN availability proof.
No source media was acquired, no public deployment was enabled, and neither
sustained/perceptual playback nor independent recovery is claimed. The owner
approved the full metadata import and private source playback coverage; independent
operator decisions and unavailable/unsupported sources remain explicit outcomes.
