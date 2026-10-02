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
