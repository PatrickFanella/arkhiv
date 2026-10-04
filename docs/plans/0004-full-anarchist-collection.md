# Full Anarchist source and verified automatic access

Owner decision, October 3, 2026: automatically approve all Anarchist collection documents for archival reading/downloads. Retain unknown rights, source attribution, correction/removal contact and no asserted reuse license. Track [issue #52](https://git.subcult.tv/subculture-collective/arkhiv/issues/52).

The measured source item `ZineArchive` has 700 unrestricted PDFs totaling 3,510,725,119 bytes. Select all exact filenames while retaining the 25 existing pilot document keys and their 26 variants. Added filename-derived titles are provisional; do not infer creators, publication dates or grouping. Existing source/editorial separation, immutable originals and backup destinations remain.

Use explicit limits of 1,000 documents, 8 GiB per acquisition batch, 250 MiB per file, concurrency two and 2 GiB free-space reserve. These limits provide room for the measured source, OCR and future editions. Source changes that exceed the bounds require a new capacity decision. Download/access restrictions, failures, corrupt PDFs and failed extraction remain visible qualification exceptions.

Automatic access is opt-in per collection and binds to the recorded source item. In one transaction per document, verify originals and reader manifests, publish eligible records and write exact variant/hash approval events and retained decision manifests. An explicit variant revocation, document withdrawal or source hold/rejection blocks automatic restoration, including later file hashes or document records under the same collection/source filename. Only an explicit manual decision can restore access. Manual collections and foreign source items are excluded. No reuse license is asserted.

Qualification order: isolated policy/budget tests and make verify; commit and linked PR; stage the operator build and new configuration on Almaz with rollback copies; acquire all selected originals and available OCR; extract/index; apply the explicit policy; qualify public catalog/search/reader/downloads; run independent NUC backup and verify its contents; record unavailable/failed files and resulting counts. Scheduled updates acquire eligible additions and apply the same verification policy. The design PR remains a separate layer.

The source preflight found two zero-byte OCR companions (`ars-print-pdf-final1_djvu.txt` and `ars-read-pdf-final2_djvu.txt`). They are excluded as unavailable optional text while their PDF originals remain selected. The inventory records this exclusion; no text coverage is invented. Usable OCR companions retain the same size, access and checksum checks.

The all-document reader probe disposes each downloaded response after hashing so qualification does not retain the entire multi-gigabyte corpus in memory.

Scheduled acquisition filters PDF filenames before applying its 500-event batch bound. Non-PDF source files remain in the review inventory and cannot permanently fill the acquisition page ahead of eligible PDFs. The regression check places 500 older non-PDF events ahead of a reviewed PDF.
