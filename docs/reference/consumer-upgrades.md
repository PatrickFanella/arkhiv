# Reviewed consumer upgrades

[consumer-registry.json](consumer-registry.json) selects repositories and dependency
files managed by the shared platform. `platform` consumers pin the package,
release receipt and shared engine together. Explicit `components` consumers pin
the package and its release provenance while retaining their own backend. Their
registry entry must omit `engine_file`; an ambiguous mode or engine pin is rejected.
A release receipt records the upstream artifact, not a claim that its engine is
running in a component consumer.

`scripts/upgrade_consumer.py` prepares an upgrade in a clean isolated checkout.
It requires the release manifest and the packed registry artifact. The manifest
must come from a successful reviewed release; checksum validation binds the
artifact to that receipt, but does not independently establish its publisher.
It rejects unsupported APIs/media/configuration, changed registry ownership,
major upgrades, downgrades, reused versions and changes to existing migrations.
The package dependency, pnpm lock and release receipt change together. Platform
consumers also update their exact engine digest. Consumer checks run before
proposal creation. Failure restores every managed file; use a disposable checkout because checks may create build outputs.
The checks require a task-owned disposable database when `make verify` does.

`scripts/propose_consumer_upgrade.py` clones only the selected registered consumer
and prepares the upgrade. It uses `ARCHIVE_UPGRADE_TOKEN` for Gitea API requests
and verifies the account is `subcult-agent`. Provision a separate repository-write
token for this job; the package publishing token cannot create proposals. Git
uses the installed agent HTTPS credential helper. Registry authentication must
be supplied through an ephemeral npm configuration outside the repository.

The branch name binds the version and complete release source revision. Repeated
runs reuse the existing PR after comparing its release receipt, including PRs
beyond the first API page. A declined PR is not reopened, and an already applied
release produces no proposal. Git pushes never force-update an existing branch.
If a push succeeded but PR creation failed, retain the branch and inspect it
before retrying; do not delete it to make the retry succeed.

The tools do not merge or deploy consumers. Local fixture tests cover guards,
checksum failure, file restoration, repeat preparation and proposal reuse.
Hosted PR creation, installation from the registry, consumer CI, rendered private
qualification and effective deployment revisions remain separate receipts.

## Initial adoption and stacked proposals

The neutral video and Democracy Now! adoption branches consume the published
0.1.0 package. Their exact revisions, merged PRs and merge revisions are recorded in the
registry. They now consume the package on their default branches; deployment
adoption remains separate. Both passed isolated registry installation, consumer verification and
image builds over the qualified engine digest.

When an adoption PR is still open, pass `--base-ref feat/shared-package-adoption`
to create the next upgrade as a separate stacked PR. The tool checks out that
remote branch and verifies its actual pins before mutation. Repeat proposals
must match the requested base and release receipt. After adoption merges,
retarget the stack to the default branch and update the registry evidence; an
existing PR with a different base requires explicit reconciliation. Normal
upgrades omit `--base-ref` and use the default branch.

## Hosted proposal automation

Committed receipts under `docs/releases/` bind published registry bytes to their
qualified source and engine digest. The proposal job runs on default-branch
receipt/registry changes or explicit dispatch. It downloads the actual registry
package, verifies the receipt checksum, and prepares consumer PRs with the pinned
pnpm toolchain. Package downloads cannot forward credentials to another registry.

CI uses separate package-read and repository-write credentials. Its Git helper
serves only the agent identity and registered consumer HTTPS paths. The job
checks compatibility and consumer builds before committing; repeat runs reuse
the matching proposal. It never merges or deploys. A dispatch with
`adoption_refs=true` stacks on the exact qualified adoption revisions. A moved
adoption branch is rejected until its evidence is refreshed.

Version 0.2.1 passed hosted release run 12652 at source
`04eb1f80a0e5d5bc92f83c0030204cafee131d96`. Its registry tarball matched the
hosted receipt; the engine digest is recorded in `docs/releases/release-0.2.1.json`.

Qualified release runs: 0.1.0 is VHNESS run 12470, source `d2a005f0ecd2a81614f1009abf2473832cc1afaf`;
0.1.1 is run 12509, source `8827514b215162e5536f2b83a65681ba70e38f30`.
Both completed every hosted release step successfully, and their registry
tarballs match the downloaded hosted receipts. Consumer deployment and browser
qualification remain separate.

The first hosted proposal attempt (run 12522) failed safely because both
adoption branches were merged and removed while the job was starting. No
upgrade branch or PR was created. The registry now records both merged
adoptions, so subsequent proposals use their default branches.
