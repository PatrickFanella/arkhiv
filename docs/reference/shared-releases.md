# Shared platform release operations

The shared viewer is built and checked through the root pnpm workspace.
`make verify` includes a fresh offline consumer installation of the packed ESM
artifact, as well as the Go/disposable-database and frontend checks.

Registry targets and compatibility are maintained in [release-contract.json](release-contract.json).
Gitea npm packages are installed from its organization registry; engine releases
use immutable container digests. See the official [npm registry documentation](https://docs.gitea.com/usage/packages/npm/).

The manual/tag release workflow requires `ARCHIVE_PACKAGE_TOKEN`, a token with
package read/write permission supplied through Gitea Actions secrets. Use the
existing secret provider for its retained recovery copy. Do not put the operator's
broad repository credential into the workflow. Gitea documents the built-in job
token's [package publishing limitation](https://docs.gitea.com/usage/actions/comparison/).
The separate consumer-upgrade automation needs its own repository-write credential;
package publishing permission is not a reason to grant cross-repository writes.
Set the repository variable `ARCHIVE_REGISTRY_USER` to the token's actual account
name; an organization registry owner is not necessarily a login identity.

The release verifies with a job-owned database, builds/pushes the engine, records
its registry digest, binds package bytes and migrations to the source revision,
then publishes the package. It never deploys a consumer. A failed publish is not
a qualified release; inspect registry state before retrying. Existing package
versions are immutable. Bump the version for changed bytes, preserve previous
artifacts, and never delete a version to make a retry succeed.

`scripts/release_manifest.py` requires clean committed source, a registry digest,
matching packed package metadata/bytes, and a new output destination. A local
image ID or mutable tag is rejected. The release receipt records migration hashes,
API/configuration compatibility, software license and source revision. Readers
must still qualify consumer behavior; a manifest is not a deployment receipt.

Retain the current and previous reviewed release during staged adoption. Review
breaking APIs/configuration and coordinate engine/migration/client order. Rollback
must use a compatible schema or an independently restored checkpoint. Public
release and collection approvals remain independent of package publication.
