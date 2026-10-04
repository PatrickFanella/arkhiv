#!/usr/bin/env python3
"""Create one reviewed upgrade PR per release in an isolated registered consumer clone."""
import argparse
import base64
import json
import os
from pathlib import Path
import subprocess
import tempfile
import urllib.error
import urllib.parse
import urllib.request

from upgrade_consumer import prepare, validate, select_consumer


def api(path, method='GET', payload=None):
    token = os.environ['ARCHIVE_UPGRADE_TOKEN']
    request = urllib.request.Request('https://git.subcult.tv/api/v1' + path,
        method=method, data=json.dumps(payload).encode() if payload is not None else None,
        headers={'Authorization': 'token ' + token, 'Content-Type': 'application/json', 'User-Agent': 'tea'})
    with urllib.request.urlopen(request, timeout=30) as response:
        raw = response.read()
        return json.loads(raw) if raw else None


def existing_proposal(base, target, branch, consumer, release, request):
    pulls = []
    page = 1
    while True:
        batch = request(base + '/pulls?state=all&limit=50&page=' + str(page))
        pulls.extend(batch)
        if len(batch) < 50:
            break
        page += 1
    for pull in pulls:
        if pull['head']['ref'] == branch:
            if pull['base']['ref'] != target:
                raise ValueError('Existing proposal targets a different base')
            if pull['state'] == 'closed' and not pull.get('merged'):
                raise ValueError('Upgrade was declined; do not reopen automatically')
            content = request(base + '/contents/' + consumer['release_file'] + '?ref=' + urllib.parse.quote(branch, safe=''))
            proposed = json.loads(base64.b64decode(content['content']))
            if proposed != release:
                raise ValueError('Existing proposal differs from this release')
            return pull['html_url']
    return None


def proposal(release, consumer, clone, artifact, request=api, base_ref=None):
    if request('/user')['login'] != 'subcult-agent':
        raise ValueError('Upgrade proposals require the subcult-agent identity')
    repository = consumer['repository']
    base = '/repos/' + repository
    metadata = request(base)
    target = base_ref or metadata['default_branch']
    api_only = consumer.get('mode') == 'api'
    branch = ('archive-core-upgrade/v' if api_only else 'archive-upgrade/v') + release['version'] + '-' + release['source_revision']
    # Validate the base consumer before consulting or creating a proposal.
    current = json.loads((clone / consumer['release_file']).read_text())
    validate(release, consumer, current, artifact)
    if current == release:
        return None
    existing = existing_proposal(base, target, branch, consumer, release, request)
    if existing:
        return existing
    subprocess.run(['git', 'switch', '-c', branch], cwd=clone, check=True)
    prepare(clone, consumer, release, artifact)
    files = [consumer[key] for key in (('release_file', 'engine_file') if api_only else ('package_file', 'lock_file', 'release_file'))]
    if consumer.get('mode', 'platform') == 'platform':
        files.append(consumer['engine_file'])
    subprocess.run(['git', 'add', '--', *files], cwd=clone, check=True)
    subprocess.run(['git', '-c', 'user.name=SUBCULT Agent', '-c', 'user.email=info@subcult.tv', '-c', 'commit.gpgsign=false',
                    'commit', '-m', 'Upgrade shared archive platform to ' + release['version']], cwd=clone, check=True)
    subprocess.run(['git', 'push', 'origin', branch], cwd=clone, check=True)
    try:
        result = request(base + '/pulls', 'POST', {
            'base': target, 'head': branch,
            'title': 'Upgrade shared archive platform to ' + release['version'],
            'body': ('Updates only the immutable headless core pin and compatibility receipt. The consumer owns its entire frontend; no frontend package is required.\n\n'
                     if api_only else
                     'Updates the frontend package, lockfile and release provenance receipt; the consumer-owned backend remains unchanged.\n\n'
                     if consumer.get('mode', 'platform') == 'components' else
                     'Updates the frontend lock, immutable engine digest and compatibility receipt together.\n\n') +
                    'Consumer checks passed in the isolated proposal checkout. Hosted checks and private qualification remain required. '
                    'This PR does not deploy the consumer.\n\nSource: `' + release['source_revision'] + '`; ' +
                    ('core digest: `' + release['engine_digest'] + '`.' if api_only else 'package SHA-256: `' + release['package_sha256'] + '`.'),
        })
    except (TimeoutError, ConnectionError, urllib.error.URLError) as error:
        if isinstance(error, urllib.error.HTTPError):
            raise
        # A timed-out POST may have created the PR. Read back its exact receipt
        # before considering it delivered; do not repeat the write blindly.
        existing = existing_proposal(base, target, branch, consumer, release, request)
        if existing:
            return existing
        raise
    return result['html_url']


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--registry', type=Path, required=True)
    parser.add_argument('--consumer', required=True)
    parser.add_argument('--release', type=Path, required=True)
    parser.add_argument('--package-artifact', type=Path)
    parser.add_argument('--base-ref', help='Explicit reviewed adoption branch for a stacked proposal; defaults to the repository default branch')
    args = parser.parse_args()
    registry = json.loads(args.registry.read_text())
    release = json.loads(args.release.read_text())
    consumer = select_consumer(registry, args.consumer, release)
    if consumer['adoption'] != ('core' if consumer.get('mode') == 'api' else 'package'):
        raise SystemExit('Consumer package adoption is pending')
    if args.base_ref:
        subprocess.run(['git', 'check-ref-format', '--branch', args.base_ref], check=True, stdout=subprocess.DEVNULL)
    with tempfile.TemporaryDirectory(prefix='archive-upgrade-') as work:
        clone = Path(work) / 'consumer'
        subprocess.run(['git', 'clone', '--quiet', 'https://git.subcult.tv/' + consumer['repository'] + '.git', str(clone)], check=True)
        if args.base_ref:
            subprocess.run(['git', 'switch', '--detach', 'origin/' + args.base_ref], cwd=clone, check=True)
        print(proposal(release, consumer, clone, args.package_artifact.resolve() if args.package_artifact else None, base_ref=args.base_ref) or 'Release already applied; no PR')


if __name__ == '__main__':
    main()
