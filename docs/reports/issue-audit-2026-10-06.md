# Issue audit and mixed-media chat qualification

This follows the October 5 cross-media report. Work began from merged main
`3cfaa6b` in `feat/issue-audit` on Sputnik. Qualification used the current dirty
source in isolated T3 `vhness` snapshots, with rootless Podman keep-id mapping.
Runtime fixtures, credentials, source snapshots and generated outputs stayed
outside Git.

## Shared changes

Audio station discussion has an explicit `media=audio` room lane. Its current
program binds the collection, schedule revision/start and catalog
item/representation. Existing requests without a media parameter remain video.
Messages, request receipts, cursors, posting restrictions and moderator rosters
are separated by media; mixed collections do not share a conversation merely
because their collection ID is equal. Print has no invented scheduled program.

Audio read/post switches default off. Enabling posts requires audio reads, an
explicit `audio_moderators` roster of issuer/subject hashes and a moderation
contact. Existing video configuration remains compatible. Viewer controls never
change either schedule. The reference audio panel requires synchronized playback
before posting; paused, buffering and on-demand states are read-only.

Personal migration `0002_media_rooms.sql` is additive; migration 0001 is unchanged.
Old video records and retry digests remain valid. Recovery checkpoints/journals
keep audio and video restrictions independently, while message/account deletions
still remove the exact opaque target. Restore upgrades a legacy personal-only
snapshot before replaying the current deletion/moderation journal. It never
resets or migrates the archival source database.

Legacy print APIs now carry `original_date` independently of editorial overrides,
and apply creator/date bounds before counts and pagination. Legacy audio APIs
apply venue/date bounds and search source/artist/taper/transfer/track titles before
pagination. Unknown/partial dates stay visible when bounds are empty. Invalid or
reversed complete date bounds are rejected. Scoped publication/transport checks
still select the eligible catalog before compatibility projection.

## Required checks

`make verify && make verify-curation` passed (exit 0) through the reviewed
rootless T3 launcher: Go formatting/vet/race and disposable PostgreSQL integration,
39 SDK tests, packed ESM consumer checks, 51 Python tests with the two independent
restore tests intentionally skipped in the ordinary suite, four reference tests,
frontend build and 19 curation tests plus lint/format/JavaScript checks.

The independent personal restore command also passed (exit 0):
`RUN_PERSONAL_RESTORE_TEST=1 python3 -m unittest discover -s scripts -p test_personal_restore.py`.
Its two real PostgreSQL 15 dump/restore cases cover a current audio withdrawal and
a legacy 0001 snapshot upgraded before both video and audio restriction replay.

Disposable integration covers same-collection lane separation, retry identity,
foreign media cursors, stale/foreign program identity, paused/on-demand posting,
rate limits, distinct moderator authorization, scoped withdrawal/restrictions,
emergency switches and checkpoint recovery. Print's 30-row fixture returns 27
bounded matches and seven records after offset 20 despite an editorial date of
1900; unbounded creator matches retain the two partial/unknown source dates.
Audio's equivalent multi-page fixture verifies 27 bounded and 29 unbounded rows.

## Browser observations

Native T3 preview at loopback port 35131 used generated video patterns, PCM tones
and a PDF, plus a generated RSA OIDC provider. The real session service verified
nonce/PKCE and used a separate personal database with a restricted serving role.
No live provider was selected or enabled.

A signed-in viewer joined the station. Audio reached readyState 4 with advancing
time; posting became enabled only in the synchronized state. A message containing
literal `<b>` text displayed as text with the original date and replay identity.
It did not appear in the concurrent video discussion. Pause displayed “Station
continues” and disabled posting. A separately signed-in fixture moderator
reported the message, blocked/unblocked its author, withdrew it and resolved the
report. Withdrawal removed it from the serving message response. An injected
chat-only 503 outage produced visible discussion errors while the station audio
remained unpaused at readyState 4; playback was independent of discussion polling.

The first report-resolution check exposed a stale client report list. The shared
hook now refreshes reports after moderation. The updated browser showed “Reopen
report” after resolution and “Resolve report” after reopening. SDK build,
typecheck/lint and all 39 tests, plus reference typecheck/lint, passed after this
follow-up correction (exit 0).

Fixture launcher retries preserved retained databases and history: schedules now
have unique revisions, existing catalog identities are reused, and the owned
proxy is refreshed after an app address changes. Earlier fixture-seeding failures
are not counted as product readiness or successful browser evidence.

## Remaining boundaries

SDK 0.4.0 and core 0.3.0 were published from source
`39a49fb45005b001e14218823c22ed19c3de1177`. An independent registry download
matched SDK SHA-256 `45ab98970b6f246e370ae26fbd7abb31b0077519b90878f353d0ae74f05ec501`.
Immutable core digest is `sha256:1a9b02ff15eee3483d14a16d859a44d5dad705510fb0bfede137696fe0c63100`;
combined reference-engine digest is
`sha256:5651c73c0f35d93b392f8ddea7660cb4d891043e124d67fc0e7fc3da1b5f905b`.
Versioned receipts are in `docs/releases/`. Publication does not adopt or deploy
these artifacts for a consumer. Native Dedhed #34, Anarchist #64 and Riot
Grrrl #15 contain the separately qualified product-owned feedback interfaces;
those consumers retain their independent backend pins.

Hosted CI remains owner-waived during the Kvant outage. These checks do not prove
physical mobile devices, full-concert human listening, complete collection
coverage, sustained operational soak, live backend cutover or public account/chat
activation. Those issues stay open until their own required evidence exists.
