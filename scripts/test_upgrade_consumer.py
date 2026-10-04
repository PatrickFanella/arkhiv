import hashlib
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

from upgrade_consumer import prepare, validate
from propose_consumer_upgrade import proposal


class UpgradeTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.artifact = self.root / 'artifact.tgz'
        self.artifact.write_bytes(b'registry-artifact')
        self.consumer = {
            'adoption': 'package', 'rollout': 'reviewed-pr', 'media': ['video'],
            'release_file': 'release.json', 'package_file': 'package.json',
            'lock_file': 'pnpm-lock.yaml', 'engine_file': 'engine.env',
            'checks': [['consumer-check']],
        }
        self.current = {
            'contract_version': 1, 'configuration_version': 1,
            'package': '@subcult/archive-viewer', 'api_versions': ['1'],
            'media': ['video'], 'registry': 'https://git.subcult.tv/npm/',
            'engine_repository': 'git.subcult.tv/subculture-collective/archive-engine',
            'engine_digest': 'git.subcult.tv/subculture-collective/archive-engine@sha256:' + 'a' * 64,
            'version': '0.1.0', 'source_revision': 'a' * 40,
            'package_sha256': hashlib.sha256(self.artifact.read_bytes()).hexdigest(),
            'migrations': {'0001.sql': 'a' * 64},
        }
        self.release = {**self.current, 'version': '0.1.1', 'source_revision': 'b' * 40,
                        'engine_digest': self.current['engine_repository'] + '@sha256:' + 'b' * 64}
        (self.root / 'release.json').write_text(json.dumps(self.current))
        (self.root / 'package.json').write_text(json.dumps({'dependencies': {self.current['package']: '0.1.0'}}))
        (self.root / 'pnpm-lock.yaml').write_text('old lock\n')
        (self.root / 'engine.env').write_text('ARCHIVE_ENGINE_IMAGE=' + self.current['engine_digest'] + '\n')
        for cmd in [['git', 'init', '-q'], ['git', 'add', '.'], ['git', '-c', 'user.name=Test', '-c', 'user.email=test@example.com', '-c', 'commit.gpgsign=false', 'commit', '-qm', 'fixture']]:
            subprocess.run(cmd, cwd=self.root, check=True)

    def test_incompatibilities_fail_before_mutation(self):
        for key, value in [('configuration_version', 2), ('api_versions', ['2']),
                           ('media', ['audio']), ('version', '1.0.0'),
                           ('engine_digest', self.current['engine_repository'] + ':latest'),
                           ('migrations', {'0001.sql': 'different'})]:
            release = {**self.release, key: value}
            with self.subTest(key=key), self.assertRaises(ValueError):
                prepare(self.root, self.consumer, release, self.artifact)
        self.assertEqual(subprocess.check_output(['git', 'status', '--porcelain'], cwd=self.root), b'')

    def test_corrupt_package_rejected(self):
        self.artifact.write_bytes(b'corrupt')
        with self.assertRaisesRegex(ValueError, 'checksum'):
            validate(self.release, self.consumer, self.current, self.artifact)

    def test_lock_and_checks_then_repeat_noop(self):
        calls = []
        def run(cmd, **kwargs):
            calls.append(cmd)
            if cmd[0] == 'pnpm':
                (self.root / 'pnpm-lock.yaml').write_text('updated lock\n')
        self.assertTrue(prepare(self.root, self.consumer, self.release, self.artifact, run))
        self.assertEqual(len(calls), 2)
        subprocess.run(['git', 'add', '.'], cwd=self.root, check=True)
        subprocess.run(['git', '-c', 'user.name=Test', '-c', 'user.email=test@example.com', '-c', 'commit.gpgsign=false', 'commit', '-qm', 'upgrade'], cwd=self.root, check=True)
        self.assertFalse(prepare(self.root, self.consumer, self.release, self.artifact, run))
        self.assertEqual(len(calls), 2)

    def test_failed_check_restores_pins_and_lock(self):
        originals = {p.name: p.read_bytes() for p in self.root.iterdir() if p.is_file()}
        def run(cmd, **kwargs):
            (self.root / 'pnpm-lock.yaml').write_text('changed')
            if cmd[0] == 'consumer-check':
                raise subprocess.CalledProcessError(1, cmd)
        with self.assertRaises(subprocess.CalledProcessError):
            prepare(self.root, self.consumer, self.release, self.artifact, run)
        self.assertEqual(originals, {p.name: p.read_bytes() for p in self.root.iterdir() if p.is_file()})

    def test_immutable_version_and_pending_adoption_rejected(self):
        with self.assertRaisesRegex(ValueError, 'reused'):
            validate({**self.release, 'version': '0.1.0'}, self.consumer, self.current, self.artifact)
        with self.assertRaisesRegex(ValueError, 'adoption'):
            validate(self.release, {**self.consumer, 'adoption': 'pending'}, self.current, self.artifact)

    def test_component_upgrade_preserves_owned_backend_and_rolls_back_failure(self):
        self.consumer['mode'] = 'components'
        del self.consumer['engine_file']
        backend = self.root / 'engine.env'
        before = backend.read_bytes()
        self.assertTrue(prepare(self.root, self.consumer, self.release, self.artifact, lambda *args, **kwargs: None))
        self.assertEqual(backend.read_bytes(), before)
        subprocess.run(['git', 'add', '.'], cwd=self.root, check=True)
        subprocess.run(['git', '-c', 'user.name=Test', '-c', 'user.email=test@example.com', '-c', 'commit.gpgsign=false', 'commit', '-qm', 'component upgrade'], cwd=self.root, check=True)
        originals = {p.name: p.read_bytes() for p in self.root.iterdir() if p.is_file()}
        def failing(cmd, **kwargs):
            (self.root / 'pnpm-lock.yaml').write_text('partial update')
            raise subprocess.CalledProcessError(1, cmd)
        with self.assertRaises(subprocess.CalledProcessError):
            prepare(self.root, self.consumer, {**self.release, 'version': '0.1.2'}, self.artifact, failing)
        self.assertEqual(originals, {p.name: p.read_bytes() for p in self.root.iterdir() if p.is_file()})

    def test_component_mode_cannot_change_engine_ownership(self):
        with self.assertRaisesRegex(ValueError, 'engine pin'):
            validate(self.release, {**self.consumer, 'mode': 'components'}, self.current, self.artifact)
        with self.assertRaisesRegex(ValueError, 'mode'):
            validate(self.release, {**self.consumer, 'mode': 'unknown'}, self.current, self.artifact)

    def test_existing_proposal_reused_without_mutation(self):
        self.consumer['repository'] = 'subculture-collective/example'
        branch = 'archive-upgrade/v0.1.1-' + 'b' * 40
        def request(path, method='GET', payload=None):
            self.assertEqual(method, 'GET')
            if path == '/user':
                return {'login': 'subcult-agent'}
            if '/pulls?' in path:
                return [{'head': {'ref': branch}, 'base': {'ref': 'main'},
                         'state': 'open', 'html_url': 'https://git.subcult.tv/example/pulls/1'}]
            if '/contents/' in path:
                return {'content': base64.b64encode(json.dumps(self.release).encode()).decode()}
            return {'default_branch': 'main'}
        for _ in range(2):
            self.assertEqual(proposal(self.release, self.consumer, self.root, self.artifact, request),
                             'https://git.subcult.tv/example/pulls/1')
        self.assertEqual(subprocess.check_output(['git', 'status', '--porcelain'], cwd=self.root), b'')

    def test_timeout_after_creation_reconciles_exact_proposal_without_second_post(self):
        self.consumer['repository'] = 'subculture-collective/example'
        branch = 'archive-upgrade/v0.1.1-' + 'b' * 40
        for outcome in ['matching', 'missing', 'different-receipt', 'different-base', 'declined']:
            calls = []
            def request(path, method='GET', payload=None):
                calls.append((path, method))
                if path == '/user':
                    return {'login': 'subcult-agent'}
                if method == 'POST':
                    raise TimeoutError('response lost after request')
                if '/pulls?' in path:
                    if not any(method == 'POST' for _, method in calls) or outcome == 'missing':
                        return []
                    return [{'head': {'ref': branch}, 'base': {'ref': 'other' if outcome == 'different-base' else 'main'},
                             'state': 'closed' if outcome == 'declined' else 'open', 'merged': False,
                             'html_url': 'https://git.subcult.tv/example/pulls/1'}]
                if '/contents/' in path:
                    value = self.current if outcome == 'different-receipt' else self.release
                    return {'content': base64.b64encode(json.dumps(value).encode()).decode()}
                return {'default_branch': 'main'}
            with self.subTest(outcome=outcome), mock.patch('propose_consumer_upgrade.prepare'), mock.patch('propose_consumer_upgrade.subprocess.run'):
                if outcome == 'matching':
                    self.assertEqual(proposal(self.release, self.consumer, self.root, self.artifact, request),
                                     'https://git.subcult.tv/example/pulls/1')
                else:
                    with self.assertRaises(TimeoutError if outcome == 'missing' else ValueError):
                        proposal(self.release, self.consumer, self.root, self.artifact, request)
            self.assertEqual(sum(method == 'POST' for _, method in calls), 1)

    def test_incompatible_release_cannot_create_proposal(self):
        self.consumer['repository'] = 'subculture-collective/example'
        def request(path, method='GET', payload=None):
            self.assertEqual(method, 'GET')
            return {'login': 'subcult-agent'} if path == '/user' else {'default_branch': 'main'}
        with self.assertRaisesRegex(ValueError, 'configuration'):
            proposal({**self.release, 'configuration_version': 2}, self.consumer, self.root, self.artifact, request)

    def test_stacked_proposal_reuse_requires_matching_reviewed_base(self):
        self.consumer['repository'] = 'subculture-collective/example'
        branch = 'archive-upgrade/v0.1.1-' + 'b' * 40
        def request(path, method='GET', payload=None):
            self.assertEqual(method, 'GET')
            if path == '/user':
                return {'login': 'subcult-agent'}
            if '/pulls?' in path:
                return [{'head': {'ref': branch}, 'base': {'ref': 'feat/adoption'},
                         'state': 'open', 'html_url': 'https://git.subcult.tv/example/pulls/2'}]
            if '/contents/' in path:
                return {'content': base64.b64encode(json.dumps(self.release).encode()).decode()}
            return {'default_branch': 'main'}
        self.assertEqual(proposal(self.release, self.consumer, self.root, self.artifact,
                                  request, base_ref='feat/adoption'), 'https://git.subcult.tv/example/pulls/2')
        with self.assertRaisesRegex(ValueError, 'different base'):
            proposal(self.release, self.consumer, self.root, self.artifact, request)


if __name__ == '__main__':
    unittest.main()
import base64
