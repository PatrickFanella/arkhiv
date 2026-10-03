#!/usr/bin/env python3
"""Serve an independent backup privately for browser recovery qualification.

Run on the backup host. The supplied binary and frontend must come from the
verified candidate. This creates only its own disposable database and API;
the backup and immutable restored originals are read without replacement.
"""
import argparse
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import time
import urllib.request

def run(args, **kwargs): return subprocess.run(args, check=True, **kwargs)

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action',choices=['start','stop','status','run'])
    parser.add_argument('--backup',required=True)
    parser.add_argument('--command',nargs=argparse.REMAINDER)
    args=parser.parse_args()
    backup=Path(args.backup).resolve();root=backup/'browser-proof';root.mkdir(mode=0o700,exist_ok=True)
    state=root/'runtime.json';name='vhness-browser-restore-'+backup.name.lower()
    if args.action=='status':
        print(state.read_text() if state.exists() else 'No owned restore API');return
    if args.action=='run':
        if not args.command or args.command[0] not in ['schedule','audit','report']:raise RuntimeError('Restore operator command must be schedule, audit, or report')
        env={**os.environ,'DATABASE_URL':f'host={root / "sockets"} user=postgres dbname=postgres sslmode=disable','VHNESS_MEDIA':str(backup/'restored-originals')}
        run([str(root/'vhness'),*args.command],env=env);return
    if args.action=='stop':
        if state.exists():
            identity=json.loads(state.read_text());pid=identity['pid'];cmd=Path(f'/proc/{pid}/cmdline')
            if cmd.exists() and cmd.read_bytes():
                if cmd.read_bytes().split(b'\0')[:2]!=[str(root/'vhness').encode(),b'serve']: raise RuntimeError('Recorded PID changed owner; preserve it')
                os.kill(pid,signal.SIGTERM)
                for _ in range(40):
                    if not cmd.exists() or not cmd.read_bytes():break
                    time.sleep(.25)
                else:raise RuntimeError('Owned restore API did not stop; no force kill')
        run(['docker','rm','-f','-v',name],stdout=subprocess.DEVNULL)
        print('Owned restore API/database stopped; independent backup retained');return
    if state.exists():raise RuntimeError('Restore runtime already recorded; inspect and stop explicitly')
    if not (backup/'receipt.json').exists() or not (backup/'metadata.dump').exists() or not (backup/'restored-originals').exists():raise RuntimeError('Complete verified backup required')
    sockets=root/'sockets';sockets.mkdir(mode=0o777);sockets.chmod(0o777)
    run(['docker','run','-d','--name',name,'--network','none','--memory','512m','--cpus','1','--mount',f'type=bind,source={sockets},target=/var/run/postgresql','-e','POSTGRES_HOST_AUTH_METHOD=trust','postgres:17'],stdout=subprocess.DEVNULL)
    try:
        for _ in range(60):
            ready=subprocess.run(['docker','exec',name,'pg_isready','-h','127.0.0.1','-U','postgres'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
            if ready.returncode==0:break
            time.sleep(.5)
        else:raise RuntimeError('Restore database readiness failed')
        with (backup/'metadata.dump').open('rb') as stream:run(['docker','exec','-i',name,'pg_restore','-U','postgres','-d','postgres','--exit-on-error','--no-owner','--no-acl'],stdin=stream)
        with socket.socket() as sock:sock.bind(('127.0.0.1',0));port=sock.getsockname()[1]
        env={**os.environ,'DATABASE_URL':f'host={sockets} user=postgres dbname=postgres sslmode=disable','API_ADDR':f'127.0.0.1:{port}','VHNESS_MEDIA':str(backup/'restored-originals'),'VHNESS_WEB':str(root/'web')}
        with (root/'api.log').open('a') as log:process=subprocess.Popen([str(root/'vhness'),'serve'],env=env,stdout=log,stderr=log,start_new_session=True)
        state.write_text(json.dumps({'pid':process.pid,'port':port,'container':name,'backup':str(backup)}));state.chmod(0o600)
        for _ in range(40):
            if process.poll() is not None:raise RuntimeError('Restore API failed; inspect private log')
            try:
                with urllib.request.urlopen(f'http://127.0.0.1:{port}/api/ready',timeout=1) as response:
                    if response.status==200:print(json.dumps({'host':socket.gethostname(),'port':port,'backup':str(backup),'scope':'independently restored metadata and originals, loopback API; browser proof pending'}));return
            except OSError:pass
            time.sleep(.25)
        raise RuntimeError('Restore API readiness failed')
    except Exception:
        # Retain logs and recorded process identity for bounded operator cleanup.
        print('Restore qualification failed; inspect browser-proof and stop its owned resources')
        raise

if __name__=='__main__':main()
