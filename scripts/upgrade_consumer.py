#!/usr/bin/env python3
"""Prepare a compatible shared release in an explicitly registered consumer checkout."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
from core_release import validate_core


def select_consumer(registry, repository, release):
    """Select the independent API or component lane for this artifact."""
    api_only = release.get('artifact_kind') == 'core'
    matches = [c for c in registry['consumers'] if c['repository'] == repository
               and (c.get('mode') == 'api') == api_only]
    if len(matches) != 1:
        raise ValueError('Expected one registered consumer for this release lane')
    return matches[0]


def validate(release, consumer, current, artifact):
    if consumer.get('mode') == 'api':
        validate_core(release, consumer, current)
        return
    if consumer['adoption'] != 'package':
        raise ValueError('Consumer has not completed package adoption')
    mode = consumer.get('mode', 'platform')
    if mode not in ('platform', 'components'):
        raise ValueError('Unsupported consumer mode')
    if mode == 'components' and consumer.get('engine_file'):
        raise ValueError('Component consumers cannot register a shared engine pin')
    if consumer['rollout'] != 'reviewed-pr':
        raise ValueError('Unsupported rollout policy')
    for key in ('contract_version', 'configuration_version', 'package'):
        if release[key] != current[key]:
            raise ValueError('Incompatible ' + key)
    if not set(current['api_versions']).issubset(release['api_versions']):
        raise ValueError('Required API version is unsupported')
    if not set(consumer['media']).issubset(release['media']):
        raise ValueError('Enabled media is unsupported')
    if release['registry'] != current['registry'] or release['engine_repository'] != current['engine_repository']:
        raise ValueError('Registry ownership changed')
    if not re.fullmatch(r'\d+\.\d+\.\d+', release['version']):
        raise ValueError('Expected a stable semantic version')
    old_version = tuple(map(int, current['version'].split('.')))
    new_version = tuple(map(int, release['version'].split('.')))
    if new_version[0] != old_version[0] or new_version < old_version:
        raise ValueError('Major changes and downgrades require a coordinated migration')
    if release['version'] == current['version'] and release != current:
        raise ValueError('Release version was reused for different metadata')
    if not re.fullmatch(re.escape(release['engine_repository']) + r'@sha256:[0-9a-f]{64}', release['engine_digest']):
        raise ValueError('Engine must be pinned to its registry digest')
    if not re.fullmatch(r'[0-9a-f]{40}', release['source_revision']):
        raise ValueError('Expected a full source revision')
    if artifact is None:
        raise ValueError('Package consumers require the verified package artifact')
    if hashlib.sha256(artifact.read_bytes()).hexdigest() != release['package_sha256']:
        raise ValueError('Package checksum mismatch')
    for name, digest in current['migrations'].items():
        if release['migrations'].get(name) != digest:
            raise ValueError('Previously applied migration changed: ' + name)


def prepare(root, consumer, release, artifact, run=subprocess.run):
    if subprocess.check_output(['git', 'status', '--porcelain'], cwd=root):
        raise ValueError('Use a clean isolated consumer checkout')
    api_only = consumer.get('mode') == 'api'
    keys = ['release_file', 'engine_file'] if api_only else ['release_file', 'package_file', 'lock_file']
    if consumer.get('mode', 'platform') == 'platform':
        keys.append('engine_file')
    paths = [root / consumer[k] for k in keys]
    for path in paths:
        if not path.resolve().is_relative_to(root.resolve()) or not path.is_file():
            raise ValueError('Registered dependency file is missing or outside the checkout')
    if len(set(p.resolve() for p in paths)) != len(paths):
        raise ValueError('Registered dependency paths must be distinct')
    if api_only:
        receipt, engine_file = paths
        current = json.loads(receipt.read_text())
        validate_core(release, consumer, current)
        if engine_file.read_text() != 'ARCHIVE_ENGINE_IMAGE=' + current['engine_digest'] + '\n':
            raise ValueError('Core pin differs from the current release receipt')
        if current == release:
            return False
        original = {p: p.read_bytes() for p in paths}
        try:
            receipt.write_text(json.dumps(release, indent=2) + '\n')
            engine_file.write_text('ARCHIVE_ENGINE_IMAGE=' + release['engine_digest'] + '\n')
            for command in consumer['checks']:
                run(command, cwd=root, check=True)
        except BaseException:
            for path, data in original.items():
                path.write_bytes(data)
            raise
        return True
    receipt, package_file, lock_file = paths[:3]
    engine_file = paths[3] if len(paths) == 4 else None
    current = json.loads(receipt.read_text())
    validate(release, consumer, current, artifact)
    package = json.loads(package_file.read_text())
    if package.get('dependencies', {}).get(release['package']) != current['version']:
        raise ValueError('Package dependency differs from the current release receipt')
    if engine_file is not None and engine_file.read_text() != 'ARCHIVE_ENGINE_IMAGE=' + current['engine_digest'] + '\n':
        raise ValueError('Engine pin differs from the current release receipt')
    if current == release:
        return False
    original = {p: p.read_bytes() for p in paths}
    try:
        package['dependencies'][release['package']] = release['version']
        package_file.write_text(json.dumps(package, indent=2) + '\n')
        receipt.write_text(json.dumps(release, indent=2) + '\n')
        if engine_file is not None:
            engine_file.write_text('ARCHIVE_ENGINE_IMAGE=' + release['engine_digest'] + '\n')
        # Caller supplies ephemeral registry authentication through npm config.
        run(['pnpm', 'install', '--lockfile-only', '--ignore-scripts', '--no-frozen-lockfile'], cwd=package_file.parent, check=True)
        for command in consumer['checks']:
            run(command, cwd=root, check=True)
    except BaseException:
        for path, data in original.items():
            path.write_bytes(data)
        raise
    return True


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--registry', type=Path, required=True)
    parser.add_argument('--consumer', required=True)
    parser.add_argument('--checkout', type=Path, required=True)
    parser.add_argument('--release', type=Path, required=True)
    parser.add_argument('--package-artifact', type=Path)
    args = parser.parse_args()
    registry = json.loads(args.registry.read_text())
    release = json.loads(args.release.read_text())
    consumer = select_consumer(registry, args.consumer, release)
    changed = prepare(args.checkout, consumer, release, args.package_artifact)
    print('Prepared reviewed upgrade' if changed else 'Release already applied; no changes')


if __name__ == '__main__':
    main()
