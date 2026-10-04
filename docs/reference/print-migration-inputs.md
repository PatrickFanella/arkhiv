# Private print migration inputs

`scripts/prepare_print_migration.py` prepares common catalog inputs from an
independently restored original print pilot. It validates the complete backup
file manifest and database dump, the restore receipt, exported table counts,
original asset sizes/hashes and the complete source metadata. It writes a new
output directory and refuses to overwrite one. It changes no database or archive.

Supply the backup directory containing `complete.json`, `catalog.dump` and
`archive/`, a JSON export of the restored `documents`, `variants`, `assets`,
`collections`, `source_snapshots` and `asset_companions` tables, and a restore
receipt with `restore_exit_code`, `dump_sha256` and `table_counts`.

```sh
python3 scripts/prepare_print_migration.py \
  --backup /private/independent-backup \
  --restored-catalog /private/restored-catalog.json \
  --restore-receipt /private/restore-receipt.json \
  --output /private/new-platform-inputs
```

Documents and variants retain their IDs and grouping. Complete document and
variant rows remain in editorial provenance; original metadata and reviewed
overrides remain distinct. OCR companions stay attached to their editions.
Unpublished listings receive no transport inputs. Retained private published
documents receive private reading/download inputs; acquisition, playback, relay,
scheduling and public access remain disabled.

Legacy snapshot IDs can refer to a different JSON serialization. The converter
checks complete semantic equality between the restored database and the retained
source file, preserves the old identity, and binds the common source to the exact
file bytes. `migration-plan.json` records both identities. It never substitutes
new network metadata or invents missing original response bytes.

Keep the original database dump and fresh legacy restore as the authority for
all prior decisions, acquisition events, text indexes and schema history. Common
catalog import adds its own migration/transport history; it does not reinterpret
legacy events as newly granted rights. Rebuild common derivatives separately and
retain original derivatives. Explicitly import the source and catalog, review
transport, then index print using the existing operator commands. Collection
limits and deployment scope still require their retained configuration.

Generated inputs are a preparation artifact. Qualify actual catalog coverage,
PDF reading/search, original-download hashes, an independent common database
restore and a rollback to the retained legacy app before any cutover decision.
