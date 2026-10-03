#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
project="arkhiv-reader-proof-${BASHPID}"
proof_dir="$(mktemp -d "${TMPDIR:-/tmp}/arkhiv-reader-proof.XXXXXXXX")"
python3 - "$proof_dir" <<'PY'
import json,sys,pathlib
p=pathlib.Path(sys.argv[1])
(p/'expected.json').write_text(json.dumps({'published':2,'queries':[{'collection':'fixture-zines','query':'knownword','minimum':1}]}))
(p/'override.json').write_text(json.dumps({'services':{'checks':{'build':{'context':str(pathlib.Path.cwd()),'dockerfile':'web/Dockerfile.readerprobe'},'volumes':[str(p)+':/proof']}}}))
PY
compose=(docker compose -p "$project" -f scripts/browser-compose.yml -f "$proof_dir/override.json")
trap '"${compose[@]}" down --volumes' EXIT
"${compose[@]}" build
"${compose[@]}" up -d --wait db api web
"${compose[@]}" run --rm checks
printf 'Reader proof retained at %s\n' "$proof_dir"
