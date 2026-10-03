#!/usr/bin/env python3
"""Qualify a tiny generated local original without acquiring archive video."""
import hashlib
import json
from pathlib import Path
import subprocess
from pilot import ROOT, cli

COLLECTION = 'vhness-local-fixture'

def write(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n'); path.chmod(0o600)

def main():
    directory = ROOT / 'generated-fixture'; directory.mkdir(exist_ok=True, mode=0o700)
    cli(['collection', '--collection', COLLECTION, '--policy', 'local_preserve', '--budget', str(10 << 20), '--reserve', str(64 << 20)])
    for number, frequency in [(1, 440), (2, 660)]:
        identity = f'vhness-generated-{number}'
        original = directory / f'{identity}.mp4'
        if not original.exists():
            subprocess.run(['ffmpeg', '-nostdin', '-hide_banner', '-loglevel', 'error', '-f', 'lavfi', '-i', 'testsrc2=size=320x180:rate=24', '-f', 'lavfi', '-i', f'sine=frequency={frequency}:sample_rate=48000', '-t', '12', '-c:v', 'libx264', '-preset', 'fast', '-crf', '30', '-pix_fmt', 'yuv420p', '-c:a', 'aac', '-b:a', '64k', '-movflags', '+faststart', str(original)], check=True)
        content = original.read_bytes()
        if len(content) > 5 << 20: raise RuntimeError('Generated fixture exceeds sample bound')
        metadata = {'metadata': {'identifier': identity, 'mediatype': 'movies', 'collection': [COLLECTION], 'title': f'Generated local qualification video {number}', 'date': '2026-10-03', 'description': 'Generated color/motion test pattern and sine tone. Synthetic qualification fixture; not a broadcast.'}, 'files': [{'name': 'original.mp4', 'size': str(len(content)), 'length': '12', 'width': '320', 'height': '180', 'md5': hashlib.md5(content).hexdigest(), 'format': 'h.264'}]}
        source = directory / f'{identity}.metadata.json'; write(source, metadata)
        cli(['import', '--id', identity, '--collection', COLLECTION, '--synthetic', '--file', str(source)])
        decision = directory / f'{identity}.review.json'; write(decision, {'acquisition': True, 'published': True, 'playback': True, 'channel': True, 'public': False, 'file': 'original.mp4', 'notes': 'Generated fixture only; 10 MiB local budget, 64 MiB reserve, independent NUC sample backup. No real collection activation.'})
        cli(['review', '--id', identity, '--file', str(decision)])
        manifest = directory / f'{identity}.manifest.json'; write(manifest, {'collection': COLLECTION, 'id': identity, 'snapshot_sha256': hashlib.sha256(source.read_bytes()).hexdigest(), 'file': 'original.mp4', 'bytes': len(content), 'sha256': hashlib.sha256(content).hexdigest(), 'notes': 'Exact generated original selected for bounded adapter qualification'})
        cli(['acquire', '--file', str(manifest), '--fixture-original', str(original), '--timeout', '1m'])
    print('Generated local originals verified and bound; no Democracy Now! video acquired')

if __name__ == '__main__': main()
