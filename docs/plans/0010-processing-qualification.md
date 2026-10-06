# Optional processing qualification

Issue #34 requires a separate acquisition and processing decision. The existing
100 selected Bill Nye MP4s already decode unchanged. The retained complete source
response lists 100 AVI originals and no `.vtt`, `.srt`, `.ass` or `.ttml` caption
files; absence of those entries does not establish whether a source contains
embedded captions. Democracy Now remains permanently source-streaming only.

## Proposed private scope

Use a distinct `bill-nye-processing-qualification` local-preservation collection,
with every public/read/play/download approval false. Select only `01 Flight.avi`
from `billnyethescienceguythecompleteseries-07-28-2021`, bound to retained source
response SHA-256
`cc7f5da848a73c022a7170e662e395c06cff4157b31947cff0c68d5b61c6670a`.
The source reports 234,848,256 bytes, MD5
`91b3b5e772b76c3a03b5238103ff114a`, SHA-1
`2e6697670a11e4fda83515d17017a3764a60dc5c`, duration 1,380.75 seconds and
512×368 pixels. These are source observations, not a verified local probe.

Cap generated derivatives at 384 MiB, the combined original/derivative/receipt
working store at 768 MiB, and reserve at least 16 GiB free on every participating
filesystem. Provide a separate 768 MiB independent-backup allowance; fresh restore
qualification needs an additional explicitly bounded 1.5 GiB workspace. Count
partial and failed output against the allowance and never prune originals or
existing qualified evidence to obtain room.

Before acquisition, refresh complete metadata and require the exact selected
filename, reported size and checksums to remain unchanged. Acquisition uses the
explicit preservation manifest and current local-preservation policy guards;
metadata workers never fetch media. Verify the received original and retain it
immutably. Probe actual codecs, frame rate, sample/display aspect ratio, audio
layout, duration, timestamps and any embedded caption streams before selecting
remux or transcode. Preserve display proportions rather than treating reported
pixel dimensions as square-pixel display instructions.

Prefer a lossless remux when a browser-compatible codec combination permits it.
Otherwise prepare a bounded H.264/AAC MP4 derivative with explicit parameters,
tool identity and original/derived SHA-256 binding. Exceeding output or workspace
caps fails the operation without making the partial result eligible for serving.
Record full decode checks, audio/video timing and any retained verified caption
tracks. No caption text is invented to fill missing source evidence.

Derived representations need independent source/editorial/access decisions and
processing receipts. The existing approved MP4 collection must continue to use
its unchanged originals and schedule. Test the derivative privately in on-demand
and scheduled viewing, preserve original dates, then restore originals, derived
bytes, captions and receipts into an independent fresh database/store before
claiming recovery. Browser playback and human audiovisual assessment remain
distinct from probe/decode checks.

## Owner decision

On October 6, 2026, the owner approved the bounded private one-AVI pilot in this
T3 thread. That approval covers the selected original, derivative, working and
independent backup/restore allowances above. Acquisition must still pass the
current metadata, manifest, policy, checksum, capacity and reserve guards.
This approval does not activate public access or deployment. Existing
unchanged-MP4 delivery remains qualified; Democracy Now stays streaming only.
