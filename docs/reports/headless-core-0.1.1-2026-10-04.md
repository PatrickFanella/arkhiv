# Independent core 0.1.1 qualification

Core 0.1.1 is published from `30454dd566f7d390a2644725b765073a447f4827`
at `git.subcult.tv/subculture-collective/archive-core@sha256:cb44da0d015ad52812f4247749be754f0e09af41b65fed9515dacac11f1c10c6`.
Hosted release run 12833 passed backend race/database checks, image verification,
registry publication and receipt upload. The 1,040-byte hosted artifact was
subsequently downloaded with the agent account and its parsed receipt exactly
matches the locally generated receipt and registry image. The image contains
no frontend assets, Node or npm runtime and no `VHNESS_WEB` setting.

The normal upgrade tool created [PR #96](https://git.subcult.tv/subculture-collective/arkhiv/pulls/96)
for the registered API-only reference consumer. Its two changed files are
`deploy/archive-core.env` and `deploy/core-release.json`. It changes no frontend
package, lockfile, source or build. Core 0.1.0 and its receipt remain immutable.
Adoption and upgrade PR merge state remain separate from this runtime proof.

An isolated clone of the retained mixed fixture database passed the additive
schema 8 to 9 migration. All original 24 table counts match the checkpoint;
publication tables initially contain no approvals. The exact retained frontend
image `sha256:8a069435b772d4635d2e0c7c2d10c4357c263b9b31816123d05f11ba0034b3d7`
runs against the new core. The old core and its original schema 8 database remain
running for rollback checks. Real HTTP catalog, collections, print status/search,
print/audio/video resolution and media bytes match between both core artifacts.

| Fixture original | Bytes | SHA-256 |
| --- | ---: | --- |
| Two-page PDF | 951 | cf0d999900a8ed3bec072d332a47bee0cde5f28f7c77b49c228dee7f2d85bba8 |
| Two-track recording | 320044 | 93ceeee2a37f989458b3635200f74ee49f8020c4d53cf571156cee7e117711d2 |
| Video | 458825 | 583a3c1275e07c1689b14f29c962c0d49993783e0a8203bfebc79dbaaf1467d0 |

The actual published core's public audience was tested privately. It initially
exposes zero items and denies print/audio transport and operator status. Explicit
approval of the exact generated PDF exposes only its print representation and
returns unchanged bytes. Audio remains denied. Withdrawal returns the catalog
to zero. These fixtures have no public ingress and do not establish media fidelity.
The database integration suite additionally covers native video approval,
withdrawal, changed source/selected file, private future schedules and unchanged
private visibility.

Browser qualification found an independent frontend defect: nginx served the
PDF worker's `.mjs` file as `application/octet-stream`. The retained failure logs
show module rejection. Frontend source `8e75f52f283165e8917ac405b146d3c296ba0e7d`
adds its JavaScript MIME mapping; image
`sha256:cd3803106e88c492bee0b16101eaa7d0e3218a2a5cfc7e6d1a5167f547531e10`
serves the same worker/build assets with `application/javascript` and `nosniff`.
The browser then loads the real worker and renders a 918 by 1188 PDF canvas with
no reader alert. One expected denied-download console response remains because
the fixture grants reading only. The core image is unchanged by this fix.
Full isolated `make verify` passed after the frontend fix and API-only registry
registration, including Go race/database suites, packed consumer import,
38 operator tests, frontend tests, typecheck, lint and production build.

Private receipts live on Kuznya beneath
`/home/ci/.local/state/arkhiv-private-qualification/mixed-reference-572d50b/headless-core011`.
Registry and hosted artifact copies live on Kvant under
`~/.local/state/vhness/releases/core-0.1.1`.

Public rollout remains in progress. Anarchist Archive's existing public catalog
and fresh recovery are separately qualified in its print repository report.
The staged combined database retains 6,725 native video entries and the initial
25 print items/26 representations. Source schemas retain every imported native
row and original per-brand schedule pointer; recorded history-id offsets preserve
identity mappings where merged sequence numbers overlap. This is a staged
catalog, not proof that all records or brands are publicly deployed. Real music
listening and any change to existing no-public-relay reviews remain owner gates.
