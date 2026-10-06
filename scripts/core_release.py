"""Independent core provenance and API-only upgrade compatibility."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess


def validate_core(release, consumer, current):
    if consumer.get('mode') != 'api' or consumer.get('adoption') != 'core':
        raise ValueError('Expected an API-only core consumer')
    if any(consumer.get(key) for key in ('package_file', 'lock_file')):
        raise ValueError('API-only consumers cannot require frontend package files')
    if consumer['rollout'] != 'reviewed-pr':
        raise ValueError('Unsupported rollout policy')
    if not consumer.get('checks'):
        raise ValueError('Core consumers require independent verification checks')
    for key in ('contract_version', 'configuration_version', 'engine_repository', 'schema_strategy'):
        if release[key] != current[key]:
            raise ValueError('Incompatible ' + key)
    for value in (current, release):
        if value['artifact_kind'] != 'core':
            raise ValueError('Expected an independent core artifact')
        if not re.fullmatch(r'\d+\.\d+\.\d+', value['version']):
            raise ValueError('Expected a stable core version')
        if not re.fullmatch(r'[0-9a-f]{40}', value['source_revision']):
            raise ValueError('Expected a full core source revision')
        if not re.fullmatch(re.escape(value['engine_repository']) + r'@sha256:[0-9a-f]{64}', value['engine_digest']):
            raise ValueError('Core image must use an immutable registry digest')
    if not set(current['api_versions']).issubset(release['api_versions']):
        raise ValueError('Required API version is unsupported')
    if not set(consumer['media']).issubset(release['media']):
        raise ValueError('Enabled media is unsupported')
    for module, version in consumer.get('modules', current['modules']).items():
        if release['modules'].get(module) != version:
            raise ValueError('Required module version is unsupported: ' + module)
    old = tuple(map(int, current['version'].split('.')))
    new = tuple(map(int, release['version'].split('.')))
    if new[0] != old[0] or new < old:
        raise ValueError('Major changes and downgrades require a coordinated migration')
    if new == old and release != current:
        raise ValueError('Core version was reused for different metadata')
    for name, digest in current['migrations'].items():
        if release['migrations'].get(name) != digest:
            raise ValueError('Previously applied migration changed: ' + name)


def manifest(root, digest):
    if subprocess.check_output(['git', 'status', '--porcelain'], cwd=root):
        raise ValueError('Core release requires a clean committed source revision')
    contract = json.loads((root / 'docs/reference/core-release-contract.json').read_text())
    if not re.fullmatch(re.escape(contract['engine_repository']) + r'@sha256:[0-9a-f]{64}', digest):
        raise ValueError('Supply the immutable registry core digest')
    version = json.loads((root / 'docs/reference/core-version.json').read_text())['version']
    if not re.fullmatch(r'\d+\.\d+\.\d+', version):
        raise ValueError('Expected a stable core version')
    return {**contract, 'version': version, 'engine_digest': digest,
            'source_revision': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip(),
            'optional_migrations': {str(p.relative_to(root / 'migrations')): hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted((root / 'migrations/community').glob('*.sql'))},
            'migrations': {p.name: hashlib.sha256(p.read_bytes()).hexdigest()
                           for p in sorted((root / 'migrations').glob('*.sql'))}}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--engine-digest', required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    result = manifest(Path(__file__).resolve().parents[1], args.engine_digest)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open('x') as output:
        output.write(json.dumps(result, indent=2) + '\n')
    print('Core release manifest:', args.output)


if __name__ == '__main__':
    main()
