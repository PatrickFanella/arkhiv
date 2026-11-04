#!/usr/bin/env python3
"""Resolve the image plan for a public deployment target from each repository's main branch.

Frontend and API services deploy the image their repository published for its current
`main` commit. Shared cores deploy the immutable digest that the owning consumer pins in
`deploy/archive-core.env` on that same commit, so a core never moves ahead of the
consumer that verified it.
"""
import argparse
import base64
import json
import os
import re
import sys
import urllib.parse
import urllib.request

API = 'https://git.subcult.tv/api/v1'
REGISTRY = 'git.subcult.tv/subculture-collective'
TARGETS = {
    'platform': {
        'reference': {'repository': 'subculture-collective/arkhiv', 'image': 'archive-reference'},
        'archive-core': {'repository': 'subculture-collective/arkhiv', 'core': True},
        'democracy-now-reference': {'repository': 'subculture-collective/democracy-now-tv', 'image': 'democracy-now-tv-frontend'},
        'democracy-core': {'repository': 'subculture-collective/democracy-now-tv', 'core': True},
        'bill-nye-reference': {'repository': 'subculture-collective/bill-nye-tv', 'image': 'bill-nye-tv-frontend'},
        'bill-core': {'repository': 'subculture-collective/bill-nye-tv', 'core': True},
        'riot-public-web': {'repository': 'subculture-collective/arkhiv-riot-grrrl', 'image': 'arkhiv-riot-grrrl-web'},
        'riot-public-api': {'repository': 'subculture-collective/arkhiv-riot-grrrl', 'image': 'arkhiv-riot-grrrl-api'},
    },
    'anarchist': {
        'public-web': {'repository': 'subculture-collective/arkhiv-anarchist', 'image': 'arkhiv-anarchist-web'},
        'public-api': {'repository': 'subculture-collective/arkhiv-anarchist', 'image': 'arkhiv-anarchist-api'},
    },
    'dedhed': {
        'web': {'repository': 'subculture-collective/dedhed', 'image': 'dedhed-web-public'},
        'api': {'repository': 'subculture-collective/dedhed', 'image': 'dedhed-api'},
        'operator': {'repository': 'subculture-collective/dedhed', 'image': 'dedhed-api'},
    },
}


def default_fetch(path):
    headers = {'User-Agent': 'arkhiv-deploy-plan/1'}
    token = os.environ.get('ARCHIVE_PACKAGE_READ_TOKEN')
    if token:
        headers['Authorization'] = 'token ' + token
    with urllib.request.urlopen(urllib.request.Request(API + path, headers=headers), timeout=30) as response:
        return json.loads(response.read())


def main_revision(fetch, repository, cache):
    if repository not in cache:
        sha = fetch('/repos/' + repository + '/branches/main')['commit']['id']
        if not re.fullmatch(r'[0-9a-f]{40}', sha):
            raise ValueError('Unexpected main revision for ' + repository)
        cache[repository] = sha
    return cache[repository]


def core_digest(fetch, repository, revision):
    content = fetch('/repos/' + repository + '/contents/deploy/archive-core.env?ref=' + revision)
    text = base64.b64decode(content['content']).decode()
    match = re.fullmatch(r'ARCHIVE_ENGINE_IMAGE=(' + re.escape(REGISTRY) + r'/archive-core@sha256:[0-9a-f]{64})\n', text)
    if not match:
        raise ValueError(repository + ' pins no immutable archive-core digest at ' + revision)
    return match.group(1)


def plan(target, fetch=default_fetch, source=None):
    if target not in TARGETS:
        raise ValueError('Unknown target ' + target)
    cache = {}
    services = {}
    for service, spec in TARGETS[target].items():
        revision = main_revision(fetch, spec['repository'], cache)
        if spec.get('core'):
            services[service] = {'image': core_digest(fetch, spec['repository'], revision), 'from': spec['repository'] + '@' + revision}
        else:
            services[service] = {'image': REGISTRY + '/' + spec['image'] + ':' + revision, 'revision': revision, 'from': spec['repository']}
    return {'target': target, 'services': services, 'source': source or {}}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--target', required=True, choices=sorted(TARGETS))
    parser.add_argument('--run-id', default='')
    args = parser.parse_args()
    print(json.dumps(plan(args.target, source={'run_id': args.run_id, 'planner': 'scripts/deploy_plan.py'}), indent=2))


if __name__ == '__main__':
    main()
