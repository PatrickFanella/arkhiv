# Combined catalog curation qualification, October 6, 2026

Arkhiv owns the copied private workbench under `curation/`; its AGPL license,
frozen lock and recorded Kollektor source provenance remain intact. Kollektor's
original service and workspace were untouched. This report qualifies the
integration tracked in Kollektor #2 through the maintained Arkhiv implementation.

The isolated workbench used Python 3.12, uv 0.12.22, disabled discovery and a new
private SQLite workspace outside Git. Its core URL pointed at the running Go
engine in the owned fixture preview, rather than a mocked HTTP transport. No
production catalog, source media or public service was changed.

The wide browser created a local collection, browsed five catalog items across
print/audio/video, selected representations and recorded review notes. Three
sources were accepted; a fourth video remained pending. Print acceptance also
worked through Enter. A 390px same-origin responsive frame browsed the combined
catalog and edited the retained print attachment through its actual form. The
frame had a 375px document width and no horizontal overflow. This is browser
responsive evidence, not a physical mobile claim.

The exported `arkhiv.curation.v1` manifest contained the accepted print, audio
and video mappings, retained source/representation evidence and notes, and
excluded the pending video. All seven top-level and per-mapping access fields
were false. The target was the existing `vhness-local-fixture` collection.

The actual `prepare-curation` operator command accepted that exported manifest
twice and returned the same draft checksum:
`5ce86384be31245b0d5252d62a83bd715efe598eba21acc434b5263714d37710`.
Both receipts reported `operator-review`, `public: false` and
`transport_granted: false`. Before/after complete row fingerprints matched for
15 tables: snapshots, observations, decisions, video schedule/history, current
schedule, catalog items/memberships/representations/reviews, representation
transport reviews/history, audio assets/history and audio schedules. Only the
operator draft was added. Generated manifest and receipt remain in the owned
runtime volume at `/data/qualification-curation-1791265041545749497`.

`make verify` passed for the current shared source. `make verify-curation`
passed all 19 tests, lint, formatting and JavaScript syntax. Tests additionally
qualify changed/withdrawn evidence rejection, retained decisions across restart,
source credits, target isolation and rejected access grants. The separate
curation Docker verify target also passed all 19 tests, lint and formatting
with its frozen lock and Python 3.13 image. The build-cache verification target
was not published or deployed. Membership,
transport approval and public activation remain separate operator decisions.
