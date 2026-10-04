# Original print pilot migration proof, October 4, 2026 UTC

The independently retained original print pilot restores into a new private
legacy database. A separate common-platform database and read-only application
now display its 25 documents and 26 PDF representations. The active original
services, collection operator and storage remain owned by their existing apps.
This proof covers the retained 25-document pilot, not the newer full corpus.

The NUC backup at `backups/arkhiv/pilot-qualified-20261002` contains 148 archive
files totaling 42,197,644 bytes and an 827,040-byte PostgreSQL dump. Every file
and the dump matched its retained checksum on NUC, Kvant and Kuznya. A fresh
`candidate` database restored the dump with 25 documents, 26 variants, 43
assets, 17 companions, 447 page-text rows, 58 review events, 156 acquisition
attempts, three schema versions and one source snapshot. An earlier incomplete
restore remains retained separately; it is not the recovery authority.

The original source file is 1,823,700 bytes with SHA-256
`e3bb4de18c1484cd8d1d6f5f9ba1a529c416f750450799cca6c9f89598648bd3`.
Its JSON content equals the restored raw source row. The historical source ID
`b1690cb1e11c133d62546d24e182530122fbc268386758dc1a2c05cb0cda9892`
used different JSON serialization. The converter records both identities and
preserves the original records; it does not claim identical serialized bytes.

The common-platform application uses engine source
`25a91e54b94f80120e1b516a712c62366e480da3`, separate PostgreSQL database
`platform`, immutable originals mounted read-only, and a new derivative root.
Existing histories remain in the restored legacy database and retained dump.
Its reviewed pilot permissions allow reading and original PDF downloads only;
public listing, acquisition, scheduling and relay remain disabled.

Isolated Playwright exercised the actual app at 390 by 844 pixels. Both
`2-by-le-Guin-READ.pdf` (24 pages) and `2-by-le-Guin-PRINT.pdf` (12 pages)
rendered through the shared PDF backend. Their canvases fit the 351-pixel reader
width without increasing the 390-pixel document width. Page navigation and a
search result jump to page 7 worked. The original READ PDF download returned
566,229 bytes, SHA-256
`fb4a2c8f9c3adab0a88dfae8a04c3c3d3c3f2724053f62249fbdd086a9509eaa`,
matching its server integrity receipt. A byte-range request returned HTTP 206
and exactly 1,024 bytes. Source identity and filenames remain visible.

The two editions exposed identical Read button labels in the catalog. A
follow-up change adds filenames for multiple representations of the same media
and displays an original-download link only after the separate download action
resolves. These changes require their own browser check and immutable release.
T3's visible preview loads, but native automation fails navigation and snapshots;
the user explicitly authorized isolated Playwright for these private checks.

Repository naming was verified on October 4: repository ID 242 is now
`subculture-collective/arkhiv` (formerly VHNESS); ID 235 is
`subculture-collective/arkhiv-alpha` (the original print product). Original Git
history and receipts keep their historical names. API writes use canonical
paths. The print adapter's task-specific remote targets ID 235; its primary
checkout and unrelated dirty work were preserved.

The common indexes cover 26 PDFs and 483 pages: 25 PDFs have page text
(463 pages), while Queering Protest Sites uses document text (20 pages).
Its real OCR search returns a document-text result without a page-jump button,
matching the coverage label. Retained legacy page-text counts are preserved
separately; reindexing does not replace that historical authority.

A common-platform backup was stored on NUC at
`backups/arkhiv-platform/print-pilot-20261004`, read back from NUC and restored
into `platform_recovery`. The 50,913,280-byte content archive and 1,378,401-byte
platform dump matched their checksums. Every public-schema table count equals
the source database, including 25 items, 26 representations, 26 indexes,
447 page-text rows, 26 transport reviews and eight schema versions. Original
legacy history remains included separately. The recovery check initially used
an incorrect table name after the restore; corrected table-by-table validation
passed without repeating or overwriting the completed restoration.

The restored app rendered the READ PDF at the same mobile width and its
original download again returned 566,229 bytes with the same SHA-256.

Remaining proof: browser verification of the new edition/download controls,
branded adapter qualification and a separate public-release decision.
Private reading and original-byte checks do not prove a public deployment or
full-corpus migration.
