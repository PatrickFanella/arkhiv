# Deployment collection scope

Use `VHNESS_COLLECTION` for an existing single-collection deployment, or
`VHNESS_COLLECTIONS=collection-one,collection-two` for an explicit allowlist of up
to 64 distinct collection IDs. Setting both fails configuration. Invalid, duplicate
or blank entries fail configuration; there is no wildcard. Omitting both retains
the legacy unrestricted operator instance. Never use that omission as a brand's
scope. An explicit empty scope in the server contract denies every collection.

The server applies the allowlist to legacy episode/media/playback/status routes,
mixed catalog membership, representation transport, print derivatives, audio
assets and stations. A request's collection selection can narrow that scope.
`/api/catalog/v1/collections` reports visible item counts and represented media
for mixed collections. The legacy `/api/collections` keeps its video-only shape.
Typed transport requires an explicit collection when more than one collection is
configured, or when the instance has no configured default.

The existing video channel retains its schedule identity and schema. Every
scheduled episode, including future entries, must exist and belong to the allowed
scope. A viewer's explicit collection selection narrows the entire channel,
including future entries. Collection-specific audio stations keep their own
revisions and independent eligibility reviews. Viewer controls never modify either
schedule.

Operator commands enforce the same allowlist on collection selection, imported
catalog memberships, transport/asset reviews, legacy episode decisions,
acquisition manifests, synthetic seeding and every imported schedule program.
Multi-collection commands that operate on one collection require `--collection`.
Database migration and serving remain instance-level commands. Existing collection
delivery policies, budgets, reserves and approvals do not change.

The shared video hook includes its selected collection on channel and playback
requests and hides a previous collection's session, channel, errors and loading
state. Collection changes invalidate pending listener actions; rendered browser
behavior is qualified separately from backend scope fixtures.
