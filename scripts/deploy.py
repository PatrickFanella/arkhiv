#!/usr/bin/env python3
"""Prepare and start an independently owned loopback deployment from verified images."""
import argparse
import json
import os
from pathlib import Path
import re
import secrets
import subprocess

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['prepare', 'start', 'stop', 'status'])
    parser.add_argument('--root', required=True)
    parser.add_argument('--project', required=True)
    parser.add_argument('--image')
    parser.add_argument('--collection')
    parser.add_argument('--media')
    parser.add_argument('--port', type=int, default=8080)
    parser.add_argument('--snapshot', help='reviewed collection-only dump; allowed only into an empty deployment DB')
    parser.add_argument('--brand', help='white-label site JSON mounted read-only over the image default')
    parser.add_argument('--brand-assets', help='owned brand asset directory mounted read-only at /brand')
    args = parser.parse_args()
    if not re.fullmatch(r'[a-z0-9][a-z0-9-]{1,50}', args.project): raise RuntimeError('Invalid owned project identity')
    root = Path(args.root).expanduser().resolve()
    if not re.fullmatch(r'/[A-Za-z0-9_./-]+', str(root)): raise RuntimeError('Use a simple absolute private root')
    repo = Path(__file__).resolve().parents[1]
    if args.action == 'prepare':
        if not args.image or not re.fullmatch(r'[A-Za-z0-9_./:@-]+', args.image) or not args.collection or not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._-]{0,199}', args.collection) or not 1024 <= args.port <= 65535: raise RuntimeError('Provide a verified image, collection, and unprivileged port')
        root.mkdir(parents=True, exist_ok=True, mode=0o700); root.chmod(0o700)
        folder = root / 'secrets'; folder.mkdir(exist_ok=True, mode=0o700)
        password = folder / 'database-password'; connection = folder / 'database-url'
        if password.exists() != connection.exists(): raise RuntimeError('Incomplete existing secret pair; preserve and inspect')
        if not password.exists():
            token = secrets.token_urlsafe(32)
            password.write_text(token); password.chmod(0o600)
            connection.write_text(f'postgres://vhness:{token}@db:5432/vhness?sslmode=disable'); connection.chmod(0o600)
        media = Path(args.media).expanduser().resolve() if args.media else root / 'media'
        media.mkdir(parents=True, exist_ok=True, mode=0o700)
        if not re.fullmatch(r'/[A-Za-z0-9_./-]+', str(media)): raise RuntimeError('Use a simple owned media path')
        config = {'APP_IMAGE': args.image, 'COLLECTION': args.collection, 'MEDIA_ROOT': str(media), 'SECRET_ROOT': str(folder), 'APP_PORT': str(args.port), 'APP_UID': str(os.getuid()), 'APP_GID': str(os.getgid())}
        receipt = root / 'deployment.json'
        if receipt.exists() and json.loads(receipt.read_text())['project'] != args.project: raise RuntimeError('Root already belongs to another project')
        (root / 'deploy.env').write_text(''.join(f'{k}={v}\n' for k,v in config.items())); (root / 'deploy.env').chmod(0o600)
        receipt.write_text(json.dumps({'project':args.project, 'config':config, 'source':str(repo)},indent=2)+'\n'); receipt.chmod(0o600)
        if args.brand:
            brand=Path(args.brand).expanduser().resolve()
            if not brand.is_file() or not re.fullmatch(r'/[A-Za-z0-9_./-]+',str(brand)): raise RuntimeError('Use an existing local site JSON path')
            json.loads(brand.read_text())
            mounts=[f'{brand}:/app/web/site.json:ro']
            if args.brand_assets:
                assets=Path(args.brand_assets).expanduser().resolve()
                if not assets.is_dir() or not re.fullmatch(r'/[A-Za-z0-9_./-]+',str(assets)): raise RuntimeError('Use an existing brand asset directory')
                mounts.append(f'{assets}:/app/web/brand:ro')
            (root/'brand.override.json').write_text(json.dumps({'services':{'app':{'volumes':mounts}}},indent=2)+'\n')
        print(f'Prepared {args.project} at {root}; secrets retained privately')
        return
    receipt = json.loads((root / 'deployment.json').read_text())
    if receipt['project'] != args.project: raise RuntimeError('Project does not match owned deployment')
    compose = ['docker','compose','--env-file',str(root/'deploy.env'),'-p',args.project,'-f',str(repo/'deploy/compose.yml')]
    if (root/'brand.override.json').exists(): compose.extend(['-f',str(root/'brand.override.json')])
    def run(*command, **kwargs): return subprocess.run([*compose,*command],check=True,**kwargs)
    if args.action == 'start':
        run('up','-d','--wait','db')
        if args.snapshot:
            snapshot=Path(args.snapshot).resolve()
            if not snapshot.is_file(): raise RuntimeError('Snapshot file unavailable')
            check=subprocess.check_output([*compose,'exec','-T','db','psql','-U','vhness','-d','vhness','-At','-c',"SELECT count(*) FROM information_schema.tables WHERE table_schema='public'"],text=True).strip()
            if check!='0': raise RuntimeError('Snapshot restore is allowed only into an empty owned database; existing data retained')
            with snapshot.open('rb') as stream: run('exec','-T','db','pg_restore','-U','vhness','-d','vhness','--exit-on-error','--no-owner','--no-acl',stdin=stream)
        run('run','--rm','--no-deps','app','migrate')
        run('up','-d','--wait','app')
        print(f"Ready at http://127.0.0.1:{receipt['config']['APP_PORT']}; no public ingress configured")
    elif args.action == 'stop': run('stop')
    else: run('ps')

if __name__ == '__main__': main()
