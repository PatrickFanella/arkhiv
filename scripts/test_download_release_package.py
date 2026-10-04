import hashlib
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from download_release_package import download, RegistryRedirect, REGISTRY


class DownloadTests(unittest.TestCase):
    def test_cross_registry_redirect_is_rejected_before_forwarding_credentials(self):
        with self.assertRaisesRegex(ValueError, 'outside'):
            RegistryRedirect().redirect_request(None, None, 302, '', {}, 'https://elsewhere.example/package.tgz')

    def test_integrity_mismatch_does_not_write_package(self):
        class Response:
            url = REGISTRY
            def __init__(self, data): self.data = data
            def __enter__(self): return self
            def __exit__(self, *args): pass
            def read(self, size): return self.data[:size]
        metadata = {'versions': {'0.1.1': {'name': '@subcult/archive-viewer', 'version': '0.1.1', 'dist': {'tarball': REGISTRY + 'package.tgz'}}}}
        release = {'package': '@subcult/archive-viewer', 'version': '0.1.1', 'registry': REGISTRY,
                   'package_sha256': hashlib.sha256(b'qualified bytes').hexdigest()}
        with tempfile.TemporaryDirectory() as directory, patch.dict(os.environ, {'ARCHIVE_PACKAGE_READ_TOKEN': 'test-only'}):
            output = Path(directory) / 'package.tgz'
            with patch('urllib.request.OpenerDirector.open', side_effect=[Response(json.dumps(metadata).encode()), Response(b'corrupt')]):
                with self.assertRaisesRegex(ValueError, 'checksum'):
                    download(release, output)
            self.assertFalse(output.exists())


if __name__ == '__main__':
    unittest.main()
