#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
project="arkhiv-browser-${BASHPID}"
compose=(docker compose -p "$project" -f scripts/browser-compose.yml)
trap '"${compose[@]}" down --volumes' EXIT
"${compose[@]}" build
"${compose[@]}" up -d db api web
"${compose[@]}" run --rm checks
