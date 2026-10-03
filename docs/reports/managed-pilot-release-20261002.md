# Private pilot release verification — 2026-10-02

The Almaz private pilot now runs merged source
`647047210c4ff6650bd849997d7f66680d325b78`, replacing its older qualified application
images without restarting the database. The primary API's archive mount is read-only.
Public ingress remains unconfigured.

Before replacement, a complete backup was created and independently verified on NUC:
`/home/onnwee/backups/arkhiv/pre-managed-20261002T233003Z`. Its 148 archive files
and portable catalog dump passed manifest verification. Earlier qualification and
restore evidence remains preserved.

The deployed API returned all 25 catalog records. Every one of the 26 PDF downloads
matched its catalog SHA-256 and byte count. The known-term solidarity query returned
14 documents. API readiness passed. The native T3 browser rendered page 2 of a local
PDF (1120 × 865 canvas with painted pixels) while a direct Archive.org fetch was
blocked. These are deployed private-service observations, independent of fixture tests.

The API image is `sha256:03e2ad6e8e78c6459f49ce9e017166722a35480693dcd9959240e82b9be8dfea`;
the web image is `sha256:ac64e29f3d4edf1bd4ee023a18ff91668002d19bc6feeb7e99dfdf7a55e713fa`.
Both source labels identify the full merged revision above. Hosted CI run 12030
passed for that revision. Private execution receipts and the prior Compose
configuration are retained under `~/.local/state/arkhiv/execution-20261003` on Kvant.

Rollback retains the pre-change snapshot and prior image/configuration receipt.
A code-only rollback can restore the prior API/web images; a schema rollback requires
a verified backup and fresh database. Do not overwrite preserved originals or restart
other Compose projects.

Canonical homelab notes could not yet be updated: the globally named Operations
vault was absent at the inspected Kvant path during the documentation migration.
This repository report preserves release evidence until the canonical owner is
available; it does not establish a second editable homelab documentation authority.

## Scheduled operations proof

The Almaz timers were installed from commit
`88933b5d5ecaad6cda44ca3f18f01cc858c0bf3d`. Their initial scheduled triggers ran
at 23:47:07 UTC (audit), 23:47:37 UTC (backup), 23:48:37 UTC (retention planning)
and 23:49:07 UTC (monitoring). These are actual timer triggers using the documented
initial activation delays; the overnight calendar cycle has not yet been observed.

Audit verified 43 assets, 26 variants and 25 published documents. Backup promoted
`auto-20261002T234738Z` on NUC after independent verification of all 148 archive
files and the 827,040-byte portable dump. The dump SHA-256 is
`8a560ff7d3aa1d2faebeee4c6646a8f1ce8a4c34d198794a01d1363d1f2c9034`.
Retention planning kept that snapshot, held all manual/qualification snapshots,
and returned no removal candidates. No bytes were deleted.

Monitoring subsequently used the existing ntfy `backups` topic on NUC. The initial
healthy message is cached as `h0b4D6xZSFPk`. An isolated qualification state directory
produced a stale `qualification-audit` failure and recovery, cached by the receiver
as `WTV752UaWcmQ` and `yL09f0X1vfgy`. The primary job state was not falsified to
produce that test. Receiver cache observations establish notification delivery;
they do not establish a user notification receipt. The original scheduled monitor
run preceded notification configuration; the configured delivery was invoked
separately through its systemd service.

Private readback receipts are `natural-jobs.json` and `notification-receiver.json`
in the execution state directory. The monitored jobs have 36-hour staleness limits,
with a 15-minute monitoring cadence. The original private reader/API revision remains
`6470472`; only the operator/job implementation uses `88933b5`.
