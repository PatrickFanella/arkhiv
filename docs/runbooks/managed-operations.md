# Managed preservation jobs

The repository provides audit, independent backup, monitoring and dry-run retention
jobs in `scripts/operations/runner.py`, with systemd units under `deploy/systemd/`.
These units target the existing Almaz private pilot. Installing code does not prove
that a timer has fired; retain timer trigger times, journal results and job receipts.

Copy the runner to `/mnt/spektr/server/data/arkhiv/operations/runner.py` on Almaz
and `/home/onnwee/backups/arkhiv/operations/runner.py` on NUC. Adapt
`deploy/operations.example.json` into Almaz's private `operations/config.json`.
Set `revision` to the full deployed source commit. Keep configuration and state
operator-owned and private. The runner reads the existing Compose configuration;
it does not copy or print database credentials.

Install only the four supplied service/timer pairs. Audit runs daily at 02:00,
backup at 03:00, retention planning at 03:40, and monitoring every 15 minutes,
in America/Chicago. The timers also have initial activation delays of 30, 60,
120 and 150 seconds. This permits an observable first scheduled run. Jobs share
a nonblocking lock; a skipped overlap preserves the previous successful receipt.
CLI writes also retain the archive's writer lock.

Audit verifies every cataloged original and published reader manifest. Backup
requires at least 2 GiB free locally and remotely, creates a new portable snapshot,
checks its dump and complete file inventory, transfers to a separate `.incoming-`
directory on NUC, verifies there, then promotes the verified directory. An interrupted
transfer remains partial evidence. The receipt names the independent verified host
and destination. Failed runs retain the prior last-success timestamp.

Retention produces a dry-run plan. It considers only complete, verified snapshots
named `auto-YYYYMMDDTHHMMSSZ`; unknown names, manual qualification snapshots,
partial transfers and explicitly pinned snapshots are held. The newest verified
snapshot is always kept. Defaults retain seven daily, four weekly and six monthly
buckets. This implementation deletes nothing. Review candidate sizes and disk
capacity before a future pruning activation. Originals are never retention targets.

Monitoring compares audit/backup results and last-success ages with configured
limits. A changed failure or recovery sends one HTTP notification when `notify_url`
is configured. Rejected or unavailable destinations retry; a previously unconfigured
notification is sent after configuration. HTTP acceptance is delivery evidence only;
receiver observation and user receipt require separate proof. Keep authenticated
notification URLs out of Git. Do not enable a destination without checking its owner.

Receipts live under `operations/state`: latest per-job result, dated history,
lock-skipped evidence, latest verified backup and notification state. Service deadlines
and memory/CPU limits bound jobs. Stop an individual timer to disable its schedule;
keep receipts and incomplete directories for investigation. To revert this automation,
stop only `arkhiv-{audit,backup,retention,monitor}.timer` and retain archive, database,
independent backups and prior application images.

NUC provides another host and disk, but shares the homelab's physical risk. A future
off-site option is an encrypted copy of complete snapshots to an operator-selected
remote storage destination, with independent checksum verification and a tested
restore. No off-site account, transfer or third copy is activated by these jobs.

Verification: `make test-operations` covers manifest corruption and extra files,
unsafe paths, overlaps, disk exhaustion, retained last-success timestamps, notification
retry/deduplication and dry-run retention. `make verify` includes this suite plus the
catalog integration checks that detect missing or mismatched reader manifests.
Recurring restore drills and source-refresh jobs remain separate roadmap work.
