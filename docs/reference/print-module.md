# Shared print module

The print module adapts the GPL-3.0-only Arkhiv reader, extraction pipeline and
synthetic PDF fixture from donor revision
`99d499b92cb8d848e0fcb37288ece8bce9de943e`. Source comments retain that provenance.
The shared viewer's `0.2.0` source version remains unreleased.

Consumers import `PrintReader` from `@subcult/archive-viewer/print` and supply a
PDF backend. Each page supplies width, height, cancellable canvas rendering and
readable text. The consumer owns its PDF library, worker asset, layout and CSS.
The root entry keeps existing video imports compatible without a PDF dependency.
Search appears only when the consumer supplies an actual search endpoint.

`vhness index-print` indexes an already preserved, reviewed PDF. `VHNESS_MEDIA`
contains immutable originals; `VHNESS_DERIVATIVES` is a separate directory for
generated covers. The server can mount both directories read-only after indexing.
Poppler reads the verified original through an inherited file descriptor, with
time, memory and output limits. Native text retains page numbers. An explicitly
reviewed OCR companion supplies document text with page zero when its page mapping
is unknown. A failed cover does not invalidate the original or extracted text.

Text and cover responses require the current reading review, source snapshot and
original hash. Revoked access or a corrupt original denies cached derivatives.
Covers have their own recorded hash and size. Reading permission does not grant
download permission.

Disposable integration tests extract a generated two-page PDF, render its cover,
exercise OCR fallback and deny revoked or corrupt originals. Package checks
install and import the print entry in a fresh consumer without a PDF library.
These checks do not establish real collection reading, browser rendering,
migration or independent recovery; those require separate qualification.
