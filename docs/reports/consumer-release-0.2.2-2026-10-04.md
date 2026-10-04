# Private video upgrade and rollback qualification, 2026-10-04

The video archive template and Democracy Now! frontend adopted the immutable
0.2.2 release in their merged PR #4s. Both built images and private upgrade
rehearsals passed. These are optional frontend integrations; they do not establish
the separate headless-core publication or API-only consumer gates in issue #86.

## Exact artifacts

Both builds pin the engine to
`git.subcult.tv/subculture-collective/archive-engine@sha256:33293d5fa8e5cc13122b7b3b1e3aae81712b81a5024c4594ab53911f654cee21`
and use `@subcult/archive-viewer` 0.2.2. The published package SHA-256 is
`3f192e7c3da5dd357325885de891035502dbe83cdcfc96df9b90d56fbc7ad920`.

| Consumer | Merged source | Built image SHA-256 |
| --- | --- | --- |
| Video archive template | `124428eaa0516e695dca393a0cc35170980d439a` | `673b602a67497c1b8786187822fba81b37537eec40a2cbf08e18c730a664a691` |
| Democracy Now! | `e85b16c8abe8bc33ffe44b7a11e33e08c47a3689` | `59e654552cbe726d69b7a2612f0ab89490f80a92a2f3be24640005de26d02f96` |

Image inspection confirmed the source revision, engine digest and viewer version.
Hosted proposals ran the registered verification gates before opening the PRs.
The local build receipts are under `~/.local/state/vhness/releases/0.2.2/`.

## Independent upgrade and rollback rehearsal

Each rehearsal verified the existing private app/database ownership, took a fresh
database checkpoint, and restored it into a new task-owned PostgreSQL service.
The new image migrated one fresh database; the retained 0.1.1 image used a second
restored database. Original private databases and production services were not
changed. No media worker ran or delivery approval changed.

| Consumer | Checkpoint bytes | Checkpoint SHA-256 |
| --- | ---: | --- |
| Video archive template | 138636117 | `578c8a99831985f6d93af10a4768a16c156ab41ff8ae21bc2c0e2692728cbc57` |
| Democracy Now! | 138637588 | `d5457456d2196575095b85e806a415ec0ff5aa446e0c7bf0f12d7fc059ad70c2` |

Original non-migration table counts matched after restore and upgrade, including
6,625 episodes, 6,830 snapshots, 6,582 reviewed decisions and 6,866 observations.
The migrated database had 24 tables and eight migration versions; the retained
image's database had 12 tables and four migration versions. The private browser
library listed 6,582 reviewed episodes in both versions.

The retained template image was
`sha256:e4ea66f8e33d33a29fa41582062cfd4526388245aec0251b188c89806fc987a9`
at source `0e2d07a515a5ac10e44ab9007e847a0125660b18`. The retained Democracy Now!
image was
`sha256:c557a16a3abefdbd8cdf151445deeac48c179b43d34f54438b00a55f6e58757b`
at source `54292e3f0954d45277debc5f35b30fe5f310a39a`.

Private checkpoints, runtime configuration and receipts remain on Kuznya under
`/home/ci/.local/state/arkhiv-private-qualification/{consumer}/upgrade-022/`.
Credential-bearing runtime files have mode 0600 and remain outside Git.

## Actual Chromium checks

The 0.2.2 template at private port 48687 played the October 2, 2026 source MP4
directly from Internet Archive: readyState 4, duration 3542.016 seconds, advancing
playback time and no media decoder error. Seeking to 90 seconds completed.
At a 390-pixel viewport the document had no horizontal overflow or alert.

Joining the existing shared replay entered September 3 mid-program. Its natural
full-duration transition to September 2 occurred at 04:56:32 UTC; the observed
playback time was 16.945828 seconds against a server offset of 17.039606285,
a difference of approximately 0.094 seconds. The viewer did not alter schedule
revision `replay-20261003T072834.411108853Z`.

The branded 0.2.2 Democracy Now! app at private port 48689 played the same
October 2 source with readyState 4, advancing time, the same duration and no media
decoder error. Seeking to 90 seconds completed. Its 390-pixel layout had no
horizontal overflow or alert and retained its own branding and navigation.

The retained template app at private port 48688 decoded that same source with
readyState 4 and advancing time. The retained Democracy Now! app at private port
48690 decoded it with readyState 4 and advancing time; after a 90-second seek,
playback advanced to 126.501533 seconds without a media decoder error. Tested
players were paused after observation.

## Remaining gates

This establishes private artifact adoption, additive migration/restore, sampled
real playback, one natural scheduled transition and sampled old-artifact recovery
for these two consumers. It does not establish subjective audiovisual fidelity,
native mobile behavior, every archived program, long-term operation, all six
consumers' 0.2.2 adoption or public deployment. Browser console errors were not
zero: an intentional nonexistent channel endpoint probe and a transition/source
request error were observed separately from media decoder state.

Issues #45, #47, #50, #51, #52 and #86 retain their distinct outstanding gates.
Democracy Now! remains source-streaming only; no video acquisition, relay, cache
or local fallback was introduced. Public rollout still requires its owner decision.
