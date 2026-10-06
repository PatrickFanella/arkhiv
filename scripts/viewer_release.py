"""Verify an independent viewer receipt against the consumer's retained core."""
import hashlib
import json
from pathlib import Path
import re

from core_release import validate_core
from download_release_package import REGISTRY


def registered_file(root, consumer, key):
    path = root / consumer[key]
    if not path.resolve().is_relative_to(root.resolve()) or not path.is_file():
        raise ValueError('Registered dependency file is missing or outside the checkout')
    return path


def validate_viewer(release, consumer, current, artifact, core):
    if consumer.get('mode') != 'viewer' or consumer.get('adoption') != 'package':
        raise ValueError('Expected an independent viewer consumer')
    if consumer.get('engine_file'):
        raise ValueError('Viewer upgrades cannot register a managed engine pin')
    if consumer.get('rollout') != 'reviewed-pr' or not consumer.get('checks'):
        raise ValueError('Viewer consumers require reviewed rollout and independent checks')
    core_consumer = {'mode': 'api', 'adoption': 'core', 'rollout': 'reviewed-pr',
                     'checks': consumer['checks'], 'media': consumer['media'],
                     'modules': consumer['modules']}
    validate_core(core, core_consumer, core)
    if core['contract_version'] != 1 or core['configuration_version'] != 1 or '1' not in core['api_versions']:
        raise ValueError('Viewer requires the reviewed version-1 API and configuration')
    for value in (current, release):
        if value.get('package') != '@subcult/archive-viewer' or value.get('registry') != REGISTRY:
            raise ValueError('Viewer package registry ownership changed')
        if value.get('license') != 'GPL-3.0-only':
            raise ValueError('Viewer license changed')
        if not re.fullmatch(r'\d+\.\d+\.\d+', value.get('version', '')):
            raise ValueError('Expected a stable viewer version')
        if value.get('package_file') != 'subcult-archive-viewer-' + value['version'] + '.tgz':
            raise ValueError('Unexpected viewer package filename')
        if not re.fullmatch(r'[0-9a-f]{40}', value.get('source_revision', '')):
            raise ValueError('Expected a full viewer source revision')
        if not re.fullmatch(r'[0-9a-f]{64}', value.get('package_sha256', '')):
            raise ValueError('Expected a full viewer package checksum')
        if value.get('compatible_core_receipt') != 'core-release-' + core['version'] + '.json':
            raise ValueError('Viewer requires a different reviewed core version')
    old, new = (tuple(map(int, value['version'].split('.'))) for value in (current, release))
    if new[0] != old[0] or new < old:
        raise ValueError('Major changes and downgrades require a coordinated migration')
    if new == old and release != current:
        raise ValueError('Viewer version was reused for different metadata')
    if artifact is None or hashlib.sha256(artifact.read_bytes()).hexdigest() != release['package_sha256']:
        raise ValueError('Viewer package checksum mismatch')


def retained_core(root, consumer):
    receipt = registered_file(root, consumer, 'core_file')
    engine = registered_file(root, consumer, 'core_engine_file')
    core = json.loads(receipt.read_text())
    if core.get('artifact_kind') != 'core':
        raise ValueError('Expected an independent retained core receipt')
    if engine.read_text() != 'ARCHIVE_ENGINE_IMAGE=' + core['engine_digest'] + '\n':
        raise ValueError('Core pin differs from the retained core receipt')
    return core, [receipt, engine]
