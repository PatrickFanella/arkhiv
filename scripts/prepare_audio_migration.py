"""Prepare private audio migration inputs from an independently restored pilot."""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import re


def digest(path):
    value = hashlib.sha256()
    with path.open('rb') as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b''):
            value.update(chunk)
    return value.hexdigest()


def encode(value):
    # Match Go's struct JSON representation for the ordered original manifest.
    return json.dumps(value, ensure_ascii=False, separators=(',', ':')).replace('&', '\\u0026').replace('<', '\\u003c').replace('>', '\\u003e').replace('\u2028', '\\u2028').replace('\u2029', '\\u2029').encode()


def prepare(backup, export, receipt, output, collection, retain_public=False):
    backup, output = Path(backup), Path(output)
    manifest = json.loads((backup / 'manifest.json').read_text())
    recovered = json.loads(Path(receipt).read_text())
    data = json.loads(Path(export).read_text())
    if not recovered.get('independent_host') or not recovered.get('restored_on') or recovered['independent_host'] == recovered['restored_on']:
        raise ValueError('Independent backup and restore evidence required')
    if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._-]{0,199}', collection):
        raise ValueError('Invalid collection identity')
    if set(data) != set(recovered['counts']):
        raise ValueError('Complete restored table export required')
    if recovered.get('files_verified') != len(manifest) or recovered.get('dump_sha256') != manifest.get('catalog.dump'):
        raise ValueError('Restore and backup manifests differ')
    for name, expected in manifest.items():
        file = backup / name
        if file.is_symlink() or not file.resolve().is_relative_to(backup.resolve()) or digest(file) != expected:
            raise ValueError('Retained backup file differs: ' + name)
    for table, rows in data.items():
        if recovered['counts'].get(table) != len(rows):
            raise ValueError('Restored table count differs: ' + table)
    recovery = digest(Path(receipt))
    snapshots = {row['sha256']: row for row in data['source_snapshots']}
    sources, inputs, mappings = [], {}, []
    approved = 0
    public_count = 0
    public_approvals = {row['recording_id']: row for row in data.get('public_approvals', [])}
    scheduled = set()
    for row in data['recordings']:
        identity = row['id']
        if not identity or '/' in identity or '\\' in identity or '..' in identity:
            raise ValueError('Unsafe recording identity')
        snapshot = snapshots[row['source_hash']]
        if snapshot['item_id'] != identity:
            raise ValueError('Snapshot recording identity differs')
        retained = backup / 'data/sources' / (identity + '.json')
        retained_bytes = retained.read_bytes()
        retained_hash = hashlib.sha256(retained_bytes).hexdigest()
        equal = json.loads(retained_bytes) == snapshot['raw']
        raw = retained_bytes if equal else encode(snapshot['raw'])
        source_hash = hashlib.sha256(raw).hexdigest()
        for value, source in [(retained_bytes, 'retained-file'), (raw, 'restored-database-json')]:
            sha = hashlib.sha256(value).hexdigest()
            name = 'source-' + sha + '.json'
            if name not in inputs:
                inputs[name] = value
                sources.append({'item_id': identity, 'file': name, 'sha256': sha, 'origin': source})
        mappings.append({'id': identity, 'legacy_snapshot_sha256': row['source_hash'], 'retained_file_sha256': retained_hash,
                         'catalog_snapshot_sha256': source_hash, 'retained_file_matches_database': equal})
        recording = copy.deepcopy(row['playback'] or row['source'])
        # Match the donor's effective catalog view without rewriting source rows.
        for key in ['kind', 'title', 'concert_id']:
            if row['review'].get(key):
                recording[key] = row['review'][key]
        if row['review'].get('sets'):
            recording['sets'] = copy.deepcopy(row['review']['sets'])
        if recording['id'] != identity or not recording['tracks']:
            raise ValueError('Recording identity or tracks missing')
        tracks = []
        for index, track in enumerate(recording['tracks']):
            section = next((s for s in recording['sets'] if s['reviewed'] and s['first'] <= index <= s['last']), None)
            mapped = {'id': 'track-' + str(index), 'file': track['file']['name'], 'title': track['title'], 'set_reviewed': bool(section)}
            if section:
                mapped['set'] = section['title']
            tracks.append(mapped)
        representation = {'id': identity, 'media': 'audio', 'source_item_id': identity, 'snapshot_sha256': source_hash,
                          'file': tracks[0]['file'], 'collections': [collection], 'extension': {'audio': {
                              'recording_id': identity, 'concert_id': recording.get('concert_id', ''), 'tracks': tracks,
                              'recording': recording, 'credits': {key: recording.get(key, '') for key in ['source', 'taper', 'transferer']}}}}
        item = {'id': identity, 'title': recording['title'], 'creator': recording['artist'], 'date': recording['date'],
                'description': recording.get('source', ''), 'collections': [collection], 'representations': [representation],
                'editorial': {'legacy_record': row, 'legacy_snapshot_mapping': mappings[-1], 'migration_scope': 'retained current decisions; public approval imported separately'}}
        inputs['item-' + identity + '.json'] = encode(item)
        review = row['review']
        if not (review.get('published') and review.get('playback') and row['playback'] and row['media']):
            continue
        acquisition = json.loads((backup / 'data' / (identity + '.acquisition.json')).read_text())
        originals = []
        files = {file['source_name']: file for file in acquisition['files']}
        for track in recording['tracks']:
            original = files[track['file']['name']]
            name = Path(original['path']).name
            path = backup / 'data/originals' / name
            if digest(path) != original['sha256'] or path.stat().st_size != original['bytes'] or original['bytes'] != track['file']['bytes']:
                raise ValueError('Original recording input differs')
            originals.append({'file': original['source_name'], 'bytes': original['bytes'], 'sha256': original['sha256'], 'path': name})
        derived = backup / 'data/media' / row['media']
        if derived.name != row['media'] or digest(derived) != row['media_sha256']:
            raise ValueError('Continuous recording differs')
        public = False
        prior_public = public_approvals.get(identity, {})
        if retain_public and prior_public.get('approved'):
            if (prior_public.get('source_hash') != row['source_hash']
                    or prior_public.get('media_sha256') != row['media_sha256']
                    or review.get('kind') != 'audience' or row['source'].get('synthetic')
                    or not prior_public.get('policy_basis') or not prior_public.get('reviewer')):
                raise ValueError('Retained public approval does not match current reviewed audience source/audio')
            public = True
            public_count += 1
        relay = bool(review.get('relay') or public)
        if relay:
            scheduled.add(identity)
        access = {'snapshot_sha256': source_hash, 'files': [track['file'] for track in tracks], 'read': False, 'playback': True,
                  'download': False, 'acquisition': False, 'relay': relay, 'schedule': relay, 'public': public,
                  'notes': review['notes'] + (' Retained public approval: ' + prior_public['policy_basis'] + '; reviewer: ' + prior_public['reviewer'] if public else ' Retained private playback and existing relay decision.')}
        preservation = {'snapshot_sha256': source_hash, 'manifest': originals, 'independent_recovery_sha256': recovery}
        inputs['review-' + identity + '.json'] = encode({'review': access, 'preservation': preservation})
        asset = {'snapshot_sha256': source_hash, 'manifest_sha256': hashlib.sha256(encode(originals)).hexdigest(),
                 'path': row['media'], 'bytes': derived.stat().st_size, 'sha256': row['media_sha256'],
                 'duration': recording['duration'], 'independent_recovery_sha256': recovery,
                 'tool': 'Retained Dedhed acquisition receipt: ' + acquisition['ffmpeg_version']}
        inputs['audio-' + identity + '.json'] = encode(asset)
        approved += 1
    plan = {'collection': collection, 'recordings': len(data['recordings']), 'private_playback_recordings': approved,
            'sources': sources, 'snapshot_mappings': mappings, 'legacy_schedule': data['station_schedule'],
            'independent_recovery_sha256': recovery, 'new_acquisition': False, 'public': public_count > 0, 'public_playback_recordings': public_count, 'schedule_eligible': sorted(scheduled)}
    inputs['migration-plan.json'] = encode(plan)
    retained_export = {'collection': collection, 'module': 'audio', 'independent_recovery_sha256': recovery,
                       'tables': data, 'counts': {table: len(rows) for table, rows in data.items()}}
    inputs['retained-export.json'] = encode(retained_export)
    output.mkdir(mode=0o700, exist_ok=False)
    for name, value in inputs.items():
        (output / name).write_bytes(value)
        (output / name).chmod(0o400)
    return plan


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['backup', 'export', 'receipt', 'output', 'collection']:
        parser.add_argument('--' + name, required=True)
    parser.add_argument('--retain-public-approvals', action='store_true', help='Preserve exact current hash-bound audience approvals from the independently restored donor')
    args = parser.parse_args()
    plan = prepare(args.backup, args.export, args.receipt, args.output, args.collection, args.retain_public_approvals)
    print(json.dumps({key: plan[key] for key in ['recordings', 'private_playback_recordings', 'new_acquisition', 'public']}))


if __name__ == '__main__':
    main()
