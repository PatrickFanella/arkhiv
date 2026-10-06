import datetime as dt
import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import personal_backup


class PersonalBackupTests(unittest.TestCase):
    def test_only_verified_personal_backups_expire(self):
        now = dt.datetime(2026, 10, 5, tzinfo=dt.timezone.utc)
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for name, age, corrupt, extra in [('personal-old', 36, False, False),
                                               ('personal-recent', 34, False, False),
                                               ('personal-corrupt', 36, True, False),
                                               ('personal-extra', 36, False, True)]:
                directory = root / name
                directory.mkdir()
                (directory / 'personal.dump').write_bytes(b'personal fixture')
                receipt = {'kind': 'arkhiv-personal-backup', 'version': 1,
                           'created_at': (now - dt.timedelta(days=age)).isoformat(),
                           'sha256': 'wrong' if corrupt else hashlib.sha256(b'personal fixture').hexdigest()}
                (directory / 'receipt.json').write_text(json.dumps(receipt))
                if extra:
                    (directory / 'original.mp4').write_bytes(b'must retain')
            (root / 'archive-originals').mkdir()
            self.assertEqual(personal_backup.prune(root, now), 1)
            self.assertFalse((root / 'personal-old').exists())
            for name in ('personal-recent', 'personal-corrupt', 'personal-extra', 'archive-originals'):
                self.assertTrue((root / name).exists())

    def test_database_inventory_rejects_archive_and_nonempty_restore(self):
        with patch('personal_backup.subprocess.run') as run:
            run.return_value.stdout = 'personal_users\nchat_messages\nepisodes\n'
            with self.assertRaises(ValueError):
                personal_backup.require_personal_database({})
            run.return_value.stdout = 'personal_users\n'
            with self.assertRaises(ValueError):
                personal_backup.require_personal_database({}, empty=True)
            run.return_value.stdout = ''
            personal_backup.require_personal_database({}, empty=True)


if __name__ == '__main__':
    unittest.main()
