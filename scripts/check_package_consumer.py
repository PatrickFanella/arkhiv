#!/usr/bin/env python3
"""Build and install the packed viewer in a consumer without sibling source code."""
import json
from pathlib import Path
import subprocess
import tarfile
import tempfile


def main():
    root = Path(__file__).resolve().parents[1]
    artifacts = root / '.artifacts'
    artifacts.mkdir(exist_ok=True)
    subprocess.run(['pnpm', 'build:packages'], cwd=root, check=True)
    subprocess.run(['pnpm', '--dir', 'packages/viewer', 'pack', '--pack-destination', str(artifacts)], cwd=root, check=True)
    version = json.loads((root / 'packages/viewer/package.json').read_text())['version']
    archive = artifacts / f'subcult-archive-viewer-{version}.tgz'
    with tarfile.open(archive) as tar:
        names = tar.getnames()
        assert 'package/dist/index.js' in names and 'package/dist/index.d.ts' in names
        assert not any('/src/' in name or name.endswith('.test.ts') for name in names)
    with tempfile.TemporaryDirectory(prefix='archive-consumer-') as directory:
        consumer = Path(directory)
        # Install the actual packed artifact in a fresh consumer. Third-party
        # runtime dependencies come from the reference lock/store, never sibling
        # application or viewer source. Retain lock resolutions for offline tests.
        dependencies = json.loads((root / 'web/package.json').read_text())['dependencies']
        dependencies['@subcult/archive-viewer'] = 'file:' + str(archive)
        (consumer / 'package.json').write_text(json.dumps({'private': True, 'type': 'module', 'dependencies': dependencies}))
        lock = (root / 'pnpm-lock.yaml').read_text()
        begin = lock.index('  web:\n')
        end = lock.index('\npackages:\n')
        # web is the last importer in this workspace; preserve all dependency
        # snapshots and let pnpm resolve only the new local tarball.
        importer = lock[begin:end].replace('  web:\n', '  .:\n', 1)
        importer = importer.replace('specifier: workspace:*', 'specifier: ' + dependencies['@subcult/archive-viewer']).replace('version: link:../packages/viewer', 'version: ' + dependencies['@subcult/archive-viewer'])
        (consumer / 'pnpm-lock.yaml').write_text(lock[:lock.index('importers:\n')] + 'importers:\n\n' + importer + lock[end:])
        store = Path(subprocess.check_output(['pnpm', 'store', 'path'], cwd=root, text=True).strip()).parent
        subprocess.run(['pnpm', '--dir', str(consumer), 'install', '--offline', '--ignore-workspace', '--no-frozen-lockfile', '--store-dir', str(store)], check=True)
        (consumer / 'check.mjs').write_text('''
import assert from 'node:assert/strict';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { Screen, playbackURL, useViewer, usePersonal, useChannelChat } from '@subcult/archive-viewer';
import { AudioPlayer, radioOffset } from '@subcult/archive-viewer/audio';
import { PrintReader, initialPrintPage } from '@subcult/archive-viewer/print';
import { useCatalog, catalogURL } from '@subcult/archive-viewer/catalog';
assert.equal(typeof useCatalog, 'function');
assert.ok(catalogURL({collection:'mixed',media:'print',query:'',page:1}).includes('collection=mixed'));
assert.equal(typeof AudioPlayer, 'function');
assert.equal(radioOffset(1,1000,0,10,2000),2);
assert.equal(typeof useViewer, 'function');
assert.equal(typeof usePersonal, 'function');
assert.equal(typeof useChannelChat, 'function');
assert.equal(initialPrintPage(999, 2), 2);
const print = renderToStaticMarkup(React.createElement(PrintReader, { title: 'Consumer document', url: '/private.pdf', backend: { open() { throw new Error('Server rendering must not open a document'); } } }));
assert.ok(print.includes('Document reader'));
assert.throws(() => playbackURL({ policy: 'source_stream', url: 'https://example.org/download/file.mp4' }));
assert.equal(playbackURL({ policy: 'local_preserve', url: '/api/media/fixture' }), '/api/media/fixture');
const html = renderToStaticMarkup(React.createElement(Screen, { viewer: { resolving: false, session: null, playError: '' } }, React.createElement('h1', {}, 'Independent consumer')));
assert.ok(html.includes('Independent consumer'));
console.log('Packed ESM consumer import, source policy and component composition passed');
''')
        subprocess.run(['node', 'check.mjs'], cwd=consumer, check=True)
    print('Packed consumer verified:', archive.name)


if __name__ == '__main__':
    main()
