#!/usr/bin/env python3
"""Migrate an unmodified generated video adapter to a reviewed package release."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--consumer', type=Path, required=True)
    parser.add_argument('--release', type=Path, required=True)
    args = parser.parse_args()
    root = args.consumer.resolve()
    if subprocess.check_output(['git', 'status', '--porcelain'], cwd=root):
        raise SystemExit('Adopt in a clean isolated consumer checkout')
    release = json.loads(args.release.read_text())
    if release['package'] != '@subcult/archive-viewer' or release['contract_version'] != 1 or release['configuration_version'] != 1 or '1' not in release['api_versions'] or 'video' not in release['media']:
        raise SystemExit('Unsupported video package release')
    if not re.fullmatch(r'git\.subcult\.tv/subculture-collective/archive-engine@sha256:[0-9a-f]{64}', release['engine_digest']):
        raise SystemExit('Use a qualified registry engine digest')
    adapter = root / 'web/src/engine'
    manifest = json.loads((adapter / 'manifest.json').read_text())
    expected = set(manifest['files']) | {'manifest.json'}
    if {p.name for p in adapter.iterdir()} != expected:
        raise SystemExit('Generated adapter contains unregistered files; preserve and review')
    for name, digest in manifest['files'].items():
        if Path(name).name != name or hashlib.sha256((adapter / name).read_bytes()).hexdigest() != digest:
            raise SystemExit('Generated adapter was edited; preserve and review: ' + name)
    app = root / 'web/src/App.tsx'
    names = []
    def replace(match):
        names.extend(n.strip() for n in match[1].split(','))
        return ''
    body = re.sub(r"^import \{ ([^\n]+) \} from './engine/[A-Za-z]+';\n", replace, app.read_text(), flags=re.MULTILINE)
    if not names or "'./engine/" in body:
        raise SystemExit('Consumer imports require a reviewed adapter migration')
    package_file = root / 'web/package.json'
    package = json.loads(package_file.read_text())
    if release['package'] in package['dependencies']:
        raise SystemExit('Consumer already has a package dependency')
    package['dependencies'][release['package']] = release['version']
    (root / 'docs/legacy-viewer-manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    package_file.write_text(json.dumps(package, indent=2) + '\n')
    app.write_text('import { ' + ', '.join(dict.fromkeys(names)) + " } from '@subcult/archive-viewer';\n" + body)
    (root / 'deploy/platform-release.json').write_text(json.dumps(release, indent=2) + '\n')
    (root / 'deploy/archive-engine.env').write_text('ARCHIVE_ENGINE_IMAGE=' + release['engine_digest'] + '\n')
    shutil.rmtree(adapter)
    (root / 'Dockerfile').write_text('''# syntax=docker/dockerfile:1
ARG ENGINE_IMAGE
FROM node:24-bookworm-slim AS frontend
RUN npm install --global pnpm@10.33.0 --force
WORKDIR /src/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN --mount=type=secret,id=npmrc,required=true NPM_CONFIG_USERCONFIG=/run/secrets/npmrc pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build
FROM ${ENGINE_IMAGE}
COPY --from=frontend /src/web/dist /app/web
''')
    checker = root / 'scripts/check_engine.py'
    original = checker.read_text()
    # Retain the existing deployment-tool and official asset integrity checks.
    tail = original[original.index('tools=json.loads'):]
    checker.write_text('''import hashlib,json,re
from pathlib import Path
repo=Path(__file__).resolve().parents[1]
release=json.loads((repo/"deploy/platform-release.json").read_text())
assert release["package"]=="@subcult/archive-viewer"
assert release["contract_version"]==release["configuration_version"]==1
assert "1" in release["api_versions"] and "video" in release["media"]
assert re.fullmatch(r"[0-9a-f]{40}",release["source_revision"])
assert re.fullmatch(r"git\\.subcult\\.tv/subculture-collective/archive-engine@sha256:[0-9a-f]{64}",release["engine_digest"])
assert (repo/"deploy/archive-engine.env").read_text()=="ARCHIVE_ENGINE_IMAGE="+release["engine_digest"]+"\\n"
package=json.loads((repo/"web/package.json").read_text())
assert package["dependencies"][release["package"]]==release["version"]
assert not (repo/"web/src/engine").exists(),"Legacy adapter is still present"
print("Shared package and immutable engine pins verified:",release["version"],release["source_revision"])
''' + tail)
    shutil.copyfile(Path(__file__).with_name('build_consumer_image.py'), root / 'scripts/build_image.py')
    (root / 'AGENTS.md').write_text('''# Standalone video app

This repository owns its frontend, design system and deployment configuration.
VHNESS owns the shared engine and `@subcult/archive-viewer` package. Import the
published package; never copy or edit shared implementation here. The exact
package dependency, pnpm lock, engine digest and release compatibility receipt
must change together through a reviewed upgrade. The former generated viewer
manifest is retained in docs/legacy-viewer-manifest.json for migration provenance.
Existing generated deployment tools remain pinned by docs/engine-tools.json.

Run make verify. Browser rendering, decoding, audiovisual review, shared-channel
timing, independent restore and deployment are separate checks. Runtime data,
media and secrets stay outside Git. Public deployment requires its separate
owner decision. Preserve original dates, source credit and replay labels.
''')
    print('Package adoption prepared; install the registry dependency, verify and qualify the consumer')


if __name__ == '__main__':
    main()
