# Private mixed interface checks

Observed October 3, 2026 in America/Chicago (October 4 UTC). These checks use
generated fixtures in a separate Kuznya database and archive. Real collection
migration, music fidelity and public cutover remain separate gates.

The neutral interface displays one catalog item with print, audio and video
representations. Two collection memberships share the PDF; each access decision
remains scoped to its collection. Listed entries without transport approval remain
visible, but their files are unavailable.

## Source and runtime

Baseline source: `572d50bb332a5ad1f63802dc9260d80fe99a2663`, image
`sha256:84782b8b5253edd21706f0c760941d2d74bf0eb7e1cea882e5e125ffb47192ec`.
Recording fixes: `a34ac11a84345456ac1b49d80e9c1603158d8ee4`, image
`sha256:2fa91aafceb3bcf1f85bd86770a1c7de7244e92875cfdaf894b2059abc9709ba`.
Both apps retain separate container owners and read-only archive mounts. They
share only the task-owned synthetic database. Neither runs acquisition workers.
Their loopback SSH forwards are `48571` and `48572` on Kvant.

The synthetic source response retains all file entries and unknown fields. Its
SHA-256 is `a5037d047a012889f82aae34bbdc1ba11ecebbb113c0fdeeaa1536f90480fef6`.
Four originals and one continuous derivative total 1,099,908 bytes. Those five
files were copied to the independent NUC disk, fetched into a separate Kuznya
recovery directory, and compared by byte count and SHA-256. The recovery receipt
is outside Git at `~/.local/state/vhness/mixed-reference-572d50b-recovery.json`
on Kvant, SHA-256
`fe7a75f49e5d35f69be11836108eea3ecf8ef1ebb2159c3b9dd45542f8c4a453`.
This proves recovery of generated media; it does not prove a real archive or
metadata database migration.

## Browser observations

T3 navigation and snapshot automation continued returning client errors after
reopening the panel and trying fresh tabs. The user confirmed the visible preview
loaded, then explicitly authorized isolated Playwright. The session
`arkhiv-platform-proof` uses its own browser profile.

| Check | Observed result |
| --- | --- |
| PDF load | PDF.js worker loaded; two pages available; canvas contained rendered dark pixels |
| Page navigation | Keyboard Next page opened page 2; readable text matched the second page |
| Document search | Search for `second` returned a page-2 match and its snippet |
| Narrow reader | At 390 × 844, fit-width canvas was 351 pixels and document width was 390; 200% zoom scrolled within its wrapper |
| Continuous audio | Actual 20-second derivative played; position advanced; second-track control sought to 10 seconds |
| Reviewed section | Baseline stopped at 10.168 seconds; the shared fix clamps the paused position to exactly 10 seconds |
| Credits | Source, taper, transfer and synthetic warnings remained visible; uncertain boundaries had no section controls |
| Shared station | Joined at the current offset; pause/rejoin left revision `synthetic-fixture-station-1` unchanged; leave removed the audio element |
| Video | Native decoder reported 320 × 180 and 20 seconds; playback advanced and a seek to 12 seconds resumed |
| Filter lifecycle | Changing the media filter removed the active video/audio element; page width stayed 390 pixels |
| Unapproved listing | Reader selection displayed `This item is unavailable for this collection.` |
| Unsupported EPUB | Fixed interface explained that its reader supports PDF files and linked the source |
| Missing recording | An intercepted continuous-resolve 404 displayed the unavailable message and created zero audio elements; it did not substitute the first original track |
| Catalog loading | A held response showed loading with zero stale item cards |
| Empty search | A completed unmatched query displayed the explicit empty result |
| Catalog failure | An intercepted 503 showed retry feedback with zero stale item cards |

The missing-recording check injects an HTTP failure through Playwright routing;
it does not remove originals or simulate an independent disk failure. Expected
404s appear in the browser console during unavailable-source checks.

## Verification and remaining gates

Full isolated `make verify` passed after the section-boundary fix: Go formatting,
vet, race tests and database integration; 19 viewer tests; a packed ESM consumer
check; 18 Python checks; four reference frontend tests; TypeScript, lint and builds.
Hosted push run 12609 and PR run 12610 succeeded for exact `a34ac11`.
PRs #63–#70 have merged; main `4532de8` passed hosted run 12611.

The generated tones establish control behavior and decoding, not real music
fidelity. These viewport checks are browser evidence, not physical-device tests.
Real print/audio migration, complete approved collection coverage, independent
database restore, branded consumer adoption and a subsequent shared upgrade
remain open in milestones 3–5. Milestone 6 retains the separate public decision.
