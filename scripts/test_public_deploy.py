import json
from pathlib import Path
import tempfile
import unittest

from public_deploy import DeployError, apply, load_config, validate_plan

REGISTRY = 'git.subcult.tv/subculture-collective'
SHA = 'a' * 40
CORE = REGISTRY + '/archive-core@sha256:' + 'b' * 64


class FakeDocker:
    """Answers the docker and compose commands the deploy issues, recording them."""
    def __init__(self, project, running, labels=None, fail_check=False):
        self.project = project
        self.running = dict(running)      # service -> image id
        self.labels = labels or {}        # image ref -> revision label
        self.calls = []
        self.fail_check = fail_check
        self.fetches = []

    def run(self, args, stdin=None):
        self.calls.append(list(args))
        if args[:2] == ['docker', 'pull']:
            return ''
        if args[:3] == ['docker', 'image', 'inspect']:
            ref = args[-1]
            revision = self.labels.get(ref, 'missing')
            if '--format' in args:
                return self.image_id(ref)
            return json.dumps([{'Id': self.image_id(ref), 'Config': {'Labels': {'org.opencontainers.image.revision': revision}}}])
        if args[:2] == ['docker', 'build']:
            return ''
        if args[:2] == ['docker', 'inspect']:
            name = args[2]
            service = name[len(self.project) + 1:-2]
            return json.dumps([{'Image': self.running[service], 'Config': {'Env': ['A=1'], 'Labels': {
                'com.docker.compose.project': self.project, 'com.docker.compose.service': service,
                'com.docker.compose.project.config_files': '/p/compose.json', 'com.docker.compose.project.environment_file': '/p/.env'}},
                'HostConfig': {k: None for k in ['Memory', 'MemorySwap', 'NanoCpus', 'PidsLimit', 'ReadonlyRootfs', 'SecurityOpt', 'CapDrop', 'CapAdd', 'PortBindings', 'LogConfig']} | {'RestartPolicy': {'Name': 'unless-stopped', 'MaximumRetryCount': 0}},
                'Mounts': []}])
        if args[:2] == ['docker', 'compose'] and 'config' in args:
            overlay = next((Path(args[i + 1]) for i, a in enumerate(args) if a == '-f' and args[i + 1].endswith('.compose.json')), None)
            override = json.loads(overlay.read_text())['services'] if overlay else {}
            services = {s: {'image': override.get(s, {}).get('image', 'old'), 'mem_limit': '256m'} for s in self.running}
            return json.dumps({'services': services})
        if args[:2] == ['docker', 'compose'] and 'up' in args:
            overlay = Path(args[args.index('-f', args.index('-f') + 1) + 1]) if args.count('-f') > 1 else Path(args[args.index('-f') + 1])
            spec = json.loads(overlay.read_text())['services']
            service = next(iter(spec))
            ref = spec[service]['image']
            self.running[service] = ref if ref.startswith('sha256:') else self.image_id(ref)
            return ''
        if args and args[0] in ('backup', 'monitor', 'migrate'):
            return ''
        raise AssertionError('unexpected command ' + ' '.join(args))

    @staticmethod
    def image_id(ref):
        return 'sha256:id-' + ref.rsplit(':', 1)[-1][:12]

    def fetch(self, url):
        self.fetches.append(url)
        return (503, '') if self.fail_check else (200, '<html>ok')


def config(state, services, extra=None):
    return load_config_dict({'state': str(state), 'registry': REGISTRY, 'targets': {'platform': {'project': 'arkhiv-platform-public-20261004', 'services': services, **(extra or {})}}})


def load_config_dict(data):
    with tempfile.NamedTemporaryFile('w', suffix='.json', delete=False) as handle:
        json.dump(data, handle)
    return load_config(handle.name)


class PlanValidationTests(unittest.TestCase):
    def setUp(self):
        self.state = tempfile.TemporaryDirectory()
        self.addCleanup(self.state.cleanup)
        self.config = config(self.state.name, {'reference': {'kind': 'frontend', 'image': 'archive-reference', 'origin': 'https://arkhiv.subcult.tv'},
                                               'archive-core': {'kind': 'core', 'image': 'archive-core'}})

    def test_accepts_registry_tags_and_core_digests_only(self):
        good = {'services': {'reference': {'image': REGISTRY + '/archive-reference:' + SHA}, 'archive-core': {'image': CORE}}}
        resolved = validate_plan(self.config, 'platform', good)
        self.assertEqual(resolved['reference']['revision'], SHA)
        self.assertIsNone(resolved['archive-core']['revision'])
        for bad in [{'reference': {'image': REGISTRY + '/archive-reference:main'}},
                    {'reference': {'image': 'docker.io/library/nginx:' + SHA}},
                    {'archive-core': {'image': REGISTRY + '/archive-core:' + SHA}},
                    {'db': {'image': REGISTRY + '/postgres:' + SHA}},
                    {'reference': {'image': REGISTRY + '/archive-reference@sha256:' + 'c' * 64}}]:
            with self.subTest(bad=bad), self.assertRaises(DeployError):
                validate_plan(self.config, 'platform', {'services': bad})

    def test_rejects_misconfigured_targets(self):
        with self.assertRaises(DeployError):
            load_config_dict({'state': '/tmp/x', 'registry': REGISTRY, 'targets': {'t': {'project': 'p', 'services': {'svc': {'kind': 'web', 'image': 'x'}}}}})


class ApplyTests(unittest.TestCase):
    def setUp(self):
        self.state = tempfile.TemporaryDirectory()
        self.addCleanup(self.state.cleanup)
        self.services = {'reference': {'kind': 'frontend', 'image': 'archive-reference', 'origin': 'https://arkhiv.subcult.tv'},
                         'democracy-now-reference': {'kind': 'frontend', 'image': 'democracy-now-tv-frontend', 'upstream': 'democracy-core:8080', 'origin': 'https://democracy-now.subcult.tv'},
                         'archive-core': {'kind': 'core', 'image': 'archive-core', 'migrate': ['migrate', '{image}']}}
        self.config = config(self.state.name, self.services, {'before_backend_change': ['backup'], 'after': ['monitor']})
        self.plan = {'services': {'reference': {'image': REGISTRY + '/archive-reference:' + SHA},
                                  'democracy-now-reference': {'image': REGISTRY + '/democracy-now-tv-frontend:' + SHA},
                                  'archive-core': {'image': CORE}}}
        self.labels = {REGISTRY + '/archive-reference:' + SHA: SHA, REGISTRY + '/democracy-now-tv-frontend:' + SHA: SHA, CORE: 'ignored'}

    def test_rejects_an_image_whose_revision_label_differs(self):
        docker = FakeDocker('arkhiv-platform-public-20261004', {'reference': 'sha256:old', 'democracy-now-reference': 'sha256:old', 'archive-core': 'sha256:old'}, {**self.labels, REGISTRY + '/archive-reference:' + SHA: 'f' * 40})
        with self.assertRaisesRegex(DeployError, 'carries revision'):
            apply(self.config, 'platform', self.plan, run=docker.run, fetch=docker.fetch)
        self.assertFalse(any('up' in c for c in docker.calls))

    def test_dry_run_pulls_and_reports_without_swapping(self):
        docker = FakeDocker('arkhiv-platform-public-20261004', {'reference': 'sha256:old', 'democracy-now-reference': 'sha256:old', 'archive-core': 'sha256:old'}, self.labels)
        receipt = apply(self.config, 'platform', self.plan, run=docker.run, fetch=docker.fetch, dry_run=True)
        self.assertEqual(receipt['status'], 'dry-run')
        self.assertEqual(sorted(receipt['changing']), ['archive-core', 'democracy-now-reference', 'reference'])
        self.assertFalse(any('up' in c or c[0] in ('backup', 'migrate') for c in docker.calls))
        self.assertTrue(any(c[:2] == ['docker', 'build'] for c in docker.calls), 'the brand wrapper is built so a dry run proves the derivation')

    def test_backend_change_backs_up_migrates_then_swaps_cores_before_frontends(self):
        docker = FakeDocker('arkhiv-platform-public-20261004', {'reference': 'sha256:old', 'democracy-now-reference': 'sha256:old', 'archive-core': 'sha256:old'}, self.labels)
        receipt = apply(self.config, 'platform', self.plan, run=docker.run, fetch=docker.fetch)
        self.assertEqual(receipt['status'], 'deployed')
        ops = [c[0] if c[0] != 'docker' else (c[1] if c[1] != 'compose' else 'compose ' + ('up' if 'up' in c else 'config')) for c in docker.calls]
        self.assertLess(ops.index('backup'), ops.index('migrate'))
        self.assertLess(ops.index('migrate'), ops.index('compose up'))
        self.assertEqual(ops[-1], 'monitor')
        self.assertEqual([s['service'] for s in receipt['services']], ['archive-core', 'reference', 'democracy-now-reference'])
        self.assertEqual(receipt['migrations'], [['migrate', CORE]])
        self.assertIn('https://democracy-now.subcult.tv/', docker.fetches)
        release = next(Path(self.state.name, 'deploys').iterdir())
        self.assertTrue((release / 'archive-core.rollback.json').exists())
        self.assertEqual(json.loads((release / 'receipt.json').read_text())['status'], 'deployed')

    def test_unchanged_images_do_nothing(self):
        docker = FakeDocker('arkhiv-platform-public-20261004', {'reference': 'sha256:id-' + SHA[:12], 'democracy-now-reference': 'sha256:id-' + SHA[:12], 'archive-core': 'sha256:id-' + ('b' * 64)[:12]}, self.labels)
        receipt = apply(self.config, 'platform', self.plan, run=docker.run, fetch=docker.fetch)
        self.assertEqual(receipt['status'], 'unchanged')
        self.assertFalse(any(c[0] in ('backup', 'migrate') or 'up' in c for c in docker.calls))

    def test_failed_origin_check_rolls_the_service_back_and_stops(self):
        docker = FakeDocker('arkhiv-platform-public-20261004', {'reference': 'sha256:old', 'democracy-now-reference': 'sha256:old', 'archive-core': 'sha256:old'}, self.labels, fail_check=True)
        import public_deploy
        original_sleep = public_deploy.time.sleep
        public_deploy.time.sleep = lambda seconds: None
        try:
            with self.assertRaisesRegex(DeployError, 'rolled back'):
                apply(self.config, 'platform', self.plan, run=docker.run, fetch=docker.fetch)
        finally:
            public_deploy.time.sleep = original_sleep
        rollbacks = [c for c in docker.calls if 'up' in c and any(a.endswith('.rollback.json') for a in c)]
        self.assertEqual(len(rollbacks), 1)
        self.assertEqual(docker.running['reference'], 'sha256:old')
        release = next(Path(self.state.name, 'deploys').iterdir())
        receipt = json.loads((release / 'receipt.json').read_text())
        self.assertEqual(receipt['status'], 'failed')
        # The core had already swapped cleanly; the failing frontend was reverted and the run stopped there.
        self.assertEqual([(s['service'], s['status']) for s in receipt['services']], [('archive-core', 'deployed')])
        self.assertEqual(docker.running['democracy-now-reference'], 'sha256:old')
        self.assertFalse(any(c[0] == 'monitor' for c in docker.calls))


if __name__ == '__main__':
    unittest.main()
