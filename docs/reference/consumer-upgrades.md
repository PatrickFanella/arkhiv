# Reviewed consumer upgrades

[consumer-registry.json](consumer-registry.json) selects the repositories and
dependency files managed by the shared platform. The initial video consumers are
marked `pending`: their current frontends still use exported source and their
deployment files do not yet consume the shared engine digest. Package adoption
must create the release receipt, exact package dependency and effective engine
configuration before an entry becomes `package`.

`scripts/upgrade_consumer.py` prepares an upgrade in a clean isolated checkout.
It requires the release manifest and the packed registry artifact. The manifest
must come from a successful reviewed release; checksum validation binds the
artifact to that receipt, but does not independently establish its publisher.
It rejects unsupported APIs/media/configuration, changed registry ownership,
major upgrades, downgrades, reused versions and changes to existing migrations.
The package dependency, pnpm lock, engine digest and compatibility receipt change
together. Consumer checks run before proposal creation. Failure restores those
four files; use a disposable checkout because checks may create build outputs.
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
0.1.0 package. Their exact revisions and PRs are recorded in the registry;
`adoption_status: pull-request` does not claim default-branch or deployment
adoption. Both passed isolated registry installation, consumer verification and
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

Qualified release runs: 0.1.0 is VHNESS run 12470, source `d2a005f0ecd2a81614f1009abf2473832cc1afaf`;
0.1.1 is run 12509, source `8827514b215162e5536f2b83a65681ba70e38f30`.
Both completed every hosted release step successfully, and their registry
tarballs match the downloaded hosted receipts. Consumer deployment and browser
qualification remain separate.
