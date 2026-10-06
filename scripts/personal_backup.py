"""Bounded personal-only backups; archive retention is never changed here."""
import argparse
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import uuid
from urllib.parse import parse_qs, unquote, urlsplit


def connection_environment(path):
    value = urlsplit(Path(path).read_text().strip())
    if value.scheme not in ('postgres', 'postgresql') or not value.hostname:
        raise ValueError('Expected a PostgreSQL URL file')
    query = parse_qs(value.query)
    if set(query) - {'sslmode'}:
        raise ValueError('Connection options require separate operator review')
    env = dict(os.environ)
    env.update(PGHOST=value.hostname, PGPORT=str(value.port or 5432),
               PGUSER=unquote(value.username or ''), PGPASSWORD=unquote(value.password or ''),
               PGDATABASE=unquote(value.path.lstrip('/')), PGSSLMODE=query.get('sslmode', ['require'])[0])
    return env


def require_personal_database(env, empty=False):
    names = subprocess.run(['psql', '-X', '-A', '-t', '-v', 'ON_ERROR_STOP=1', '-c',
                            "SELECT tablename FROM pg_tables WHERE schemaname NOT IN ('pg_catalog','information_schema') ORDER BY tablename"],
                           env=env, capture_output=True, text=True, check=True).stdout.splitlines()
    if empty and names:
        raise ValueError('Restore target must be a new, empty personal database')
    if not empty and (not names or any(not name.startswith(('personal_', 'chat_')) and name != 'schema_versions' for name in names)):
        raise ValueError('Refusing a database containing archive or unknown tables')


def now_utc():
    return dt.datetime.now(dt.timezone.utc)


def require_matching_config(args):
    config = json.loads(Path(os.environ['ARKHIV_COMMUNITY_CONFIG_FILE']).read_text())
    if Path(config['database_url_file']).resolve() != args.database_url_file.resolve() or config['deployment'] != args.deployment:
        raise ValueError('CLI configuration must name this exact personal database and deployment')


def prune(destination, now):
    removed = 0
    for directory in destination.glob('personal-*'):
        if directory.is_symlink() or not directory.is_dir():
            continue
        receipt_path, dump = directory / 'receipt.json', directory / 'personal.dump'
        if receipt_path.is_symlink() or dump.is_symlink():
            continue
        try:
            receipt = json.loads(receipt_path.read_text())
            created = dt.datetime.fromisoformat(receipt['created_at'])
            if receipt['kind'] != 'arkhiv-personal-backup' or receipt['version'] != 1 or created.tzinfo is None:
                continue
            if now - created < dt.timedelta(days=35):
                continue
            if hashlib.sha256(dump.read_bytes()).hexdigest() != receipt['sha256']:
                continue
            if set(p.name for p in directory.iterdir()) != {'receipt.json', 'personal.dump'}:
                continue
        except (OSError, ValueError, KeyError):
            continue
        shutil.rmtree(directory)
        removed += 1
    return removed


def backup(args):
    require_matching_config(args)
    env = connection_environment(args.database_url_file)
    require_personal_database(env)
    destination = args.destination.resolve()
    checkpoints = args.checkpoint_directory.resolve()
    if destination == checkpoints or destination in checkpoints.parents or checkpoints in destination.parents:
        raise ValueError('Current recovery checkpoints require a separate destination')
    destination.mkdir(parents=True, exist_ok=True, mode=0o700)
    checkpoints.mkdir(parents=True, exist_ok=True, mode=0o700)
    now = now_utc()
    checkpoint = subprocess.run([args.binary, 'community-checkpoint'], env=env,
                                capture_output=True, text=True, check=True).stdout
    parsed = json.loads(checkpoint)
    if parsed['deployment'] != args.deployment or parsed['version'] != 1:
        raise ValueError('Checkpoint deployment mismatch')
    with tempfile.TemporaryDirectory(prefix='.personal-', dir=destination) as temporary:
        root = Path(temporary)
        dump = root / 'personal.dump'
        subprocess.run(['pg_dump', '--format=custom', '--no-owner', '--no-acl', '--file', str(dump)], env=env, check=True)
        receipt = {'version': 1, 'kind': 'arkhiv-personal-backup', 'deployment': args.deployment,
                   'created_at': now.isoformat(), 'sha256': hashlib.sha256(dump.read_bytes()).hexdigest()}
        (root / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n')
        for path in root.iterdir():
            path.chmod(0o600)
        # Refresh CURRENT deletion/moderation evidence independently of an older dump.
        checkpoint_tmp = checkpoints / ('.checkpoint-' + uuid.uuid4().hex)
        checkpoint_tmp.write_text(checkpoint)
        checkpoint_tmp.chmod(0o600)
        checkpoint_tmp.replace(checkpoints / 'current-personal-recovery.json')
        final = destination / ('personal-' + now.strftime('%Y%m%dT%H%M%SZ') + '-' + root.name.rsplit('-', 1)[-1])
        root.rename(final)
    print(json.dumps({'backup': str(final), 'sha256': receipt['sha256'], 'expired_backups_removed': prune(destination, now)}))


def restore(args):
    require_matching_config(args)
    env = connection_environment(args.database_url_file)
    require_personal_database(env, empty=True)
    receipt = json.loads((args.backup / 'receipt.json').read_text())
    dump = args.backup / 'personal.dump'
    if receipt['kind'] != 'arkhiv-personal-backup' or receipt['deployment'] != args.deployment or receipt['version'] != 1:
        raise ValueError('Backup deployment mismatch')
    age = now_utc() - dt.datetime.fromisoformat(receipt['created_at'])
    if age < dt.timedelta(0) or age > dt.timedelta(days=35):
        raise ValueError('Personal backup is outside accepted retention')
    if hashlib.sha256(dump.read_bytes()).hexdigest() != receipt['sha256']:
        raise ValueError('Personal backup checksum mismatch')
    # No --clean, drop, reset, or archive restore is permitted.
    subprocess.run(['pg_restore', '--exit-on-error', '--no-owner', '--no-acl', '--dbname', env['PGDATABASE'], str(dump)], env=env, check=True)
    # Upgrade an older personal-only checkpoint before replaying current media scopes.
    # The target was required to be empty; archive databases are never migrated here.
    subprocess.run([args.binary, 'community-migrate'], env=env, check=True)
    subprocess.run([args.binary, 'community-recover', '--file', str(args.checkpoint)], env=env, check=True)
    print('Personal restore complete; sessions revoked and current deletions/moderation reapplied. Keep serving disabled until independently verified.')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest='command', required=True)
    for name in ('backup', 'restore'):
        command = commands.add_parser(name)
        command.add_argument('--database-url-file', type=Path, required=True)
        command.add_argument('--deployment', required=True)
        command.add_argument('--binary', default='vhness')
    save = commands.choices['backup']
    save.add_argument('--destination', type=Path, required=True)
    save.add_argument('--checkpoint-directory', type=Path, required=True)
    load = commands.choices['restore']
    load.add_argument('--backup', type=Path, required=True)
    load.add_argument('--checkpoint', type=Path, required=True)
    args = parser.parse_args()
    (backup if args.command == 'backup' else restore)(args)


if __name__ == '__main__':
    main()
