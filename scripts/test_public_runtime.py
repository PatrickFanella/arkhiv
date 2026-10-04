"""Exercise public HTTP failures and transient recovery through the operator CLI."""
import http.server
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import threading
import unittest


class PublicMonitorTests(unittest.TestCase):
    def exercise(self, responses):
        class Handler(http.server.BaseHTTPRequestHandler):
            attempts = 0

            def do_GET(self):
                status = responses[min(type(self).attempts, len(responses) - 1)]
                type(self).attempts += 1
                self.send_response(status)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(b'{"audience":"public","total":23}')

            def log_message(self, *_):
                pass

        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            docker = root / 'inspect-owner'
            docker.write_text('#!/usr/bin/env python3\nprint(\'[ {"Config":{"Labels":'
                              '{"com.docker.compose.project":"owned"}}} ]\')\n')
            docker.chmod(0o700)
            server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Handler)
            thread = threading.Thread(target=server.serve_forever, daemon=True)
            thread.start()
            try:
                url = f'http://127.0.0.1:{server.server_port}/api/documents'
                cfg = {'root': str(root), 'state': str(root / 'state'), 'project': 'owned',
                       'database_container': 'owned-db', 'docker': [str(docker)],
                       'sites': [{'url': url, 'total': 23, 'equals': {'audience': 'public'}}]}
                config = root / 'config.json'
                config.write_text(json.dumps(cfg))
                result = subprocess.run([sys.executable, str(Path(__file__).with_name('public_runtime.py')),
                                         'monitor', '--config', str(config)], capture_output=True, text=True)
                files = {p.name: json.loads(p.read_text()) for p in (root / 'state').glob('*.json')}
                return result, files, Handler.attempts, url
            finally:
                server.shutdown()
                server.server_close()
                thread.join()

    def test_transient_service_failure_retries_and_checks_contract(self):
        result, files, attempts, _ = self.exercise([503, 200])
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(attempts, 2)
        self.assertEqual(files['monitor.json']['sites'][0]['total'], 23)
        self.assertNotIn('monitor-failure.json', files)

    def test_access_failure_is_not_hidden_or_marked_success(self):
        result, files, attempts, url = self.exercise([403])
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(attempts, 1)
        self.assertIn(url, result.stderr)
        self.assertIn('HTTP 403', result.stderr)
        self.assertEqual(files['monitor-failure.json']['http_status'], '403')
        self.assertNotIn('monitor.json', files)
