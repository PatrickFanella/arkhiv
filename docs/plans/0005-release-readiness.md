# Arkhiv and branded app release readiness

Owner clarification, October 5, 2026: only Kollektor functionality moves into
Arkhiv. Branded apps retain independent repositories and their original frontend
ownership. Arkhiv releases the shared engine; consumers adopt compatible pinned
versions through reviewed upgrade PRs.

## Work and acceptance

| Unit | Required behavior | Verification |
| --- | --- | --- |
| Copied curation workbench | Saved searches, bounded resumable discovery, review decisions, notes, exports, and combined-catalog curation run from Arkhiv without a Kollektor app | Retained workflow tests, container build/runtime, desktop and narrow-screen review flows |
| Curation recovery | A consistent online checkpoint restores decisions, search progress, and source evidence into a new workspace | WAL-aware backup, integrity/relationship checks, restarted restored app and accepted-only export |
| Retained print/audio core | Original brand APIs, identity, reading/playback, range behavior, and schedules work against the combined engine | Full isolated gate, independently restored donor comparisons, browser readers/players, withdrawal and scope rejection |
| Independent brands | Each brand retains its source, frontend, database/media ownership, and release pin | Per-repository checks and built frontend comparison; no frontend-source import into Arkhiv |
| Easy engine upgrades | Every engine consumer has an API-only registry lane, immutable release receipt, and meaningful consumer checks | Compatibility/reused-version rejection, repeatable upgrade proposal, exact-head hosted CI and independent rollback receipts |
| Release assessment | Each app has named evidence and explicit unresolved gates | Current release matrix; distinguish code checks, hosted CI, browser, media, recovery, natural runs, and deployment |

The engine remains headless. The copied workbench is an Arkhiv-owned private
operator artifact with its own locked Python dependencies and AGPL source
provenance. Its local curation state is separate from public catalog approval.
Existing branded services and original databases remain available through
qualification. New publication or acquisition scope is not required for this work.

## Commands and evidence

The core/reference gate is `t3-dev --profile vhness verify` from the release
worktree. Curation uses `make verify-curation` and the separate curation CI
workflow. Branded repositories retain their required gates and use isolated
snapshots. Record fresh results in the dated release report. Keep credential
values, database dumps, media, and detailed runtime receipts outside Git.

Remaining runtime qualification must use new databases and retained read-only
media, never resets of running preview or production data. Any release upgrade
preserves applied migration hashes and the prior immutable artifact/checkpoint.
