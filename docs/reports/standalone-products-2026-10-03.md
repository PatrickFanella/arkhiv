# Standalone product qualification — October 3, 2026

Three independently deployable frontends use the shared VHNESS engine at revision
`1980c704e7495557c06fb9e6ac63e47ecc244dab`. Private deployments on Kvant use
separate databases, credentials and immutable UTC replay schedules. Public ingress
is not configured. VHNESS naming and its eventual identity remain undecided.

| Product | Repository and reviewed frontend revision | Private preview |
| --- | --- | --- |
| Configurable white-label template | [video-archive-template](https://git.subcult.tv/subculture-collective/video-archive-template), `0cd10bbe7a8fdf1371da1d8123ff5036637dd1d0` | http://127.0.0.1:48071 |
| Democracy Now! broadcast archive | [democracy-now-tv](https://git.subcult.tv/subculture-collective/democracy-now-tv), `03195eabf9ebc7e086f9ec63babba0511c08036b` | http://127.0.0.1:48072 |
| Bill Nye science television | [bill-nye-tv](https://git.subcult.tv/subculture-collective/bill-nye-tv), `ae0933cf2d1373bacd3fec94f0cc5c0aab1e6bf1` | http://127.0.0.1:48073 |

The template accepts site JSON with a collection, title, introductory copy, local
logo and six colors. A browser configuration fixture changed the document title
and accent color, then restored the default. Democracy Now! uses the supplied
vector identity and Amy Goodman photograph, recorded with filename/hash
provenance. Bill Nye ships the supplied series logo; other inspiration is recorded
separately from shipped assets. Its CSS CRT cabinet, power control and optional
static picture effect preserve the original video proportions. Supplied assets
do not establish official affiliation or public-release clearance.

## Catalog and preservation

Final metadata reconciliation accounted for all 6,626 currently discovered
Democracy Now! identifiers: 6,625 imported and one explicitly rejected.
`dn2008-0717_vid` has the invalid source date `2008-17-17`; its complete raw
response remains retained. No correction was invented. There are 460,624 complete
file entries, 6,830 raw snapshot records including 202 collection/discovery
responses, and 6,866 observations. These are dated source counts, not permanent
collection limits. Full metadata includes records without playable media.

The standalone library has 6,582 privately reviewed playback/channel entries.
Another 43 normalized items have no eligible MP4 with a reported duration and
remain unenabled. Review history is independent of raw source data. This review
does not prove decoding or availability for every one of those 6,582 videos.

Democracy Now! remains `source_stream` with zero acquisition budget. Its two
standalone deployments have empty media roots. Actual browser video requests go
directly to Archive.org; the API returns references and never proxies,
persistently caches or transcodes those videos.

The owner's full-series instruction superseded the initial two-episode Bill Nye
pilot. The reviewed selection is exactly all 100 source-provided MP4 derivatives,
13,763,456,996 media bytes (about 12.82 GiB). The 100 duplicate AVI originals were
not acquired. A compare-and-swap budget review raised 512 MiB to 16 GiB, retained
the 1 GiB free-space reserve, and stored the authorization/history. Measured free
space on both hosts passed the primary and independent backup/extraction reserve
checks before transfer. Originals remain under
`~/.local/state/vhness/private-pilot/media/bill-nye`, outside Git.

All 100 files passed supplied MD5/SHA-1 checks, local SHA-256 verification and the
integrity audit. All 100 also have readable durations, H.264 video and an audio
stream according to bounded ffprobe inspection. This does not prove perceptual
fidelity for every frame of the full series. Episode titles follow the source
filenames in numeric order; individual broadcast dates remain unknown rather
than inheriting the series upload date.

## Recovery and release artifacts

NUC backup `20261003T063503Z` independently restored all 100 Bill Nye originals
plus two clearly generated adapter fixtures, with SHA-256 checks for all 102
originals. The 13,764,433,920-byte preservation archive has SHA-256
`0bf1aafe198e0b7926bf98d1f031ed89b444f10bddf20c0cdf35319ec6811beb`.
Its metadata was an earlier partial Democracy Now! checkpoint, not full coverage.

The completed inventory backup `20261003T065219Z` contains full source coverage.
Its metadata dump SHA-256 is
`449060e9106cc8a6d77590beacf17b19d4c4eaed5c3d4a072d3add7077b2bf3a`.
Local and independent NUC restores matched all 12 table fingerprints, found zero
invalid raw snapshot hashes, and again verified all 102 preserved originals.
The metadata completion job finished at 07:27:23 UTC.

Checkpoint `20261003T070130Z` additionally includes the private full-library
reviews and Bill Nye display-title reviews. Its 148,490,108-byte dump SHA-256 is
`64ede27e8366c09966e35ca43bbfa52e6ae886d81971a3d3f88b3a292244fe79`;
all 12 independently restored table fingerprints matched.

Collection-only v2 exports preserve each product's complete raw snapshots,
rejected-item evidence, response history, decisions and applicable budget history.
The export tool restores into its own disposable database, verifies the retained
raw identity set before/after filtering, and never edits the source database.
Democracy Now! export: 138,633,866 bytes, SHA-256
`c2de112c93ea3c43d3675cc83814ba5f507603c9bd6ca6d42c005d8356326109`.
Bill Nye export: 9,874,948 bytes, SHA-256
`e5814e103195e4ef8e66db7841564f1c8ab5605923013831ae9c55ea799841a8`.
Earlier v1 exports omitted collection responses; they are retained as failed
evidence and excluded from the qualified release bundle.

The private qualified release directory is
`~/.local/state/vhness/releases/standalone-20261003/qualified`.
It contains pinned source archives, engine/product images, the two v2 dumps,
deployment instructions, proof summaries and a SHA-256 manifest. Its independent
copy is on NUC below `/home/onnwee/Backups/vhness-private-pilot/standalone-20261003/qualified`.
Credential values and media bodies are excluded from source and release Git
commits. Preserved Bill Nye originals remain in the separate verified backup.

## Executed checks and proof boundaries

The full isolated engine gate passed formatting, vet, race/build, disposable
PostgreSQL and HTTP regressions, frontend type/lint/tests and production build.
Each frontend passed frozen dependency, generated adapter/tool/asset hash,
type/lint/test and build checks. Template tests include configuration validation.
Local Docker builds succeeded. Final engine hosted runs 12215/12216 passed. Final
Democracy Now! runs 12217/12218, Bill Nye 12219/12220 and template
12221/12222 passed. Earlier concurrent global-pnpm failures are retained; the app
workflows now install job-local pnpm. One setup-node cache failure passed on retry.

All three private deployment health checks pass. Containers run as user 1000
with read-only roots and read-only media mounts, dropped capabilities and bounded
resources. Foreign collection/catalog/playback requests return 404; operator
commands reject a different collection. A replay renewal preserved the exact
ongoing program tuple. Empty-database restore guards prevent replacing an active
deployment database.

Fresh Chromium contexts decoded source video in both streaming apps and local
Bill Nye video. ReadyState 4, advancing video frames and audio bytes were
observed. Two independent sessions of each branded app passed mid-program joins,
pause/Rejoin, on-demand seeking, volume and empty search. A real-duration
near-end schedule transitioned from Friday to Thursday and Flight to Earth's
Crust at 07:26:09 UTC; both sessions decoded the new media without a viewer
changing the shared clock. After-boundary drift was below 0.08 seconds, against
the declared three-second threshold.

Screenshots of the actual deployed layouts were inspected at desktop and 390
pixels. Both branded mobile layouts have no horizontal overflow and use contain
video sizing. Keyboard Tab reaches the skip link; semantic buttons, inputs,
replay labels, dates/source links and optional controls are present. Bill Nye
power-off unmounts the player; power-on restores it. The optional effect can be
enabled/disabled. Reduced-motion episode selection uses automatic scrolling;
all three final adapters were checked with the reduced-motion media preference. Video fullscreen entry and exit passed in the source app.
This is Linux Chromium evidence, not physical mobile, Safari or Firefox proof.

A synthetic browser-only reviewed-caption/VTT fixture produced a native showing
text track with the expected active cue, using a temporary blob while video
continued as a normal non-CORS media request. Off disables the track. Malformed
VTT produced an explicit caption failure while video/audio decoding continued.
No fixture caption decision was persisted, and no source-provided caption
availability is claimed. A catalog 503 and incorrect engine collection produced
explicit unavailable states.

The actual branded Bill Nye image ran on NUC against the independently restored
100-episode database and originals. With Archive.org blocked in that browser,
episodes 1, 2 and 100 decoded and sought successfully through local media URLs.
NUC's exhausted Docker network pool was not pruned or reconfigured: its bounded
disposable database uses network-none and a private Unix socket. The restored
app binds loopback only.

The earlier full-episode qualification reached natural end after 3,542.016
seconds at speed 1, with 116 continuity samples, 106,154 decoded frames, zero
dropped frames, advancing audio and no nonterminal pause/backward seek. It
qualified the earlier engine/player candidate; fresh standalone playback and
transition checks supplement it. It is periodic automation, not human
frame-by-frame viewing. The owner's earlier confirmation, “I checked; picture
and audio are correct,” applies to the private pilot they viewed, not all 100
Bill Nye episodes or every new screen.

## Operating limits and next decision

The private channels have reviewed schedules through October 10. The operator
must renew them with `plan-replay --days 7`; no new host renewal, ingestion or
backup timers are silently installed. Deployment instructions explain scoped
content refresh, private review and explicit schedule renewal. Gaps, exhausted
schedules, unavailable source media and autoplay restrictions remain visible.

Public domains, ingress, approved asset/source use and concurrency remain
[#22](https://git.subcult.tv/subculture-collective/VHNESS/issues/22).
Optional remux/transcode and locally preserved captions remain extensions in
[#34](https://git.subcult.tv/subculture-collective/VHNESS/issues/34); the selected
compatible source MP4s do not require processing. There is no automatic deletion
of originals, history or backups.

The old port-47071 qualification API was stopped after its metadata job and
receipts completed; its database, media, logs and immutable evidence remain.
The three standalone previews are the current review targets. Raw machine
receipts and screenshots live under `~/.local/state/vhness/standalone`.

Final release checks: a reviewed template-image rollback and forward upgrade preserved all 6,582 catalog entries and the exact ongoing program. The final pinned Bill Nye image repeated NUC source-blocked decoding/seeking for episodes 1, 2 and 100. All 27 final release artifacts passed independent NUC SHA-256 verification. Compatible-MP4 adapter #20 and independent preservation qualification #21 are closed; their optional processing criteria were moved intact to #34.
