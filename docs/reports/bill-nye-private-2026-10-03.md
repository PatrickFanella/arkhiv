# Bill Nye private preservation qualification — October 3, 2026

The owner approved exactly two source-provided MP4 derivatives from
[the Bill Nye archive](https://archive.org/details/billnyethescienceguythecompleteseries-07-28-2021).
The application preserves their original bytes. It retains the AVI relationships
and all 1,040 parent file entries in the metadata snapshot; no AVI or other video
was acquired. Episode dates remain unknown because the parent's series debut
date is not an individual episode broadcast date.

| Selected file | Bytes | Verified local SHA-256 |
| --- | ---: | --- |
| `01 Flight.mp4` | 138,274,449 | `a7d7a4a962af986c3fc359d1c6c6d33364825e89aa7c646692429d16ece74563` |
| `02 Earth's Crust.mp4` | 136,506,408 | `c0c5fdfefcbf43b65f40884663b1d73454c11265ff6b4865b3191b8dbce54c7d` |

Combined media is 274,780,857 bytes. The immutable collection configuration sets
a 536,870,912-byte budget and 1,073,741,824-byte free-space reserve. Acquisition
enforces both before and during transfer. Originals and receipts live under
`~/.local/state/vhness/private-pilot/media/bill-nye`. Manifest source snapshot
`9c516f9854ce21fdb854a3b7a7f353163cf4f297fabddd23fd31d64ee6cd3769`
pins source MD5 and SHA1; both matched after transfer, and SHA-256 was computed
and recorded before promotion. The integrity audit reported two originals and
zero failures. Private review permits playback and channel use, with public
release disabled.

The owner authorized a separate local qualification browser. With Archive.org
blocked in that browser, on-demand Flight decoded 512×368 video and audio.
Seeks to 5, 690, and 1,370 seconds completed with readyState 4 and increasing
audio/video decoding counters. Two independent browser contexts joined a
mid-program local schedule, then changed automatically from Flight to Earth's
Crust at 04:52:13 UTC. All steady playing samples before the boundary were within
0.36 seconds of their server timeline. After the boundary both contexts continued
decoding the second original. An intentional local pause left the other session
playing; the paused session exposed Rejoin current program. These controls did
not change the stored schedule.

Independent recovery receipt `20261003T044829Z` lives on NUC at
`/home/onnwee/Backups/vhness-private-pilot/20261003T044829Z`. Its 14,503,884-byte
metadata dump has SHA-256
`610a2cbd27ce3ac6a9e9fa70d318ea6d37537f1333ce64228581eed51b3dbfd4`.
All 11 table fingerprints matched local and NUC disposable restores. Every raw
snapshot passed its stored SHA-256 check. The preservation archive has SHA-256
`d6b5d5707deffc4ab5548cd61486fe2b447453870c4f7ee1fc9b79d15a3aba1d`;
both extracted Bill Nye originals matched their recorded SHA-256 values.

The verified candidate binary and frontend then served that independently
restored database and the extracted originals on NUC, through a loopback API
and an SSH forward. With Archive.org blocked, the restored library rendered,
Flight played and sought at beginning/middle/end, and the restored local channel
changed to Earth's Crust at 04:59:01 UTC. Actual restored video/audio decoding
and a rendered screenshot were captured. The same restored library retained
Democracy Now! source identity and dates; blocking its source produced an
explicit playback error with a source link. Metadata recovery does not recover
Democracy Now! video.

Evidence stays outside Git under the pilot's `evidence/` and `backups/` folders:
owner selection, source snapshot, per-file manifests/reviews, acquisition
receipts, channel observations, screenshots, and independent restore receipt.
The full isolated `make verify` gate passed, including Go race/PostgreSQL/HTTP
checks, TypeScript/lint, five frontend tests, and production build. Hosted CI
must qualify the preservation PR's exact committed head separately.

This evidence proves byte integrity, decoder progress, browser seeking, the
measured channel boundary, and independently restored application playback.
The owner separately confirmed clear audio, matching picture/audio timing,
and original picture proportions at the private pilot URL. That confirmation
did not name individual viewed titles. This report does not claim qualification
of other browser engines, a public deployment, or complete Democracy Now! item
metadata coverage. Continuous full-episode source viewing and collection-wide
inventory remain separate qualification gates. No remux/transcode was needed
for these compatible source MP4s; processing and caption acquisition require
their own manifest and storage budget before use.
