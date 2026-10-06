import copy
import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from process_video import DERIVATIVE_CAP, RESERVE, WORKING_CAP, preflight, process, validate_probes


def probe(video='mpeg4', audio='mp3'):
    return {'format': {'duration': '20'}, 'streams': [
        {'codec_type': 'video', 'codec_name': video, 'pix_fmt': 'yuv420p',
         'width': 512, 'height': 368, 'display_aspect_ratio': '32:23',
         'sample_aspect_ratio': '1:1', 'avg_frame_rate': '30000/1001',
         'duration': '20', 'start_time': '0'},
        {'codec_type': 'audio', 'codec_name': audio, 'duration': '20', 'start_time': '0',
         'sample_rate': '48000', 'channels': 2, 'channel_layout': 'stereo'}]}


class Tools:
    image = 'a' * 64
    oversized = False
    truncated = False
    corrupt = False
    compatible = False
    def __init__(self, root, output, image, cap):
        self.root, self.output, self.cap = root, output, cap

    def run(self, args):
        if args[0] == 'ffprobe':
            result = probe('h264', 'aac') if args[-1].startswith('/output/') or self.compatible else probe()
            if self.truncated and args[-1].startswith('/output/'):
                result['format']['duration'] = '10'
            return json.dumps(result)
        if args[-1] == '/output/partial.mp4':
            (self.output / 'partial.mp4').write_bytes(b'x' * (self.cap + 1) if self.oversized else b'derivative')
            if self.corrupt:
                (self.root / 'original.avi').write_bytes(b'changed original')
        return 'ffmpeg test tool\n'


class ProcessingTests(unittest.TestCase):
    def fixture(self, root):
        content = b'preserved original'
        (root / 'original.avi').write_bytes(content)
        snapshot = json.dumps({'metadata': {'identifier': 'source-item'}, 'files': [
            {'name': 'Flight.avi', 'source': 'original', 'size': str(len(content)),
             'md5': hashlib.md5(content).hexdigest()}], 'unknown': 'retained'}).encode()
        (root / 'snapshot.json').write_bytes(snapshot)
        return {'approved': True, 'approval_notes': 'Owner approved bounded private pilot',
                'policy': 'local_preserve', 'collection': 'private-processing', 'source_file': 'Flight.avi',
                'snapshot_sha256': hashlib.sha256(snapshot).hexdigest(), 'snapshot_path': 'snapshot.json',
                'source_item': 'source-item', 'original_sha256': hashlib.sha256(content).hexdigest(),
                'original_path': 'original.avi', 'original_bytes': len(content),
                'derivative_cap': 100, 'working_cap': WORKING_CAP, 'reserve': RESERVE,
                'public': False, 'published': False, 'playback': False, 'channel': False}

    def test_success_retains_binding_without_access_approval(self):
        with tempfile.TemporaryDirectory() as name, patch('process_video.shutil.disk_usage') as free:
            free.return_value.free = 100 << 30
            root = Path(name); manifest = self.fixture(root)
            receipt = process(root, manifest, root / 'new', 'image', Tools)
            self.assertEqual(receipt['original_sha256'], manifest['original_sha256'])
            self.assertEqual(receipt['derivative']['sha256'], hashlib.sha256(b'derivative').hexdigest())
            self.assertEqual(receipt['operation'], 'transcode')
            self.assertEqual(receipt['independent_recovery'], 'pending')
            for permission in ('public', 'playback', 'channel'):
                self.assertFalse(receipt[permission])
            self.assertTrue((root / 'new/receipt.json').is_file())
            self.assertFalse((root / 'new/video.mp4').stat().st_mode & 0o222)
            with self.assertRaisesRegex(ValueError, 'existing output'):
                process(root, manifest, root / 'new', 'image', Tools)

    def test_policy_and_integrity_fail_before_output_allocation(self):
        changes = [{'approved': False}, {'collection': 'democracy_now_vid'}, {'policy': 'stream_only'},
                   {'public': True}, {'channel': True}, {'playback': True}, {'original_sha256': 'b' * 64},
                   {'snapshot_sha256': 'b' * 64}, {'source_file': 'Different.avi'}, {'source_item': 'other'},
                   {'derivative_cap': DERIVATIVE_CAP + 1}, {'reserve': RESERVE - 1},
                   {'original_path': '../outside.avi'}]
        for change in changes:
            with self.subTest(change=change), tempfile.TemporaryDirectory() as name:
                root = Path(name); manifest = self.fixture(root); manifest.update(change)
                with self.assertRaises(ValueError):
                    process(root, manifest, root / 'new', 'image', Tools)
                self.assertFalse((root / 'new').exists())

    def test_compatible_original_is_remuxed_and_source_policy_cannot_be_relabelled(self):
        with tempfile.TemporaryDirectory() as name, patch('process_video.shutil.disk_usage') as free:
            free.return_value.free = 100 << 30
            root = Path(name); manifest = self.fixture(root)
            receipt = process(root, manifest, root / 'remux', 'image', type('CompatibleTools', (Tools,), {'compatible': True}))
            self.assertEqual(receipt['operation'], 'remux')
            self.assertIn('copy', receipt['command'])
            self.assertNotIn('libx264', receipt['command'])
            snapshot = json.loads((root / 'snapshot.json').read_text())
            snapshot['metadata']['collection'] = ['democracy_now_vid']
            (root / 'snapshot.json').write_text(json.dumps(snapshot))
            manifest['snapshot_sha256'] = hashlib.sha256((root / 'snapshot.json').read_bytes()).hexdigest()
            with self.assertRaisesRegex(ValueError, 'Democracy Now'):
                process(root, manifest, root / 'blocked', 'image', Tools)
            self.assertFalse((root / 'blocked').exists())

    def test_capacity_accounts_for_failed_output_and_reserve(self):
        with tempfile.TemporaryDirectory() as name, patch('process_video.shutil.disk_usage') as free:
            root = Path(name); manifest = self.fixture(root)
            free.return_value.free = RESERVE
            with self.assertRaisesRegex(ValueError, 'reserve'):
                preflight(root, manifest, root / 'new')
            free.return_value.free = 100 << 30
            manifest['working_cap'] = 100
            (root / 'old-partial').write_bytes(b'x' * 100)
            with self.assertRaisesRegex(ValueError, 'working allowance'):
                preflight(root, manifest, root / 'new')

    def test_failed_conversion_never_writes_success_receipt(self):
        for flag in ('oversized', 'truncated', 'corrupt'):
            with self.subTest(flag=flag), tempfile.TemporaryDirectory() as name, patch('process_video.shutil.disk_usage') as free:
                free.return_value.free = 100 << 30
                root = Path(name); manifest = self.fixture(root)
                tools = type('FailingTools', (Tools,), {flag: True})
                with self.assertRaises(ValueError):
                    process(root, manifest, root / 'new', 'image', tools)
                self.assertTrue((root / 'new/request.json').is_file())
                self.assertTrue((root / 'new/partial.mp4').is_file())
                self.assertFalse((root / 'new/receipt.json').exists())
                self.assertFalse((root / 'new/video.mp4').exists())

    def test_proportions_timing_and_unpreserved_captions_are_rejected(self):
        source, derived = probe(), probe('h264', 'aac')
        validate_probes(source, derived)
        for key, value in [('display_aspect_ratio', '16:9'), ('avg_frame_rate', '25/1'), ('start_time', '2')]:
            changed = copy.deepcopy(derived); changed['streams'][0][key] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                validate_probes(source, changed)
        changed = copy.deepcopy(source); changed['streams'].append({'codec_type': 'subtitle'})
        with self.assertRaisesRegex(ValueError, 'caption'):
            validate_probes(changed, derived)


if __name__ == '__main__':
    unittest.main()
