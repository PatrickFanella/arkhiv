# Private music migration and interface checks, 2026-10-04 UTC

The common engine imported eleven retained Dedhed recordings from the independent
NUC pilot backup. Two already reviewed private recordings received playback
access. No download, public relay, acquisition or active common schedule was
approved. The original station state remains in the donor dump and complete
four-table export.

The converter retains all tracks, credits, warnings, review decisions and legacy
identities. It applies reviewed titles and sections over the playback view while
preserving the original rows. This was necessary because Cornell's stored
playback view had no sections while its effective donor catalog had Set I,
Set II and Encore. A regression check covers that distinction.

One source has a retained-file/database discrepancy: Darby 1991-11-03. Both
versions remain separate, with their respective hashes; the exported database
serialization is explicitly a derived observation. The remaining ten retained
source files match their restored database source bytes.

The independently restored input contains 101 files totaling 601,174,116 bytes.
Both continuous derivatives were registered only after the server verified every
original track input, derivative checksum and measured duration. Cornell has
21 tracks and a 10,166.24-second AAC recording; Fillmore 1969-06-21 has twelve
tracks and a 3,418.3025-second recording. Known source cuts and splices remain
visible, alongside taper and transfer credits.

Isolated Chromium at 390 pixels rendered the reviewed sections and credits.
Cornell Set II started at its reviewed bound with decoder ready state 4. Playback
advanced through the Scarlet Begonias / Fire on the Mountain boundary using the
same continuous media URL, without a decoder error. Fillmore similarly advanced
through a track boundary. These are transport/decode observations; subjective
sound quality and listening remain a separate owner check.

The first cold audio resolution exceeded the old ten-second frontend limit.
Source `17565d01903c019ba6610943ce5f1ac90910e93b` permits a bounded sixty seconds
for continuous integrity checking, retains normal metadata deadlines, and offers
an explicit retry. An injected twenty-millisecond browser deadline produced the
readable error and no stale audio. Restoring the normal deadline and pressing
Try again loaded Cornell with all three reviewed sections and no alert. Download
resolution and a different collection scope returned 404; approved audio
resolution returned 200.

The corrected common database and complete retained media were backed up to NUC,
read back, then restored into `platform_v3_recovery` on Kuznya. All 24 table
counts matched: eleven catalog items/representations, twelve raw snapshots,
two audio assets and their histories, and eight schema versions. The recovery
archive is 603,443,200 bytes, SHA-256
`69f58d48a213a5b74771ea9e6a2d0304cabb0fe2e967476e6e06d8d9c4a42aa8`.
The 563,670-byte common database dump has SHA-256
`5dba68600253eb9ca995db9560bff520702a49c6b1ce6babb123250e9a752a11`.
The corrected image was started against that fresh recovery database and its
read-only restored media. Fillmore decoded with ready state 4 and advanced to
0.926487 seconds, with a 3,418.3025-second duration and no decoder error. The
corrected preview independently played Fillmore at ready state 4 and advanced
to 1.13412 seconds. The original preview URL now targets the corrected build.
Original and earlier rehearsal apps remain available. Old donor artifact
rollback, branded listening and sustained station runs remain separate
qualification work.
