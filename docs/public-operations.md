# Public deployment maintenance

The public API processes use read-only database roles. Replay planning and backups
run as separate operators with private credentials; the frontends remain separate
images. `scripts/public_runtime.py` maintains one explicitly named deployment.
It never acquires media, approves publication, or deletes originals or checkpoints.

Keep its JSON configuration, credentials, outputs, and checkpoints outside Git,
owned by the service user with directory mode 0700 and file mode 0600. Configuration
selects the Compose project, database/core containers, Docker command prefix,
network, databases, replay scopes, URL secret files, public JSON contracts,
checkpoint budget, write allowance, free-space reserve, and independent mirror.
The script checks container project ownership before acting. Replay URL files
must resolve inside the private deployment root.

Run each action before enabling its timer:

```sh
python3 scripts/public_runtime.py monitor --config /private/ops/config.json
python3 scripts/public_runtime.py replay --config /private/ops/config.json
python3 scripts/public_runtime.py backup --config /private/ops/config.json
```

Monitoring checks the public audience, collection scopes and expected catalog
counts. A changed count requires reconciliation; it does not silently update the
baseline. Replay checks current source hashes and selected files against public
reviews before planning each collection. Unreviewed eligible entries fail the job.
It uses the active core's immutable image and plans seven days without rebuilding
a frontend. Backup preserves all configured public and imported source tables,
records row counts and dump hashes, copies to an independent host, and checks
those remote hashes. Copy success explicitly records `restore_verified=false`.
Restore each dump into a fresh independently owned database and compare every
manifest count before recording recovery qualification.

All actions share a blocking maintenance lock. Jobs queue behind the current
owner; do not delete or bypass that lock. The checkpoint budget fails closed and
requires an operator retention review when exhausted. It never prunes recovery
artifacts. Almaz's public deployment uses a 16 GiB checkpoint cap, 2 GiB write
allowance and 1 GiB free-space reserve.

The installed services run as `onnwee` with `Type=oneshot`, `UMask=0077`,
`PrivateTmp=true`, and a 15-minute startup timeout. They execute the reviewed
script from the deployment's private `ops` directory. The proposed recurring
schedule is monitoring every five minutes, replay at 00:10 UTC, and backup at
00:30 UTC. Daily timers are persistent. Enable them only after their first actual
runs and independent restore checks succeed. Inspect service results, receipts,
and timer next-run times separately; a manual run does not prove a natural daily
cycle or long-term reliability.

For a release, retain the previous image pins, current ingress/configuration
hashes and independent checkpoints. Change only this deployment's services and
owned ingress blocks. Restore public access reviews and source histories along
with catalog rows. A historical private server image is not automatically a safe
public rollback target.
