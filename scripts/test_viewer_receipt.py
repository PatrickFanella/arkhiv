import io
import json
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import unittest

from viewer_receipt import receipt


class ViewerReceiptTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='viewer-receipt-')
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        for path in ['packages/viewer', 'docs/releases']:
            (self.root / path).mkdir(parents=True)
        (self.root / 'packages/viewer/package.json').write_text(json.dumps({'name': '@subcult/archive-viewer', 'version': '0.4.5', 'license': 'GPL-3.0-only'}))
        (self.root / 'docs/releases/core-release-0.3.0.json').write_text('{}')
        (self.root / 'docs/releases/viewer-release-0.4.4.json').write_text(json.dumps({'version': '0.4.4', 'compatible_core_receipt': 'core-release-0.3.0.json'}))
        subprocess.run(['git', 'init', '-q'], cwd=self.root, check=True)
        subprocess.run(['git', 'add', '.'], cwd=self.root, check=True)
        subprocess.run(['git', '-c', 'user.name=T', '-c', 'user.email=t@example.invalid', '-c', 'commit.gpgsign=false', 'commit', '-qm', 'fixture'], cwd=self.root, check=True)
        self.package = self.root / 'subcult-archive-viewer-0.4.5.tgz'
        self.pack({'name': '@subcult/archive-viewer', 'version': '0.4.5', 'license': 'GPL-3.0-only'})

    def pack(self, metadata):
        with tarfile.open(self.package, 'w:gz') as archive:
            data = json.dumps(metadata).encode()
            info = tarfile.TarInfo('package/package.json')
            info.size = len(data)
            archive.addfile(info, io.BytesIO(data))

    def test_receipt_binds_published_bytes_and_carries_the_compatible_core(self):
        result = receipt(self.root, self.package)
        self.assertEqual(result['version'], '0.4.5')
        self.assertEqual(result['compatible_core_receipt'], 'core-release-0.3.0.json')
        self.assertEqual(result['package_file'], 'subcult-archive-viewer-0.4.5.tgz')
        self.assertEqual(len(result['package_sha256']), 64)
        self.assertEqual(result['source_revision'], subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=self.root, text=True).strip())

    def test_mismatched_or_unknown_core_is_rejected(self):
        self.pack({'name': '@subcult/archive-viewer', 'version': '0.4.6', 'license': 'GPL-3.0-only'})
        with self.assertRaises(SystemExit):
            receipt(self.root, self.package)
        self.pack({'name': '@subcult/archive-viewer', 'version': '0.4.5', 'license': 'GPL-3.0-only'})
        with self.assertRaises(SystemExit):
            receipt(self.root, self.package, 'core-release-9.9.9.json')

    def test_dirty_source_is_rejected(self):
        (self.root / 'packages/viewer/extra.ts').write_text('export {};')
        with self.assertRaises(SystemExit):
            receipt(self.root, self.package)


if __name__ == '__main__':
    unittest.main()
