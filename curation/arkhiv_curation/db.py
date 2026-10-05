"""SQLite storage. Each operation owns its connection."""

import sqlite3
from contextlib import contextmanager
from pathlib import Path

SCHEMA = """
CREATE TABLE IF NOT EXISTS collections (
    id INTEGER PRIMARY KEY, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '',
    created_at REAL NOT NULL
);
CREATE TABLE IF NOT EXISTS core_bindings (
    collection_id INTEGER PRIMARY KEY REFERENCES collections(id),
    target_collection TEXT NOT NULL, updated_at REAL NOT NULL
);
CREATE TABLE IF NOT EXISTS core_attachments (
    collection_id INTEGER NOT NULL REFERENCES collections(id), core_item_id TEXT NOT NULL,
    core_url TEXT NOT NULL, evidence TEXT NOT NULL, evidence_sha256 TEXT NOT NULL,
    notes TEXT NOT NULL, attached_at REAL NOT NULL,
    PRIMARY KEY(collection_id,core_item_id)
);
CREATE TABLE IF NOT EXISTS searches (
    id INTEGER PRIMARY KEY, collection_id INTEGER NOT NULL REFERENCES collections(id),
    name TEXT NOT NULL, query TEXT NOT NULL, media_type TEXT NOT NULL DEFAULT '',
    source_collection TEXT NOT NULL DEFAULT '', year_from INTEGER, year_to INTEGER,
    interval_hours INTEGER NOT NULL DEFAULT 24, enabled INTEGER NOT NULL DEFAULT 1,
    cursor TEXT, next_run REAL NOT NULL, last_run REAL, last_error TEXT,
    scanned_total INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS items (
    collection_id INTEGER NOT NULL REFERENCES collections(id), identifier TEXT NOT NULL,
    title TEXT NOT NULL, creator TEXT NOT NULL, description TEXT NOT NULL,
    media_type TEXT NOT NULL, year TEXT NOT NULL, subjects TEXT NOT NULL,
    source_collections TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending'
        CHECK(status IN ('pending','accepted','rejected')),
    notes TEXT NOT NULL DEFAULT '', discovered_at REAL NOT NULL, reviewed_at REAL,
    PRIMARY KEY(collection_id, identifier)
);
CREATE TABLE IF NOT EXISTS matches (
    collection_id INTEGER NOT NULL, identifier TEXT NOT NULL,
    search_id INTEGER NOT NULL REFERENCES searches(id), matched_at REAL NOT NULL,
    FOREIGN KEY(collection_id,identifier) REFERENCES items(collection_id,identifier),
    PRIMARY KEY(collection_id,identifier,search_id)
);
CREATE TABLE IF NOT EXISTS runs (
    id INTEGER PRIMARY KEY, search_id INTEGER NOT NULL REFERENCES searches(id),
    state TEXT NOT NULL CHECK(state IN ('queued','running','completed','failed')),
    created_at REAL NOT NULL, started_at REAL, finished_at REAL,
    scanned INTEGER NOT NULL DEFAULT 0, added INTEGER NOT NULL DEFAULT 0,
    continued INTEGER NOT NULL DEFAULT 0, error TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS one_active_run ON runs(search_id)
    WHERE state IN ('queued','running');
CREATE INDEX IF NOT EXISTS item_queue ON items(collection_id,status,discovered_at);
"""


class Database:
    def __init__(self, path: str):
        self.path = path
        Path(path).parent.mkdir(parents=True, exist_ok=True)
        with self.connect() as db:
            db.execute("PRAGMA journal_mode=WAL")
            db.executescript(SCHEMA)

    @contextmanager
    def connect(self):
        db = sqlite3.connect(self.path, timeout=10)
        db.row_factory = sqlite3.Row
        db.execute("PRAGMA foreign_keys=ON")
        try:
            yield db
            db.commit()
        except BaseException:
            db.rollback()
            raise
        finally:
            db.close()
