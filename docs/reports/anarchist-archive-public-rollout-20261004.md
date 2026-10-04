# Anarchist Archive public rollout, October 4, 2026 UTC

The existing print product now serves `https://anarchist-archive.subcult.tv`.
The owner requested this hostname while authorizing continued public rollout.
The existing `arkhiv.subcult.tv` route remains available during preparation of
the mixed reference app; its document links have not yet moved to redirects.

The deployed frontend and API bind source
`af1f358774ee083e1818a05cebb4e7b542bddd49`, with shared reader 0.2.2.
Hosted PR run 12818 passed all required checks and all 12 desktop/narrow browser
checks. The final change updates the reader's accessible home-link assertion
to the renamed product. The built frontend assets match the private candidate
that displayed the catalog and advanced the sample PDF from page 2 to page 3.

Only `public-api` and `public-web` in Almaz's existing `arkhiv-pilot` Compose
project were replaced. The database, private reader/API, operator, workers,
archive ownership and content decisions remain in their existing scope. The
public API retains its `arkhiv_public` role, `ARKHIV_PUBLIC=1` filter and read-only
archive mount. The new Caddy route uses the existing `arkhiv-public-web` alias.
Caddy configuration validation and reload passed without restarting Caddy.

The candidate matched every page of the existing 699-document public catalog.
The public hostname returned HTTP 200, rendered LittleGreenSongbook in Chromium
and served its original 9,337,316-byte PDF with SHA-256
`ae58f7cdf74f6fac70fe433499a806f06e01fb389ae7585fc1be45cad4edef02`.
The public reader reported a 918 by 594 canvas on page 2, with no alert and no
console errors. Its one PDF rendering warning remains in the retained browser
log. Querying the private `a-info` collection returned zero documents. These
checks establish the public path and representative reading/download behavior;
they do not constitute a long observation window or a fresh native-device run.

Before replacement, a 19,061,219-byte production catalog checkpoint was retained
with SHA-256
`f5fcf14d589dadd044579aba00118dfbe9412987e18458d80ba56b09849be0f6`.
The protected qualification directory is
`/mnt/spektr/server/data/arkhiv/qualification/anarchist-public-98071f3`.
It contains the original Compose configuration, catalog dump, previous Caddy
site and activation/ingress receipts. Existing independent archive/restore
evidence remains retained; this new dump has not yet had its own restore drill.
No schema migration occurred during this frontend/API cutover.

Rollback selects the retained public API/web images from `compose-before.json`
and recreates only those two services with `--no-deps`. Preserve subsequent
configuration changes: compare the activation receipt's Compose hash before
restoring a complete file. Remove only the new hostname block if later rollout
has added other Caddy routes; validate and reload the resulting configuration.
A whole-file Caddy rollback is appropriate only while its current hash still
matches this activation's ingress receipt. Keep the newer catalog checkpoint
and archive files; rollback does not delete or replace the database.
