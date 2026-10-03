import hashlib
import json
from pathlib import Path
import subprocess
import tempfile
import unittest


class AdoptionTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name) / 'consumer'
        self.root.mkdir()
        adapter = self.root / 'web/src/engine'
        adapter.mkdir(parents=True)
        (adapter / 'model.ts').write_text('export const request = () => {};\n')
        (adapter / 'manifest.json').write_text(json.dumps({'files': {'model.ts': hashlib.sha256((adapter / 'model.ts').read_bytes()).hexdigest()}, 'revision': 'a' * 40}))
        self.body = 'export default function App() { return <main className="brand-page">Distinct brand</main>; }\n'
        (self.root / 'web/src/App.tsx').write_text("import { request } from './engine/model';\n" + self.body)
        (self.root / 'web/package.json').write_text(json.dumps({'dependencies': {'react': '^19.2.0'}}))
        (self.root / 'scripts').mkdir()
        (self.root / 'scripts/check_engine.py').write_text('tools=json.loads((repo/"docs/engine-tools.json").read_text())\n')
        (self.root / 'docs').mkdir()
        (self.root / 'deploy').mkdir()
        release = {'package': '@subcult/archive-viewer', 'contract_version': 1, 'configuration_version': 1,
                   'api_versions': ['1'], 'media': ['video'], 'version': '0.1.0',
                   'engine_digest': 'git.subcult.tv/subculture-collective/archive-engine@sha256:' + 'b' * 64}
        self.release = Path(self.temp.name) / 'release.json'
        self.release.write_text(json.dumps(release))
        for cmd in [['git', 'init', '-q'], ['git', 'add', '.'], ['git', '-c', 'user.name=Test', '-c', 'user.email=test@example.com', '-c', 'commit.gpgsign=false', 'commit', '-qm', 'fixture']]:
            subprocess.run(cmd, cwd=self.root, check=True)

    def migrate(self):
        return subprocess.run(['python3', str(Path(__file__).with_name('adopt_package.py')),
                               '--consumer', str(self.root), '--release', str(self.release)], capture_output=True, text=True)

    def test_preserves_brand_and_moves_to_package(self):
        r = self.migrate()
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertEqual((self.root / 'web/src/App.tsx').read_text(), "import { request } from '@subcult/archive-viewer';\n" + self.body)
        self.assertFalse((self.root / 'web/src/engine').exists())
        self.assertTrue((self.root / 'docs/legacy-viewer-manifest.json').is_file())
        self.assertTrue((self.root / 'scripts/build_image.py').is_file())
        self.assertEqual(json.loads((self.root / 'web/package.json').read_text())['dependencies']['@subcult/archive-viewer'], '0.1.0')

    def test_preserves_an_edited_generated_adapter(self):
        (self.root / 'web/src/engine/model.ts').write_text('custom change')
        r = self.migrate()
        self.assertNotEqual(r.returncode, 0)
        self.assertIn('clean isolated', r.stderr)
        subprocess.run(['git', 'add', '.'], cwd=self.root, check=True)
        subprocess.run(['git', '-c', 'user.name=Test', '-c', 'user.email=test@example.com', '-c', 'commit.gpgsign=false', 'commit', '-qm', 'custom'], cwd=self.root, check=True)
        r = self.migrate()
        self.assertNotEqual(r.returncode, 0)
        self.assertIn('was edited', r.stderr)
        self.assertTrue((self.root / 'web/src/engine/model.ts').is_file())
        self.assertEqual(subprocess.check_output(['git', 'status', '--porcelain'], cwd=self.root), b'')


if __name__ == '__main__':
    unittest.main()
