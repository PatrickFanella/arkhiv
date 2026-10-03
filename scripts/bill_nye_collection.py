#!/usr/bin/env python3
"""Preserve the owner-authorized complete series as source-provided MP4s."""
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import shlex
import shutil
import subprocess
import time
from datetime import datetime, timezone
from pilot import ROOT, REPO, environment

ITEM = 'billnyethescienceguythecompleteseries-07-28-2021'
BUDGET = 16 << 30
RESERVE = 1 << 30
EXPECTED_BYTES = 13763456996

def write(path, value):
    temporary = path.with_suffix(path.suffix + '.tmp')
    temporary.write_text(json.dumps(value, indent=2) + '\n')
    temporary.chmod(0o600)
    temporary.replace(path)

def main():
    folder = ROOT / 'evidence/bill-nye-collection'
    folder.mkdir(exist_ok=True, mode=0o700)
    with (folder / '.lock').open('w') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        source = ROOT / 'evidence/bill-nye-source.json'
        raw = source.read_bytes(); payload = json.loads(raw)
        files = sorted((f for f in payload['files'] if f['name'].lower().endswith('.mp4')), key=lambda f: f['name'])
        numbers = {int(re.match(r'^(\d+) ', f['name']).group(1)) for f in files}
        if payload['metadata']['identifier'] != ITEM or len(files) != 100 or numbers != set(range(1, 101)):
            raise RuntimeError('Complete-series source identity or episode selection changed')
        total = sum(int(f['size']) for f in files)
        if total != EXPECTED_BYTES or total >= BUDGET or any(f.get('private') for f in files):
            raise RuntimeError('Reviewed source sizes or eligibility changed')
        remote = 'import shutil,json; print(json.dumps({"free":shutil.disk_usage("/home/onnwee/Backups/vhness-private-pilot").free}))'
        backup_free = json.loads(subprocess.check_output(['ssh', 'nuc', 'python3 -c ' + shlex.quote(remote)], text=True))['free']
        local_free = shutil.disk_usage(ROOT / 'media').free
        if local_free < total + RESERVE or backup_free < 3 * total + RESERVE:
            raise RuntimeError('Insufficient primary or independent backup/restore capacity')
        plan = {'source_item': ITEM, 'snapshot_sha256': hashlib.sha256(raw).hexdigest(),
                'files': [{'name': f['name'], 'bytes': int(f['size']), 'md5': f.get('md5'), 'sha1': f.get('sha1')} for f in files],
                'bytes': total, 'budget_bytes': BUDGET, 'reserve_bytes': RESERVE,
                'primary_free_at_start': local_free, 'backup_free_at_start': backup_free,
                'destination': str(ROOT / 'media/bill-nye'), 'backup': 'nuc:/home/onnwee/Backups/vhness-private-pilot',
                'authorization': 'Owner requested all three deployable products and authorized pulling the entire Bill Nye episode collection if space permits. Use all 100 source-provided MP4s without duplicate AVI copies. Democracy Now! remains streaming-only.',
                'selected_at': datetime.now(timezone.utc).isoformat()}
        write(folder / 'selection.json', plan)
        binary = ROOT / 'bin/vhness-full-preservation-reviewed'
        if not binary.exists(): raise RuntimeError('Build the verified budget-review engine before acquisition')
        def cli(args):
            return subprocess.check_output([str(binary), *args], env=environment(), cwd=REPO / 'backend', text=True)
        cli(['collection', '--collection', 'bill-nye', '--policy', 'local_preserve', '--budget', str(BUDGET), '--reserve', str(RESERVE), '--source-item', ITEM])
        completed = []
        for file in files:
            name = file['name']
            identity = json.loads(cli(['import-item', '--id', ITEM, '--collection', 'bill-nye', '--file', str(source), '--episode-file', name]))['episode_id']
            review = folder / f'{identity}.review.json'
            write(review, {'acquisition': True, 'published': True, 'playback': True, 'channel': True, 'public': False, 'file': name,
                           'notes': 'Owner authorized complete-series private preservation after capacity review. Unchanged source MP4; 16 GiB budget and 1 GiB reserve, independent NUC backup. No AVI duplicate. Episode date unknown. Public rollout is separate.'})
            cli(['review', '--id', identity, '--file', str(review)])
            manifest = folder / f'{identity}.manifest.json'
            write(manifest, {'collection': 'bill-nye', 'id': identity, 'snapshot_sha256': plan['snapshot_sha256'], 'file': name, 'bytes': int(file['size']), 'sha256': '', 'notes': 'Owner-authorized complete-series manifest; supplied MD5/SHA1 verified and local SHA256 computed.'})
            for attempt in range(3):
                try:
                    receipt = json.loads(cli(['acquire', '--file', str(manifest), '--timeout', '30m']))
                    break
                except subprocess.CalledProcessError:
                    if attempt == 2: raise
                    time.sleep(60 * (attempt + 1))
            completed.append({'episode_id': identity, 'file': name, 'bytes': int(file['size']), 'sha256': receipt['sha256']})
            write(folder / 'progress.json', {'completed': completed, 'count': len(completed), 'total': len(files), 'at': datetime.now(timezone.utc).isoformat()})
            print(f'Verified {len(completed)}/100: {name}', flush=True)
        audit = json.loads(cli(['audit', '--collection', 'bill-nye']))
        if audit['originals'] != 100 or audit['failures'] != 0: raise RuntimeError('Complete series integrity audit failed')
        write(folder / 'complete.json', {'selection': plan, 'originals': completed, 'audit': audit, 'completed_at': datetime.now(timezone.utc).isoformat(), 'independent_backup': 'pending; completion does not itself prove backup or restore'})

if __name__ == '__main__': main()
