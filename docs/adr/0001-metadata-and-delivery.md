# Metadata and delivery contracts

Decision: implement issue #3 with Go domain contracts, PostgreSQL persistence,
React/Vite UI, and delivery adapters independent of viewing mode.

Collections have `source_stream` or `local_preserve` policy. Democracy Now! is
locked to `source_stream` in domain validation and the database. Acquisition,
catalog publication, on-demand viewing, channel use, and public publication are
independent decisions. Public publication remains disabled for this private slice.

Store exact raw response bytes in `bytea` and parsed data in `jsonb`, keyed by
SHA-256. Preserve every file entry and unknown field. Normalized episodes and
search fields are rebuildable indexes. Source updates append observations and
snapshots without overwriting editorial decisions. Decisions and schedule
revisions retain history. SQL migrations have recorded checksums and an advisory
lock; changed applied migrations fail rather than being silently re-executed.

The metadata HTTP client is implemented here, following the reference projects'
patterns. No Arkhiv/Dedhed application source is copied. It requests only metadata
and search, has bounded response sizes, validates redirects, and returns full
source backoff deadlines to the persistent worker. Retry deadlines are not
shortened to fit a process lifetime. Conditional validators are stored with
observations. Worker locks and durable queues separate discovery progress from
item success; no failed item is deleted.

Read-only APIs provide `/api/episodes`, `/api/episodes/{id}`,
`/api/playback/{id}`, `/api/channel`, `/api/health`, `/api/ready`, and operator
status. The playback result includes item/file identity, delivery policy,
resolved URL, source URL, and snapshot hash. Streaming URLs point directly to
Internet Archive. Local URLs use an access-checked Range endpoint. Operator
mutations use the CLI rather than unauthenticated web endpoints.

Programs pin episode ID, chosen file, policy, absolute UTC start, and duration.
The server clock determines the current slot and offset. Schedule gaps, withdrawn
assets, and exhausted schedules remain explicit; viewer controls cannot mutate
the timeline. Clients account for load time, rejoin after suspension, and use
the server clock for boundaries. Synchronization tolerance is a qualification
measurement rather than a frame-accuracy promise.

Versioned API shapes and database migration changes must retain client and stored
decision compatibility. Metadata refresh may invalidate a selected file and
make playback unavailable; it never silently chooses a different scheduled asset.
Local preservation is a separate manifest-driven operator workflow with positive
quotas and free-space reserve. It cannot be used by streaming collections.
