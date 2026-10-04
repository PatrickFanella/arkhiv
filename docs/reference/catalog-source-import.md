# Catalog source imports and module compatibility

`vhness import-source --collection COLLECTION --id SOURCE_ID --file metadata.json`
stores a complete Internet Archive metadata response for a print, audio, video
or future source media type. The selected collection must already exist inside
the instance's configured scope. The command validates the exact string source
identifier and an object-based file inventory, with a 32 MiB input limit. It
retains private entries and unknown fields in the original snapshot bytes.

Importing a snapshot does not list an item, create an episode, acquire media,
approve reading or playback, register a derivative, or publish content. The
operator separately uses `import-catalog` to review identity and memberships,
then `review-transport` for each representation's collection-bound permissions.
The older `import` command continues to require a movie source and create a
compatible video episode.

`/api/version` keeps legacy video API version `1` and adds a `modules` map with
the implemented catalog, print, audio and video contract versions. The mixed
reference interface requires version `1` for those modules before it loads.
Legacy video configurations remain compatible with engines that predate the
module map; an engine's explicit collection allowlist still constrains their
configured collection. Site settings cannot expand server scope.

The unreleased `0.2.0` release contract now names all three media types and the
four compiled package entries. A release receipt records additive migration
checksums, exact package bytes and immutable engine digest. Source capability
does not imply private real-media qualification or public release approval.
