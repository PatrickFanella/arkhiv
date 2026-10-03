#!/usr/bin/env python3
"""Bind packed frontend and immutable engine artifacts to a clean source revision."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import tarfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--package', type=Path, required=True)
    parser.add_argument('--engine-digest', required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    status = subprocess.check_output(['git', 'status', '--porcelain'], cwd=root, text=True)
    if status:
        raise SystemExit('Release requires a clean committed source revision:\n' + status)
    contract = json.loads((root / 'docs/reference/release-contract.json').read_text())
    if not re.fullmatch(re.escape(contract['engine_repository']) + r'@sha256:[0-9a-f]{64}', args.engine_digest):
        raise SystemExit('Supply the immutable registry engine digest, not a mutable tag or local image ID')
    if args.output.exists():
        raise SystemExit('Do not overwrite a release manifest')
    with tarfile.open(args.package) as archive:
        member = archive.extractfile('package/package.json')
        if member is None:
            raise SystemExit('Packed artifact has no package metadata')
        package = json.load(member)
    source_package = json.loads((root / 'packages/viewer/package.json').read_text())
    if package['name'] != contract['package'] or package['version'] != source_package['version']:
        raise SystemExit('Packed artifact does not match the source package release')
    # Bind every shipped file to the built source; tarball metadata can vary.
    with tarfile.open(args.package) as archive:
        for member in archive.getmembers():
            if member.isdir():
                continue
            if not member.isfile():
                raise SystemExit('Package contains an unsupported link or special file')
            name = Path(member.name)
            if name.parts[0] != 'package' or '..' in name.parts:
                raise SystemExit('Invalid package member')
            relative = Path(*name.parts[1:])
            expected = root / 'packages/viewer' / relative
            actual = archive.extractfile(member)
            if actual is None or not expected.is_file():
                raise SystemExit('Packed bytes differ from built source: ' + str(relative))
            data = actual.read()
            # pnpm normalizes package.json formatting during packing. Compare
            # its complete metadata, while executable/other files remain exact.
            matches = json.loads(data) == json.loads(expected.read_bytes()) if relative == Path('package.json') else data == expected.read_bytes()
            if not matches:
                raise SystemExit('Packed bytes differ from built source: ' + str(relative))
    manifest = {
        **contract,
        'source_revision': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip(),
        'version': package['version'],
        'license': package['license'],
        'package_sha256': hashlib.sha256(args.package.read_bytes()).hexdigest(),
        'package_file': args.package.name,
        'engine_digest': args.engine_digest,
        'migrations': {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted((root / 'migrations').glob('*.sql'))},
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(manifest, indent=2) + '\n')
    print('Release manifest:', args.output)


if __name__ == '__main__':
    main()
