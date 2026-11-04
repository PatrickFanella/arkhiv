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

    def test_metadata_url_keeps_the_scope_prefix(self):
        requested = []
        class Response:
            url = REGISTRY
            def __enter__(self): return self
            def __exit__(self, *args): pass
            def read(self, size): return b'{}'
        def open(self, request, timeout=None):
            requested.append(request.full_url)
            return Response()
        release = {'package': '@subcult/archive-viewer', 'version': '0.1.1', 'registry': REGISTRY, 'package_sha256': '0' * 64}
        with tempfile.TemporaryDirectory() as directory, patch.dict(os.environ, {'ARCHIVE_PACKAGE_READ_TOKEN': 'test-only'}):
            with patch('urllib.request.OpenerDirector.open', open):
                with self.assertRaises(KeyError):
                    download(release, Path(directory) / 'package.tgz')
        self.assertEqual(requested, [REGISTRY + '@subcult%2Farchive-viewer'])


if __name__ == '__main__':
    unittest.main()
