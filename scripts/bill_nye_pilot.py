#!/usr/bin/env python3
"""Acquire only the owner's approved two-episode Bill Nye preservation sample."""
import hashlib
import json
import subprocess
from datetime import datetime, timezone
from pilot import ROOT, REPO, cli, environment

ITEM = 'billnyethescienceguythecompleteseries-07-28-2021'
COLLECTION = 'bill-nye'
FILES = ['01 Flight.mp4', "02 Earth's Crust.mp4"]

def write(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n'); path.chmod(0o600)

def main():
    source = ROOT / 'evidence/bill-nye-source.json'
    raw = source.read_bytes(); payload = json.loads(raw)
    if payload.get('metadata', {}).get('identifier') != ITEM: raise RuntimeError('Wrong source metadata')
    selected = {file['name']: file for file in payload['files'] if file['name'] in FILES}
    if set(selected) != set(FILES) or sum(int(file['size']) for file in selected.values()) != 274780857:
        raise RuntimeError('Approved sample size or selection changed; operator review needed')
    write(ROOT / 'evidence/bill-nye-owner-selection.json', {'selected_at': datetime.now(timezone.utc).isoformat(), 'source_item': ITEM, 'files': FILES, 'bytes': 274780857, 'budget_bytes': 512 << 20, 'reserve_bytes': 1 << 30, 'destination': str(ROOT / 'media/bill-nye'), 'independent_backup': 'nuc:/home/onnwee/Backups/vhness-private-pilot', 'authorization': 'Owner selected this source and accepted the exact two-file pilot, destination, budget, reserve, and independent backup in the T3 thread. Democracy Now! remains source streaming only.'})
    cli(['collection', '--collection', COLLECTION, '--policy', 'local_preserve', '--budget', str(512 << 20), '--reserve', str(1 << 30), '--source-item', ITEM])
    for name in FILES:
        result = subprocess.check_output([str(ROOT / 'bin/vhness'), 'import-item', '--id', ITEM, '--collection', COLLECTION, '--file', str(source), '--episode-file', name], env=environment(), cwd=REPO / 'backend', text=True)
        identity = json.loads(result)['episode_id']
        decision = ROOT / f'evidence/{identity}.review.json'
        write(decision, {'acquisition': True, 'published': True, 'playback': True, 'channel': True, 'public': False, 'file': name, 'notes': 'Owner approved this exact Bill Nye source-provided MP4 derivative for private qualification. Preserve bytes unchanged, retain AVI relationship in full metadata. Two-file limit, 512 MiB budget, 1 GiB disk reserve; independent NUC recovery. Episode broadcast date is unknown; series debut date stays in source metadata.'})
        cli(['review', '--id', identity, '--file', str(decision)])
        manifest = ROOT / f'evidence/{identity}.manifest.json'
        write(manifest, {'collection': COLLECTION, 'id': identity, 'snapshot_sha256': hashlib.sha256(raw).hexdigest(), 'file': name, 'bytes': int(selected[name]['size']), 'sha256': '', 'notes': 'Exact owner-selected source snapshot pins supplied MD5 and SHA1; compute and record SHA256 after verified transfer. Source derivative retained unchanged.'})
        cli(['acquire', '--file', str(manifest), '--timeout', '30m'])
    cli(['audit', '--collection', COLLECTION])
    print('Approved Bill Nye originals verified; no other source video acquired')

if __name__ == '__main__': main()
