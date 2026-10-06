# Viewer implementation qualification

This record tracks implementation for Arkhiv #105–#109 and #112–#113, plus the
linked Bill Nye TV, Democracy Now! TV and template issues. Designs #110 and #111
were accepted by the owner. Provider selection and public activation remain
separate decisions.

## Local evidence on Sputnik

Source is the `feat/improve-video-viewing` worktree under
`/home/onnwee/.t3/worktrees/arkhiv/t3code-e5a2f777`. Consumer source is on
`feat/viewer-feedback` in the three task-owned clones under
`/tmp/arkhiv-consumers`. No public deployment was performed.

The shared T3 `vhness` launcher runs copied snapshots, isolated networks and a
separate disposable database. Rootless Podman supplies the container API with
keep-id mapping; no host container socket is mounted into tests. The local
wrapper supplies frozen uv 0.12.22 and Python 3.12 for curation.

The latest `make verify && make verify-curation` passed for the 0.3.0 source.
It ran Go vet, race tests including disposable database integration, 34 SDK
tests, packed ESM consumer installation, 49 Python checks, four reference
frontend tests, frontend builds and 19 curation tests plus lint/format checks.
The independent restore test is explicitly separate from this gate.

`RUN_PERSONAL_RESTORE_TEST=1 python3 -m unittest discover -s scripts -p
 test_personal_restore.py` passed using a disposable PostgreSQL 15 service and
matching installed clients. It created two new personal databases, packed a
real pg_dump backup, restored with pg_restore through the operator tool, then
used a separate psql process to check that current independent journal events
suppressed deleted accounts, favorites and bodies, revoked sessions, retained
withdrawals and reapplied posting restrictions. Only its owned disposable
databases were removed. This does not prove off-host storage recovery.

The consumer source qualification copied each frontend into the isolated
snapshot and installed the packed SDK tarball. All three passed typecheck,
lint, tests and build. Their committed pins and release receipts still require
the qualified registry release; provisional tarball installation is not adoption
or a substitute for each repository's `make verify`.

Earlier generated H.264/AAC browser fixtures decoded in the collaborative
preview. Volume and mute survived on-demand/channel switches and a program
boundary. Pause and fresh tune showed explicit paused/synchronized labels and
rejoined the current channel offset without a native scrubber. This generated
fixture does not prove fidelity of real source programs. Original source video
was not downloaded for Democracy Now! metadata work.

## Outstanding evidence

Complete real browser checks of the branded controls, reviewed artwork,
filters, sharing, favorites and optional account/chat behavior. Qualify the new
registry package and immutable engine/core images, upgrade consumer pins and
receipts together, and run each consumer's required gate. Record source-program
fidelity and collection coverage separately where the issue requires them.

Hosted verification runs 13451–13454 for the initial source commit remained
queued. Read-only runner discovery found the `kvant` and `kuznya` runners with
`ubuntu-latest` labels offline. Other online runners carry unrelated production
or host labels. Do not relabel or borrow those to force a pass. Hosted CI and the
curation container workflow remain outstanding. Issues stay open until their
required evidence exists.

The independent curation verify container passed its 19 tests and lint/format checks. Its runtime image passed health with worker_alive=true under a read-only filesystem, no network, dropped capabilities and bounded temporary storage. Podman does not accept Docker's tmpfs uid/gid options here, so the private task container used bounded sticky temporary mounts instead. The hosted curation workflow remains unverified.

Bill Nye browser checks confirmed the source logo loads with a 24-pixel top gap, the power control uses centered grid alignment, and theater mode expands to one column and hides the sidebar. Escape exits theater mode. Subsequent mobile preview navigation failed with connection refusal despite the local readiness endpoint returning 200. Mobile and further interaction evidence remains outstanding.
