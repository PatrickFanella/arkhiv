"""Local review routes. The core connection never writes or approves access."""

import json
import time
from typing import Literal

from fastapi import HTTPException
from fastapi.responses import Response
from pydantic import BaseModel, Field, field_validator

from .core import DENIED, ID, ITEM_ID, SCHEMA, CoreError, evidence_hash


class BindingInput(BaseModel):
    target_collection: str = Field(pattern="^" + ID + "$")


class AttachmentInput(BaseModel):
    item_id: str = Field(pattern="^" + ITEM_ID + "$")
    representations: list[str] = Field(min_length=1, max_length=64)
    notes: str = Field(min_length=1, max_length=4000)

    @field_validator("notes")
    @classmethod
    def reviewed_note(cls, value):
        if not value.strip():
            raise ValueError("Record why this item belongs in the collection.")
        return value.strip()


def install_curation_routes(app, database, core):
    def collection(db, collection_id):
        row = db.execute("SELECT * FROM collections WHERE id=?", (collection_id,)).fetchone()
        if not row:
            raise HTTPException(404, "Collection not found.")
        return dict(row)

    def read_core(method, *args, **kwargs):
        try:
            return method(*args, **kwargs)
        except CoreError as exc:
            raise HTTPException(503, str(exc)) from exc

    @app.get("/api/core/status")
    def status():
        return {
            "configured": bool(core.base_url),
            "workflow": "curation",
            "access_approval": "separate",
        }

    @app.get("/api/core/catalog")
    def catalog(
        q: str = "",
        media: Literal["", "print", "audio", "video"] = "",
        collection: str = "",
        page: int = 1,
    ):
        if page < 1 or page > 100000 or len(q) > 200 or len(collection) > 200:
            raise HTTPException(422, "Invalid catalog filter.")
        return read_core(core.browse, q=q, media=media, collection=collection, page=page)

    @app.get("/api/collections/{collection_id}/core")
    def attachments(collection_id: int):
        with database.connect() as db:
            collection(db, collection_id)
            binding = db.execute(
                "SELECT target_collection FROM core_bindings WHERE collection_id=?",
                (collection_id,),
            ).fetchone()
            rows = db.execute(
                "SELECT * FROM core_attachments WHERE collection_id=? ORDER BY attached_at,core_item_id",
                (collection_id,),
            ).fetchall()
            return {
                "target_collection": binding[0] if binding else "",
                "attachments": [
                    dict(row) | {"evidence": json.loads(row["evidence"])} for row in rows
                ],
            }

    @app.put("/api/collections/{collection_id}/core")
    def bind(collection_id: int, value: BindingInput):
        with database.connect() as db:
            collection(db, collection_id)
            db.execute(
                "INSERT INTO core_bindings VALUES (?,?,?) ON CONFLICT(collection_id) DO UPDATE SET target_collection=excluded.target_collection,updated_at=excluded.updated_at",
                (collection_id, value.target_collection, time.time()),
            )
        return {"target_collection": value.target_collection, "access_approval": "separate"}

    @app.post("/api/collections/{collection_id}/core/attachments", status_code=201)
    def attach(collection_id: int, value: AttachmentInput):
        # Fetch evidence on the server; never trust a browser-supplied snapshot or credits.
        with database.connect() as db:
            collection(db, collection_id)
        item = read_core(core.item, value.item_id)
        selected = set(value.representations)
        available = {rep["id"] for rep in item["representations"]}
        if len(selected) != len(value.representations) or not selected <= available:
            raise HTTPException(422, "Choose representations present in the combined catalog.")
        evidence = item | {
            "representations": [rep for rep in item["representations"] if rep["id"] in selected]
        }
        now = time.time()
        with database.connect() as db:
            collection(db, collection_id)
            for source_id in sorted({rep["source_item_id"] for rep in evidence["representations"]}):
                media = sorted(
                    {
                        rep["media"]
                        for rep in evidence["representations"]
                        if rep["source_item_id"] == source_id
                    }
                )
                # Existing acceptance/rejection, notes, and discovery provenance survive attachment.
                db.execute(
                    "INSERT OR IGNORE INTO items(collection_id,identifier,title,creator,description,media_type,year,subjects,source_collections,discovered_at) VALUES (?,?,?,?,?,?,?,?,?,?)",
                    (
                        collection_id,
                        source_id,
                        item["title"],
                        item["creator"],
                        item["description"],
                        ", ".join(media),
                        item["date"][:4],
                        "",
                        ", ".join(item["collections"]),
                        now,
                    ),
                )
            db.execute(
                "INSERT INTO core_attachments VALUES (?,?,?,?,?,?,?) ON CONFLICT(collection_id,core_item_id) DO UPDATE SET core_url=excluded.core_url,evidence=excluded.evidence,evidence_sha256=excluded.evidence_sha256,notes=excluded.notes,attached_at=excluded.attached_at",
                (
                    collection_id,
                    item["id"],
                    core.base_url,
                    json.dumps(evidence, ensure_ascii=False),
                    evidence_hash(evidence),
                    value.notes,
                    now,
                ),
            )
        return {"item_id": item["id"], "review_required": True, "access_approval": "separate"}

    @app.delete("/api/collections/{collection_id}/core/attachments/{item_id}")
    def detach(collection_id: int, item_id: str):
        with database.connect() as db:
            collection(db, collection_id)
            db.execute(
                "DELETE FROM core_attachments WHERE collection_id=? AND core_item_id=?",
                (collection_id, item_id),
            )
        return {"item_id": item_id}

    @app.get("/api/collections/{collection_id}/manifest")
    def manifest(collection_id: int):
        with database.connect() as db:
            selected_collection = collection(db, collection_id)
            binding = db.execute(
                "SELECT target_collection FROM core_bindings WHERE collection_id=?",
                (collection_id,),
            ).fetchone()
            if not binding:
                raise HTTPException(
                    409, "Choose a target core collection before exporting the migration manifest."
                )
            accepted = {
                row["identifier"]: dict(row)
                for row in db.execute(
                    "SELECT * FROM items WHERE collection_id=? AND status='accepted' ORDER BY identifier",
                    (collection_id,),
                )
            }
            rows = db.execute(
                "SELECT * FROM core_attachments WHERE collection_id=? ORDER BY core_item_id",
                (collection_id,),
            ).fetchall()
        mappings, mapped = [], set()
        for row in rows:
            evidence = json.loads(row["evidence"])
            sources = {rep["source_item_id"] for rep in evidence["representations"]}
            if not sources <= accepted.keys():
                continue
            if row["core_url"] != core.base_url:
                raise HTTPException(
                    409,
                    "The catalog connection changed. Reattach items from the current catalog before exporting.",
                )
            fresh = read_core(core.item, row["core_item_id"])
            reps = {rep["id"] for rep in evidence["representations"]}
            fresh["representations"] = [
                rep for rep in fresh["representations"] if rep["id"] in reps
            ]
            if evidence_hash(fresh) != row["evidence_sha256"]:
                raise HTTPException(
                    409, "Catalog evidence changed. Reattach and review the item before exporting."
                )
            mappings.append(
                {
                    "catalog_item": evidence,
                    "evidence_sha256": row["evidence_sha256"],
                    "curation_notes": row["notes"],
                    "attached_at": row["attached_at"],
                    "access": DENIED.copy(),
                }
            )
            mapped.update(sources)
        result = {
            "schema": SCHEMA,
            "intent": "reviewed-curation",
            "exported_at": time.time(),
            "collection": selected_collection,
            "target_collection": binding[0],
            "access": DENIED.copy(),
            "mappings": mappings,
            "unresolved_items": [
                accepted[key]
                | {"reason": "Core source snapshot and selected-file mapping required."}
                for key in sorted(accepted.keys() - mapped)
            ],
        }
        return Response(
            json.dumps(result, ensure_ascii=False, indent=2),
            media_type="application/json",
            headers={
                "Content-Disposition": f'attachment; filename="arkhiv-curation-{collection_id}-core-manifest.json"'
            },
        )
