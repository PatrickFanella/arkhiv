# Private video processing

`scripts/process_video.py` processes one previously acquired AVI original under
an explicit local-preservation manifest. It cannot acquire media, update a
catalog, grant playback access or publish an asset. Democracy Now source media
is rejected even when the manifest assigns it another collection name.

Keep the original, complete source snapshot, manifest and new output directory
inside one bounded working store outside Git. Existing output directories,
symlinks, source checksum changes and mismatched source selections fail before
the operator allocates output. Failed output remains in place and counts against
the next run's working allowance. Never remove an original or prior evidence to
make a run fit.

The manifest contains these fields. The values below describe the approved
one-AVI pilot; use the exact retained original and source snapshot paths and
hashes from its acquisition receipt.

```json
{
  "approved": true,
  "approval_notes": "Owner approved the bounded private one-AVI pilot on October 6, 2026",
  "policy": "local_preserve",
  "collection": "bill-nye-processing-qualification",
  "source_item": "billnyethescienceguythecompleteseries-07-28-2021",
  "source_file": "01 Flight.avi",
  "snapshot_path": "evidence/source.json",
  "snapshot_sha256": "cc7f5da848a73c022a7170e662e395c06cff4157b31947cff0c68d5b61c6670a",
  "original_path": "media/EXACT_PATH_FROM_ACQUISITION_RECEIPT",
  "original_sha256": "EXACT_HASH_FROM_ACQUISITION_RECEIPT",
  "original_bytes": 234848256,
  "derivative_cap": 402653184,
  "working_cap": 805306368,
  "reserve": 17179869184,
  "public": false,
  "published": false,
  "playback": false,
  "channel": false
}
```

The approval field records a prior owner decision. Writing `true` does not
authorize additional acquisition, processing or storage. The operator verifies
the complete snapshot hash, selected original entry, source checksums, local
SHA-256 and byte count. Its hard maximum allowances are the approved pilot's
384 MiB derivative and 768 MiB working store, with at least 16 GiB free reserve.
It reserves another 16 MiB within the working allowance for reports and receipts.

Install the reviewed tools image separately. Processing resolves its immutable
local image ID and runs Podman without network access, with a read-only original
mount, a new writable output directory, a read-only container filesystem,
1 GiB memory, two CPUs, bounded file sizes and a one-hour runtime limit. A host
timeout or tool failure removes only that invocation's uniquely named container
before returning, so a failed processing attempt does not leave a writer running.

```sh
python3 scripts/process_video.py \
  --root /absolute/private/working-store \
  --manifest /absolute/private/working-store/manifest.json \
  --output /absolute/private/working-store/new-processing-attempt \
  --image REVIEWED_LOCALLY_INSTALLED_TOOLS_IMAGE
```

Browser-compatible H.264 yuv420p and AAC streams are remuxed. Other supported
one-video, one-audio inputs use H.264 CRF 21 and AAC at 128 kbps. The operator
does not resize, change frame rate or silently discard additional streams.
Embedded captions require a separate preservation plan and cause this operator
to refuse the input. It never generates caption text.

Success requires matching dimensions, sample/display proportions, frame rate,
audio layout, timing and durations, plus full original and derivative decode.
The operator rechecks the original and source snapshot before writing its final
read-only receipt. That receipt retains parameters, probes, tool identity and
input/output hashes. Playback, channel use, independent restore and human
audiovisual review remain separate checks.

The already qualified pilot derivative remains untouched. Running this operator
again beside that original and derivative would exceed its conservative working
preflight. No additional storage or second real-media conversion was authorized.

## Registering a private derivative

`vhness register-video --id EPISODE_ID --file /absolute/review.json` registers
one verified derivative against the current acquisition-only original. Generic
`review` cannot grant this access. Set `VHNESS_MEDIA`,
`VHNESS_VIDEO_DERIVATIVES` and `VHNESS_VIDEO_RECEIPTS` to three existing absolute,
non-overlapping roots. Keep all stores outside Git and the web asset directory.

The review pins `snapshot_sha256`, `original_file`, `original_sha256` and
`original_bytes`, then `path`, `sha256`, `bytes`, `duration`, `width` and `height`
for the derivative. Receipt fields are `processing_receipt_path`,
`processing_receipt_sha256`, `processing_receipt_bytes`,
`recovery_receipt_path`, `recovery_receipt_sha256` and `recovery_receipt_bytes`.
Paths are relative to their configured store. Supply explicit `published`,
`playback`, `channel`, `public` and `notes` fields. This pilot requires
`public: false`. Publication and channel use require playback approval.
Original publication/playback/channel/public flags remain false.

The processing receipt must bind the exact original and derivative hashes and
byte counts, successful full decodes and tool/image identity. The independent
recovery receipt must record `independent_host`, `fresh_restore_root`,
`fresh_database`, `all_hashes_match`, `original_immutable`,
`derivative_and_processing_receipts_restored`, `public: false`,
`original_sha256`, `derivative_sha256`, `processing_receipt_sha256` and an
ordered `caption_sha256` array when captions exist. The receipt records actual
completed recovery. It is not a substitute for performing recovery.

Registration rehashes each asset and receipt, probes both original and derived
streams under time/memory/output limits, checks storage/reserve allowances, then
revalidates the current source binding while locking the episode row. A changed
source, original approval or local preservation binding refuses registration.
Catalog and decision history retain the original AVI inventory. Playback
responses identify the processed MP4 separately with `processed: true`.
The pilot is excluded from public catalog projections and public approval.

Serving rehashes the original, derivative, receipts and any captions before
Range or conditional responses. A corrupt asset returns unavailable even when
a client supplies a previously valid ETag. Channel revocation does not revoke
on-demand access; playback or original acquisition revocation removes playback.
Viewer controls do not change schedules.

## Already preserved source captions

An optional `captions` array contains `file`, `label`, `language`, `path`,
`sha256` and `bytes` per sidecar. The file must be a non-private WebVTT entry in
the current complete source snapshot, with a pinned source MD5 or SHA-1. Its
actual preserved bytes must match those source checksums and the local SHA-256.
Each UTF-8 WebVTT file is limited to 1 MiB, with at most 16 tracks, and counts
against both the collection and working allowances. Every caption also needs
an exact hash in the independent recovery receipt.

Private playback exposes verified tracks through
`/api/video-captions/EPISODE_ID/INDEX`. The endpoint checks the complete media
review before serving a track, including conditional requests. These controls
do not acquire, convert or invent captions. A new sidecar acquisition requires
its own explicit manifest and storage decision. The approved Flight pilot has
no reviewed caption source and keeps an empty track list.
