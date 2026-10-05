"""Read-only Archive searches with query-bound identifier checkpoints."""

import base64
import hashlib
import html
import json
import re

import httpx

FIELDS = "identifier,title,creator,description,mediatype,year,date,subject,collection,publicdate"
IDENTIFIER = re.compile(r"^[A-Za-z0-9_.-]{1,1000}$")


def query_for(search: dict) -> str:
    parts = [f"({search['query']})", "NOT mediatype:collection"]
    for key, field in (("media_type", "mediatype"), ("source_collection", "collection")):
        if search.get(key):
            value = search[key].replace("\\", "\\\\").replace('"', '\\"')
            parts.append(f'{field}:"{value}"')
    if search.get("year_from") or search.get("year_to"):
        parts.append(f"year:[{search.get('year_from') or '*'} TO {search.get('year_to') or '*'}]")
    return " AND ".join(parts)


def text(value) -> str:
    if isinstance(value, list):
        return "; ".join(text(part) for part in value)
    return html.unescape(re.sub(r"<[^>]*>", " ", str(value or ""))).strip()


def normalize(item: dict) -> dict | None:
    identifier = item.get("identifier")
    if not isinstance(identifier, str) or not IDENTIFIER.fullmatch(identifier):
        return None
    return {
        "identifier": identifier,
        "title": text(item.get("title")) or identifier,
        "creator": text(item.get("creator")),
        "description": text(item.get("description"))[:12000],
        "media_type": text(item.get("mediatype")),
        "year": text(item.get("year") or item.get("date"))[:40],
        "subjects": text(item.get("subject"))[:4000],
        "source_collections": text(item.get("collection"))[:4000],
    }


class ArchiveClient:
    def fetch(self, query: str, cursor: str | None) -> dict:
        digest = hashlib.sha256(query.encode()).hexdigest()
        effective_query = query
        if cursor:
            try:
                checkpoint = json.loads(base64.b64decode(cursor, validate=True))
                after = checkpoint["after"]
                if checkpoint["query_hash"] != digest or not IDENTIFIER.fullmatch(after):
                    raise ValueError
            except (ValueError, KeyError, TypeError) as exc:
                raise ValueError("Saved scan cursor is incompatible. Restart this scan.") from exc
            # Inclusive range plus explicit exclusion works with Archive's search parser.
            # Always start at offset zero to avoid the 10,000-offset paging ceiling.
            effective_query = f"({query}) AND identifier:[{after} TO *] AND NOT identifier:{after}"
        params = {
            "q": effective_query,
            "fl[]": FIELDS,
            "sort[]": "identifier asc",
            "rows": "100",
            "page": "1",
            "output": "json",
        }
        with httpx.Client(timeout=20, follow_redirects=False) as client:
            response = client.get(
                "https://archive.org/advancedsearch.php",
                params=params,
                headers={"User-Agent": "Arkhiv-Curation/0.1 (private Archive curation workbench)"},
            )
            response.raise_for_status()
            payload = response.json()
        if not isinstance(payload, dict):
            raise TypeError("Archive returned an unexpected search response.")
        if payload.get("error"):
            raise ValueError("Archive could not execute this query. Check its syntax.")
        result = payload.get("response", {})
        header = payload.get("responseHeader", {})
        echoed = header.get("params", {}).get("qin")
        if not isinstance(echoed, str) or html.unescape(echoed) != effective_query:
            raise ValueError("Archive returned results for a different query. No items were saved.")
        items = result.get("docs")
        total = result.get("numFound")
        if not isinstance(items, list) or not isinstance(total, int):
            raise TypeError("Archive returned an unexpected search response.")
        if total > 0 and not items:
            raise ValueError("Archive returned no page for a nonempty search. Retry this scan.")
        next_cursor = None
        if items and total > len(items):
            last = items[-1].get("identifier")
            if not isinstance(last, str) or not IDENTIFIER.fullmatch(last):
                raise ValueError("Archive returned an invalid item checkpoint.")
            next_cursor = base64.b64encode(
                json.dumps(
                    {
                        "after": last,
                        "query_hash": digest,
                    }
                ).encode()
            ).decode()
        return {"items": items, "cursor": next_cursor, "total": total}
