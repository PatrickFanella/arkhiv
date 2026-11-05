#!/usr/bin/env bash
set -euo pipefail
image=${1:?image is required}
docker run --rm --network none --read-only --tmpfs /tmp:rw,noexec,nosuid,size=32m \
  --cap-drop ALL --security-opt no-new-privileges:true "$image" nginx -t
fixture=$(docker create --network none --read-only --tmpfs /tmp:rw,noexec,nosuid,size=32m \
  --memory 128m --memory-swap 128m --cpus 0.5 --pids-limit 64 \
  --cap-drop ALL --security-opt no-new-privileges:true "$image")
trap 'docker rm -f "$fixture" >/dev/null' EXIT
docker start "$fixture" >/dev/null
for _attempt in {1..20}; do
  if docker exec "$fixture" wget -q -T 2 -O /dev/null http://127.0.0.1:8080/healthz; then
    break
  fi
  sleep 0.2
done
html=$(docker exec "$fixture" wget -q -T 2 -O - http://127.0.0.1:8080/)
grep -Fq 'An archive across formats.' <<< "$html"
grep -Fq 'href="https://arkhiv.subcult.tv/"' <<< "$html"
grep -Fq 'name="twitter:card" content="summary_large_image"' <<< "$html"
for path in favicon.ico apple-touch-icon.png og.png site.json; do
  docker exec "$fixture" wget -q -T 2 -O /dev/null "http://127.0.0.1:8080/$path"
done
