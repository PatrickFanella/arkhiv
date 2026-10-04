import copy
from pathlib import Path
import socket
import tempfile
import unittest

from bind_upgrade_database import bind


class AliasBindingTests(unittest.TestCase):
    def setUp(self):
        self.inspected = {'Config': {'Labels': {'arkhiv.upgrade-database': '1-1'},
            'Env': ['POSTGRES_USER=dev', 'POSTGRES_PASSWORD=disposable-dev-only']},
            'NetworkSettings': {'Networks': {'job': {'IPAddress': '172.18.0.2'}}}}
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.hosts = Path(self.temporary.name) / 'hosts'
        self.hosts.write_text('127.0.0.1 localhost\n')

    def test_missing_alias_appends_only_the_verified_job_address(self):
        def missing(name):
            self.assertEqual(name, 'db')
            raise socket.gaierror('unknown')
        bind(self.inspected, '1-1', self.hosts, missing)
        self.assertEqual(self.hosts.read_text(), '127.0.0.1 localhost\n172.18.0.2 db\n')

    def test_matching_alias_keeps_hosts_unchanged(self):
        bind(self.inspected, '1-1', self.hosts, lambda name: '172.18.0.2')
        self.assertEqual(self.hosts.read_text(), '127.0.0.1 localhost\n')

    def test_other_job_or_existing_alias_cannot_change_hosts(self):
        for outcome in ['job', 'alias']:
            inspected = copy.deepcopy(self.inspected)
            if outcome == 'job':
                inspected['Config']['Labels']['arkhiv.upgrade-database'] = 'other'
            with self.subTest(outcome=outcome), self.assertRaises(ValueError):
                bind(inspected, '1-1', self.hosts, lambda name: '172.18.0.99')
            self.assertEqual(self.hosts.read_text(), '127.0.0.1 localhost\n')
