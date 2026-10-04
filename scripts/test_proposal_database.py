import copy
import unittest

from propose_release_upgrades import consumer_test_database


class ProposalDatabaseTests(unittest.TestCase):
    def setUp(self):
        self.inspected = {'Config': {'Labels': {'arkhiv.upgrade-database': 'run-1'},
            'Env': ['POSTGRES_USER=dev', 'POSTGRES_PASSWORD=disposable-dev-only']},
            'NetworkSettings': {'Networks': {'job': {'IPAddress': '172.18.0.2'}}}}

    def test_job_owned_database_uses_exact_launcher_identity_and_fresh_consumer_name(self):
        calls = []
        def resolve(name):
            calls.append(name)
            return '172.18.0.2'
        database, url = consumer_test_database(self.inspected, 'run-1', 'subculture-collective/arkhiv-alpha', resolve)
        self.assertEqual(database, 'upgrade_arkhiv_alpha')
        self.assertEqual(url, 'postgres://dev:disposable-dev-only@db:5432/upgrade_arkhiv_alpha?sslmode=disable')
        self.assertEqual(calls, ['db'])

    def test_rejects_other_job_identity_alias_network_and_database_name(self):
        for failure in ['job', 'user', 'password', 'alias', 'networks', 'name']:
            inspected = copy.deepcopy(self.inspected)
            repository = 'subculture-collective/arkhiv-alpha'
            if failure == 'job':
                inspected['Config']['Labels']['arkhiv.upgrade-database'] = 'other-job'
            if failure == 'user':
                inspected['Config']['Env'][0] = 'POSTGRES_USER=production'
            if failure == 'password':
                inspected['Config']['Env'][1] = 'POSTGRES_PASSWORD=other'
            if failure == 'networks':
                inspected['NetworkSettings']['Networks']['other'] = {'IPAddress': '172.19.0.2'}
            if failure == 'name':
                repository = 'owner/' + 'x' * 64
            with self.subTest(failure=failure), self.assertRaises(ValueError):
                consumer_test_database(inspected, 'run-1', repository,
                    lambda name: '172.18.0.99' if failure == 'alias' else '172.18.0.2')
