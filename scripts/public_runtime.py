#!/usr/bin/env python3
"""Check and maintain an explicitly configured public deployment.

Config, credentials, checkpoints and receipts belong outside the checkout.
This operator never acquires media, approves access or removes originals/backups.
"""
import argparse
import datetime
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import shlex
import subprocess


def run(args, **kwargs):
    result = subprocess.run(args, capture_output=True, **kwargs)
    if result.returncode:
        raise RuntimeError(f"{args[0]} failed with exit {result.returncode}")
    return result.stdout


def digest(path):
    h = hashlib.sha256()
    with path.open("rb") as f:
        for block in iter(lambda: f.read(1024 * 1024), b""):
            h.update(block)
    return h.hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=["monitor", "replay", "backup"])
    parser.add_argument("--config", required=True, type=Path)
    args = parser.parse_args()
    cfg = json.loads(args.config.read_text())
    root = Path(cfg["root"]).resolve()
    state = Path(cfg["state"]).resolve()
    state.mkdir(mode=0o700, parents=True, exist_ok=True)
    lock = (state / "maintenance.lock").open("a")
    # Queue this deployment's jobs; a timed monitor can overlap a checkpoint.
    # Keep the shared lock intact until the active owner completes.
    fcntl.flock(lock, fcntl.LOCK_EX)
    docker = cfg.get("docker", ["docker"])
    project = cfg["project"]
    database = cfg["database_container"]
    inspect = json.loads(run(docker + ["inspect", database]))[0]
    if inspect["Config"]["Labels"].get("com.docker.compose.project") != project:
        raise RuntimeError("Database belongs to another deployment")

    def sql(db, query):
        if not re.fullmatch(r"[a-z_]+", db):
            raise RuntimeError("Invalid configured database")
        return run(docker + ["exec", database, "psql", "-v", "ON_ERROR_STOP=1",
                            "-U", cfg["database_owner"], "-d", db, "-Atc", query],
                   text=True).strip()

    now = datetime.datetime.now(datetime.timezone.utc)
    result = {"action": args.action, "observed_at": now.isoformat(), "project": project}
    if args.action == "monitor":
        observations = []
        for site in cfg["sites"]:
            request = subprocess.run(
                ["curl", "--silent", "--show-error", "--max-time", "30", "--retry", "2",
                 "--retry-delay", "2", "--retry-max-time", "90", "--write-out", "\n%{http_code}",
                 site["url"]], capture_output=True, text=True)
            body, _, status = request.stdout.rpartition("\n")
            if request.returncode or status != "200":
                failure = {**result, "url": site["url"], "http_status": status,
                           "transport_exit": request.returncode}
                (state / "monitor-failure.json").write_text(json.dumps(failure, indent=2) + "\n")
                raise RuntimeError(f"Public check failed at {site['url']}: HTTP {status}; "
                                   f"transport exit {request.returncode}")
            data = json.loads(body)
            if site.get("total") is not None and data.get("total") != site["total"]:
                raise RuntimeError(f"Catalog count changed at {site['url']}")
            for key, value in site.get("equals", {}).items():
                if data.get(key) != value:
                    raise RuntimeError(f"Public contract differs at {site['url']}")
            observations.append({"url": site["url"], "total": data.get("total"),
                                 "audience": data.get("audience"), "public": data.get("public")})
        result["sites"] = observations
    elif args.action == "replay":
        results = []
        core = json.loads(run(docker + ["inspect", cfg["core_container"]]))[0]
        if core["Config"]["Labels"].get("com.docker.compose.project") != project:
            raise RuntimeError("Core belongs to another deployment")
        image = core["Image"]
        for job in cfg["replays"]:
            scope = job["collection"]
            if not re.fullmatch(r"[A-Za-z0-9_-]+", scope):
                raise RuntimeError("Invalid configured collection")
            # Planning must fail before writing if any eligible program lacks
            # an exact current public review. New source/selection needs review.
            query = ("SELECT count(*) FROM episodes e WHERE collection_id='" + scope +
                     "' AND decision->>'published'='true' AND decision->>'playback'='true' "
                     "AND decision->>'channel'='true' AND NOT unavailable AND NOT EXISTS "
                     "(SELECT 1 FROM episode_public_reviews p WHERE p.episode_id=e.id AND "
                     "p.approved AND p.source_hash=e.source_hash AND p.file=e.decision->>'file')")
            if sql(job["database"], query) != "0":
                raise RuntimeError(f"Unreviewed public replay candidate in {scope}")
            secret = (root / job["database_url_file"]).resolve()
            if not secret.is_relative_to(root) or not secret.is_file():
                raise RuntimeError("Operator URL must be inside the private deployment")
            raw = run(docker + ["run", "--rm", "--network", cfg["network"],
                               "--read-only", "--cap-drop", "ALL", "--security-opt",
                               "no-new-privileges:true", "--memory", "512m", "--cpus", "1",
                               "--tmpfs", "/tmp:size=64m", "-v", str(secret) + ":/run/secrets/operator-url:ro",
                               "-e", "DATABASE_URL_FILE=/run/secrets/operator-url", "-e",
                               "VHNESS_COLLECTION=" + scope, image, "plan-replay", "--days", "7"])
            receipt = state / ("replay-" + job["database"] + ".json")
            receipt.write_bytes(raw)
            results.append({"database": job["database"], "collection": scope, "saved": True})
        result["replays"] = results
    else:
        checkpoints = state / "checkpoints"
        checkpoints.mkdir(mode=0o700, exist_ok=True)
        used = sum(p.stat().st_size for p in checkpoints.rglob("*") if p.is_file())
        budget = cfg["checkpoint_budget_bytes"]
        reserve = cfg["free_space_reserve_bytes"]
        if used + cfg["checkpoint_write_allowance_bytes"] > budget:
            raise RuntimeError("Checkpoint budget requires an operator retention review")
        fs = os.statvfs(checkpoints)
        if fs.f_bavail * fs.f_frsize < reserve + cfg["checkpoint_write_allowance_bytes"]:
            raise RuntimeError("Checkpoint free-space reserve would be crossed")
        destination = checkpoints / now.strftime("public-%Y%m%dT%H%M%SZ")
        destination.mkdir(mode=0o700)
        entries = []
        for db in cfg["databases"]:
            tables = sql(db, "SELECT schemaname||'.'||tablename FROM pg_tables WHERE "
                         "schemaname='public' OR schemaname LIKE 'rollout_stage_%' ORDER BY 1").splitlines()
            counts = {}
            for table in tables:
                if not re.fullmatch(r"[a-z_]+\.[a-z_]+", table):
                    raise RuntimeError("Unexpected table identifier")
                counts[table] = int(sql(db, "SELECT count(*) FROM " + table))
            path = destination / (db + ".dump")
            with path.open("wb") as stream:
                p = subprocess.run(docker + ["exec", database, "pg_dump", "-U", cfg["database_owner"],
                                            "-d", db, "-Fc"], stdout=stream, stderr=subprocess.PIPE)
            if p.returncode:
                raise RuntimeError("Public catalog checkpoint failed")
            path.chmod(0o600)
            entries.append({"database": db, "filename": path.name, "bytes": path.stat().st_size,
                            "sha256": digest(path), "table_counts": counts})
        manifest = {"format": 1, "owner": project, "observed_at": now.isoformat(), "databases": entries,
                    "media": "Immutable originals retain their separately verified independent backups"}
        (destination / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
        (destination / "manifest.json").chmod(0o600)
        mirror = cfg["mirror"]
        remote_dir = mirror["path"].rstrip("/") + "/" + destination.name
        run(["ssh", mirror["host"], "mkdir -p -m 700 " + shlex.quote(remote_dir)])
        for path in destination.iterdir():
            run(["scp", "-q", str(path), mirror["host"] + ":" + remote_dir + "/"])
        for entry in entries:
            remote = run(["ssh", mirror["host"], "sha256sum " + shlex.quote(remote_dir + "/" + entry["filename"])], text=True)
            if remote.split()[0] != entry["sha256"]:
                raise RuntimeError("Independent checkpoint checksum differs")
        result.update({"checkpoint": str(destination), "mirror": mirror["host"] + ":" + remote_dir,
                       "independent_metadata_checksums_match": True, "databases": len(entries),
                       "restore_verified": False})
    (state / (args.action + ".json")).write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps(result))


if __name__ == "__main__":
    main()
