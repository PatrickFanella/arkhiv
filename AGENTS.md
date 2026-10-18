# VHNESS implementation instructions

Read the design and ROADMAP.md. Democracy Now! (`democracy_now_vid`) is permanently
source-streaming only. Preserve complete source responses, every file entry,
history, decisions, and schedules. Metadata workers must never request video.
Local preservation requires an explicit different collection, manifest, quotas,
free-space reserve, and independent recovery evidence.

Go lives in `backend/`, React/TypeScript in `web/`, SQL in `migrations/`, and
operator tools in `scripts/`. Runtime data, source snapshots, media, credentials,
and generated outputs stay outside Git. Originals are never silently deleted.
Source fields and editorial decisions are independent.

Read the shared T3 agent guide before previews or tests; use profile `vhness`.
Required gate: `make verify`. Database integration requires the explicitly
disposable `TEST_DATABASE_URL`; no production/preview database may be reset.
Browser playback, actual audiovisual fidelity, hosted CI, complete collection
coverage, sustained qualification, and restore proof are distinct checks.

Arkhiv owns the copied private curation workbench under `curation/`. Use its
frozen uv lockfile and Python 3.12 or newer. Its additional required gate is
`make verify-curation`; the curation workflow verifies its container separately.
Keep its AGPL license and source provenance. Branded frontend source remains in
the consumer repositories; engine releases must not replace those interfaces.

Keep Gitea issues open until required evidence exists. Original dates and replay
labels must be visible. Viewer controls cannot alter shared schedules. Public
deployment requires its separate owner decision.

Source-catalog listing decisions use `set-source-catalog-listing`; do not delete
retained source metadata to hide a recording. Exclusion survives re-import and
keeps source history and import accounting independent of catalog visibility.
