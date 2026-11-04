#!/usr/bin/env python3
"""Apply a reviewed image plan to one owned public deployment on this host, swapping one service at a time with rollback.

The plan names, per compose service, the exact registry image to run: a content-hashed
tag `<registry>/<name>:<40-hex revision>` whose image label must carry that revision, or
an immutable `<registry>/<name>@sha256:<digest>` for the shared core. Each swap is a
compose overlay that changes only the image, is checked for configuration equivalence,
waited for health, checked over HTTPS, and reverted on any failure. Previous images are
never deleted; every run leaves a receipt.
"""
import argparse
import datetime
import fcntl
import hashlib
import json
import os
import re
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

DIGEST = re.compile(r'^(?P<registry>[a-z0-9.-]+(?::\d+)?/[a-z0-9._-]+)/(?P<name>[a-z0-9._-]+)@sha256:(?P<digest>[0-9a-f]{64})$')
TAG = re.compile(r'^(?P<registry>[a-z0-9.-]+(?::\d+)?/[a-z0-9._-]+)/(?P<name>[a-z0-9._-]+):(?P<revision>[0-9a-f]{40})$')
SERVICE = re.compile(r'^[a-z0-9][a-z0-9_-]{0,63}$')
DOCKER = ['docker']
HOST_KEYS = ['Memory', 'MemorySwap', 'NanoCpus', 'PidsLimit', 'ReadonlyRootfs', 'SecurityOpt', 'CapDrop', 'CapAdd', 'PortBindings', 'RestartPolicy', 'LogConfig']


class DeployError(Exception):
    pass


def default_run(args, stdin=None):
    result = subprocess.run(args, input=stdin, capture_output=True, text=True)
    if result.returncode != 0:
        detail = (result.stderr or result.stdout).strip().splitlines()
        raise DeployError(' '.join(args[:6]) + ('...' if len(args) > 6 else '') + ' failed: ' + ' | '.join(detail[-3:]))
    return result.stdout.strip()


def default_fetch(url, timeout=20):
    request = urllib.request.Request(url, headers={'User-Agent': 'arkhiv-deploy/1'})
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            return response.status, response.read(1 << 20).decode('utf-8', 'replace')
    except urllib.error.HTTPError as error:
        return error.code, ''


def load_config(path):
    config = json.loads(Path(path).read_text())
    for key in ('state', 'registry', 'targets'):
        if key not in config:
            raise DeployError('Deployment configuration lacks ' + key)
    if not re.fullmatch(r'[a-z0-9.-]+(:\d+)?/[a-z0-9._-]+', config['registry']):
        raise DeployError('Registry must be host/owner')
    for name, target in config['targets'].items():
        for key in ('project', 'services'):
            if key not in target:
                raise DeployError('Target ' + name + ' lacks ' + key)
        for service, spec in target['services'].items():
            if not SERVICE.match(service) or spec.get('kind') not in ('frontend', 'api', 'core') or not re.fullmatch(r'[a-z0-9._-]+', spec.get('image', '')):
                raise DeployError('Target ' + name + ' service ' + service + ' is misconfigured')
    return config


def validate_plan(config, target_name, plan):
    target = config['targets'][target_name]
    services = plan.get('services')
    if not isinstance(services, dict) or not services:
        raise DeployError('Plan names no services')
    resolved = {}
    for service, entry in services.items():
        spec = target['services'].get(service)
        if spec is None:
            raise DeployError('Service ' + service + ' is not deployable in target ' + target_name)
        image = entry.get('image', '') if isinstance(entry, dict) else ''
        if spec['kind'] == 'core':
            match = DIGEST.match(image)
            if not match or match.group('registry') != config['registry'] or match.group('name') != spec['image']:
                raise DeployError('Service ' + service + ' requires an immutable ' + config['registry'] + '/' + spec['image'] + '@sha256 digest')
            resolved[service] = {'image': image, 'revision': None}
        else:
            match = TAG.match(image)
            if not match or match.group('registry') != config['registry'] or match.group('name') != spec['image']:
                raise DeployError('Service ' + service + ' requires ' + config['registry'] + '/' + spec['image'] + ':<revision>')
            resolved[service] = {'image': image, 'revision': match.group('revision')}
    return resolved


def pull_and_verify(run, image, revision):
    run(DOCKER + ['pull', '--quiet', image])
    inspected = json.loads(run(DOCKER + ['image', 'inspect', image]))[0]
    labels = inspected['Config'].get('Labels') or {}
    if revision is not None and labels.get('org.opencontainers.image.revision') != revision:
        raise DeployError('Image ' + image + ' carries revision ' + str(labels.get('org.opencontainers.image.revision')) + ', not ' + revision)
    return inspected['Id']


def wrap_upstream(run, image, upstream, tag):
    """Derive a brand frontend whose proxy names this deployment's core service."""
    if not re.fullmatch(r'[a-z0-9][a-z0-9_-]*:\d{2,5}', upstream):
        raise DeployError('Invalid core upstream ' + upstream)
    dockerfile = ('FROM ' + image + '\nUSER root\n'
                  "RUN grep -q 'api:8080' /etc/nginx/nginx.conf && sed -i 's#api:8080#" + upstream + "#g' /etc/nginx/nginx.conf\n"
                  'USER 101:101\n')
    run(DOCKER + ['build', '--quiet', '--label', 'tv.subcult.arkhiv-deploy.source=' + image, '-t', tag, '-'], stdin=dockerfile)
    return run(DOCKER + ['image', 'inspect', '--format', '{{.Id}}', tag])


def container_name(project, service):
    return project + '-' + service + '-1'


def compose_base(run, project, service):
    old = json.loads(run(DOCKER + ['inspect', container_name(project, service)]))[0]
    labels = old['Config']['Labels']
    if labels.get('com.docker.compose.project') != project or labels.get('com.docker.compose.service') != service:
        raise DeployError('Container for ' + service + ' does not belong to ' + project)
    base = DOCKER + ['compose', '-p', project]
    environment = labels.get('com.docker.compose.project.environment_file')
    if environment:
        base += ['--env-file', environment]
    for file in labels['com.docker.compose.project.config_files'].split(','):
        base += ['-f', file]
    return old, base


def equivalent(run, base, overlay, service):
    before = json.loads(run(base + ['config', '--format', 'json']))['services'][service]
    after = json.loads(run(base + ['-f', str(overlay), 'config', '--format', 'json']))['services'][service]
    strip = lambda value: {k: v for k, v in value.items() if k not in ('image', 'build', 'restart')}
    if strip(before) != strip(after):
        raise DeployError('Overlay for ' + service + ' changes more than the image')


def restart_policy(container):
    policy = container['HostConfig']['RestartPolicy']
    name = policy.get('Name') or 'no'
    if name == 'on-failure' and policy.get('MaximumRetryCount'):
        return name + ':' + str(policy['MaximumRetryCount'])
    return name


def swap_service(run, fetch, release, project, service, image_id, image_ref, origin=None, check=None, wait_timeout=120):
    old, base = compose_base(run, project, service)
    if old['Image'] == image_id:
        return {'service': service, 'status': 'unchanged', 'image': image_id}
    restart = restart_policy(old)
    overlay = release / (service + '.compose.json')
    rollback = release / (service + '.rollback.json')
    overlay.write_text(json.dumps({'services': {service: {'image': image_ref, 'restart': restart}}}) + '\n')
    rollback.write_text(json.dumps({'services': {service: {'image': old['Image'], 'restart': restart}}}) + '\n')
    equivalent(run, base, overlay, service)
    up = ['up', '-d', '--no-build', '--no-deps', '--pull', 'never', '--wait', '--wait-timeout', str(wait_timeout), service]
    record = {'service': service, 'status': 'prepared', 'previous_image': old['Image'], 'image': image_id, 'image_ref': image_ref,
              'command': base + ['-f', str(overlay)] + up, 'rollback_command': base + ['-f', str(rollback)] + up}
    try:
        run(record['command'])
        current = json.loads(run(DOCKER + ['inspect', container_name(project, service)]))[0]
        if current['Image'] != image_id:
            raise DeployError('Container for ' + service + ' did not switch to the planned image')
        if sorted(current['Config']['Env']) != sorted(old['Config']['Env']):
            raise DeployError('Environment changed for ' + service)
        for key in HOST_KEYS:
            if current['HostConfig'].get(key) != old['HostConfig'].get(key):
                raise DeployError('Host configuration ' + key + ' changed for ' + service)
        mounts = lambda c: {(m['Source'], m['Destination'], m['RW']) for m in c['Mounts']}
        if mounts(current) != mounts(old):
            raise DeployError('Mounts changed for ' + service)
        if origin:
            path, expected = (check or {'path': '/', 'expect': '<html'})['path'], (check or {'expect': '<html'})['expect']
            last = None
            for attempt in range(20):
                status, body = fetch(origin + path)
                if status == 200 and expected in body:
                    break
                last = status
                time.sleep(1)
            else:
                raise DeployError('Origin ' + origin + path + ' answered ' + str(last) + ' after the swap')
        record['status'] = 'deployed'
    except Exception as error:
        record['status'] = 'rolled-back'
        record['error'] = str(error)
        run(record['rollback_command'])
        raise DeployError('Service ' + service + ' rolled back: ' + str(error))
    return record


def apply(config, target_name, plan, run=default_run, fetch=default_fetch, dry_run=False, now=None):
    global DOCKER
    target = config['targets'][target_name]
    DOCKER = list(target.get('docker') or config.get('docker') or ['docker'])
    if 'docker' not in DOCKER or not all(re.fullmatch(r'[A-Za-z0-9_./=-]+', part) for part in DOCKER):
        raise DeployError('The docker command must name docker and may add only plain flags such as --config <dir>')
    resolved = validate_plan(config, target_name, plan)
    stamp = (now or datetime.datetime.now(datetime.timezone.utc)).strftime('%Y%m%dT%H%M%SZ')
    state = Path(config['state'])
    state.mkdir(parents=True, exist_ok=True, mode=0o700)
    release = state / 'deploys' / (stamp + '-' + target_name)
    release.mkdir(parents=True, exist_ok=False, mode=0o700)
    receipt = {'target': target_name, 'project': target['project'], 'started_at': stamp, 'dry_run': dry_run, 'plan': plan,
               'script_sha256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest(), 'services': [], 'status': 'prepared'}
    (release / 'plan.json').write_text(json.dumps(plan, indent=2) + '\n')
    def save():
        (release / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n')
    save()
    with (state / 'deploy.lock').open('a') as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            raise DeployError('Another deployment holds the lock')
        maintenance = None
        if target.get('maintenance_lock'):
            maintenance = open(target['maintenance_lock'], 'a')
            fcntl.flock(maintenance, fcntl.LOCK_EX)
        try:
            images = {}
            for service, entry in resolved.items():
                spec = target['services'][service]
                image_id = pull_and_verify(run, entry['image'], entry['revision'])
                ref = entry['image']
                if spec.get('upstream'):
                    ref = 'local/arkhiv-deploy/' + target_name + '-' + service + ':' + (entry['revision'] or image_id.split(':')[-1][:12])
                    image_id = wrap_upstream(run, entry['image'], spec['upstream'], ref)
                images[service] = (image_id, ref)
            changing = [s for s, (image_id, _) in images.items() if json.loads(run(DOCKER + ['inspect', container_name(target['project'], s)]))[0]['Image'] != image_id]
            receipt['changing'] = changing
            if dry_run:
                receipt['status'] = 'dry-run'
                save()
                return receipt
            if not changing:
                receipt['status'] = 'unchanged'
                save()
                return receipt
            backend_change = any(target['services'][s]['kind'] in ('core', 'api') for s in changing)
            if backend_change and target.get('before_backend_change'):
                run(target['before_backend_change'])
                receipt['backup'] = target['before_backend_change']
            order = sorted(changing, key=lambda s: 0 if target['services'][s]['kind'] in ('core', 'api') else 1)
            for service in order:
                spec = target['services'][service]
                image_id, ref = images[service]
                if spec.get('migrate'):
                    command = [part.replace('{image}', resolved[service]['image']) for part in spec['migrate']]
                    run(command)
                    receipt.setdefault('migrations', []).append(command)
                    save()
                record = swap_service(run, fetch, release, target['project'], service, image_id, ref, spec.get('origin'), spec.get('check'))
                receipt['services'].append(record)
                save()
            if target.get('after'):
                run(target['after'])
                receipt['after'] = target['after']
            receipt['status'] = 'deployed'
            receipt['finished_at'] = datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ')
            save()
            return receipt
        except Exception as error:
            receipt['status'] = 'failed'
            receipt['error'] = str(error)
            save()
            raise
        finally:
            if maintenance:
                maintenance.close()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='action', required=True)
    do = sub.add_parser('apply', help='Read a plan JSON document on stdin and apply it')
    do.add_argument('--config', required=True)
    do.add_argument('--target', required=True)
    do.add_argument('--dry-run', action='store_true')
    args = parser.parse_args()
    config = load_config(args.config)
    if args.target not in config['targets']:
        raise SystemExit('Unknown target ' + args.target)
    plan = json.load(sys.stdin)
    try:
        receipt = apply(config, args.target, plan, dry_run=args.dry_run)
    except DeployError as error:
        print(json.dumps({'status': 'failed', 'error': str(error)}))
        raise SystemExit(1)
    print(json.dumps({'status': receipt['status'], 'target': args.target, 'changing': receipt.get('changing', []),
                      'services': [{'service': s['service'], 'status': s['status']} for s in receipt['services']]}))


if __name__ == '__main__':
    main()
