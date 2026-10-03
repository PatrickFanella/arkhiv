#!/usr/bin/env python3
"""Build a branded frontend over its reviewed shared engine digest."""
import argparse
import json
from pathlib import Path
import re
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--npmrc', type=Path, required=True, help='Private registry configuration outside the checkout')
    parser.add_argument('--tag', required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    config = args.npmrc.resolve()
    if not config.is_file() or config.is_relative_to(root):
        raise SystemExit('Supply a private npm configuration outside the Docker context')
    if not re.fullmatch(r'[a-zA-Z0-9][a-zA-Z0-9._/:-]+', args.tag):
        raise SystemExit('Invalid consumer image tag')
    if subprocess.check_output(['git', 'status', '--porcelain'], cwd=root):
        raise SystemExit('Build the committed reviewed consumer source')
    subprocess.run(['python3', 'scripts/check_engine.py'], cwd=root, check=True)
    release = json.loads((root / 'deploy/platform-release.json').read_text())
    revision = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip()
    subprocess.run(['docker', 'build', '--build-arg', 'ENGINE_IMAGE=' + release['engine_digest'],
                    '--secret', 'id=npmrc,src=' + str(config),
                    '--label', 'org.opencontainers.image.revision=' + revision,
                    '--label', 'org.subcult.archive-engine=' + release['engine_digest'],
                    '--label', 'org.subcult.archive-viewer=' + release['version'],
                    '-t', args.tag, '.'], cwd=root, check=True)
    print('Consumer image:', subprocess.check_output(['docker', 'image', 'inspect', '--format', '{{.Id}}', args.tag], text=True).strip())


if __name__ == '__main__':
    main()
