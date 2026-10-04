# Mixed reference interface

The neutral Arkhiv frontend selects `"interface": "catalog"` in `site.json`.
Existing configurations with no interface field retain the video library and
scheduled replay layout. A brand can compose its own interface from the shared
`@subcult/archive-viewer`, `/catalog`, `/print` and `/audio` package entries.

The catalog exposes only the engine's configured collection allowlist. The
frontend cannot expand it through site configuration. Browsing all collections
does not grant transport rights: every representation button carries its exact
listed collection into the read or playback resolver. Items with several media
types appear once, with a button for each listed format and collection binding.

Catalog search covers titles and descriptions. Document search is offered only
when a current, reviewed print index reports page or document text. OCR without
page mapping remains labeled as document text. Collection changes, media filters,
search changes and pagination hide the previous catalog response and close the
selected reader or player. Request cleanup cancels superseded work.

PDF.js, its worker and font, character-map and WASM assets belong to the reference
frontend. The shared print entry does not bundle a PDF runtime. The reference
loads the PDF runtime when a document is opened. Both development and built
frontends serve the pinned runtime's assets. Readable page text, page navigation,
zoom and indexed search use the shared reader controls.

Continuous recordings use the reviewed audio derivative resolver and shared
player. Only reviewed set boundaries appear as recording-section choices. Other
reviewed single-file recordings use native audio controls. Source credits and
recording warnings remain visible. Legacy video items use the shared player,
including reviewed source captions and original dates; generic MP4 representations
use native video controls. Both retain a link to the source item.

Selecting one collection exposes its scheduled audio and video programs. Tune in,
local seeking, pausing, leaving and rejoining operate on the listener's session;
the interface does not write schedules. Changing collections unmounts the prior
channel controls and stops their requests and media sessions.

This implementation does not establish real-media browser qualification,
subjective audiovisual fidelity, donor migration, independent recovery or public
deployment approval. Those remain separate roadmap gates.
