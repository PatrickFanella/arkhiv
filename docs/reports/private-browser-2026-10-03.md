# Private browser evidence — October 3, 2026

The owner authorized a separate local browser after the native T3 browser failed
to reach loopback. Qualification uses isolated named Chromium contexts against
the private Go API and verified React build, rather than the synthetic preview.
Media recording and response-body capture were not used for Democracy Now!.

Two source-streaming contexts joined `dn2026-1002_vid` near its scheduled end,
then changed to `dn2026-1001_vid` at 05:01:30 UTC. Eighteen paired observations
recorded direct Archive.org video URLs, decoded frames/audio, and each player's
offset against the server timeline. All ready, playing samples were within
0.28 seconds of that timeline; after transition both were within 0.08 seconds.
Each browser owns its player and local controls. On-demand playback continued
independently while the shared schedule changed.

The source on-demand player sought `dn2026-1001_vid.mp4` to 5, 1,770, and 3,530
seconds. Each seek completed at readyState 4 with increasing audio/video decoder
counters. Source CDN requests returned 206. A few requests logged
`ERR_NETWORK_CHANGED` during the broader run; playback resumed and continued.
This report does not turn a recovered source/network event into a zero-error
claim or a source availability guarantee.

The local Bill Nye sample passed source-blocked on-demand seeking and a two-session
scheduled transition. Its independently restored NUC application rendered local
metadata and played the recovered originals with Archive.org blocked. Blocking
the source-streaming Democracy Now! video instead produced an explicit error
and source link while preserving the restored catalog. See the
[preservation qualification report](bill-nye-private-2026-10-03.md) for the exact
selection, byte budget, checksums, and recovery receipt.

The owner confirmed in the T3 thread: "I checked; picture and audio are correct."
The question covered clear audio, matching picture/audio timing, and original
picture proportions at `http://127.0.0.1:47071`. Individual viewed titles were
not separately named. That perceptual confirmation is distinct from automated
decoder, transport, and clock evidence.

The 390-pixel layout rendered without horizontal overflow; native video controls,
source link, unknown episode-date label, and collection selection remained
available. Bill Nye video uses contain sizing to preserve its source proportions.
Rendered screenshots were inspected. This is Chromium on Linux evidence, not
Safari/Firefox or mobile-device qualification.

Earlier viewing windows were closed by the owner and their interrupted runs do
not count as complete-episode proof. A separate background context now observes
a new Democracy Now! episode at normal playback speed from the beginning, with
audio/video decoding and periodic continuity receipts. Its natural end and the
complete 6,626-identity metadata queue remain pending. No issue requiring those
checks is closed by the shorter observations in this report.

Raw evidence remains outside Git under
`~/.local/state/vhness/private-pilot/evidence/`: source-channel observations,
local and restored-channel observations, source HTTP headers, screenshots,
owner perceptual confirmation, and full-episode observations. The operator
reports retain source identity and proof boundaries without committing source
metadata or media.
