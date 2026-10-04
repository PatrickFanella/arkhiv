#!/usr/bin/env python3
"""Snapshot the owned private pilot and prove restore on a separate host."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shlex
import subprocess
import time
from datetime import datetime, timezone

from pilot import ROOT, PROJECT, REPO

TABLES = ['collections', 'snapshots', 'observations', 'episodes', 'decision_history', 'schedules', 'current_schedule', 'worker_state', 'metadata_queue', 'schema_versions', 'preservation_history', 'collection_budget_reviews', 'catalog_items', 'catalog_memberships', 'catalog_representations', 'catalog_representation_memberships', 'catalog_review_history', 'representation_transport_reviews', 'representation_transport_history', 'print_indexes', 'print_page_text', 'audio_assets', 'audio_asset_history', 'audio_schedules']

def shell_arguments(args):
    if args[0] == 'ssh' and len(args) > 3:
        return args[:2] + [shlex.join(args[2:])]
    return args

def run(args, **kwargs):
    return subprocess.run(shell_arguments(args), check=True, **kwargs)

def capture(args, **kwargs):
    return subprocess.check_output(shell_arguments(args), **kwargs)

def checksum(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()

def fingerprint(prefix):
    result = {}
    for table in TABLES:
        query = f"SELECT count(*),md5(coalesce(string_agg(md5(row_to_json(t)::text),'' ORDER BY md5(row_to_json(t)::text)),'')) FROM {table} t"
        value = capture([*prefix, 'psql', '-U', 'postgres', '-d', 'postgres', '-At', '-c', query], text=True).strip()
        count, digest = value.split('|'); result[table] = {'rows': int(count), 'digest': digest}
    query = "SELECT count(*) FROM snapshots WHERE encode(sha256(raw),'hex')<>sha256"
    invalid = int(capture([*prefix, 'psql', '-U', 'postgres', '-d', 'postgres', '-At', '-c', query], text=True).strip())
    if invalid: raise RuntimeError('Restored raw snapshots failed byte checksum')
    return {'tables': result, 'invalid_raw_snapshots': invalid}

def restore(dump, name, host=None, remote_dump=None):
    prefix = ['ssh', host, 'docker'] if host else ['docker']
    run([*prefix, 'run', '-d', '--name', name, '--network', 'none', '--memory', '512m', '--cpus', '1', '-e', 'POSTGRES_HOST_AUTH_METHOD=trust', 'postgres:17'], stdout=subprocess.DEVNULL)
    try:
        for _ in range(60):
            # The image starts a temporary Unix-socket server during initdb.
            # TCP readiness identifies the final server after that restart.
            ready = subprocess.run(shell_arguments([*prefix, 'exec', name, 'pg_isready', '-h', '127.0.0.1', '-U', 'postgres']), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            if ready.returncode == 0: break
            time.sleep(.5)
        else: raise RuntimeError('Disposable restore database readiness failed')
        if host:
            run(['ssh', host, f'docker exec -i {name} pg_restore -U postgres -d postgres --exit-on-error --no-owner --no-acl < {remote_dump}'])
        else:
            with dump.open('rb') as stream:
                run([*prefix, 'exec', '-i', name, 'pg_restore', '-U', 'postgres', '-d', 'postgres', '--exit-on-error', '--no-owner', '--no-acl'], stdin=stream)
        return fingerprint([*prefix, 'exec', name])
    finally:
        run([*prefix, 'rm', '-f', '-v', name], stdout=subprocess.DEVNULL)

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--host', required=True, help='independent SSH backup host')
    parser.add_argument('--destination', required=True, help='absolute independent backup directory')
    parser.add_argument('--media', help='optional exact preservation root to include')
    parser.add_argument('--media-budget', type=int, default=10 << 20, help='explicit preservation backup bound in bytes')
    args = parser.parse_args()
    if not re.fullmatch(r'[a-zA-Z0-9_.-]+', args.host) or not re.fullmatch(r'/[a-zA-Z0-9_./-]+', args.destination) or '/..' in args.destination:
        raise RuntimeError('Use a simple SSH host and absolute destination')
    stamp = datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ')
    directory = ROOT / 'backups' / stamp; directory.mkdir(parents=True, mode=0o700)
    dump = directory / 'metadata.dump'
    with dump.open('wb') as output:
        run(['docker', 'compose', '-p', PROJECT, '-f', str(ROOT / 'compose.json'), 'exec', '-T', 'db', 'pg_dump', '-U', 'vhness', '-d', 'vhness', '-Fc'], stdout=output)
    dump.chmod(0o600)
    if dump.stat().st_size > 1 << 30: raise RuntimeError('Metadata backup exceeded 1 GiB operator bound; preserved for review')
    digest = checksum(dump)
    baseline = restore(dump, 'vhness-restore-local-' + stamp.lower())
    remote = args.destination.rstrip('/') + '/' + stamp
    run(['ssh', args.host, 'mkdir', '-p', '-m', '700', remote])
    run(['scp', '-q', str(dump), args.host + ':' + remote + '/metadata.dump'])
    actual = capture(['ssh', args.host, 'sha256sum', remote + '/metadata.dump'], text=True).split()[0]
    if actual != digest: raise RuntimeError('Independent dump checksum mismatch')
    recovered = restore(dump, 'vhness-restore-remote-' + stamp.lower(), args.host, remote + '/metadata.dump')
    if baseline != recovered: raise RuntimeError('Independent restore content mismatch')
    media_result = None
    if args.media:
        media = Path(args.media).resolve()
        # Only immutable originals and receipts belong to this bounded sample backup.
        originals = list(media.rglob('originals/*.mp4'))
        if args.media_budget <= 0 or args.media_budget > 32 << 30 or not originals or any(p.is_symlink() for p in originals) or sum(p.stat().st_size for p in originals) > args.media_budget:
            raise RuntimeError('Preservation exceeds the explicit media backup bound')
        archive = directory / 'preservation.tar'
        run(['tar', '-cf', str(archive), '-C', str(media), *[str(p.relative_to(media)) for p in originals], *[str(p.relative_to(media)) for p in media.rglob('*.receipt.json')]])
        archive.chmod(0o600)
        run(['scp', '-q', str(archive), args.host + ':' + remote + '/preservation.tar'])
        archive_hash = checksum(archive)
        remote_hash = capture(['ssh', args.host, 'sha256sum', remote + '/preservation.tar'], text=True).split()[0]
        if archive_hash != remote_hash: raise RuntimeError('Independent preservation archive mismatch')
        run(['ssh', args.host, 'mkdir', '-m', '700', remote + '/restored-originals'])
        run(['ssh', args.host, 'tar', '-xf', remote + '/preservation.tar', '-C', remote + '/restored-originals'])
        files = {}
        for original in originals:
            expected = checksum(original)
            actual = capture(['ssh', args.host, 'sha256sum', remote + '/restored-originals/' + str(original.relative_to(media))], text=True).split()[0]
            if actual != expected: raise RuntimeError('Independent original checksum mismatch')
            files[str(original.relative_to(media))] = expected
        media_result = {'archive_sha256': archive_hash, 'restored_originals': files}
    receipt = {'created_at': stamp, 'source_host': capture(['hostname'], text=True).strip(), 'backup_host': capture(['ssh', args.host, 'hostname'], text=True).strip(), 'destination': remote, 'dump_sha256': digest, 'dump_bytes': dump.stat().st_size, 'local_restore': baseline, 'independent_restore': recovered, 'preservation': media_result, 'proof_boundary': 'database/raw-byte and optional original recovery; no browser or AV qualification'}
    path = directory / 'receipt.json'; path.write_text(json.dumps(receipt, indent=2) + '\n'); path.chmod(0o600)
    run(['scp', '-q', str(path), args.host + ':' + remote + '/receipt.json'])
    print(json.dumps(receipt, indent=2))

if __name__ == '__main__': main()
