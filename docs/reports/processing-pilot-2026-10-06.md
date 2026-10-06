# Bounded private AVI pilot, October 6, 2026

The owner approved the exact scope in docs/plans/0010-processing-qualification.md.
A separate `bill-nye-processing-qualification` collection in an isolated
PostgreSQL 17 database retains the selected `01 Flight.avi` original. At the initial acquisition checkpoint, all publication, playback, channel and
public decisions remained false. The later private derivative review is recorded
below. The existing
100-MP4 collection and Democracy Now source-streaming policy are unchanged.

Refreshed complete metadata SHA-256 remained
`cc7f5da848a73c022a7170e662e395c06cff4157b31947cff0c68d5b61c6670a`.
Acquisition passed the engine's explicit manifest, local policy, quota, reserve
and source-checksum guards. The 234,848,256-byte original matched its source MD5
and SHA-1; local SHA-256 is
`ac7d70a74c69e6a83173b828cf374f3cca888688fff97ef59066f4bb9a1ee401`.
Source responses and every file entry remain retained. The original was acquired
before the storage-extension correction: its immutable opaque path ends in
`.mp4`, although its actual container is AVI. Its bytes/path/receipt remain
unchanged; new acquisitions preserve the actual selected filename's extension.

## Probe and processing

Source metadata labels the format Cinepack. The actual probe reports AVI with
MPEG-4 Part 2 Advanced Simple/XVID video and stereo MP3 at 48 kHz. Both
observations remain retained. Video is 512×368 at 29.97 fps, square pixels,
display aspect 32:23, duration 1,380.747414 seconds. Source audio duration is
1,380.744 seconds; both streams start at zero. No embedded subtitle stream was
found, and the complete source response lists no reviewed caption sidecar.
That evidence does not assert that every kind of caption encoding is absent.

FFmpeg 5.1.9 in the identified shared tools image transcoded with libx264 medium,
CRF 21, yuv420p, two threads, AAC 128 kbit/s and MP4 faststart. Explicit video/audio
maps preserve the measured display proportions; no frame-rate conversion,
picture resizing or invented caption text is applied. The output cap is
384 MiB; partial output counts against the 768 MiB working allowance and is
ineligible until probe/decode verification completes. Both original and derived
streams passed complete decode with fatal errors enabled. Derivative and source
container durations differ by less than 0.1 seconds, as do derivative audio/video
durations. This timing check does not establish perceptual lip synchronization.

The derivative is 198,483,729 bytes, SHA-256
`e5c2188519fa3c37d0ff64294a1ffed0f343f6b0b127252753fd6a028af25745`.
Processing took 516.466 seconds. Originals, derivative, metadata, probes and
receipts used 433,669,548 bytes before the small catalog dump. The measured store
stays below 768 MiB with more than 16 GiB free. The derivative is read-only.
Runtime data, raw responses, credentials and generated outputs remain outside Git.

## Remaining evidence

Independent NUC backup and fresh database/store restore completed: 13 exact
files totaling 434,015,328 bytes were verified on NUC's independent `/srv`
filesystem, frozen read-only, then copied back to a separate fresh local store.
All original, derivative, raw metadata, probe, processing and acquisition receipt
hashes matched. A fresh PostgreSQL 17 database restored the independently copied
dump and matched one snapshot, one decision-history row and one preservation
history row. Dump SHA-256 is
`6e03fb77a310c9873c54f9c04e944e524540032bb0ab974c26ecaa08975cf033`.
Both the 768 MiB independent backup and 1.5 GiB restore allowances were respected,
with more than 16 GiB free on each participating filesystem.

The first restore connection hit initdb's temporary-server shutdown. It wrote
no public tables; the retry waited for the final server and refused to restore
over a nonempty database. No existing database was reset. The completed recovery
receipt remains outside Git. Derived-media registration, private on-demand
and scheduled browser checks, caption integrity/serving behavior and human
picture/audio review are still required. A probe/decode result alone does not
close #34 or authorize serving the new derivative.

Required `make verify` and `make verify-curation` passed for the explicit original
approval path, including policy/refusal tests, real disposable PostgreSQL tests,
45 SDK tests, 56 Python tests and 19 frozen-lock curation tests. Hosted CI is
owner-waived during the Kvant outage. No live service, backup schedule, provider
or public access was activated.

## Reusable processing operator qualification

The reusable `scripts/process_video.py` operator passed a separate three-second
generated MPEG-4/MP3 AVI conversion using the retained tools image. Its actual
Podman run had no network, mounted the original read-only, enforced the processing
limits, and produced a 216,642-byte H.264/AAC MP4. Both complete decode checks
passed. The derivative SHA-256 was
`12c9b93c26f7182cd8c36676bf1aa99849cf3314e51b9b530325d9f866a8d55b`.

Seven operator tests cover source/policy/checksum refusals, compatible-codec remux
selection, source collection relabelling, capacity and reserve checks, failed
output retention, altered display proportions/timing, and unpreserved caption
streams and exact owned-container cleanup after a runtime failure. The fixture
uses generated test patterns and tones; it does not claim
human audiovisual qualification of the Bill Nye original. The real pilot
original, derivative and independently restored evidence remain unchanged.
See the [operator instructions](../reference/private-video-processing.md).

## Private registration and restored playback follow-up

The October 6 follow-up registered the exact previously qualified derivative in
the isolated pilot database. `register-video` verified the immutable original,
processing receipt and independently recovered original/derivative hashes, then
compared real probes under bounded resources. Source files and raw responses
remain unchanged. The original stays acquisition-only; a separate nested review
approves private derivative publication, on-demand playback and channel use.
Public approval is refused. No branded interface, live collection or shared
production schedule changed.

A private two-program schedule uses the processed file on the pilot collection
only. Desktop on-demand playback decoded 474 frames with zero dropped frames
and 263,038 audio bytes in the recorded sample. Desktop channel playback reached
2,101 decoded frames and 1,062,169 audio bytes, with six dropped frames in the
later sample. Both report the actual 512 × 368 dimensions and approximately
1,380.781-second container duration. The player distinguishes on-demand access
from scheduled replay and displays the unknown original broadcast date.

Android 15 Chrome on the isolated qualification emulator decoded 2,014 frames
and 1,105,053 audio bytes during on-demand playback. Its scheduled sample
decoded 2,223 frames and 1,299,190 audio bytes. These samples dropped 102 and
129 frames respectively. They establish browser decoding, not smooth playback
on a physical device. Recorded resource entries in desktop/Android checks
contained no external requests. The pilot exposes zero caption tracks because
its complete source inventory has no reviewed caption source.

The registration, recovery binding and schedule were backed up as a new small
read-only addition beside the existing independent NUC copy. The combined
independent store measured 434,367,918 bytes with more than 16 GiB free. The
new catalog dump SHA-256 is
`02c50e128a113f67cff53c50de946fc0c3f53b45ca62acf0c192c435920fb935`.
Old snapshots and receipts remain unchanged. Restoring that independently copied
dump into a new empty PostgreSQL 17 database retained one snapshot, two decision
history entries, one preservation history entry and one schedule. All transferred
files matched; the original and derivative were reverified against the earlier
independent restore. Combined recovered data measured 434,371,647 bytes, below
the 1.5 GiB restore allowance, with more than 16 GiB free.

The recovered browser decoded 376 on-demand frames and 208,050 audio bytes with
zero dropped frames. Scheduled recovery also decoded advancing video/audio;
its sample reached 545 frames and 364,016 audio bytes, with two dropped frames,
then reported buffering. A later sample resumed scheduled replay at ready state
4 with 877 decoded frames and 1,314,703 audio bytes. This is a successful recovery and browser decode check,
not a sustained playback qualification. Human picture/audio synchronization and
perceptual review remain outstanding.

Source WebVTT handling has separate domain, integrity and disposable PostgreSQL
API tests. They cover source checksum binding, labels/language, quota, independent
receipt binding, actual track responses, corruption before cached responses,
revocation and public refusal. These synthetic fixtures do not claim a recovered
real caption track or authorize additional acquisition.

The final `make verify` and `make verify-curation` passed through the shared
`vhness` profile with its disposable PostgreSQL database, Go race/vet/format
checks, 45 SDK tests, 63 Python tests, frontend build and 19 frozen-lock curation
tests. The profile tooling and owned Unix API socket were recovered after the
T3 restart. Hosted CI remains owner-waived during the Kvant outage. Issue #34
remains open for perceptual review, sustained qualification and real caption
recovery evidence when a separately approved caption source exists.
