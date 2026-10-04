# Representation access

The mixed catalog separates listing from transport. A listed print representation
has no reader or download access until an operator records its exact source
snapshot and files. Reading, playback, downloads, acquisition, relay and scheduling
are independent flags. Public access is disabled for this private integration;
existing deployed donor products keep their current public decisions and services.

Operator commands use database authorization, with no web mutation endpoint:

```sh
vhness import-catalog --file reviewed-item.json --notes 'Reviewed identity mapping'
vhness review-transport --id ITEM --representation REPRESENTATION \
  --collection COLLECTION --file reviewed-transport.json
```

The transport JSON contains `review` and `preservation`. Review notes, exact source
digest and an explicit file list are required for enabled actions. A local
preservation record includes the same source digest, bounded original manifest
and the digest of independently qualified recovery evidence. Recording that
digest does not itself perform or prove recovery. The migration qualification
workflow supplies and checks the evidence before using these records.

Each review is transactional and appends its complete payload to transport
history. Revocation replaces current flags while preserving previous reviews.
Snapshots remain immutable. Source refresh or an edited file binding cannot
inherit a stale transport decision. No brand configuration grants these flags.

The private resolver is
`GET /api/catalog/v1/items/{item}/representations/{representation}/resolve`.
An explicit `collection` is required when the server has no single configured
collection. Optional `action` is `read`, `play` or `download`, with a reviewed
`file`; defaults select the representation's primary file and appropriate mode.
Unsupported actions and absent/revoked access return 404.

Source-streaming resolution returns a direct Archive.org URL. Its media endpoint
returns 404, so the server cannot relay those bytes. Democracy Now! protection
also reads the retained source collection identity: mapping its source into a
mixed collection cannot activate local delivery, acquisition, downloads or relay.
Metadata workers and the existing Bill Nye acquisition path are unchanged.

Local resolution returns the access-checked `/media` endpoint. Every request,
including Range and conditional requests, checks the current review before
opening the exact original under the media root. Its hash, size and regular-file
type must match the approved manifest. Reading does not enable the download
action. Review withdrawal prevents an ETag response from exposing a withdrawn
asset. The manifest's total must fit the collection budget, with its existing
positive reserve retained. Actual free-space checks remain in the acquisition
workflow; this change creates no generic acquisition or relay endpoint.

This is the common policy/transport foundation for the print and audio modules.
Real PDF reading, audio continuity, OCR/covers, shared audio scheduling, donor
migration and independent recovery require their later module qualification.
Synthetic policy fixtures establish access and byte behavior, not media fidelity.
