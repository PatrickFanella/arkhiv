# Private audio migration inputs

`scripts/prepare_audio_migration.py` prepares catalog, representation review and
continuous-audio receipts from an independently restored legacy music pilot.
It does not acquire media, change originals, import a database or activate a
schedule. It verifies every retained backup checksum and complete exported table
counts before allocating an exclusive output directory.

Pass the backup directory, complete legacy table export, restore receipt, a new
output directory and the private collection ID. The backup contains
`manifest.json`, `catalog.dump`, and retained `data/sources`, `data/originals`,
`data/media` and acquisition receipts. The restore receipt records independent
and recovery hosts, verified file count, dump checksum and every table count.

Original recording identities, concert identities, source and reviewed playback
rows, tracks, file variants, reviewed sets, warnings, source/taper/transferer
credits and legacy schedule stay retained. Metadata-only recordings get no
transport review. Existing published private playback gets a new playback-only
review. Download, acquisition, relay, scheduling and public publication are
separate actions and remain disabled in this migration.

A retained source file may differ from its restored database observation. Both
JSON responses are included with separate hashes. The catalog binds to the
restored observation; a database-only observation is explicitly serialized as
new migration bytes. The plan retains the historical database hash, the retained
file hash, the new catalog hash and their equality result. It never describes
new serialization as preserved original network bytes.

Each continuous derivative binds the independently verified selected original
tracks, ordered manifest checksum, retained derivative checksum and duration,
recovery receipt and original extraction tool. `register-audio` must still
verify every input, current rights, byte counts and the actual derivative's
bounded duration probe. Original and derivative mounts must be separate roots.
The existing app, dump and restored legacy database own historical decisions
and schedules until a separately reviewed cutover.

The required repository gate covers corrupted originals, unreviewed recordings,
differing source observations and action separation. Actual private browser
decoding, independent common-platform restore, audio fidelity and public release
are separate evidence.
