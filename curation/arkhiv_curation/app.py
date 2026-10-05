import asyncio
import csv
import fcntl
import io
import json
import os
import threading
import time
from contextlib import asynccontextmanager, contextmanager
from pathlib import Path
from typing import Literal
from urllib.parse import urlsplit

from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import FileResponse, Response
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel, Field, model_validator

from .archive import ArchiveClient, query_for
from .core import CoreClient
from .curation import install_curation_routes
from .db import Database
from .worker import Worker

STATIC = Path(__file__).parent / "static"


@contextmanager
def worker_lock(path):
    with open(path + ".lock", "a") as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as exc:
            raise RuntimeError(
                "Another Arkhiv curation worker owns this database. Use one process."
            ) from exc
        yield


class CollectionInput(BaseModel):
    name: str = Field(min_length=1, max_length=120)
    description: str = Field(default="", max_length=4000)

    @model_validator(mode="after")
    def clean(self):
        self.name = self.name.strip()
        if not self.name:
            raise ValueError("Give the collection a name.")
        return self


class SearchInput(BaseModel):
    name: str = Field(min_length=1, max_length=120)
    query: str = Field(min_length=1, max_length=2000)
    media_type: Literal["", "texts", "movies", "audio", "image", "software", "data", "etree"] = ""
    source_collection: str = Field(default="", max_length=200)
    year_from: int | None = Field(default=None, ge=1, le=9999)
    year_to: int | None = Field(default=None, ge=1, le=9999)
    interval_hours: int = Field(default=24, ge=1, le=720)
    enabled: bool = True

    @model_validator(mode="after")
    def clean(self):
        self.name, self.query = self.name.strip(), self.query.strip()
        self.source_collection = self.source_collection.strip()
        if not self.name or not self.query:
            raise ValueError("Give the search a name and a query.")
        if self.year_from and self.year_to and self.year_from > self.year_to:
            raise ValueError("The start year must be no later than the end year.")
        return self


class DecisionInput(BaseModel):
    status: Literal["pending", "accepted", "rejected"]
    notes: str | None = Field(default=None, max_length=4000)


def create_app(db_path=None, archive=None, start_worker=True, core=None):
    database = Database(db_path or os.getenv("ARKHIV_CURATION_DB", "data/curation.sqlite"))
    worker = Worker(database, archive or ArchiveClient())

    @asynccontextmanager
    async def lifespan(app):
        if not start_worker:
            yield
            return
        with worker_lock(database.path):
            thread = threading.Thread(target=worker.loop, name="discovery", daemon=True)
            app.state.worker_thread = thread
            thread.start()
            try:
                yield
            finally:
                worker.stop.set()
                worker.wake.set()
                await asyncio.to_thread(thread.join)

    app = FastAPI(title="Arkhiv curation", lifespan=lifespan)
    app.state.database = database
    app.state.worker = worker

    @app.middleware("http")
    async def same_origin(request: Request, call_next):
        # Mutations require JSON and reject browser requests from other origins.
        if request.method in {"POST", "PATCH", "PUT", "DELETE"}:
            origin = request.headers.get("origin")
            if origin and urlsplit(origin).netloc != request.headers.get("host"):
                return Response("Cross-origin writes are not allowed.", status_code=403)
            if request.headers.get("content-type", "").split(";")[0] != "application/json":
                return Response("Use application/json.", status_code=415)
        response = await call_next(request)
        response.headers["X-Content-Type-Options"] = "nosniff"
        response.headers["Referrer-Policy"] = "no-referrer"
        response.headers["Content-Security-Policy"] = (
            "default-src 'self'; img-src 'self' https://archive.org; "
            "style-src 'self'; script-src 'self'; connect-src 'self'; "
            "object-src 'none'; base-uri 'self'; frame-ancestors 'self'"
        )
        return response

    def collection_exists(db, collection_id):
        row = db.execute("SELECT * FROM collections WHERE id=?", (collection_id,)).fetchone()
        if not row:
            raise HTTPException(404, "Collection not found.")
        return dict(row)

    def search_exists(db, search_id):
        row = db.execute("SELECT * FROM searches WHERE id=?", (search_id,)).fetchone()
        if not row:
            raise HTTPException(404, "Search not found.")
        return dict(row)

    @app.get("/api/health")
    def health():
        thread = getattr(app.state, "worker_thread", None)
        return {"status": "ok", "worker_alive": bool(thread and thread.is_alive())}

    @app.get("/api/collections")
    def collections():
        with database.connect() as db:
            rows = db.execute("SELECT * FROM collections ORDER BY created_at,id").fetchall()
            result = []
            for row in rows:
                value = dict(row)
                value["counts"] = {"pending": 0, "accepted": 0, "rejected": 0}
                for count in db.execute(
                    "SELECT status,count(*) AS n FROM items WHERE collection_id=? GROUP BY status",
                    (row["id"],),
                ):
                    value["counts"][count["status"]] = count["n"]
                value["search_count"] = db.execute(
                    "SELECT count(*) FROM searches WHERE collection_id=?", (row["id"],)
                ).fetchone()[0]
                result.append(value)
        return result

    @app.post("/api/collections", status_code=201)
    def create_collection(value: CollectionInput):
        with database.connect() as db:
            result = db.execute(
                "INSERT INTO collections(name,description,created_at) VALUES (?,?,?)",
                (value.name, value.description, time.time()),
            )
            return collection_exists(db, result.lastrowid)

    @app.put("/api/collections/{collection_id}")
    def edit_collection(collection_id: int, value: CollectionInput):
        with database.connect() as db:
            collection_exists(db, collection_id)
            db.execute(
                "UPDATE collections SET name=?,description=? WHERE id=?",
                (value.name, value.description, collection_id),
            )
            return collection_exists(db, collection_id)

    @app.get("/api/collections/{collection_id}/searches")
    def searches(collection_id: int):
        with database.connect() as db:
            collection_exists(db, collection_id)
            rows = db.execute(
                """SELECT searches.*, (SELECT state FROM runs WHERE search_id=searches.id
                AND state IN ('queued','running') LIMIT 1) AS active_state
                FROM searches WHERE collection_id=? ORDER BY id""",
                (collection_id,),
            ).fetchall()
            return [dict(row) | {"effective_query": query_for(dict(row))} for row in rows]

    @app.post("/api/collections/{collection_id}/searches", status_code=201)
    def create_search(collection_id: int, value: SearchInput):
        fields = value.model_dump()
        with database.connect() as db:
            collection_exists(db, collection_id)
            result = db.execute(
                f"INSERT INTO searches(collection_id,{','.join(fields)},next_run) "
                f"VALUES ({','.join('?' for _ in range(len(fields) + 2))})",
                (collection_id, *fields.values(), time.time()),
            )
            search = search_exists(db, result.lastrowid)
        if start_worker:
            worker.wake.set()
        return search

    @app.put("/api/searches/{search_id}")
    def edit_search(search_id: int, value: SearchInput):
        fields = value.model_dump()
        with database.connect() as db:
            # Lock out edits while a worker is using this configuration/cursor.
            db.execute("BEGIN IMMEDIATE")
            previous = search_exists(db, search_id)
            if db.execute(
                "SELECT id FROM runs WHERE search_id=? AND state IN ('queued','running')",
                (search_id,),
            ).fetchone():
                raise HTTPException(
                    409, "Wait for the current discovery run before editing this search."
                )
            reset = query_for(previous) != query_for(fields)
            db.execute(
                f"UPDATE searches SET {','.join(f'{key}=?' for key in fields)},cursor=?,"
                "next_run=?,last_error=NULL WHERE id=?",
                (
                    *fields.values(),
                    None if reset else previous["cursor"],
                    time.time() + value.interval_hours * 3600,
                    search_id,
                ),
            )
            return search_exists(db, search_id)

    @app.post("/api/searches/{search_id}/run", status_code=202)
    def run_search(search_id: int):
        with database.connect() as db:
            search_exists(db, search_id)
        return {"run_id": worker.enqueue(search_id)}

    @app.post("/api/searches/{search_id}/restart", status_code=202)
    def restart_search(search_id: int):
        with database.connect() as db:
            db.execute("BEGIN IMMEDIATE")
            search_exists(db, search_id)
            if db.execute(
                "SELECT id FROM runs WHERE search_id=? AND state IN ('queued','running')",
                (search_id,),
            ).fetchone():
                raise HTTPException(409, "Wait for the current run before restarting the scan.")
            db.execute("UPDATE searches SET cursor=NULL,last_error=NULL WHERE id=?", (search_id,))
        return {"run_id": worker.enqueue(search_id)}

    @app.get("/api/collections/{collection_id}/runs")
    def runs(collection_id: int):
        with database.connect() as db:
            collection_exists(db, collection_id)
            return [
                dict(row)
                for row in db.execute(
                    """SELECT runs.*, searches.name AS search_name FROM runs
                JOIN searches ON searches.id=runs.search_id WHERE searches.collection_id=?
                ORDER BY runs.id DESC LIMIT 50""",
                    (collection_id,),
                )
            ]

    @app.get("/api/collections/{collection_id}/items")
    def items(
        collection_id: int,
        status: Literal["pending", "accepted", "rejected"] = "pending",
        page: int = 1,
        q: str = "",
    ):
        if page < 1 or len(q) > 200:
            raise HTTPException(422, "Invalid page or filter.")
        with database.connect() as db:
            collection_exists(db, collection_id)
            where = "collection_id=? AND status=?"
            params = [collection_id, status]
            if q.strip():
                where += " AND (instr(lower(title),lower(?))>0 OR instr(lower(creator),lower(?))>0)"
                params.extend([q.strip(), q.strip()])
            total = db.execute(f"SELECT count(*) FROM items WHERE {where}", params).fetchone()[0]
            rows = db.execute(
                f"SELECT * FROM items WHERE {where} ORDER BY discovered_at DESC,identifier "
                "LIMIT 24 OFFSET ?",
                (*params, (page - 1) * 24),
            ).fetchall()
            result = []
            for row in rows:
                item = dict(row)
                item["matches"] = [
                    dict(match)
                    for match in db.execute(
                        """SELECT searches.id,searches.name,searches.query,matches.matched_at FROM matches
                    JOIN searches ON searches.id=matches.search_id
                    WHERE matches.collection_id=? AND matches.identifier=?""",
                        (collection_id, row["identifier"]),
                    )
                ]
                result.append(item)
            return {"items": result, "total": total, "page": page, "page_size": 24}

    @app.patch("/api/collections/{collection_id}/items/{identifier}")
    def decide(collection_id: int, identifier: str, value: DecisionInput):
        with database.connect() as db:
            existing = db.execute(
                "SELECT notes FROM items WHERE collection_id=? AND identifier=?",
                (collection_id, identifier),
            ).fetchone()
            if not existing:
                raise HTTPException(404, "Item not found in this collection.")
            db.execute(
                "UPDATE items SET status=?,notes=?,reviewed_at=? WHERE collection_id=? AND identifier=?",
                (
                    value.status,
                    existing["notes"] if value.notes is None else value.notes,
                    time.time(),
                    collection_id,
                    identifier,
                ),
            )
        return {"status": value.status}

    @app.get("/api/collections/{collection_id}/export")
    def export(collection_id: int, format: Literal["json", "csv", "inquiry"] = "json"):
        with database.connect() as db:
            collection = collection_exists(db, collection_id)
            rows = [
                dict(row)
                for row in db.execute(
                    "SELECT * FROM items WHERE collection_id=? AND status='accepted' ORDER BY title,identifier",
                    (collection_id,),
                )
            ]
            search_rows = [
                dict(row)
                for row in db.execute(
                    "SELECT * FROM searches WHERE collection_id=? ORDER BY id", (collection_id,)
                )
            ]
        for row in rows:
            row["url"] = "https://archive.org/details/" + row["identifier"]
        if format == "json":
            body = json.dumps(
                {
                    "collection": collection,
                    "exported_at": time.time(),
                    "searches": search_rows,
                    "items": rows,
                },
                indent=2,
                ensure_ascii=False,
            )
            media_type, extension = "application/json", "json"
        elif format == "csv":
            output = io.StringIO()
            fields = ["identifier", "title", "creator", "year", "media_type", "url", "notes"]
            writer = csv.DictWriter(output, fields, extrasaction="ignore")
            writer.writeheader()
            # Prevent spreadsheet formula execution from upstream metadata or notes.
            for row in rows:
                writer.writerow(
                    {
                        key: (
                            "'" + str(row[key])
                            if str(row[key]).lstrip().startswith(("=", "+", "-", "@"))
                            else row[key]
                        )
                        for key in fields
                    }
                )
            body, media_type, extension = output.getvalue(), "text/csv", "csv"
        else:
            body = (
                "To: info@archive.org\nSubject: Curation destination inquiry: "
                + collection["name"]
                + "\n\nHello,\n\nI am curating existing items uploaded by multiple accounts. "
                "Could you advise whether a collection with a permitted Simple Lists destination "
                "is appropriate for this selection, and how to provision it?\n\n"
                "This is a curation request, not a claim that I uploaded these items or meet "
                "the standard minimum-upload requirement.\n\n"
                "Collection: "
                + collection["name"]
                + "\nDescription: "
                + collection["description"]
                + "\nMy Archive account: [add your account]\n"
                "Suggested identifier: [add an identifier]\nSubject tags: [add tags]\n\n"
                + f"Reviewed selection ({len(rows)} items):\n"
                + "\n".join(row["url"] for row in rows)
                + "\n"
            )
            media_type, extension = "text/plain", "txt"
        return Response(
            body,
            media_type=media_type,
            headers={
                "Content-Disposition": f'attachment; filename="arkhiv-curation-{collection_id}.{extension}"'
            },
        )

    install_curation_routes(
        app,
        database,
        core
        or CoreClient(
            os.getenv("ARKHIV_CURATION_CORE_URL", ""), os.getenv("ARKHIV_CURATION_CORE_TOKEN", "")
        ),
    )

    app.mount("/static", StaticFiles(directory=STATIC), name="static")

    @app.get("/")
    def index():
        return FileResponse(STATIC / "index.html")

    return app
