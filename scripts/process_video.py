"""Process one explicitly approved preserved original; never grant playback access."""
import argparse
from fractions import Fraction
import hashlib
import json
import math
from pathlib import Path
import re
import resource
import shutil
import subprocess
import time
import uuid


DERIVATIVE_CAP = 384 << 20
WORKING_CAP = 768 << 20
RESERVE = 16 << 30
REPORT_ALLOWANCE = 16 << 20


def digest(path):
    value = hashlib.sha256()
    with path.open('rb') as stream:
        for block in iter(lambda: stream.read(8 << 20), b''):
            value.update(block)
    return value.hexdigest()


def local_file(root, relative):
    path = Path(relative)
    if path.is_absolute() or '..' in path.parts or not path.parts:
        raise ValueError('file must be relative to the working store')
    current = root
    for part in path.parts:
        current = current / part
        if current.is_symlink():
            raise ValueError('symlinks are not permitted in the working store')
    if not current.is_file():
        raise ValueError('expected a regular preserved file')
    return current


def store_bytes(root):
    total = 0
    for path in root.rglob('*'):
        if path.is_symlink():
            raise ValueError('symlinks are not permitted in the working store')
        if path.is_file():
            total += path.stat().st_size
    return total


def preflight(root, manifest, output):
    if len(json.dumps(manifest).encode()) > 64 << 10:
        raise ValueError('processing manifest exceeds 64 KiB')
    if manifest.get('approved') is not True or not manifest.get('approval_notes'):
        raise ValueError('explicit acquisition and processing approval required')
    if manifest.get('policy') != 'local_preserve' or manifest.get('collection') == 'democracy_now_vid':
        raise ValueError('processing requires a different local-preservation collection')
    if any(manifest.get(key) is not False for key in ('public', 'published', 'playback', 'channel')):
        raise ValueError('processing does not grant viewing or publication permission')
    if not manifest.get('collection') or not manifest.get('source_file', '').lower().endswith('.avi'):
        raise ValueError('an exact AVI source selection is required')
    for key in ('snapshot_sha256', 'original_sha256'):
        if not re.fullmatch('[0-9a-f]{64}', manifest.get(key, '')):
            raise ValueError('exact source snapshot and original SHA-256 required')
    for key, maximum in [('derivative_cap', DERIVATIVE_CAP), ('working_cap', WORKING_CAP)]:
        value = manifest.get(key)
        if type(value) is not int or not 0 < value <= maximum:
            raise ValueError('processing allowance exceeds bounded private pilot')
    if type(manifest.get('reserve')) is not int or manifest['reserve'] < RESERVE:
        raise ValueError('at least 16 GiB free-space reserve required')
    if type(manifest.get('original_bytes')) is not int or manifest['original_bytes'] <= 0:
        raise ValueError('positive exact original byte count required')
    original = local_file(root, manifest['original_path'])
    if original.stat().st_size != manifest['original_bytes'] or digest(original) != manifest['original_sha256']:
        raise ValueError('preserved original differs from approved manifest')
    snapshot = local_file(root, manifest['snapshot_path'])
    if snapshot.stat().st_size > 8 << 20 or digest(snapshot) != manifest['snapshot_sha256']:
        raise ValueError('complete source snapshot differs from approved manifest')
    source = json.loads(snapshot.read_text())
    collections = source.get('metadata', {}).get('collection', [])
    if isinstance(collections, str):
        collections = [collections]
    if 'democracy_now_vid' in collections:
        raise ValueError('Democracy Now source media cannot be processed')
    if source.get('metadata', {}).get('identifier') != manifest.get('source_item'):
        raise ValueError('source item differs from approved manifest')
    matches = [row for row in source.get('files', []) if row.get('name') == manifest['source_file']]
    if len(matches) != 1 or matches[0].get('source') != 'original' or matches[0].get('private'):
        raise ValueError('exact non-private original source entry required')
    selected = matches[0]
    if int(selected.get('size', 0)) != manifest['original_bytes']:
        raise ValueError('source byte count differs from preserved original')
    pinned = False
    for algorithm, length in [('md5', 32), ('sha1', 40)]:
        expected = selected.get(algorithm, '').lower()
        if not expected:
            continue
        if not re.fullmatch(f'[0-9a-f]{{{length}}}', expected):
            raise ValueError('invalid original source checksum')
        value = hashlib.new(algorithm)
        with original.open('rb') as stream:
            for block in iter(lambda: stream.read(8 << 20), b''):
                value.update(block)
        if value.hexdigest() != expected:
            raise ValueError('source checksum differs from preserved original')
        pinned = True
    if not pinned:
        raise ValueError('original source checksum required')
    if output.exists() or output.is_symlink():
        raise ValueError('preserve existing output, including failed or partial output')
    if output.parent != root or root.is_symlink():
        raise ValueError('output must be a new directory immediately inside the working store')
    if store_bytes(root) + manifest['derivative_cap'] + REPORT_ALLOWANCE > manifest['working_cap']:
        raise ValueError('insufficient working allowance, including retained partials')
    if shutil.disk_usage(root).free < manifest['reserve'] + manifest['derivative_cap'] + REPORT_ALLOWANCE:
        raise ValueError('free-space reserve would be exceeded')
    return original


def streams(probe):
    video = [row for row in probe['streams'] if row['codec_type'] == 'video']
    audio = [row for row in probe['streams'] if row['codec_type'] == 'audio']
    if len(video) != 1 or len(audio) != 1:
        raise ValueError('exactly one video and one audio stream required')
    if any(row['codec_type'] not in ('video', 'audio') for row in probe['streams']):
        raise ValueError('additional streams require a separate caption or stream preservation plan')
    return video[0], audio[0]


def seconds(value):
    result = float(value)
    if not math.isfinite(result):
        raise ValueError('finite timing measurements required')
    return result


def validate_probes(source, derived):
    before, before_audio = streams(source)
    after, after_audio = streams(derived)
    if after['codec_name'] != 'h264' or after_audio['codec_name'] != 'aac' or after.get('pix_fmt') != 'yuv420p':
        raise ValueError('derivative must be H.264 yuv420p and AAC')
    if (before['width'], before['height']) != (after['width'], after['height']):
        raise ValueError('pixel dimensions changed')
    for key in ('sample_aspect_ratio', 'display_aspect_ratio'):
        if Fraction(before[key].replace(':', '/')) != Fraction(after[key].replace(':', '/')):
            raise ValueError('display proportions changed')
    if abs(float(Fraction(before['avg_frame_rate'])) - float(Fraction(after['avg_frame_rate']))) > .001:
        raise ValueError('frame rate changed')
    if (before_audio['sample_rate'], before_audio['channels']) != (after_audio['sample_rate'], after_audio['channels']):
        raise ValueError('audio sample rate or channel layout changed')
    if before_audio.get('channel_layout') != after_audio.get('channel_layout'):
        raise ValueError('audio channel layout changed')
    for left, right in [(source['format']['duration'], derived['format']['duration']),
                        (before['duration'], after['duration']),
                        (before_audio['duration'], after_audio['duration'])]:
        if seconds(left) <= 0 or abs(seconds(left) - seconds(right)) > .1:
            raise ValueError('duration differs from original')
    before_offset = seconds(before['start_time']) - seconds(before_audio['start_time'])
    after_offset = seconds(after['start_time']) - seconds(after_audio['start_time'])
    if abs(before_offset - after_offset) > .1:
        raise ValueError('audio/video start timing changed')


class ContainerTools:
    def __init__(self, root, output, image, cap):
        self.root, self.output, self.cap = root, output, cap
        self.sequence = 0
        self.image = subprocess.check_output(['podman', 'image', 'inspect', image, '--format', '{{.Id}}'], text=True).strip()
        if not re.fullmatch('(sha256:)?[0-9a-f]{64}', self.image):
            raise ValueError('immutable local tool image identity required')

    def run(self, args):
        name = 'arkhiv-processing-' + uuid.uuid4().hex
        command = ['podman', 'run', '--rm', '--name', name,
                   '--label', 'arkhiv.qualification=video-processing', '--network', 'none', '--read-only',
                   '--memory', '1g', '--cpus', '2', '--pids-limit', '128', '--timeout', '3600',
                   '--ulimit', f'fsize={self.cap}:{self.cap}',
                   '-v', f'{self.root}:/input:ro', '-v', f'{self.output}:/output:rw', self.image, *args]
        self.sequence += 1
        if self.sequence > 6:
            raise ValueError('tool invocation allowance exceeded')
        # Host-side stdout/stderr each have a hard 1 MiB file limit. The container
        # receives its separate derivative limit; no captured pipe can grow in RAM.
        def limit_reports():
            resource.setrlimit(resource.RLIMIT_FSIZE, (1 << 20, resource.getrlimit(resource.RLIMIT_FSIZE)[1]))
        stdout = self.output / f'tool-{self.sequence}.stdout'
        stderr = self.output / f'tool-{self.sequence}.stderr'
        with stdout.open('xb') as report, stderr.open('xb') as log:
            try:
                subprocess.run(command, stdout=report, stderr=log, check=True,
                               timeout=3660, preexec_fn=limit_reports)
            except BaseException:
                # A host-side timeout or log limit must not leave a writer running.
                # Only this invocation's unique owned container may be removed.
                subprocess.run(['podman', 'rm', '--force', '--time', '5', name],
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                               check=False, timeout=30)
                raise
        return stdout.read_text()


def process(root, manifest, output, image, tool_factory=ContainerTools):
    root, output = Path(root).absolute(), Path(output).absolute()
    original = preflight(root, manifest, output)
    output.mkdir(mode=0o700)
    # This marker is retained on every failure. A successful receipt is written last.
    (output / 'request.json').write_text(json.dumps(manifest, indent=2) + '\n')
    tools = tool_factory(root, output, image, manifest['derivative_cap'])
    source_path = '/input/' + str(original.relative_to(root))
    def probe(path):
        return json.loads(tools.run(['ffprobe', '-v', 'error', '-show_streams', '-show_format', '-of', 'json', path]))
    started = time.monotonic()
    source = probe(source_path)
    video, audio = streams(source)
    remux = video['codec_name'] == 'h264' and video.get('pix_fmt') == 'yuv420p' and audio['codec_name'] == 'aac'
    encoding = ['-c', 'copy'] if remux else ['-c:v', 'libx264', '-preset', 'medium', '-crf', '21',
               '-pix_fmt', 'yuv420p', '-threads', '2', '-c:a', 'aac', '-b:a', '128k', '-ar', str(audio['sample_rate'])]
    command = ['ffmpeg', '-nostdin', '-hide_banner', '-v', 'warning', '-xerror', '-n', '-i', source_path,
               '-map', '0:v:0', '-map', '0:a:0', '-map_metadata', '-1', *encoding,
               '-movflags', '+faststart', '-fs', str(manifest['derivative_cap']), '/output/partial.mp4']
    tools.run(command)
    partial = output / 'partial.mp4'
    if not partial.is_file() or partial.stat().st_size > manifest['derivative_cap']:
        raise ValueError('derivative cap exceeded')
    derived = probe('/output/partial.mp4')
    validate_probes(source, derived)
    for path in (source_path, '/output/partial.mp4'):
        tools.run(['ffmpeg', '-nostdin', '-v', 'error', '-xerror', '-threads', '2', '-i', path,
                   '-map', '0:v:0', '-map', '0:a:0', '-f', 'null', '-'])
    if digest(original) != manifest['original_sha256'] or original.stat().st_size != manifest['original_bytes']:
        raise ValueError('preserved original changed during processing')
    if digest(local_file(root, manifest['snapshot_path'])) != manifest['snapshot_sha256']:
        raise ValueError('complete source snapshot changed during processing')
    receipt = {'schema': 1, 'manifest': manifest, 'tool_image': tools.image,
               'tool': tools.run(['ffmpeg', '-version']).splitlines()[0], 'command': command,
               'operation': 'remux' if remux else 'transcode', 'original_probe': source,
               'derivative_probe': derived, 'original_sha256': manifest['original_sha256'],
               'derivative': {'path': str((output / 'video.mp4').relative_to(root)),
                              'bytes': partial.stat().st_size, 'sha256': digest(partial)},
               'elapsed_seconds': time.monotonic() - started,
               'full_decode_original': True, 'full_decode_derivative': True,
               'public': False, 'playback': False, 'channel': False,
               'independent_recovery': 'pending', 'browser_and_human_qualification': 'pending'}
    encoded = (json.dumps(receipt, indent=2) + '\n').encode()
    if store_bytes(root) + len(encoded) > manifest['working_cap'] or shutil.disk_usage(root).free < manifest['reserve'] + len(encoded):
        raise ValueError('working cap or free-space reserve exceeded')
    partial.chmod(0o444)
    partial.rename(output / 'video.mp4')
    with (output / 'receipt.json').open('xb') as stream:
        stream.write(encoded)
    (output / 'receipt.json').chmod(0o444)
    return receipt


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', type=Path, required=True)
    parser.add_argument('--manifest', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--image', required=True, help='Already installed immutable processing tools image')
    args = parser.parse_args()
    receipt = process(args.root, json.loads(args.manifest.read_text()), args.output, args.image)
    print(json.dumps({'receipt': str(args.output / 'receipt.json'), 'derivative': receipt['derivative']}))


if __name__ == '__main__':
    main()
