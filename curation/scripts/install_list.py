"""Install a reviewed search-list definition into a local Arkhiv curation workspace."""

import argparse
import json
import urllib.request
from pathlib import Path


def request(base, path, payload=None):
    body = None if payload is None else json.dumps(payload).encode()
    req = urllib.request.Request(
        base + "/api" + path,
        data=body,
        headers={"Content-Type": "application/json"},
    )
    with urllib.request.urlopen(req, timeout=20) as response:
        return json.load(response)


def install(spec, base):
    collections = request(base, "/collections")
    collection = next((c for c in collections if c["name"] == spec["name"]), None)
    if not collection:
        collection = request(
            base,
            "/collections",
            {
                "name": spec["name"],
                "description": spec["description"],
            },
        )
    existing = request(base, f"/collections/{collection['id']}/searches")
    excluded = spec.get("baseline_identifiers", [])
    suffix = " AND NOT identifier:(" + " OR ".join(excluded) + ")" if excluded else ""
    for search in spec["searches"]:
        query = "(" + search["query"] + ")" + suffix
        found = next((s for s in existing if s["name"] == search["name"]), None)
        if found:
            print(f"Kept existing search: {search['name']} (no edits)")
            continue
        request(
            base,
            f"/collections/{collection['id']}/searches",
            {
                "name": search["name"],
                "query": query,
                "media_type": "texts",
                "interval_hours": 168,
                "enabled": True,
            },
        )
        print(f"Created weekly search: {search['name']}")
    print(f"Collection: {collection['name']} (local ID {collection['id']})")
    return collection["id"]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("list", type=Path)
    parser.add_argument("--url", default="http://127.0.0.1:18764")
    parser.add_argument(
        "--apply", action="store_true", help="Create missing local collection/searches"
    )
    args = parser.parse_args()
    spec = json.loads(args.list.read_text())
    if args.apply:
        install(spec, args.url.rstrip("/"))
    else:
        print(f"Collection: {spec['name']}")
        print(f"Exclude {len(spec.get('baseline_identifiers', []))} existing source identifiers")
        for search in spec["searches"]:
            print(f"Weekly: {search['name']}\n  {search['query']}")
        print("Use --apply to create missing entries in the local workspace.")


if __name__ == "__main__":
    main()
