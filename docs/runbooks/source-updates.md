# Source inventories and update review

`arkhiv refresh --config COLLECTION.yaml` fetches a complete item inventory,
preserves its raw source snapshot and records added, changed-content, metadata,
removed and uncertain files. Files without source checksums remain uncertain;
matching size alone does not prove identical bytes. Failed, truncated, duplicate
or unsafe inventories retain the last successful baseline and record a failed run.
Inventory advancement and review-event creation commit in one database transaction.

Apply migration `0004_source_updates.sql` before using these commands. Back up the
catalog first. The migration adds inventories, review events and run receipts;
it does not change publication state or remove any existing record.

```sh
arkhiv refresh --config COLLECTION.yaml
arkhiv changes --config COLLECTION.yaml --offset 0
arkhiv changes --config COLLECTION.yaml --offset 500
arkhiv review-change --id CHANGE_ID --decision approve --review 'Acquisition evidence'
arkhiv review-change --id CHANGE_ID --decision reject --review 'Deliberate exclusion'
arkhiv review-change --id CHANGE_ID --decision hold --review 'Needs further review'
arkhiv acquire-changes --config COLLECTION.yaml
```

The queue returns 500 events per page, ordered by creation time and ID. Decisions
persist across repeated refreshes. Changed-content versions create separate
unpublished editions with deterministic keys; they cannot append a new variant
to an existing published document. Earlier originals and editorial overrides remain
preserved. Acquisition checkpoints resume interrupted downloads. A repeated import
reuses an existing edition with the same original hash. Failed extraction retains
an unpublished edition for investigation. Review/group/publish remain explicit
operator commands; no refresh approves public release.

Acquisition accepts checked, nonrestricted PDFs with known size and a source
checksum. Collection file, total-byte and document limits apply; at least 2 GiB
must remain free by default. Formats outside this policy and changed/removal events
that cannot safely acquire remain in review. The current update path indexes PDFs;
new OCR companion acquisition remains follow-up work. Existing OCR assets are retained.

A collection may set `updates.enabled`, `updates.policy` (`review` or `acquire`),
`updates.cadence_hours` and `updates.minimum_free_bytes`. The default policy is
review, cadence 24 hours, and scheduled updates disabled. `refresh --scheduled`
honors enable and cadence. An explicit manual refresh ignores cadence but respects
a persisted source Retry-After boundary. Long backoffs return a retry time instead
of retrying early. The HTTP client otherwise keeps its bounded retry policy.

The update runner accepts a `collections` list of operator-container configuration
paths. Its timer proposes 04:00 America/Chicago, with an initial 180-second activation
delay. Install and enable it only after migration, collection configuration and
exact operator image identity have been verified. Include update in the monitor's
staleness limits. A successful not-due or disabled result means the schedule checked
policy; it does not prove a source inventory or an acquisition occurred.

`make test-updates` uses disposable PostgreSQL and synthetic HTTP fixtures to check
complete inventories, persistent exclusions, failed baseline protection, evolving
source bytes, separate editions, original retention, replay, cadence and backoff.
`make verify` includes the inventory/acquisition integration suite. Actual unchanged
source checks and timer runs must be recorded separately from synthetic additions.
Multi-item query discovery and second-archive qualification remain roadmap work.
