#!/usr/bin/env python3
"""Complete the owned metadata queue in bounded calls, honoring persisted backoff."""
import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import subprocess
import time
import urllib.request
from pilot import ROOT, REPO, environment, settings


def status():
    with urllib.request.urlopen(f"http://127.0.0.1:{settings()['port']}/api/status", timeout=10) as response:
        return json.load(response)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--hours', type=float, default=6)
    parser.add_argument('--resume', action='store_true', help='resume an already reconciled discovery queue')
    args = parser.parse_args()
    if not 0 < args.hours <= 12: raise RuntimeError('Inventory run must be bounded to 12 hours or less')
    deadline = time.monotonic() + args.hours * 3600
    evidence = ROOT / 'evidence/inventory-progress.json'
    reconciled = args.resume
    final_reconciliation_started = False
    if reconciled and not status()['worker']['discovery_complete']:
        raise RuntimeError('Resume requires a complete identifier discovery checkpoint')
    while time.monotonic() < deadline:
        current = status()
        current['observed_at'] = datetime.now(timezone.utc).isoformat()
        temporary = evidence.with_suffix('.tmp'); temporary.write_text(json.dumps(current, indent=2) + '\n'); temporary.chmod(0o600); temporary.replace(evidence)
        print(json.dumps(current), flush=True)
        worker = current['worker']; counts = current['items']
        retry = worker.get('retry_at')
        if retry:
            wait = (datetime.fromisoformat(retry) - datetime.now(timezone.utc)).total_seconds()
            if wait > 0: time.sleep(min(60, wait, max(0, deadline-time.monotonic()))); continue
        if reconciled and worker['discovery_complete'] and not counts.get('pending') and not counts.get('retry'):
            if not final_reconciliation_started:
                # The source can add/backfill items during a long inventory.
                # Repeat discovery once before declaring point-in-time coverage.
                final_reconciliation_started = True
                result = subprocess.run([str(ROOT / 'bin/vhness'), 'refresh', '--pages', '100', '--items', '0', '--timeout', '10m'], env=environment(), cwd=REPO / 'backend')
                if result.returncode:
                    time.sleep(min(30, max(0, deadline-time.monotonic())))
                    continue
                continue
            total = sum(counts.values())
            if total != worker['source_count']: raise RuntimeError('Coverage count differs from source discovery; operator reconciliation needed')
            subprocess.run(['python3', str(REPO / 'scripts/recovery.py'), '--host', 'nuc', '--destination', '/home/onnwee/Backups/vhness-private-pilot', '--media', str(ROOT / 'media'), '--media-budget', str(512 << 20)], check=True)
            current['completed_at'] = datetime.now(timezone.utc).isoformat()
            (ROOT / 'evidence/inventory-complete.json').write_text(json.dumps(current, indent=2) + '\n')
            print('All discovered identities accounted for; complete metadata backup/restore passed. Rejected/unavailable items remain explicit.', flush=True)
            return
        # Reconciliation also retains complete search responses on this verified implementation.
        command = 'refresh' if not reconciled else 'discover'
        needs_discovery = not reconciled or not worker['discovery_complete']
        result = subprocess.run([str(ROOT / 'bin/vhness'), command, '--pages', '100' if needs_discovery else '0', '--items', '0' if needs_discovery else '1000', '--timeout', '10m'], env=environment(), cwd=REPO / 'backend', stdout=subprocess.PIPE, text=True)
        if result.stdout: print(result.stdout.strip(), flush=True)
        if command == 'refresh' and result.returncode == 0: reconciled = True
        if result.returncode: time.sleep(min(30, max(0, deadline-time.monotonic())))
        elif command == 'discover':
            summary = json.loads(result.stdout)
            if not any(summary.get(key, 0) for key in ['pages', 'imported', 'failed']):
                # Network retries have per-item deadlines even without collection
                # backoff. Avoid a hot loop while those retained deadlines wait.
                time.sleep(min(30, max(0, deadline-time.monotonic())))
    raise RuntimeError('Bounded inventory duration exhausted; all progress retained for explicit resume')

if __name__ == '__main__': main()
