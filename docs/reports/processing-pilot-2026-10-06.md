# Bounded private AVI pilot, October 6, 2026

The owner approved the exact scope in docs/plans/0010-processing-qualification.md.
A separate `bill-nye-processing-qualification` collection in an isolated
PostgreSQL 17 database retains the selected `01 Flight.avi` original. All
publication, playback, channel and public decisions remain false. The existing
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
