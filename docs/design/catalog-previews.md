# Catalog design: Front Cover

The owner selected A, **Front Cover**, on October 3, 2026. A large sans-serif masthead, a featured scanned publication, a compact contents column and a three-column cover catalog. The selected layout now uses the catalog API. The comparison switcher, six sample records, sample cover artwork and unused B/C layouts have been removed.

## Content and interaction

The opening feature uses the first returned document on the unfiltered first page. It is a browsing entry, not an editorial ranking. The contents column links to the next three returned records. Search and subsequent pages omit this opening feature and display the actual results, including page-text links. Collection selection, query, subject and pagination remain URL state handled by the server.

Cover images come from the existing local cover endpoint. Missing images show a clearly labeled title fallback. The layouts introduce no fabricated issue numbers, publication dates, scan artwork or creator information. Document pages retain the local PDF reader, download links, text-coverage labels, original source links, access statements, hashes and correction/removal contact.

At narrow widths the feature stacks, the cover catalog reduces to two columns, and forms remain labeled. The reader controls retain their existing responsive behavior. The second configured archive keeps its own collection name and introduction.

## Preview

Use the isolated Arkhiv launcher from this exact checkout for synthetic application development. For a frontend review against an already qualified API, set `ARKHIV_DEV_API` when starting Vite; it defaults to `http://127.0.0.1:8080`. This option changes only the development proxy. Production keeps its existing API route and access policy.

The retained private review URL is `http://10.0.0.50:54431/`. It proxies the existing public API; it displays only public-approved documents. No `design` parameter is needed.

Owner-supplied references remain in [docs/inspo](../inspo/). Their original bytes are tracked as research material. Source URLs and reuse licenses were not supplied. Reference images are not bundled into the application.

## Release boundary

This change prepares the selected interface for review. Verification and hosted CI are recorded in the PR. Deployment is a separate action; passing frontend checks does not establish production rollout. Existing preservation storage, publication approvals and update jobs are unchanged.
