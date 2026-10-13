# Core 0.4.0 qualification, October 13, 2026

Core 0.4.0 publishes the source metadata catalog from merged
[PR 126](https://git.subcult.tv/subculture-collective/arkhiv/pulls/126).
Its exact source is `6a14a59c970e935433b79dbc2757c3d97e151afa`.
The immutable registry image is
`git.subcult.tv/subculture-collective/archive-core@sha256:89f6fa6acb6150afe975f581e06498b72ad486cb17567fff9c54b61c2c0ceed6`.
The release receipt is [core-release-0.4.0.json](../releases/core-release-0.4.0.json).
All core 0.3.0 migration checksums remain unchanged; migrations 0013 and 0014 are
additive. Source catalog module 1 is independent of media acquisition and reviews.

The shared T3 launcher on Dozor passed `make verify && make test-integration`
using its owned disposable database. Hosted verification and curation passed
for the merged source. Earlier `make verify-curation` passed 19 tests; this
release changes no curation code.

The tag-triggered
[release attempt](https://git.subcult.tv/subculture-collective/arkhiv/actions/runs/15212)
was canceled after the assigned Kuznya runner failed to start its first step.
Runner logs reported cache writes failing with `no space left on device`.
Unrelated runner jobs and storage were left in place.

Publication used a clean detached checkout of the exact merged source on Dozor,
the official Dockerfile `core` target, and the agent registry identity with
temporary authentication removed after the push. Image labels match source,
version and artifact kind. The image contains no bundled frontend, Node or npm
and has no `VHNESS_WEB` environment setting. The standard `core_release.py`
generated the receipt from that clean source and the published registry digest.

The registry image was pulled by digest and started against the retained private
Live Music preview database. `/api/version` reported source catalog module 1;
local metadata search returned stored records. Consumer `make verify` passed
with core 0.4.0 and viewer 0.4.2. Earlier browser playback passed through the
shared resolver; source CDN failures were displayed as explicit player errors.

The viewer keeps its original core 0.3.0 qualification receipt. Upgrade tooling
now validates that receipt's contracts and locked migrations against an additive
newer core. Tests reject major changes and modified qualified migrations before
consumer mutation. Core and kit upgrades remain separate reviewed proposals.

This qualifies artifact publication and the private metadata/playback foundation.
It does not establish full etree import, a final frontend design, public rollout,
gapless playback, sustained playback or independent media recovery. Existing
consumer deployments are unchanged. Rollback requires the prior matching core
pin and receipt plus a database checkpoint from before any newer migrations.
