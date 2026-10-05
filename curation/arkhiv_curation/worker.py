"""Durable discovery runs with bounded, resumable scans and one worker."""

import logging
import sqlite3
import threading
import time

import httpx

from .archive import normalize, query_for

log = logging.getLogger(__name__)


class Worker:
    def __init__(self, database, archive, pages_per_run=3):
        self.database = database
        self.archive = archive
        self.pages_per_run = pages_per_run
        self.stop = threading.Event()
        self.wake = threading.Event()

    def enqueue(self, search_id: int) -> int:
        with self.database.connect() as db:
            db.execute("BEGIN IMMEDIATE")
            active = db.execute(
                "SELECT id FROM runs WHERE search_id=? AND state IN ('queued','running')",
                (search_id,),
            ).fetchone()
            if active:
                return active["id"]
            try:
                result = db.execute(
                    "INSERT INTO runs(search_id,state,created_at) VALUES (?,'queued',?)",
                    (search_id, time.time()),
                )
            except sqlite3.IntegrityError:
                return db.execute(
                    "SELECT id FROM runs WHERE search_id=? AND state IN ('queued','running')",
                    (search_id,),
                ).fetchone()["id"]
            run_id = result.lastrowid
        self.wake.set()
        return run_id

    def recover(self):
        with self.database.connect() as db:
            db.execute("UPDATE runs SET state='queued' WHERE state='running'")

    def schedule(self, now=None):
        now = time.time() if now is None else now
        with self.database.connect() as db:
            due = db.execute(
                "SELECT id FROM searches WHERE enabled=1 AND next_run<=?", (now,)
            ).fetchall()
        for search in due:
            self.enqueue(search["id"])

    def run_one(self) -> bool:
        with self.database.connect() as db:
            run = db.execute(
                "SELECT * FROM runs WHERE state='queued' ORDER BY id LIMIT 1"
            ).fetchone()
            if not run:
                return False
            db.execute(
                "UPDATE runs SET state='running',started_at=? WHERE id=?", (time.time(), run["id"])
            )
            search = dict(
                db.execute("SELECT * FROM searches WHERE id=?", (run["search_id"],)).fetchone()
            )
        cursor = search["cursor"]
        scanned, added = run["scanned"], run["added"]
        error = None
        try:
            for _ in range(self.pages_per_run):
                if self.stop.is_set():
                    # Leave this run recoverable; successful pages are already checkpointed.
                    return True
                payload = self.archive.fetch(query_for(search), cursor)
                next_cursor = payload.get("cursor") or None
                if cursor and next_cursor == cursor:
                    raise ValueError("Archive repeated its search cursor. Retry this search later.")
                page = [item for raw in payload["items"] if (item := normalize(raw))]
                now = time.time()
                with self.database.connect() as db:
                    for item in page:
                        result = db.execute(
                            """INSERT INTO items(collection_id,identifier,title,creator,description,
                            media_type,year,subjects,source_collections,discovered_at)
                            VALUES (?,?,?,?,?,?,?,?,?,?) ON CONFLICT(collection_id,identifier)
                            DO NOTHING""",
                            (search["collection_id"], *item.values(), now),
                        )
                        added += result.rowcount
                        db.execute(
                            """INSERT INTO matches VALUES (?,?,?,?)
                            ON CONFLICT(collection_id,identifier,search_id)
                            DO UPDATE SET matched_at=excluded.matched_at""",
                            (search["collection_id"], item["identifier"], search["id"], now),
                        )
                    scanned += len(payload["items"])
                    db.execute(
                        "UPDATE searches SET cursor=?,scanned_total=scanned_total+? WHERE id=?",
                        (next_cursor, len(payload["items"]), search["id"]),
                    )
                    db.execute(
                        "UPDATE runs SET scanned=?,added=? WHERE id=?", (scanned, added, run["id"])
                    )
                cursor = next_cursor
                if not cursor:
                    break
                if self.stop.wait(0.75):
                    return True
        except (httpx.HTTPError, ValueError, KeyError, TypeError) as exc:
            error = self.error_message(exc)
        except Exception:
            log.exception("Discovery failed for run %s", run["id"])
            error = "Discovery failed unexpectedly. See the server log and retry."
        now = time.time()
        with self.database.connect() as db:
            db.execute(
                "UPDATE runs SET state=?,finished_at=?,continued=?,error=? WHERE id=?",
                ("failed" if error else "completed", now, bool(cursor), error, run["id"]),
            )
            db.execute(
                "UPDATE searches SET last_run=?,last_error=?,next_run=? WHERE id=?",
                (now, error, now + search["interval_hours"] * 3600, search["id"]),
            )
        return True

    @staticmethod
    def error_message(exc):
        if isinstance(exc, httpx.HTTPStatusError):
            code = exc.response.status_code
            if code == 429:
                return "Archive is rate limiting requests. Progress is saved; retry later."
            return (
                f"Archive returned HTTP {code}. Progress is saved; check the query or retry later."
            )
        if isinstance(exc, httpx.HTTPError):
            return "Could not reach Archive. Progress is saved; retry when the connection returns."
        return str(exc)[:500]

    def loop(self):
        self.recover()
        while not self.stop.is_set():
            try:
                self.schedule()
                if self.run_one():
                    continue
            except Exception:
                log.exception("Discovery worker iteration failed; retrying in 15 seconds")
            self.wake.wait(15)
            self.wake.clear()
