"""Independent pg_dump/pg_restore qualification on explicitly disposable databases."""
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
import uuid

from personal_backup import connection_environment


@unittest.skipUnless(os.environ.get('RUN_PERSONAL_RESTORE_TEST') == '1' and os.environ.get('TEST_DATABASE_URL'),
                     'Independent restore requires disposable TEST_DATABASE_URL and matching PostgreSQL clients')
class PersonalRestore(unittest.TestCase):
    def test_old_dump_cannot_resurrect_deleted_data_or_sessions(self):
        root = Path(__file__).resolve().parents[1]
        binary = root / 'backend/vhness'
        subprocess.run(['go', 'build', '-o', str(binary), './cmd/vhness'], cwd=root / 'backend', check=True)
        with tempfile.TemporaryDirectory(prefix='personal-independent-restore-') as temporary:
            directory = Path(temporary)
            admin_url = directory / 'disposable-url'
            admin_url.write_text(os.environ['TEST_DATABASE_URL'])
            env = connection_environment(admin_url)
            databases = ['personal_proof_' + uuid.uuid4().hex for _ in range(2)]
            def sql(statement, database=None):
                target = dict(env)
                if database:
                    target['PGDATABASE'] = database
                return subprocess.run(['psql', '-X', '-A', '-t', '-v', 'ON_ERROR_STOP=1', '-c', statement],
                                      env=target, capture_output=True, text=True, check=True).stdout.strip()
            for database in databases:
                sql('CREATE DATABASE ' + database)
            try:
                config_path, url_path = directory / 'config.json', directory / 'personal-url'
                journal = directory / 'journal'
                journal.mkdir(mode=0o700)
                config = {'deployment': 'restore-proof', 'database_url_file': str(url_path), 'recovery_directory': str(journal)}
                config_path.write_text(json.dumps(config))
                command_env = dict(os.environ, ARKHIV_COMMUNITY_CONFIG_FILE=str(config_path),
                                   ARKHIV_COMMUNITY_MIGRATIONS=str(root / 'migrations/community'))
                def point(database):
                    # TEST_DATABASE_URL is the only connection authority for this test.
                    from urllib.parse import urlsplit, urlunsplit
                    value = urlsplit(os.environ['TEST_DATABASE_URL'])
                    url_path.write_text(urlunsplit(value._replace(path='/' + database)))
                    url_path.chmod(0o600)
                point(databases[0])
                subprocess.run([str(binary), 'community-migrate'], env=command_env, check=True)
                user, message, blocked, withdrawn = ['1' * 64, '2' * 64, '3' * 64, '4' * 64]
                sql(f"INSERT INTO personal_users(id,deployment,issuer,subject,alias) VALUES('{user}','restore-proof','provider','subject','Viewer');"
                    f"INSERT INTO personal_favorites(user_id,collection,item,revision) VALUES('{user}','room','episode',1);"
                    "INSERT INTO personal_sessions VALUES('session','restore-proof','provider','subject',now(),now()+interval '1 hour');"
                    f"INSERT INTO chat_messages(id,deployment,collection,user_id,alias,body,revision,program_start,episode,original_date,idempotency_key) VALUES('{message}','restore-proof','room','{user}','Viewer','deleted body','r',now(),'episode','1989','a'),('{withdrawn}','restore-proof','room','{blocked}','Viewer','withdrawn body','r',now(),'episode','1989','b');", databases[0])
                sql(f"UPDATE chat_messages SET media='audio',item='recording',representation='continuous' WHERE id='{withdrawn}'", databases[0])
                output = subprocess.run(['python3', str(root / 'scripts/personal_backup.py'), 'backup',
                                         '--database-url-file', str(url_path), '--deployment', 'restore-proof',
                                         '--destination', str(directory / 'dumps'), '--checkpoint-directory', str(directory / 'checkpoints'),
                                         '--binary', str(binary)], env=command_env, capture_output=True, text=True, check=True)
                backup = Path(json.loads(output.stdout)['backup'])
                at = dt.datetime.now(dt.timezone.utc)
                # Journal is independently retained after the older backup/checkpoint.
                for kind, opaque_id in [('account', user), ('message', message), ('hide', withdrawn), ('restriction', blocked)]:
                    event = {'version': 1, 'deployment': 'restore-proof', 'at': at.isoformat(), 'kind': kind, 'opaque_id': opaque_id}
                    if kind == 'restriction':
                        event['until'] = (at + dt.timedelta(hours=24)).isoformat()
                    raw = json.dumps(event).encode()
                    path = journal / ('recovery-' + hashlib.sha256(raw).hexdigest() + '.json')
                    path.write_bytes(raw)
                    path.chmod(0o600)
                point(databases[1])
                subprocess.run(['python3', str(root / 'scripts/personal_backup.py'), 'restore',
                                '--database-url-file', str(url_path), '--deployment', 'restore-proof', '--backup', str(backup),
                                '--checkpoint', str(directory / 'checkpoints/current-personal-recovery.json'), '--binary', str(binary)],
                               env=command_env, check=True)
                # A separate PostgreSQL client inspects the recovered database.
                self.assertEqual(sql('SELECT count(*) FROM personal_users', databases[1]), '0')
                self.assertEqual(sql('SELECT count(*) FROM personal_favorites', databases[1]), '0')
                self.assertEqual(sql('SELECT count(*) FROM personal_sessions', databases[1]), '0')
                self.assertEqual(sql(f"SELECT count(*) FROM chat_messages WHERE id='{message}'", databases[1]), '0')
                self.assertEqual(sql(f"SELECT hidden FROM chat_messages WHERE id='{withdrawn}'", databases[1]), 't')
                self.assertEqual(sql(f"SELECT count(*) FROM chat_restrictions WHERE user_id='{blocked}' AND until_at>now()", databases[1]), '1')
            finally:
                for database in databases:
                    sql('DROP DATABASE ' + database)

    def test_legacy_video_dump_upgrades_before_media_recovery(self):
        root = Path(__file__).resolve().parents[1]
        binary = root / 'backend/vhness'
        subprocess.run(['go', 'build', '-o', str(binary), './cmd/vhness'], cwd=root / 'backend', check=True)
        with tempfile.TemporaryDirectory(prefix='personal-legacy-restore-') as temporary:
            directory = Path(temporary)
            url_path = directory / 'personal-url'
            url_path.write_text(os.environ['TEST_DATABASE_URL'])
            url_path.chmod(0o600)
            admin_env = connection_environment(url_path)
            databases = ['personal_legacy_' + uuid.uuid4().hex for _ in range(2)]
            def sql(statement, database=None):
                env = dict(admin_env)
                if database:
                    env['PGDATABASE'] = database
                return subprocess.run(['psql', '-X', '-A', '-t', '-v', 'ON_ERROR_STOP=1', '-c', statement],
                                      env=env, capture_output=True, text=True, check=True).stdout.strip()
            def point(database):
                from urllib.parse import urlsplit, urlunsplit
                value = urlsplit(os.environ['TEST_DATABASE_URL'])
                url_path.write_text(urlunsplit(value._replace(path='/' + database)))
            for database in databases:
                sql('CREATE DATABASE ' + database)
            try:
                journal = directory / 'journal'
                journal.mkdir(mode=0o700)
                config_path = directory / 'config.json'
                config_path.write_text(json.dumps({'deployment': 'legacy-proof', 'database_url_file': str(url_path), 'recovery_directory': str(journal)}))
                legacy = directory / 'legacy-migrations'
                legacy.mkdir()
                (legacy / '0001_personal_chat.sql').write_bytes((root / 'migrations/community/0001_personal_chat.sql').read_bytes())
                env = dict(os.environ, ARKHIV_COMMUNITY_CONFIG_FILE=str(config_path), ARKHIV_COMMUNITY_MIGRATIONS=str(legacy))
                point(databases[0])
                subprocess.run([str(binary), 'community-migrate'], env=env, check=True)
                user, message = '5' * 64, '6' * 64
                sql(f"INSERT INTO personal_users(id,deployment,issuer,subject,alias) VALUES('{user}','legacy-proof','provider','subject','Viewer');"
                    f"INSERT INTO chat_messages(id,deployment,collection,user_id,alias,body,revision,program_start,episode,original_date,idempotency_key) VALUES('{message}','legacy-proof','mixed','{user}','Viewer','legacy video','r',now(),'episode','1977','old-key');"
                    f"INSERT INTO chat_requests VALUES('legacy-proof','{user}','old-key','{message}','retained-digest',now());", databases[0])
                backup = directory / 'legacy-backup'
                backup.mkdir()
                dump = backup / 'personal.dump'
                subprocess.run(['pg_dump', '--format=custom', '--no-owner', '--no-acl', '--file', str(dump)],
                               env=connection_environment(url_path), check=True)
                now = dt.datetime.now(dt.timezone.utc)
                (backup / 'receipt.json').write_text(json.dumps({'version': 1, 'kind': 'arkhiv-personal-backup', 'deployment': 'legacy-proof',
                    'created_at': now.isoformat(), 'sha256': hashlib.sha256(dump.read_bytes()).hexdigest()}))
                checkpoint = directory / 'checkpoint.json'
                checkpoint.write_text(json.dumps({'version': 1, 'deployment': 'legacy-proof', 'at': now.isoformat(), 'deletions': [],
                    'withdrawals': [], 'restrictions': [{'opaque_user_id': user, 'until': (now + dt.timedelta(hours=24)).isoformat()}]}))
                event = {'version': 1, 'deployment': 'legacy-proof', 'at': now.isoformat(), 'kind': 'restriction', 'opaque_id': user,
                         'media': 'audio', 'until': (now + dt.timedelta(hours=1)).isoformat()}
                raw = json.dumps(event).encode()
                (journal / ('recovery-' + hashlib.sha256(raw).hexdigest() + '.json')).write_bytes(raw)
                point(databases[1])
                env['ARKHIV_COMMUNITY_MIGRATIONS'] = str(root / 'migrations/community')
                subprocess.run(['python3', str(root / 'scripts/personal_backup.py'), 'restore', '--database-url-file', str(url_path),
                    '--deployment', 'legacy-proof', '--backup', str(backup), '--checkpoint', str(checkpoint), '--binary', str(binary)], env=env, check=True)
                self.assertEqual(sql(f"SELECT media||':'||episode FROM chat_messages WHERE id='{message}'", databases[1]), 'video:episode')
                self.assertEqual(sql("SELECT media||':'||digest FROM chat_requests WHERE key='old-key'", databases[1]), 'video:retained-digest')
                self.assertEqual(sql(f"SELECT string_agg(media,',' ORDER BY media) FROM chat_restrictions WHERE user_id='{user}'", databases[1]), 'audio,video')
            finally:
                for database in databases:
                    sql('DROP DATABASE ' + database)
