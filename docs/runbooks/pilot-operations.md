# Pilot operations and recovery

## Destinations and access

The user authorized homelab placement on October 2, 2026. The working archive is
Almaz `/mnt/spektr/server/data/arkhiv/pilot` on the Spektr SSD. Independent dated
backups live on NUC `/home/onnwee/backups/arkhiv`. These are different hosts and
disks; both remain in the same homelab. A future off-site copy would protect against
loss of the whole site.

Almaz's private pilot service/configuration lives under
`/mnt/spektr/server/data/arkhiv`. The site binds to Almaz loopback port 54610.
Use an SSH tunnel to inspect it. No public hostname or ingress is configured.
The API/database network has no outbound route. Only the separately invoked
operator container joins the acquisition network. Preserve those boundaries.
A record published in this private pilot is not an approval for public release.

Keep database connection settings in the existing private environment file;
do not print, commit, or copy them into reports. Restrict archive/backup directories
to the archive operator. CLI commands serialize changes using `.writer.lock` and
fail if another operation is active. Run batch review, acquisition, extraction,
and backup sequentially.

## Commands

Build `backend/cmd/arkhiv` as the `arkhiv` operator binary. Set `DATABASE_URL` and
`ARKHIV_ARCHIVE` in the private operator environment. The runtime image supplies
Poppler, `prlimit`, and matching PostgreSQL backup tools. Run from `backend/`, or
pass explicit configuration/migration paths.

```sh
arkhiv migrate --migrations ../migrations
arkhiv plan --config ../collections/anarchist-zines.yaml --output /private/plan.json
arkhiv acquire --config ../collections/anarchist-zines.yaml --output /private/acquisition.json
arkhiv extract
arkhiv list --config ../collections/anarchist-zines.yaml
arkhiv inspect --id DOCUMENT_ID
arkhiv edit --id DOCUMENT_ID --metadata '{"creator":"Reviewed creator"}' --review 'Evidence for correction'
arkhiv group --variant VARIANT_ID --id TARGET_DOCUMENT_ID --review 'Evidence for grouping'
arkhiv publish --id DOCUMENT_ID --review 'Recorded publication review'
arkhiv unpublish --id DOCUMENT_ID --review 'Reason for withdrawal'
arkhiv audit
arkhiv backup --destination /private/new-dated-backup
```

Review the exact inventory and byte total before acquisition. Unselected files
are not fetched. Matching `_djvu.txt` companions are included in the batch cap.
Missing size, private files, unsafe selection paths, and excessive limits fail.
The CLI transfers sequentially, below the configured concurrency ceiling.
A failed batch retains checkpoints and receipts; rerun the same `acquire` command.
Strong ETags plus valid Range responses allow resume. A changed source identity or
ignored Range response restarts that file; source checksums and local SHA-256
must pass before immutable promotion. Existing versions remain preserved.

Acquisition leaves documents unpublished. Inspect content/title, variants,
missing metadata, rights/access statements, and text coverage before publication.
The CLI verifies original hashes, PDF headers, successful extraction/page counts,
and a review note. Grouping requires both records to be unpublished and in the
same collection. Source refresh updates normalized source fields while preserving
editorial overrides, rights, review, grouping, and publication decisions.

Extraction reads verified PDFs under time, memory, output-size, and CPU bounds.
It produces covers and page text without changing originals. A plain OCR companion
can supply document-level text at page `0`; the reader never invents a page link
for that text. `no-text` is a valid readable-PDF state; `failed` blocks publication.
Rerun `extract` to rebuild derivatives and search text. Cover failure is recorded
without invalidating the original.

## Independent backup and restore

Before backup, copy the reviewed collection YAML into the archive's `config/`
directory. The catalog also retains the full collection configuration. `backup`
verifies every cataloged original, writes a portable database dump, and copies
originals, raw source snapshots, receipts, checkpoints, configuration, and generated
derivatives into a new directory. It writes `complete.json` only after success,
with the catalog dump checksum and checksums for every copied archive file.
An incomplete directory remains evidence and is never treated as a valid backup.

Transfer the completed directory to a new dated directory on NUC using SSH/rsync.
Verify the transferred manifest before relying on it. Keep existing backups;
retention/deletion policy and scheduled backup automation are later operating work.
Fetch the recovery input from NUC into a new directory on Almaz for restore proof.

Create a fresh empty database with the matching PostgreSQL major version and set
its private `DATABASE_URL`. Use a new archive directory; restore refuses an
existing destination or database with tables.

```sh
arkhiv restore --backup /private/backup-fetched-from-nuc --destination /private/fresh-restored-archive
ARKHIV_ARCHIVE=/private/fresh-restored-archive arkhiv audit
```

Restore verifies the entire backup before copying bytes or applying the dump.
It then audits every cataloged original. Compare document/variant/asset counts,
publication and editorial state, and all asset hashes with the primary receipt.
Start a separate private API/site against the restored database and archive;
repeat metadata/text search, page navigation, PDF rendering, and download-hash
checks with outbound source access blocked. Preserve failures for diagnosis.
Do not point restore at the primary database or archive.

## Verification and rollback

Use `make verify`, the explicit disposable PostgreSQL integration/recovery gates,
and `make test-browser` before reviewing implementation changes. Browser fixtures
check desktop and 390px layouts, empty/loading/error states, page search, actual
PDF canvas rendering, navigation, zoom, original downloads, and a second brand.
Live acquisition and independent restore need separate dated receipts.

To stop this private pilot, stop only Compose project `arkhiv-pilot` without
removing its database or archive data. Keep the dated backup and recovery input.
Before replacing its application image, retain the previous image ID and run an
integrity audit. Migrations are forward-only, checksum-locked, and transactional;
a rollback requiring an older schema uses a verified backup and fresh database.
Never remove originals to reverse a code change.
