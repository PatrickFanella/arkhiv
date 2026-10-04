# Arkhiv rollout review

This review belongs to milestone P6 and issue #52. On October 4, 2026 the owner
instructed: "continue all the way to public rollout, don't stop unless you
encounter a true roadblock." This authorizes development, qualification and
public activation of the reviewed products without another generic deployment
permission request. Existing content decisions, source-streaming restrictions,
allowlists and storage limits still define the content each product can expose.
Select exact qualified artifacts and a recoverable checkpoint before activation.

The owner selected `anarchist-archive.subcult.tv` for the existing print product,
`arkhiv.subcult.tv` for the mixed reference product, and
`arkhiv-api.subcult.tv` for the headless core. Existing document links under the
old print hostname need redirects preserving document identifiers and query
parameters. Other proposed brand hostnames are `democracy-now.subcult.tv`,
`bill-nye.subcult.tv`, `dedhed.subcult.tv` and `riot-grrrl.subcult.tv`; inspect
existing DNS/ingress ownership before adding their routes.

## Products and delivery boundaries

| Product | Shared adoption | Content and runtime boundary |
| --- | --- | --- |
| Arkhiv neutral app | Common engine and viewer | Scoped mixed collections; independent read/play/download/schedule review per representation. No default public collection or acquisition. |
| Video archive template | Published package and engine pin | Configurable distinct interface. A chosen collection keeps its own database, schedule, credentials and approved delivery policy. |
| Democracy Now! | Published package and engine pin | Metadata retained locally; video streams directly from Archive.org. No application download, proxy, cache or transcoding. |
| Bill Nye TV | Published package and engine pin | Approved 100 source MP4s, 13,763,456,996 bytes; 16 GiB cap and 1 GiB reserve. Independent NUC recovery and its own collection-only database. |
| Original print / arkhiv-alpha | Shared print components | Existing document/variant URLs, public allowlist, correction contact and automatic access policy remain in its backend. Latest independent backup contains 4,600 files and 3,822,509,172 archive bytes. |
| Riot Grrrl | Shared print components | Separate archive, database, reviewed source/item policy and Riot interface. Latest independent backup contains 218 files and 948,567,360 archive bytes. |
| Dedhed | Shared audio hook and timing helpers | Tape Shelf, calendar, original recording URLs and server schedule remain owned by Dedhed. Private reviewed audio only; public relay and subjective sound quality remain unapproved/unqualified. |

The registry records exact adoption revisions and PRs. Component consumers retain
upstream release receipts as package provenance; those receipts do not assert
that the upstream engine replaced their backend. Future backend migration is a
separate reviewed change with complete donor history retained.

## Artifact and recovery review

Release 0.2.1 passed hosted run 12652. Its verified receipt is
`docs/releases/release-0.2.1.json`; it binds the exact source, registry tarball and
engine digest. Release 0.2.2 passed hosted run 12680 from source
`17565d01903c019ba6610943ce5f1ac90910e93b`. Its registry package matches the
verified manifest in `docs/releases/release-0.2.2.json`; the new release adds
bounded slow audio resolution and readable request errors. Automated consumer
proposals and actual private adoption must be qualified before selection.

For each selected product, record the final package version, source revision,
effective frontend image and backend image/digest, database identity, storage
roots, ingress target and correction/removal contact. The revision matrix must
point to rendered behavior, not just an image build or liveness check.

Recovery uses the retained independent donor backup and a fresh target database.
Verify complete dump/media hashes, table counts, raw source identity and original
file integrity before startup. Common print and audio rehearsal receipts also
retain their original donor dumps, review history and schedule authority. Start
the prior artifact against a separate restored pre-migration checkpoint for an
actual rollback check; keep newer checkpoints and existing services available.
Public recovery must use the selected production checkpoint and its retained
artifact, with a scoped rollback receipt.

## Remaining decisions and activation

Complete issue #51's consumer matrix, automated second improvement, branded
reading/playback, reviewed music listening, schedule timing, independent restored
behavior and old-artifact rollback first. Responsive Chromium evidence is
separate from unavailable iOS/Android devices and from sustained natural runs.

The owner's October 4 instruction authorizes rollout. Resolve each product's
audience, existing source-use decisions, asset provenance, domain/ingress,
collection allowlist, byte limits/reserve and relay policy from current project
and deployment evidence. Select the concrete qualified artifacts and recovery
checkpoint. Record the exact activation scope in issue #52 and the product's
existing gate before deployment. A content decision that cannot be inferred
from those records is a specific unresolved boundary; proceed with independent
work while obtaining only the missing decision.

After authorized activation, inspect the actual deployed revisions and verify
representative public catalog, reading/download, on-demand playback, scheduled
replay, denied content and correction/removal behavior. Monitor the real data
path and quota/backoff state alongside process health. Leave pending approval
or unavailable evidence open rather than treating deployment readiness as release.
