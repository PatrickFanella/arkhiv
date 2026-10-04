#!/usr/bin/env python3
"""Prepare private catalog inputs from an independently restored print archive.

Never changes an archive or database. The retained legacy dump owns original
history; mapped items retain document/variant IDs and complete editorial rows.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re


def digest(data):
    return hashlib.sha256(data).hexdigest()


def sha(value):
    if not isinstance(value, str) or not re.fullmatch(r'[0-9a-f]{64}', value):
        raise ValueError('Invalid content identity')
    return value


def filename(value):
    if not value or Path(value).name != value or '\\' in value or '\x00' in value:
        raise ValueError('Invalid source filename')
    return value


def prepare(backup, catalog_file, restore_file, output):
    receipt = json.loads((backup / 'complete.json').read_text())
    restore = json.loads(restore_file.read_text())
    catalog = json.loads(catalog_file.read_text())
    archive = backup / 'archive'
    dump = (backup / 'catalog.dump').read_bytes()
    if (receipt.get('complete') is not True or restore.get('restore_exit_code') != 0
            or restore.get('dump_sha256') != receipt['catalog_sha256']
            or digest(dump) != receipt['catalog_sha256'] or len(dump) != receipt['catalog_size']):
        raise ValueError('Independent dump/restore evidence does not match')
    for name, expected in receipt['files'].items():
        relative = Path(name)
        if relative.is_absolute() or '..' in relative.parts:
            raise ValueError('Invalid recovery path')
        file = archive / relative
        if file.is_symlink() or digest(file.read_bytes()) != expected:
            raise ValueError('Archive differs from the independent backup: ' + name)
    for table, rows in catalog.items():
        if not isinstance(rows, list) or restore['table_counts'].get(table) != len(rows):
            raise ValueError('Restored table coverage differs: ' + table)
    assets = {sha(row['sha256']): row for row in catalog['assets']}
    snapshots = {}
    for row in catalog['source_snapshots']:
        key = sha(row['id'])
        raw = (archive / 'snapshots' / (key + '.json')).read_bytes()
        # Legacy snapshot IDs can identify a prior JSON serialization. Bind the
        # shared source to the exact independently retained file bytes, while
        # preserving the old identity and checking complete semantic equality.
        if json.loads(raw) != row['raw']:
            raise ValueError('Original source response differs from restored metadata')
        snapshots[key] = (row['item_id'], raw, digest(raw))
    collections = {row['id']: row['config'] for row in catalog['collections']}
    planned = []
    reviewed = []
    used_variants = set()
    recovery = digest(restore_file.read_bytes())

    def original(name, key):
        filename(name)
        key = sha(key)
        relative = 'originals/' + key[:2] + '/' + key
        file = archive / relative
        if key not in assets or file.is_symlink() or file.stat().st_size != assets[key]['size'] or digest(file.read_bytes()) != key:
            raise ValueError('Unverified preserved original: ' + name)
        return {'file': name, 'bytes': assets[key]['size'], 'sha256': key, 'path': relative}

    for document in catalog['documents']:
        sha(document['id'])
        collection = document['collection_id']
        if collection not in collections:
            raise ValueError('Document has no retained collection configuration')
        metadata = {**document['metadata'], **document['overrides']}
        legacy_variants = [v for v in catalog['variants'] if v['document_id'] == document['id']]
        item = {'id': document['id'], 'title': metadata['title'], 'description': document['rights'],
                'date': metadata.get('date', ''), 'creator': metadata.get('creator', ''),
                'collections': [collection], 'editorial': {'legacy_document': document, 'legacy_variants': legacy_variants}, 'representations': []}
        for variant in catalog['variants']:
            if variant['document_id'] != document['id']:
                continue
            if variant['id'] in used_variants or not variant['filename'].lower().endswith('.pdf'):
                raise ValueError('Duplicate or unsupported legacy variant')
            used_variants.add(variant['id'])
            sha(variant['id'])
            source, raw, bound_snapshot = snapshots[variant['source_snapshot']]
            if source != document['source_item']:
                raise ValueError('Variant source differs from its document')
            manifest = [original(variant['filename'], variant['asset_sha'])]
            companions = []
            for companion in catalog['asset_companions']:
                if companion['asset_sha'] != variant['asset_sha']:
                    continue
                if companion['source_snapshot'] != variant['source_snapshot']:
                    raise ValueError('OCR companion source differs from its edition')
                companions.append({'file': filename(companion['filename']), 'role': 'ocr'})
                manifest.append(original(companion['filename'], companion['companion_sha']))
            inventory = {entry['name'] for entry in json.loads(raw)['files']}
            if any(entry['file'] not in inventory for entry in manifest):
                raise ValueError('Preserved original absent from complete source response')
            item['representations'].append({'id': variant['id'], 'media': 'print', 'source_item_id': source,
                'snapshot_sha256': bound_snapshot, 'file': variant['filename'], 'collections': [collection],
                'extension': {'print': {'document_id': document['id'], 'variant': variant['kind'], 'pages': 0, 'companions': companions}}})
            if document['published']:
                reviewed.append({'item_id': document['id'], 'representation_id': variant['id'], 'collection': collection,
                    'input': {'review': {'snapshot_sha256': bound_snapshot, 'files': [entry['file'] for entry in manifest],
                        'read': True, 'download': True, 'playback': False, 'acquisition': False, 'relay': False,
                        'schedule': False, 'public': False, 'notes': 'Retained private pilot reading/download decision. ' + document['review_note']},
                        'preservation': {'snapshot_sha256': bound_snapshot, 'manifest': manifest,
                                         'independent_recovery_sha256': recovery}}})
        if not item['representations']:
            raise ValueError('Document has no mapped representation')
        planned.append(item)
    if len(used_variants) != len(catalog['variants']):
        raise ValueError('Unmapped legacy variants')
    # Only after all validation succeeds, allocate a new output directory.
    output.mkdir(mode=0o700, parents=False, exist_ok=False)
    def save(name, value):
        target = output / name
        target.write_text(json.dumps(value, indent=2) + '\n')
        target.chmod(0o600)
    for _, raw, bound_snapshot in snapshots.values():
        target = output / ('source-' + bound_snapshot + '.json')
        target.write_bytes(raw)
        target.chmod(0o600)
    for item in planned:
        save('item-' + sha(item['id']) + '.json', item)
    for review in reviewed:
        save('review-' + sha(review['representation_id']) + '.json', review['input'])
    plan = {'legacy_dump_sha256': receipt['catalog_sha256'], 'legacy_export_sha256': digest(catalog_file.read_bytes()),
            'independent_restore_sha256': recovery, 'documents': len(planned), 'variants': len(used_variants),
            'collections': collections, 'sources': [{'item_id': item, 'legacy_snapshot_id': key, 'sha256': bound_snapshot}
                for key, (item, _, bound_snapshot) in snapshots.items()],
            'reviews': [{k: v for k, v in review.items() if k != 'input'} for review in reviewed],
            'history_owner': 'Retained original dump and independently restored legacy database',
            'scope': 'Private existing pilot only; no acquisition, relay or public approval'}
    save('migration-plan.json', plan)
    return plan


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--backup', required=True, type=Path)
    parser.add_argument('--restored-catalog', required=True, type=Path)
    parser.add_argument('--restore-receipt', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    plan = prepare(args.backup, args.restored_catalog, args.restore_receipt, args.output)
    print('Private migration inputs:', plan['documents'], 'documents,', plan['variants'], 'variants')
