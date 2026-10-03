#!/usr/bin/env python3
"""Own one loopback-only metadata pilot; never acquire Democracy Now! video."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import secrets
import shutil
import signal
import socket
import subprocess
import time
import urllib.request

REPO = Path(__file__).resolve().parents[1]
ROOT = Path(os.environ.get('VHNESS_PILOT_ROOT', str(Path.home() / '.local/state/vhness/private-pilot'))).resolve()
PROJECT = 'vhness-private-pilot'

def command(args, **kwargs):
    return subprocess.run(args, check=True, **kwargs)

def settings():
    return json.loads((ROOT / 'runtime.json').read_text())

def environment():
    return {**os.environ, **json.loads((ROOT / 'environment.json').read_text())}

def cli(args):
    return command([str(ROOT / 'bin/vhness'), *args], env=environment(), cwd=REPO / 'backend')

def compose(*args):
    return command(['docker', 'compose', '-p', PROJECT, '-f', str(ROOT / 'compose.json'), *args])

def live_pid():
    if not (ROOT / 'runtime.json').exists(): return None
    pid = settings().get('pid')
    if not pid or not Path(f'/proc/{pid}/cmdline').exists(): return None
    arguments = Path(f'/proc/{pid}/cmdline').read_bytes().split(b'\0')
    if arguments[:2] != [str(ROOT / 'bin/vhness').encode(), b'serve']:
        raise RuntimeError('Recorded PID belongs to another process; preserve it')
    return pid

def prepare():
    ROOT.mkdir(parents=True, exist_ok=True, mode=0o700)
    ROOT.chmod(0o700)
    for part in ['bin', 'logs', 'evidence']: (ROOT / part).mkdir(exist_ok=True, mode=0o700)
    password = ROOT / 'database-password'
    if not password.exists():
        password.write_text(secrets.token_urlsafe(32)); password.chmod(0o600)
    definition = {'services': {'db': {'image': 'postgres:17', 'environment': {'POSTGRES_USER': 'vhness', 'POSTGRES_DB': 'vhness', 'POSTGRES_PASSWORD_FILE': '/run/secrets/password'}, 'secrets': ['password'], 'ports': ['127.0.0.1::5432'], 'volumes': ['metadata:/var/lib/postgresql/data'], 'healthcheck': {'test': ['CMD-SHELL', 'pg_isready -U vhness -d vhness'], 'interval': '2s', 'timeout': '3s', 'retries': 20}, 'mem_limit': '512m', 'cpus': 1}}, 'secrets': {'password': {'file': str(password)}}, 'volumes': {'metadata': {}}}
    (ROOT / 'compose.json').write_text(json.dumps(definition, indent=2) + '\n')
    compose('up', '-d', '--wait')
    result = subprocess.check_output(['docker', 'compose', '-p', PROJECT, '-f', str(ROOT / 'compose.json'), 'port', 'db', '5432'], text=True).strip()
    host, port = result.rsplit(':', 1)
    if host != '127.0.0.1': raise RuntimeError('Pilot DB is not loopback-only')
    with socket.socket() as sock:
        sock.bind(('127.0.0.1', 0)); api_port = sock.getsockname()[1]
    if (ROOT / 'runtime.json').exists(): api_port = settings()['port']
    env = {'DATABASE_URL': f'postgres://vhness:{password.read_text()}@127.0.0.1:{port}/vhness?sslmode=disable', 'API_ADDR': f'127.0.0.1:{api_port}', 'VHNESS_WEB': str(ROOT / 'web'), 'VHNESS_MIGRATIONS': str(REPO / 'migrations'), 'VHNESS_MEDIA': str(ROOT / 'media')}
    p = ROOT / 'environment.json'; p.write_text(json.dumps(env)); p.chmod(0o600)
    return api_port

def build():
    if live_pid(): raise RuntimeError('Stop this pilot API before replacing its executable')
    command(['go', 'build', '-o', str(ROOT / 'bin/vhness'), './cmd/vhness'], cwd=REPO / 'backend')
    # Use only this worktree's already-verified isolated frontend output.
    key = hashlib.sha256(str(REPO).encode()).hexdigest()[:12]
    output = Path.home() / f'.local/state/t3-dev-environments/fleet/t3dev-vhness-{key}-test/source/web/dist'
    if not (output / 'index.html').exists(): raise RuntimeError('Run isolated make verify before pilot build')
    snapshot = output.parent
    for path in [* (REPO / 'web/src').rglob('*.ts'), * (REPO / 'web/src').rglob('*.tsx'), * (REPO / 'web/src').rglob('*.css'), REPO / 'web/index.html', REPO / 'web/package.json']:
        candidate = snapshot / path.relative_to(REPO / 'web')
        if not candidate.exists() or candidate.read_bytes() != path.read_bytes():
            raise RuntimeError('Frontend differs from verified snapshot; run isolated make verify before pilot build')
    shutil.copytree(output, ROOT / 'web', dirs_exist_ok=True)
    identity = {'source': str(REPO), 'revision': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=REPO, text=True).strip(), 'dirty': bool(subprocess.check_output(['git', 'status', '--porcelain'], cwd=REPO)), 'binary_sha256': hashlib.sha256((ROOT / 'bin/vhness').read_bytes()).hexdigest(), 'video_storage': 'explicit collection manifests only; Democracy Now! source streaming only'}
    (ROOT / 'evidence/artifact.json').write_text(json.dumps(identity, indent=2) + '\n')

def start():
    if live_pid(): raise RuntimeError('This pilot is already running')
    port = prepare(); build(); cli(['migrate']); cli(['collection', '--collection', 'democracy_now_vid', '--policy', 'source_stream'])
    with (ROOT / 'logs/api.log').open('a') as log:
        process = subprocess.Popen([str(ROOT / 'bin/vhness'), 'serve'], env=environment(), stdout=log, stderr=log, start_new_session=True)
    (ROOT / 'runtime.json').write_text(json.dumps({'pid': process.pid, 'port': port, 'source': str(REPO)}))
    url = f'http://127.0.0.1:{port}'
    for _ in range(40):
        if process.poll() is not None: raise RuntimeError('Pilot API failed; inspect its private log')
        try:
            with urllib.request.urlopen(url + '/api/ready', timeout=1) as response:
                if response.status == 200: print(url); return
        except OSError: pass
        time.sleep(.25)
    raise RuntimeError('Pilot readiness timed out')

def stop():
    pid = live_pid()
    if pid:
        os.kill(pid, signal.SIGTERM)
        for _ in range(40):
            if not Path(f'/proc/{pid}/cmdline').exists() or not Path(f'/proc/{pid}/cmdline').read_bytes(): break
            time.sleep(.25)
        else: raise RuntimeError('Pilot did not stop; no force kill performed')
    print('Stopped owned pilot API; metadata database and recovery state retained')

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['start', 'stop', 'status', 'run'])
    parser.add_argument('args', nargs=argparse.REMAINDER)
    args = parser.parse_args()
    if args.action == 'start': start()
    elif args.action == 'stop': stop()
    elif args.action == 'run': cli(args.args)
    else:
        print(json.dumps({'api_running': bool(live_pid()), 'url': f"http://127.0.0.1:{settings()['port']}" if (ROOT / 'runtime.json').exists() else None, 'root': str(ROOT), 'video_storage': 'explicit collection manifests only; Democracy Now! source streaming only'}, indent=2))

if __name__ == '__main__': main()
