#!/usr/bin/env python3
"""Export a pinned, generated viewer adapter; this repository owns its source."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess

FILES = ['Player.tsx', 'model.ts', 'model.test.ts', 'useViewer.ts', 'Viewer.tsx', 'Compatibility.tsx']
def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--destination', required=True)
    args = parser.parse_args()
    repo = Path(__file__).resolve().parents[1]
    if subprocess.check_output(['git', 'status', '--porcelain'], cwd=repo): raise RuntimeError('Export a clean committed engine revision')
    destination = Path(args.destination).resolve()
    destination.mkdir(parents=True, exist_ok=True)
    hashes = {}
    for name in FILES:
        source = repo / 'web/src' / name
        shutil.copy2(source, destination / name)
        hashes[name] = hashlib.sha256(source.read_bytes()).hexdigest()
    manifest = {'repository': 'https://git.subcult.tv/subculture-collective/VHNESS', 'revision': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=repo, text=True).strip(), 'api_version': '1', 'files': hashes, 'license': 'GPL-3.0', 'ownership': 'Generated copy; edit upstream, export at a reviewed revision, then verify and commit the app upgrade.'}
    (destination / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')

if __name__ == '__main__': main()
