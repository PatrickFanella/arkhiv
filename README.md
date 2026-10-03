# VHNESS

VHNESS is a proposed video library with on-demand viewing and a shared scheduled
television channel. Its first collection is
[Democracy Now! on Internet Archive](https://archive.org/details/democracy_now_vid).
It brings together the preservation approach of
[Arkhiv](https://git.subcult.tv/subculture-collective/arkhiv) and the library plus
shared-station approach of
[Dedhed](https://git.subcult.tv/subculture-collective/dedhed).

Viewers would browse episodes and play a selected broadcast from the beginning.
**Tune in** would join the channel at its current program position. Original
broadcast dates, descriptions, source links, and credits would stay with each
episode, and archived programs would be labeled as replays.

Delivery is configured per collection: **stream from the source** or **preserve
selected video locally**. Both modes retain complete source metadata and support
on-demand viewing and scheduled playback. Democracy Now! will use source-hosted
streaming only because local video storage is unavailable. Its video files will
not be downloaded, cached by VHNESS, or transcoded locally.

Status: metadata foundation implemented and under verification. The isolated
preview shows synthetic metadata; real source playback and channel qualification
are not complete. See [development instructions](DEVELOPMENT.md).
Both viewing modes and both delivery designs are in scope; Democracy Now!'s
streaming-only policy is confirmed. Local preservation remains an explicit,
budgeted choice for other collections.

See the [roadmap](ROADMAP.md) and
[Gitea tracking issue #2](https://git.subcult.tv/subculture-collective/VHNESS/issues/2)
for milestone order and implementation tasks. The
[initial design proposal](docs/plans/0001-video-archive-and-channel.md) covers
scope, reuse boundaries, delivery policies, and verification gates.
