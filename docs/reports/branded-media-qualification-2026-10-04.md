# Private branded media checks, 2026-10-04 UTC

These checks use independently restored donor databases and retained media on
Kuznya. All previews are loopback-only. No acquisition, public cutover, approval
expansion or production restart occurred. Browser evidence uses isolated Chromium;
iOS and Android devices are unavailable in this Linux environment.

## Print consumers

Original print PR #59 and Riot Grrrl PR #10 merged after their Reader.tsx conflicts
were resolved against current main. The shared PrintReader adapters preserve
existing document/variant/page URLs and server search. Each PDF.js bridge retains
the new 32 MiB decoded scan-area bound. Both exact worktrees passed the full
isolated `make verify`, including 15 operations tests and their frontend, Go,
database, lint, build and Compose gates.

The original print preview uses source
`86680b30a75a7fbe24932b624cf6d9ad013daba8`, frontend image
`sha256:a6fd77adb93658bcd988e64c62c328993fd872e6703f31dcc49a704d44cf102c`
and API image
`sha256:6db78ad971d6bdf0daf547994114e3a7a0b39ccf450784a60df9c7efd5830aba`.
The restored public-filtered catalog lists 699 documents. Little Green Songbook
SCREEN opened locally, moved to page 2, and found two matching pages for “green”.
The second search result jumped to page 8 and updated the URL. At 390 pixels,
the rendered canvas was 355 by 230 pixels with no horizontal overflow or alert.
Its original download returned HTTP 200, 9,337,316 bytes, attachment disposition,
and SHA-256 `ae58f7cdf74f6fac70fe433499a806f06e01fb389ae7585fc1be45cad4edef02`,
matching the catalog and ETag. No browser resource request reached Archive.org.

Riot Grrrl uses source `99db53d1f7ae84704a18418d6384c143dbb08ec0`,
frontend image
`sha256:2f0546d4ddf203ecab9a11d1d9a49edcbb24a1c9a22e96aad9c5549b742ce1d6`
and API image
`sha256:aae6c98c7856d10ef0db26365a66234296d557bc1ebc1efa024c8371f2f2b4dc`.
All 23 restored variants retain false public approval flags. Public-filter mode
returned no collection; the private donor mode lists all 23 documents without
changing those flags. Violet Volume 4 rendered its original scan at 390 pixels
with a 355 by 474 canvas, no horizontal overflow and no reader alert. Its
whole-document OCR search found one “violet” section and explicitly reported
that page mapping is unavailable, without a fabricated page-jump control.
The original PDF download returned HTTP 200 and attachment disposition,
3,327,470 bytes and SHA-256
`2d2e7a833cfff4f0eadf9dc0194b65e59be7af815467f6c5981ebbc25c929bb5`,
matching its catalog checksum while public approval remains false.

The restored archive directories retain their owner and permissions. API
containers run as that owner with read-only mounts; Nginx runs as its own UID
with owned temporary directories and no extra capabilities. Failed startup
attempts remain alongside the corrected previews and receipts.

## Print rollback with retained images

The actual running original print and Riot Grrrl API/frontend images were exported
read-only from Almaz and loaded on Kuznya. Their image IDs matched exactly. The
retained image archive is 96,316,416 bytes, SHA-256
`d8015d28a7883f9391987789c60f4298f459a9373d2cd4fc0e01666b1a3b9bca`.
Existing Almaz containers were neither stopped nor restarted.

Each current private qualification database was checkpointed before a further
upgrade. Separate new PostgreSQL instances restored these checkpoints with
`pg_restore --exit-on-error`; every row count across all 17 public tables matched.
Original print checkpoint: 19,064,025 bytes, SHA-256
`ac5edc88f2002d250493b10cfb35b2e1029fb08ba735ac9c19b2670e54381d00`.
Riot Grrrl checkpoint: 855,413 bytes, SHA-256
`d81d6131883d0fcfcd59df9001dd96828af2822cac83b531d27e81f5b68292d2`.
These copies preserve snapshots, discoveries, inventories, reviews, source changes,
public-access history and original asset identities, alongside all retained files.

Previous original-print API image
`sha256:03e2ad6e8e78c6459f49ce9e017166722a35480693dcd9959240e82b9be8dfea`
and frontend image
`sha256:cc570cf164b190b2c579e8343f56a6f262601d56432e0caba428f0d4c2048693`
started against the original-print rollback copy. The old reader opened Little
Green Songbook SCREEN and advanced to page 2, with a 918 by 594 rendered canvas
and no reader alert. Its original download returned HTTP 200, 9,337,316 bytes and
the same `ae58f7cd…edef02` checksum recorded above.

Previous Riot Grrrl API image
`sha256:672ad7d16af1debf70664d6118c84d43e1ddf983785c208e399d2069fc320567`
and frontend image
`sha256:11d60ecb840ea9014355787667cad27ed469835ae997400ab2e0aa9024eae53e`
started against the Riot Grrrl rollback copy. Violet Volume 4 opened and advanced
to page 2, with an 864 by 1,152 canvas and no reader alert. Its original download
returned HTTP 200, 3,327,470 bytes and the same `2d2e7a83…929bb5` checksum above.
Public approval remains false. All rollback apps are private and mount the
independently restored archives read-only.

These rehearsals prove the previous print artifacts can read the preserved
checkpoint and files. They do not activate a production rollback or qualify an
unimplemented destructive schema migration. Video and audio rollback remain open.

## Music consumer

Dedhed PR #27 merged the shared transport hook into the existing Tape Shelf.
Runtime image source `def0ff59df292f4896f3fb51109e12877ea00f90` adds independent
API/frontend image definitions in PR #28 and passed full isolated verification.
The frontend image is
`sha256:d7fa76350aba1279f480f0bf9162c4203fb86c084d4b3358798e2bfe68415f5f`;
the API image is
`sha256:0991e228dd2c2badf3f21b2433ac9798c48b220af30bedfb5fac77f5867eb6a2`.
The fresh database retains all eleven pilot recording rows and original snapshots,
then applies the current additive inventory migration. The approved library
shows Fillmore and Cornell with their sets, taper/transfer credits and original
recording URLs. The old pilot lacks the later collection-wide inventory;
calendar coverage cannot be claimed from this restore.

Fillmore decoded at ready state 4, duration 3,418.3025 seconds, and advanced to
35.787529 seconds without a decoder error. Playback crossed the first reviewed
track boundary at 324.3015 seconds and reached 326.009028 with the same continuous
media URL. This proves transport continuity, not subjective fidelity.

The restored station schedule was expired and returned HTTP 503 with an off-air
interface. Before a private rehearsal, its original JSON and complete fresh
checkpoint were retained. A new private schedule uses the already reviewed
private recordings; private-only remains true and public relay remains false.
Late joining Fillmore produced server offset 3,340.851770973 and playback time
3,340.740412. With no further selection, the schedule advanced to Cornell;
playback was active at ready state 4, time 164.420763 against offset 164.548799466,
0.128036466 seconds apart, with no decoder error. Public relay and owner listening
remain separate gates.

## Local video consumer

All 100 Bill Nye originals were restored from the independent NUC archive and
passed their retained checksums. Their total is 13,763,456,996 bytes, within the
reviewed 16 GiB cap. The separate collection-only database was restored and
migrated before starting source `d77639cdbff9a09766b51cfbf4dba668232c77e7`;
frontend/engine image
`sha256:3a9d21ab00f598e49e02185032098fa4f01e6c6ba67bc1f1cf4a3a622c977791`
retains published viewer/engine 0.1.1. This is initial adoption evidence;
0.2.2 adoption requires its own artifact and checks.

The branded episode guide listed 100 episodes. Flight played from the local
restored media endpoint at ready state 4, duration 1,380.780781 seconds, and
advanced to 24.463695. Seeking to 90 seconds succeeded. At 390 pixels there was
no horizontal overflow, the optional CRT effect was off, and no Archive.org
resource request occurred. Turning the television off removed the video and
reported the off state; turning it on restored the interface.

A private reviewed schedule used Flight followed by Earth's Crust, preserving
full source durations. After tuning in, the schedule advanced to Earth's Crust
without another selection. Playback was active at ready state 4, time 17.197837
against server offset 17.301998833: 0.104161833 seconds apart, with no decoder error.

## Remaining qualification

Release 0.2.2 passed hosted run 12680 and registry-byte verification. The first
automated consumer run verified and pushed the video-template upgrade, then
lost the HTTP response while creating its PR. Readback confirmed PR #4 exists.
The remaining five proposals and all six actual 0.2.2 consumer adoptions still
need qualification. Video and audio deployed-artifact rollback, reviewed music listening and sustained
natural runs remain open. The print rollback checks above cover the retained
artifacts and fresh checkpoints separately from 0.2.2 adoption.
Source-stream-only Democracy Now! remains outside local preservation. Public
activation requires the concrete per-product decision recorded in the P6 review.
