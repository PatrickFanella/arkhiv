#!/usr/bin/env python3
"""Open (or reuse) the pull request that lands a release receipt on main, merging it once checks pass."""
import argparse
import json
import os
import urllib.error
import urllib.request

REPOSITORY = 'subculture-collective/arkhiv'


def api(path, method='GET', payload=None):
    token = os.environ['ARCHIVE_UPGRADE_TOKEN']
    request = urllib.request.Request('https://git.subcult.tv/api/v1' + path, method=method,
        data=json.dumps(payload).encode() if payload is not None else None,
        headers={'Authorization': 'token ' + token, 'Content-Type': 'application/json', 'User-Agent': 'tea'})
    with urllib.request.urlopen(request, timeout=30) as response:
        raw = response.read()
        return json.loads(raw) if raw else None


def land(branch, title, body, request=api, base='main'):
    if request('/user')['login'] != 'subcult-agent':
        raise ValueError('Receipt pull requests require the subcult-agent identity')
    repo = '/repos/' + REPOSITORY
    pull = None
    page = 1
    while pull is None:
        batch = request(repo + '/pulls?state=open&limit=50&page=' + str(page))
        pull = next((p for p in batch if p['head']['ref'] == branch and p['base']['ref'] == base), None)
        if len(batch) < 50:
            break
        page += 1
    if pull is None:
        pull = request(repo + '/pulls', 'POST', {'base': base, 'head': branch, 'title': title, 'body': body})
    try:
        request(repo + '/pulls/' + str(pull['number']) + '/merge', 'POST', {'do': 'merge', 'merge_when_checks_succeed': True, 'delete_branch_after_merge': True})
    except urllib.error.HTTPError as error:
        if error.code not in (405, 409, 423):
            raise
        print(pull['html_url'], 'left for review: merge request answered HTTP', error.code, flush=True)
    return pull['html_url']


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--branch', required=True)
    parser.add_argument('--title', required=True)
    parser.add_argument('--body', required=True)
    args = parser.parse_args()
    print(land(args.branch, args.title, args.body))


if __name__ == '__main__':
    main()
