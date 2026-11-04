# Automated update and deploy

Status: approved design, 2026-11-04. Owner decisions: upgrade proposals merge
automatically on green CI; public deployment runs from a one-click dispatch
first and becomes automatic once a few cycles pass; the targets are the Almaz
public project, the Anarchist project and the Dedhed pilot; the deploy job
runs on Almaz's own runner; the agent account owns the credentials.

## Why

Every lane today ends at a reviewed PR and a manual deployment. Merging the
eight bug-fix PRs on 2026-11-03 produced no release, no image and no
deployment. The hosted proposal job had also failed on every automatic run
since 2026-10-20 (registry URL encoding, fixed in PR #148). Consumers pin core
0.3.0 while 0.4.4 is published, and the public cores still run 0.1.1.

## Chain

1. **Release.** A merge on `main` that changes `packages/viewer/package.json`
   publishes that viewer version and opens a receipt PR; a merge that changes
   `docs/reference/core-version.json` builds and publishes that core image and
   opens its receipt PR. Receipt PRs merge automatically when `verify` passes.
   Existing tag triggers keep working. A version that already exists in the
   registry is never republished.
2. **Propose.** A receipt landing on `main` runs the existing proposal job for
   the matching lane (viewer or core), which clones each registered consumer,
   bumps its pins, runs its checks and opens an upgrade PR.
3. **Adopt.** Each upgrade PR is created with Gitea's merge-when-checks-succeed
   flag. Branch protection on every consumer `main` already requires the
   `verify / verify (pull_request)` status, so the merge waits for green CI.
4. **Build.** A merge on a consumer `main` builds that consumer's own image(s)
   with its committed Dockerfile(s) after `verify` passes, and pushes them to
   the Gitea container registry tagged with the merge commit. The image label
   `org.opencontainers.image.revision` is the only identity the deploy trusts.
5. **Deploy.** `deploy-public` in this repository is a `workflow_dispatch`
   on the `almaz` runner. It connects to Almaz as `onnwee` through an SSH key
   whose `authorized_keys` entry forces a single command, and runs
   `scripts/public_deploy.py` from the deployment's private `ops` directory.
   The script backs up, pulls the exact digests, runs core migrations with the
   operator credential when the core changes, swaps each service through a
   compose overlay with configuration-equivalence checks (the pattern the
   2026-10-20 activations used), waits for health, checks the public origin,
   runs the existing monitor, and writes a receipt. Any failure restores the
   previous overlay and image. The previous images are never deleted.

## Targets

| Target | Project on Almaz | Services deployed |
| --- | --- | --- |
| platform | `arkhiv-platform-public-20261004` | `archive-core`, `bill-core`, `democracy-core`, `reference`, `bill-nye-reference`, `democracy-now-reference`, `riot-public-api`, `riot-public-web` |
| anarchist | `arkhiv-pilot` | `public-api`, `public-web` |
| dedhed | `dedhed-public-20261004` | `api`, `web`, `operator` |

The current "support-seo" frontend overlays are plain rebuilt frontend files
from each repository's merged revision, so a consumer's own image supersedes
them without losing content.

## Boundaries kept

- No tool deletes originals, checkpoints or previous images.
- Public cores keep SELECT-only roles; migrations use the private operator
  credential only inside the deploy script.
- The deploy script refuses a digest whose image revision label does not match
  the requested consumer revision, and refuses to run while another deploy or
  maintenance lock is held.
- Democracy Now! stays source-streaming; nothing here changes collection policy.

## Credentials

Existing: `ARCHIVE_PACKAGE_TOKEN` (publish), `ARCHIVE_PACKAGE_READ_TOKEN`,
`ARCHIVE_UPGRADE_TOKEN` (consumer PRs) on this repository; `REGISTRY_PAT` and
`RELEASE_TOKEN` (write:package, write:repository) on the organization. New:
`ALMAZ_DEPLOY_KEY`, a dedicated SSH private key stored on this repository,
whose public half is restricted on Almaz to the deploy entry command.

## Rollout order

1. PR #148 (registry URL) merges.
2. Release and proposal automation in this repository.
3. Auto-merge on proposals; image builds in each consumer.
4. Deploy job plus the Almaz entry command and `public_deploy.py`.
5. First real cycle: viewer 0.4.5 and core 0.4.4 reach every consumer and the
   public hosts through the chain, with receipts and public checks recorded.
