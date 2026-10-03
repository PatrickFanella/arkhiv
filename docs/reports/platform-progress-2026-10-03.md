# Multi-media platform implementation progress

Observed 2026-10-03 on Kvant. This records completed source work and the first
blocked qualification gate; it does not claim a combined platform is delivered.

## Delivered source changes

- [PR #54](https://git.subcult.tv/subculture-collective/VHNESS/pulls/54): canonical
  ownership/API/package ADR, seven-repository source/runtime inventory and pinned
  Arkhiv/Dedhed/VHNESS donor file hashes. The PR merged during implementation.
- [PR #55](https://git.subcult.tv/subculture-collective/VHNESS/pulls/55): shared
  `@subcult/archive-viewer` workspace package, compiled ESM/types, reusable
  components, typed playback/schedule logic and consumer-owned navigation option.
  The reference app consumes the package. The workspace lock retains every
  pre-existing web dependency resolution. Docker and Make build the workspace;
  legacy export compatibility is retained.
- [PR #56](https://git.subcult.tv/subculture-collective/VHNESS/pulls/56): manual/tag
  artifact release workflow, compatibility contract, source/migration/package
  manifest checks and regression coverage. This PR continues from #55 without
  waiting for its merge.

## Verification

Full isolated `t3-dev --profile vhness verify` passed from
`/home/onnwee/.t3/worktrees/vhness/multimedia-platform-roadmap`. Its disposable
snapshot checked Go formatting/vet/race/unit/database behavior, package build/
typecheck/lint/nine tests, reference typecheck/lint/two tests/build, fresh packed
ESM installation/composition/source policy and four release-manifest tests.
After detecting pnpm's package metadata formatting normalization, the manifest
comparison was corrected; five focused release-manifest tests passed.

The packed consumer installed into a fresh temporary directory using the
isolated dependency store, with no viewer source checkout. No registry package
installation, Docker registry push, fresh browser playback, migration, restore,
consumer deployment or public rollout is claimed. The existing private video
containers and other active previews were retained. Task-owned disposable test
databases were cleaned up by the launcher.

Release workflow YAML and every embedded shell block parse. Executable artifacts
are compared byte-for-byte; normalized package JSON is compared as complete
metadata. Mutable engine tags, dirty sources, replaced artifacts and overwritten
receipts are rejected by regression checks. Hosted check state belongs to the
Gitea runs, not these local results.

## Blocker: scoped package publishing credential

[Issue #39](https://git.subcult.tv/subculture-collective/VHNESS/issues/39) cannot
complete hosted package/container publication with current credentials:

- VHNESS repository and organization Actions secret lists are empty.
- After the owner promoted `subcult-agent`, the authenticated account reports
  `is_admin: true` and its token includes `write:admin`, alongside read
  notification/organization/user and write issue/repository scopes. The token
  still has no package access. Repeating scoped-token creation after promotion
  returned the missing `write:user` permission error. Account administration and
  token scopes are separate gates.
- The existing operator token's token-creation endpoint returned `auth required`.
- The collaborative Gitea browser is signed out; no authenticated settings session
  is available for token provisioning.
- Vaultwarden CLI is locked. No credential values were logged, copied into CI or
  changed in the vault.

The concrete release workflow requires a retained package-read/write token in
the existing secret provider, installed as the repository secret
`ARCHIVE_PACKAGE_TOKEN`, and repository variable `ARCHIVE_REGISTRY_USER` set to
that token's actual account. The existing operator repository token is not
substituted into an unattended release job.

The token-creation endpoint currently requires `write:user` on the device token;
its remaining authentication requirements have not been verified past that gate.
The direct unblock is to provision a separate token with `read:package` and
`write:package` into `ARCHIVE_PACKAGE_TOKEN` and set
`ARCHIVE_REGISTRY_USER` to that token's account. Keep credential values in the
secret provider or Gitea Actions settings; do not put them in an issue or chat.

Once the credential is available, run the release workflow and verify actual
registry artifacts/manifest before closing #39. Continue with upgrade PR automation
(#40), observable neutral/branded video propagation (#41), then the dependent
catalog/print/audio and migration/qualification milestones. Software package
publishing does not grant media acquisition or public deployment.

## Inventory limitations retained for migration

No separately named print/audio template repositories were found in the
authenticated organization inventory. Arkhiv/Dedhed are the engines for extraction;
the neutral combined app belongs to VHNESS. The historical Dedhed port 18577 was
not listening, while its isolated preview was active. Arkhiv public API requests
from this client returned 403. Current print production revision/worker ownership
and independent backup byte verification remain open evidence, rather than being
inferred from older reports. See the [inventory](../reference/platform-consumers.json).
