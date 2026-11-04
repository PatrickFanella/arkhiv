import base64
import unittest

from deploy_plan import REGISTRY, plan

CORE = REGISTRY + '/archive-core@sha256:' + 'b' * 64


class DeployPlanTests(unittest.TestCase):
    def fetch(self, path):
        if path.endswith('/branches/main'):
            repository = path.split('/repos/')[1].split('/branches')[0]
            return {'commit': {'id': ('1' if 'democracy' in repository else '2' if 'bill' in repository else '3') * 40}}
        if '/contents/deploy/archive-core.env' in path:
            return {'content': base64.b64encode(('ARCHIVE_ENGINE_IMAGE=' + CORE + '\n').encode()).decode()}
        raise AssertionError(path)

    def test_platform_plan_names_each_consumer_revision_and_its_pinned_core(self):
        result = plan('platform', self.fetch)
        services = result['services']
        self.assertEqual(services['democracy-now-reference']['image'], REGISTRY + '/democracy-now-tv-frontend:' + '1' * 40)
        self.assertEqual(services['bill-nye-reference']['image'], REGISTRY + '/bill-nye-tv-frontend:' + '2' * 40)
        self.assertEqual(services['democracy-core']['image'], CORE)
        self.assertEqual(services['archive-core']['from'], 'subculture-collective/arkhiv@' + '3' * 40)
        self.assertEqual(services['riot-public-api']['revision'], '3' * 40)

    def test_rejects_a_mutable_core_pin(self):
        def fetch(path):
            if '/contents/' in path:
                return {'content': base64.b64encode(b'ARCHIVE_ENGINE_IMAGE=git.subcult.tv/subculture-collective/archive-core:latest\n').decode()}
            return self.fetch(path)
        with self.assertRaises(ValueError):
            plan('platform', fetch)

    def test_dedhed_and_anarchist_targets(self):
        self.assertEqual(set(plan('dedhed', self.fetch)['services']), {'web', 'api', 'operator'})
        self.assertEqual(set(plan('anarchist', self.fetch)['services']), {'public-web', 'public-api'})
        with self.assertRaises(ValueError):
            plan('everything', self.fetch)


if __name__ == '__main__':
    unittest.main()
