import io
import json
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import unittest


class ReleaseManifestTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='archive-release-test-')
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        for path in ['scripts', 'docs/reference', 'packages/viewer/dist', 'migrations']:
            (self.root / path).mkdir(parents=True)
        source = Path(__file__).with_name('release_manifest.py')
        shutil.copy2(source, self.root / 'scripts/release_manifest.py')
        self.contract = {'package': '@subcult/archive-viewer', 'engine_repository': 'git.subcult.tv/subculture-collective/archive-engine', 'api_versions': ['1']}
        (self.root / 'docs/reference/release-contract.json').write_text(json.dumps(self.contract))
        (self.root / 'packages/viewer/package.json').write_text(json.dumps({'name': self.contract['package'], 'version': '0.1.0', 'license': 'GPL-3.0-only'}))
        (self.root / 'packages/viewer/dist/index.js').write_text('export const version = "0.1.0";\n')
        (self.root / 'migrations/0001.sql').write_text('SELECT 1;\n')
        (self.root / '.gitignore').write_text('.artifacts/\n')
        self.git('init', '-q')
        self.git('add', '.')
        self.git('-c', 'user.name=Release Test', '-c', 'user.email=release-test@example.invalid', 'commit', '-qm', 'fixture')
        self.artifacts = self.root / '.artifacts'
        self.artifacts.mkdir()
        self.package = self.artifacts / 'viewer.tgz'
        self.pack()

    def git(self, *args):
        return subprocess.check_output(['git', *args], cwd=self.root, text=True).strip()

    def pack(self, changed=False):
        with tarfile.open(self.package, 'w:gz') as archive:
            for file in ['package.json', 'dist/index.js']:
                data = (self.root / 'packages/viewer' / file).read_bytes()
                if changed and file.endswith('.js'):
                    data += b'// substituted artifact\n'
                info = tarfile.TarInfo('package/' + file)
                info.size = len(data)
                archive.addfile(info, io.BytesIO(data))

    def run_manifest(self, digest=None):
        return subprocess.run(['python3', 'scripts/release_manifest.py', '--package', str(self.package), '--engine-digest', digest or self.contract['engine_repository'] + '@sha256:' + 'a' * 64, '--output', str(self.artifacts / 'release.json')], cwd=self.root, capture_output=True, text=True)

    def test_binds_clean_source_and_migration_hashes(self):
        result = self.run_manifest()
        self.assertEqual(result.returncode, 0, result.stderr)
        receipt = json.loads((self.artifacts / 'release.json').read_text())
        self.assertEqual(receipt['source_revision'], self.git('rev-parse', 'HEAD'))
        self.assertEqual(len(receipt['migrations']['0001.sql']), 64)
        self.assertNotEqual(self.run_manifest().returncode, 0, 'must not replace an existing release receipt')

    def test_rejects_mutable_tag(self):
        result = self.run_manifest(self.contract['engine_repository'] + ':latest')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('immutable registry engine digest', result.stderr)

    def test_rejects_dirty_source(self):
        (self.root / 'migrations/0001.sql').write_text('SELECT 2;\n')
        result = self.run_manifest()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('clean committed source', result.stderr)

    def test_rejects_substituted_package_bytes(self):
        self.pack(changed=True)
        result = self.run_manifest()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Packed bytes differ', result.stderr)

    def test_accepts_packer_metadata_formatting_without_accepting_changes(self):
        with tarfile.open(self.package, 'w:gz') as archive:
            data = json.dumps(json.loads((self.root / 'packages/viewer/package.json').read_text()), indent=2).encode()
            info = tarfile.TarInfo('package/package.json')
            info.size = len(data)
            archive.addfile(info, io.BytesIO(data))
            archive.add(self.root / 'packages/viewer/dist/index.js', arcname='package/dist/index.js')
        result = self.run_manifest()
        self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == '__main__':
    unittest.main()
