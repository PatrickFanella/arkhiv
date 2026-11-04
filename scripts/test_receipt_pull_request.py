import unittest
import urllib.error

from receipt_pull_request import land


class ReceiptPullRequestTests(unittest.TestCase):
    def test_creates_then_reuses_and_schedules_merge(self):
        calls = []
        created = []
        def request(path, method='GET', payload=None):
            calls.append((path, method, payload))
            if path == '/user':
                return {'login': 'subcult-agent'}
            if '/pulls?' in path:
                return list(created)
            if method == 'POST' and path.endswith('/pulls'):
                created.append({'number': 5, 'html_url': 'https://git.subcult.tv/subculture-collective/arkhiv/pulls/5', 'head': {'ref': payload['head']}, 'base': {'ref': payload['base']}})
                return created[-1]
            if method == 'POST' and path.endswith('/merge'):
                return None
            raise AssertionError(path)
        for _ in range(2):
            self.assertEqual(land('release/viewer-0.4.5', 'Record viewer 0.4.5', 'body', request), 'https://git.subcult.tv/subculture-collective/arkhiv/pulls/5')
        self.assertEqual(sum(1 for path, method, _ in calls if method == 'POST' and path.endswith('/pulls')), 1)
        merges = [payload for path, method, payload in calls if method == 'POST' and path.endswith('/merge')]
        self.assertEqual(merges, [{'do': 'merge', 'merge_when_checks_succeed': True, 'delete_branch_after_merge': True}] * 2)

    def test_unmergeable_answer_leaves_the_pull_request_for_review(self):
        def request(path, method='GET', payload=None):
            if path == '/user':
                return {'login': 'subcult-agent'}
            if '/pulls?' in path:
                return [{'number': 6, 'html_url': 'https://git.subcult.tv/subculture-collective/arkhiv/pulls/6', 'head': {'ref': 'release/core-0.4.5'}, 'base': {'ref': 'main'}}]
            if method == 'POST' and path.endswith('/merge'):
                raise urllib.error.HTTPError(path, 405, 'not mergeable', {}, None)
            raise AssertionError(path)
        self.assertEqual(land('release/core-0.4.5', 't', 'b', request), 'https://git.subcult.tv/subculture-collective/arkhiv/pulls/6')

    def test_rejects_other_identity(self):
        with self.assertRaises(ValueError):
            land('release/x', 't', 'b', lambda path, method='GET', payload=None: {'login': 'someone'})


if __name__ == '__main__':
    unittest.main()
