# Private headless HTTP qualification, 2026-10-04 UTC

The owner clarified that Arkhiv must be a headless core and end users must own
the entire frontend. PR #87 adds a core-only image target, backend verification
command and plain HTTP contract regression. The neutral app and React package
remain optional consumers.

## Exact artifact and independent runtime

Source: `94f9385634fdb4b0e86d7a20beed92aff9598f46`.
Core image: `sha256:55becf6e7e4f6be72d02855b563dae6a13d6651a2f87040d54a7b3023231a491`.
The `--target core` build executed only Go/backend and media-runtime stages.
Node/frontend stages were absent from its build. Container inspection checked
that `/app/web`, `node` and `npm` were absent and the source label matched.
An earlier local candidate with an incorrect source label remains unqualified;
this report uses only the corrected exact-source artifact above.

The runtime uses a new database copied from the existing independently restored
mixed fixture. Checkpoint SHA-256:
`3eaa9990fe766e7a2a18dafc0353a925cb7bb8501980aab1e0aaa2c32ac9ec00`.
All 24 table counts matched, preserving source snapshots, representation/access
history, print indexes, audio derivative records and schedule history. Media and
derivative mounts remain read-only. No acquisition worker or public endpoint was
started. Existing private apps and production services remained in place.

## HTTP client proof

A Python `urllib` client used HTTP directly, with no JavaScript, React or frontend
package. It verified:

- `/api/version` returned API version 1 and catalog, print, audio and video modules.
- `/api/ready` passed database readiness.
- Scoped catalog browsing returned three items, including the reviewed mixed item
  with print, audio and video representations.
- Each permitted representation resolved and delivered its preserved local bytes.
- Byte Range requests returned HTTP 206 and the same first 32 bytes as each full
  print, audio and video response.
- Continuous audio resolved separately and delivered the registered derivative.
- A foreign collection and unsupported transport action returned HTTP 404.
- `/`, `/library` and `/assets/frontend.js` returned HTTP 404.

| Media response | Bytes | SHA-256 |
| --- | ---: | --- |
| PDF original | 951 | `cf0d999900a8ed3bec072d332a47bee0cde5f28f7c77b49c228dee7f2d85bba8` |
| First audio track | 160,044 | `63003ab1e59c3522723a73ddabe189d9e73ab59dce29a4cc67bdb49a8927641e` |
| Video original | 458,825 | `583a3c1275e07c1689b14f29c962c0d49993783e0a8203bfebc79dbaaf1467d0` |
| Continuous audio derivative | 320,044 | `93ceeee2a37f989458b3635200f74ee49f8020c4d53cf571156cee7e117711d2` |

These are deliberately synthetic media and prove contracts, scope and byte
transport. They do not prove audiovisual fidelity. The separate real print,
music and video reports retain those evidence boundaries.

## Validation and remaining release work

Full isolated `make verify` passed before the source commit: Go formatting, vet,
race/database checks and compilation; 21 viewer tests; packed client import;
29 Python tests; four web tests; typecheck, lint and frontend build. The new
headless regression uses the HTTP handler without a web directory and verifies
capability discovery, absent frontend routes and collection enforcement.

Issue #86 remains open for independently published core artifacts, API-only
consumer upgrade receipts and hosted artifact qualification. This private image
is not a registry release or a public deployment. Existing published 0.2.2
package and bundled engine artifacts remain unchanged. Core updates reach custom
HTTP clients through their backend upgrade; optional React fixes reach only
package consumers after their package upgrade.
