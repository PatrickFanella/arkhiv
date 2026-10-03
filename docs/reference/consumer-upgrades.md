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
