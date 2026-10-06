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
