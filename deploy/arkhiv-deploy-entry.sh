#!/bin/sh
# Forced command for the deploy key: only `deploy --target <name> [--dry-run]` reaches the host script.
set -eu
command="${SSH_ORIGINAL_COMMAND:-}"
case "$command" in
  "deploy --target platform"|"deploy --target anarchist"|"deploy --target dedhed") target="${command#deploy --target }"; dry="" ;;
  "deploy --target platform --dry-run"|"deploy --target anarchist --dry-run"|"deploy --target dedhed --dry-run") target="${command#deploy --target }"; target="${target% --dry-run}"; dry="--dry-run" ;;
  *) echo "refused: $command" >&2; exit 2 ;;
esac
home="${HOME:?}"
exec python3 "$home/.local/state/arkhiv-deploy/public_deploy.py" apply --config "$home/.local/state/arkhiv-deploy/deploy.json" --target "$target" $dry
