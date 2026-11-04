#!/usr/bin/env python3
"""Write the independent viewer (SDK) receipt for a packed, published artifact."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import tarfile

from download_release_package import REGISTRY


def latest_viewer_receipt(releases):
    receipts = sorted(releases.glob('viewer-release-*.json'), key=lambda p: tuple(int(x) for x in p.stem.rsplit('-', 1)[1].split('.')))
    return receipts[-1] if receipts else None


def receipt(root, package, compatible_core=None):
    status = subprocess.check_output(['git', 'status', '--porcelain', '--', 'packages', 'migrations', 'backend'], cwd=root, text=True)
    if status:
        raise SystemExit('Release requires a clean committed source revision:\n' + status)
    with tarfile.open(package) as archive:
        try:
            member = archive.extractfile('package/package.json')
        except KeyError:
            member = None
        if member is None:
            raise SystemExit('Packed artifact has no package metadata')
        metadata = json.load(member)
    source = json.loads((root / 'packages/viewer/package.json').read_text())
    if metadata['name'] != '@subcult/archive-viewer' or metadata['version'] != source['version'] or metadata.get('license') != source.get('license'):
        raise SystemExit('Packed artifact does not match the source package')
    if not re.fullmatch(r'\d+\.\d+\.\d+', metadata['version']):
        raise SystemExit('Expected a stable semantic version')
    releases = root / 'docs/releases'
    if compatible_core is None:
        previous = latest_viewer_receipt(releases)
        if previous is None:
            raise SystemExit('Name the compatible core receipt; no previous viewer receipt exists')
        compatible_core = json.loads(previous.read_text())['compatible_core_receipt']
    if not re.fullmatch(r'core-release-\d+\.\d+\.\d+\.json', compatible_core) or not (releases / compatible_core).is_file():
        raise SystemExit('Compatible core receipt must be a committed core release receipt')
    return {
        'package': metadata['name'],
        'version': metadata['version'],
        'source_revision': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip(),
        'package_file': package.name,
        'package_sha256': hashlib.sha256(package.read_bytes()).hexdigest(),
        'license': metadata['license'],
        'scope': 'Frontend SDK patch; core artifacts are versioned independently',
        'compatible_core_receipt': compatible_core,
        'registry': REGISTRY,
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--package', type=Path, required=True, help='Packed tarball exactly as published')
    parser.add_argument('--compatible-core', help='Core receipt filename; defaults to the previous viewer receipt\'s value')
    parser.add_argument('--output', type=Path, help='Defaults to docs/releases/viewer-release-<version>.json')
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    result = receipt(root, args.package.resolve(), args.compatible_core)
    output = args.output or root / 'docs/releases' / ('viewer-release-' + result['version'] + '.json')
    if output.name != 'viewer-release-' + result['version'] + '.json':
        raise SystemExit('Receipt filename must carry the package version')
    with output.open('x') as handle:
        handle.write(json.dumps(result, indent=2) + '\n')
    print('Viewer release receipt:', output)


if __name__ == '__main__':
    main()
