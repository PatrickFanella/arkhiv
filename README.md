# Arkhiv

Arkhiv is the shared archive engine for independent print, audio, and video apps.
It preserves source metadata, editorial history, reviewed access decisions,
media identities, search indexes, and shared playback schedules.

The branded apps own their frontend source, design, and deployment in separate
repositories: Anarchist Archive (`arkhiv-alpha`), Riot Grrrl Zine Archive,
Dedhed, Bill Nye TV, and Democracy Now! TV. Arkhiv supplies a versioned headless
engine and optional shared viewer components. Updating the engine does not
replace a brand's interface.

Engine releases use immutable container digests, API and module compatibility
versions, and checksum-locked additive migrations. Each consumer records its
pin and upgrades through a reviewed, independently verified PR. See the
[consumer registry](docs/reference/consumer-registry.json),
[core release contract](docs/reference/core-release-contract.json), and
[retained backend migration guide](docs/retained-backend-migration.md).

The private curation workbench under [`curation/`](curation/README.md) contains
copied Kollektor code, its license, and exact source provenance. It provides
saved searches, bounded discovery, collection review, notes, and exports.
It runs from this repository and connects directly to Arkhiv's catalog;
it does not require a running Kollektor app. Reviewed exports do not grant
acquisition, playback, or public access.

Print compatibility preserves original document and edition IDs, PDF readers,
searchable page text, OCR companions, and covers. Audio compatibility preserves
recording credits, sets, calendar browsing, original audio bytes, and shared
station playback. Explicit public station renewal verifies every selected
recording before extending its timeline.

Democracy Now! (`democracy_now_vid`) remains permanently source-streaming only.
Local preservation for other collections requires explicit review, bounded
storage, verified original bytes, and independent recovery evidence. Original
rights and source credits remain with each item.

The core 0.1.2 source has passed isolated database integration and hosted CI.
Release qualification uses recovered catalogs and the original branded
frontends. Production rollout, owner audiovisual review, and natural operation
are separate evidence. See [development instructions](DEVELOPMENT.md),
[deployment operations](docs/DEPLOYMENT.md), and the [roadmap](ROADMAP.md).
