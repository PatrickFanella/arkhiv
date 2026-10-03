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

## Update timer qualification in progress

PR #41 merged after hosted checks passed. Operator revision `5bdf863` was installed
following another independently verified NUC snapshot, `auto-20261003T000214Z`.
Migration 0004 applied without restarting the reader or primary database.
The first update timer trigger at 00:06:10 UTC returned `disabled`, revealing
that the existing JSON-format collection file had a trailing YAML settings block
outside the decoded document. The operator configuration was rewritten as a single
JSON document with updates enabled and review-only acquisition. The timer was
restarted to schedule another initial trigger. The config loader now rejects
trailing/multiple documents, with regression coverage. The disabled run does not
establish a completed source inventory.

The corrected timer triggered at 00:12:00 UTC and completed a real source inventory
at 00:12:14 UTC: 5,337 files, 5,338 pending review events including item metadata,
and no acquisition under the review-only policy. A separate repeat refresh completed
with zero added/content/metadata/removed/uncertain files. This is unchanged-source
proof; evolving additions and editions are proven separately by fixtures.

The post-inventory NUC snapshot `auto-20261003T001300Z` independently verified
149 archive files and a 2,236,230-byte dump with SHA-256
`379c5f1aeb05bd42ecbb48c3ee1b6e7b2dd305a432902e4be6c59baa43ea0c30`.
The additional archive file is the retained raw source snapshot. The reader remains
at revision `6470472`; operator image `0fd557f` supplies the stricter configuration
reader. The restore runner source is `d6e2669`, while its unchanged probe image
identifies `0fd557f`. The first restore timer trigger is scheduled for 00:17:21 UTC.

## First recurring restore failure

The first actual restore timer triggered at 00:17:22 UTC and failed at catalog
restoration. It retained the independently fetched snapshot and fresh archive at
`qualification/arkhiv-drill-20261003t001723087569z`; the isolated stack was removed.
Inspection of the portable dump showed owner role `arkhiv`, while the fresh drill
database had been created with role `drill`. The correction uses the original role
name with synthetic drill-only credentials. Saved phase receipts now distinguish
database startup, catalog restoration, audit and reader proof. A corrected scheduled
run remains required; this failed run is not a successful restore qualification.

The corrected restore timer triggered at 00:24:15 UTC and completed at 00:24:49.
It independently fetched `auto-20261003T001423Z` from NUC, verified 149 archive
files and the 2,236,304-byte dump, and restored into
`qualification/arkhiv-drill-20261003t002416200747z`. Audit matched 43 assets,
26 variants and 25 published records. The browser probe matched all 26 download
hashes and rendered all 26 variants with Archive.org blocked. Search qualification
passed. Screenshots and reader receipts remain in the drill's `proof/` directory;
the isolated database/API/web/check stack was removed. Job source is `dbb8fcdf`,
operator/probe image source `0fd557f`, and web image source `6470472`.
The initial failure remains retained and is not overwritten by this successful run.
