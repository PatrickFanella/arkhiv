# Arkhiv curation and reviewed manifests

Arkhiv's [private curation workbench](../curation/README.md) contains copied
Kollektor discovery, review, and export code maintained in this repository.
It requires no separate Kollektor application. It reads the combined
`/api/catalog/v1/items` API on the server and retains
selected representations with their source identifiers, snapshot hashes, filenames,
media extensions and credits. Attachments enter the local review queue. Existing
acceptance, rejection, notes and discovery provenance survive later attachments.

The `arkhiv.curation.v1` export names a target core collection. A mapping is included
only when every selected Archive source is accepted and the current core evidence
still matches the retained attachment. Accepted items discovered through Archive
search without core evidence appear under `unresolved_items`; they require a
separate source snapshot and file selection before import. A listing is not proof
that media can be read or played.

The operator can prepare a draft in an instance that already holds the source catalog:

```sh
vhness prepare-curation --file arkhiv-curation-1-core-manifest.json --notes "Reviewed collection selection"
```

The target collection must already exist and lie within the configured instance
scope. That scope must also include the source collections being reviewed; a target-only
instance first needs a separately qualified source/catalog import. The command checks current source identity, selected files, typed extensions
and credits before storing the exact manifest in `curation_drafts`, keyed by its
SHA-256. Reimporting identical bytes is idempotent. Drafts do not appear in catalog
or transport views. Catalog mapping and transport/public reviews remain separate
operator steps. The seven access fields must all be present and false; adding any
permission to the manifest causes rejection.

This workflow transfers curation evidence. It does not acquire media, change Archive
items, run readers or players inside the workbench, or authorize a public collection.
