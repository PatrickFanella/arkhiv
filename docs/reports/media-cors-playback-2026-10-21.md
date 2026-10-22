# Cross-origin playback and volume extensions — 2026-10-21

Lineage's player advanced normally but produced no audible output in the owner's
Zen session. The same MP3 played directly from Internet Archive. Player volume,
element mute, tab mute, a private window and a full browser restart did not
resolve the silence. WirePlumber, PipeWire and pipewire-pulse were active.

## Confirmed cause and repair

The installed Volume Master 1.0.1 extension routes media elements through
`AudioContext.createMediaElementSource`. Cross-origin responses fetched without
CORS are silent in that graph. The original media elements did not request CORS.
A new media element in the same page therefore had the same failure.

An isolated Zen 1.23.4b process on Sputnik reproduced the issue with the owner's
installed extension, without copying their browser profile or changing its
settings. At one minute into the December 29, 2024 Billy Strings opening track,
the original player produced zero-valued output samples. Reloading the same
source with `crossOrigin="anonymous"` restored output at -28.0 dB RMS. The source
CDN returned `Access-Control-Allow-Origin: *`. The owner independently confirmed
that the same CORS change restored sound in their existing Lineage tab.

Set the attribute before assigning a source. The shared audio/video components,
Arkhiv's native mixed-media elements, and custom consumer media elements need the
same request mode. Remote servers must permit the consumer origin. Same-origin
engine media keeps its normal session behavior.

## Published SDK

Viewer 0.4.4 is an independent SDK patch. It changes no engine image, database
migration, collection approval or acquisition policy. The package registry
artifact was downloaded after publication and matched its receipt.

- Source revision: `65d090236f584ba2166b03f7dcfa83ea9911c6d7`.
- Package SHA-256: `6a39e478117a419dc9258b2e82c635a4b7b4ced68acd63996fd8b2ab1f10f241`.
- Receipt: [viewer-release-0.4.4.json](../releases/viewer-release-0.4.4.json).

## Verification

The shared full `make verify` gate passed in the isolated `vhness` test snapshot,
including backend race/database checks, 46 viewer tests, 75 release-tooling tests
(two skipped), the packed offline consumer check, and the reference frontend's
five tests and production build. The separate `make verify-curation` gate passed
19 tests, Ruff checks and JavaScript syntax checking.

Consumer source checks, PR-head CI, merged pins and effective deployments remain
separate evidence. The all-consumer adoption results are recorded below.

## Consumer adoption PRs

- [subculture-collective/video-archive-template #16](https://git.subcult.tv/subculture-collective/video-archive-template/pulls/16): required `make verify` passed; head `2c119378ab35c3d1ecd28e14c19fcff3796427dc`.
- [subculture-collective/democracy-now-tv #18](https://git.subcult.tv/subculture-collective/democracy-now-tv/pulls/18): required `make verify` passed; head `0eebffa6ad2cf35bb667051e612749d8245b547a`.
- [subculture-collective/arkhiv-anarchist #72](https://git.subcult.tv/subculture-collective/arkhiv-anarchist/pulls/72): required `make verify` passed; head `6d0881c311354c0ff8ec8b06dc5d967298b3b284`.
- [subculture-collective/arkhiv-riot-grrrl #23](https://git.subcult.tv/subculture-collective/arkhiv-riot-grrrl/pulls/23): required `make verify` passed; head `d101be79a755efe3ea2d1c48e1249bf36170dd01`.
- [subculture-collective/bill-nye-tv #18](https://git.subcult.tv/subculture-collective/bill-nye-tv/pulls/18): required `make verify` passed; head `e613f02a467e0e32ec86763ccf8100f174b25c92`.
- [subculture-collective/dedhed #39](https://git.subcult.tv/subculture-collective/dedhed/pulls/39): required `make verify` passed; head `3f6773e21c73299b79d7a78ca218835c5f4067ca`.
- [PatrickFanella/arkhiv-live-music #20](https://git.subcult.tv/PatrickFanella/arkhiv-live-music/pulls/20): required `make verify` passed; head `331969521452479296c7f45c01eb6a277b424c7a`.

Lineage and Dedhed own their audio markup and include the CORS attribute locally.
The other consumers use the released shared components. All seven pin the exact
0.4.4 package, lockfile and release receipt. The retained core receipts and image
pins were checked as read-only inputs during upgrade preparation.

The PRs remain reviewable proposals. Merge the consumer updates before the
shared receipt PR so release automation observes the adopted receipt rather
than preparing duplicate proposals. No production service is activated by these
source changes.
