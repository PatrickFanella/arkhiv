"""Bind the disposable database alias only inside an owned CI job container."""
import json
import os
from pathlib import Path
import re
import socket
import subprocess

from propose_release_upgrades import consumer_test_database


def bind(inspected, run, hosts=Path('/etc/hosts'), resolve=socket.gethostbyname):
    # Reuse every identity/address guard before writing a temporary job mapping.
    address = next(iter(inspected['NetworkSettings']['Networks'].values()))['IPAddress']
    consumer_test_database(inspected, run, 'qualification/alias', lambda name: address)
    try:
        current = resolve('db')
    except socket.gaierror:
        current = None
    if current is not None:
        if current != address:
            raise ValueError('Refusing to replace an existing database alias')
        return
    line = address + ' db\n'
    try:
        with hosts.open('a') as output:
            output.write(line)
    except PermissionError:
        subprocess.run(['sudo', '-n', 'tee', '-a', str(hosts)], input=line, text=True,
                       stdout=subprocess.DEVNULL, check=True)


def main():
    identity = os.environ['ARCHIVE_UPGRADE_DATABASE_CONTAINER']
    run = os.environ['ARCHIVE_UPGRADE_DATABASE_RUN']
    if not re.fullmatch(r'[0-9a-f]{64}', identity) or not re.fullmatch(r'[0-9]+-[0-9]+', run):
        raise ValueError('Expected one job-owned database and run identity')
    # Host mappings belong only to the ephemeral Actions container, never a host.
    if not Path('/.dockerenv').exists():
        raise ValueError('Database alias binding requires the isolated CI container')
    inspected = json.loads(subprocess.check_output(['docker', 'inspect', identity]))[0]
    bind(inspected, run)
    consumer_test_database(inspected, run, 'qualification/alias')
    print('Disposable database alias verified inside the owned CI job')


if __name__ == '__main__':
    main()
