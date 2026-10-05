"""Create a consistent online SQLite checkpoint or restore into a new path."""

import argparse
import hashlib
import json
import os
import sqlite3
import tempfile
from contextlib import closing
from pathlib import Path

TABLES = {
    "collections",
    "searches",
    "items",
    "matches",
    "runs",
    "core_bindings",
    "core_attachments",
}


def checkpoint(source, destination):
    source, destination = Path(source), Path(destination)
    if source.is_symlink() or not source.is_file():
        raise ValueError("Select an existing regular curation database.")
    if destination.exists() or destination.is_symlink():
        raise ValueError("The destination already exists; select a new recovery path.")
    destination.parent.mkdir(parents=True, exist_ok=True)
    descriptor, temporary = tempfile.mkstemp(prefix=".curation-checkpoint-", dir=destination.parent)
    os.close(descriptor)
    temporary = Path(temporary)
    try:
        with (
            closing(sqlite3.connect(source.resolve().as_uri() + "?mode=ro", uri=True)) as original,
            closing(sqlite3.connect(temporary)) as recovered,
        ):
            original.backup(recovered)
            if recovered.execute("PRAGMA integrity_check").fetchall() != [("ok",)]:
                raise ValueError("The checkpoint failed SQLite integrity verification.")
            if recovered.execute("PRAGMA foreign_key_check").fetchall():
                raise ValueError("The checkpoint has invalid curation relationships.")
            tables = {
                row[0]
                for row in recovered.execute("SELECT name FROM sqlite_master WHERE type='table'")
            }
            if not TABLES <= tables:
                raise ValueError("The source does not contain a complete curation workspace.")
            counts = {
                table: recovered.execute(f'SELECT count(*) FROM "{table}"').fetchone()[0]
                for table in sorted(TABLES)
            }
        with temporary.open("rb") as snapshot:
            os.fsync(snapshot.fileno())
        # Atomic no-overwrite publication, including a concurrent destination creator.
        os.link(temporary, destination)
        return {
            "database": str(destination),
            "sha256": hashlib.sha256(destination.read_bytes()).hexdigest(),
            "bytes": destination.stat().st_size,
            "table_counts": counts,
            "integrity": "ok",
        }
    finally:
        temporary.unlink(missing_ok=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    parser.add_argument("destination", type=Path)
    args = parser.parse_args()
    print(json.dumps(checkpoint(args.source, args.destination), indent=2))


if __name__ == "__main__":
    main()
