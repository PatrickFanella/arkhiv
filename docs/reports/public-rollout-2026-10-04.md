# Arkhiv public rollout — October 4, 2026

The owner authorized continuation through public rollout and selected the final
hostnames. The former print site is Anarchist Archive; Arkhiv names the mixed
reference interface and its independently released headless core.

| Public endpoint | Actual verified behavior |
| --- | --- |
| https://arkhiv.subcult.tv/ | 6,707 approved items: 6,582 Democracy Now! programs, 100 Bill Nye episodes and the original 25-document print pilot with 26 PDF representations |
| https://arkhiv-api.subcult.tv/api/version | Public API v1; catalog, print, audio and video modules v1; no frontend assets exposed on this hostname |
| https://anarchist-archive.subcult.tv/ | Existing 699-document public print catalog retained; original document links redirect here with reader query parameters preserved |
| https://bill-nye.subcult.tv/ | Distinct Bill Nye interface, 100 original MP4s, on-demand playback and a seven-day scheduled replay |
| https://riot-grrrl.subcult.tv/ | Distinct Cut & Paste interface, 23 hash-bound reviewed PDFs, unknown source rights retained |
| https://democracy-now.subcult.tv/ | Distinct news interface, 6,582 reviewed programs, direct Archive.org playback and a seven-day scheduled replay |

The mixed reference contains the initial qualified print migration, not the
entire 699-document donor catalog. Private audio remains absent from the public
catalog. Riot Grrrl is live. The owner’s private music listening and public
Dedhed access decisions remain pending; the private music preview is healthy
but T3’s collaborative preview navigation still fails.

All three new video/mixed API instances run the same published core 0.1.1 image:
`git.subcult.tv/subculture-collective/archive-core@sha256:cb44da0d015ad52812f4247749be754f0e09af41b65fed9515dacac11f1c10c6`.
Each public server uses a SELECT-only PostgreSQL role and explicit collection
scope. Bill Nye and Democracy Now! have separate restored schedule pointers.
No acquisition worker or public operator endpoint was started. Democracy Now!
has an empty local media mount; its local relay endpoint returns 404.

The mixed frontend image is
`sha256:122175672e8c401e3190ffa1181e24b356657e48dbfff282fbd24f8d158bf48e`,
built from `d3c48289a3039adeb9e9046432614cc9066e735a`. Bill Nye’s frontend image is
`sha256:910c849176edfd6b7c8de0440c9a78d11b7dd64ac139d00b60f3223491ecdee4`;
Democracy Now!’s is
`sha256:7b322d81bc79dcb979d7bc7a5b443258f726f1e1c6abab4b60222c16d2da8b74`.
The branded wrappers come from `5c2a422d015719e0972273f29cf9b39f25138a66`.
Their complete static file manifests match their qualified consumer images
exactly. Their final images contain the interface and Nginx, without Go or Node.
[The extraction recipe](../../deploy/branded-frontend.Dockerfile) rebuilds only
this wrapper; consumers keep ownership of their frontend code and dependencies.

Qualification caught Nginx retaining old core addresses after container
replacement, including a Bill Nye proxy reaching the news core when Docker
reused an address. The fixed proxy resolves service DNS every ten seconds.
Both cores were then replaced while the exact frontend images remained running;
each proxy still returned the correct collection and catalog. The earlier
failure and subsequent proof remain in the private qualification receipts.

Actual HTTPS checks rendered a 1,152 by 891 original PDF canvas on Arkhiv with
no reader alert or console error. Bill Nye’s Flight video advanced on the public
domain and was paused after inspection; its candidate additionally sought to
217 seconds and continued to 233.59. Democracy Now!’s October 2 recording
advanced directly from Archive.org to 59.19 seconds on its public domain and
was paused. These checks establish playback behavior, not owner listening
fidelity or a long-term observation window.

An independently hosted browser fetched the headless API and received HTTP 200
with 25 print items. Public CORS allows anonymous GET/HEAD and range headers,
without credentials; its real OPTIONS preflight returns 204. Private audience
rules remain unchanged. Some default Python HTTP requests receive an edge 403;
the actual browser and curl requests passed. This edge behavior is not an API
server availability failure and remains a client qualification limitation.

Representative HTTP checks confirmed a 799,756-byte original PDF with SHA-256
`fdd6f8e946f28b2607048c8edaaf3cf6fe56963c8c1866f440bd0e2d8eeb5be2`,
Bill Nye range responses (206, 1,024 bytes), denied foreign collection queries,
denied operator status, no public audio and denied local Democracy Now! relay.
The required isolated `make verify` passed after the Nginx DNS change.

The new public runtime is isolated from retained private services. Its initial
metadata restore contains every original imported table and both source schemas,
including historical schedule pointers. The independently copied 299,068,322-byte
checkpoint has SHA-256
`fd48eff47083ce45582d9d5e6a7421c0682402cfc922ab53c4d8c0558ba1eb5a`.
All 100 Bill Nye originals were verified again after transfer: 13,763,456,996
bytes, within the approved 16 GiB cap and 1 GiB reserve.

Spektr I/O pressure blocked initialization of the first unused candidate
database and slowed its media copy. Only that task’s media copy was cancelled;
the partial destination and failed startup evidence remain. The active new
metadata and media use Almaz’s root volume. Existing print/video/music stores,
services and backups were not relocated or deleted. The unused first candidate
is stopped with its data retained; the active database remains separate.

The post-replay public checkpoint `public-20261004T084653Z` contains four databases.
It reached NUC with matching hashes and was fetched from that independent host
into fresh Kuznya databases. Every recorded public and imported source table
count matched. Restored public services returned the same 6,707/100/6,582/23 item
counts, denied operator access, exposed no audio, served the same original PDF
hash and Bill Nye range bytes, and denied local Democracy Now! relay. All 23
Riot Grrrl restored HTTP PDFs matched their hashes and sizes. Temporarily
withdrawing a review in the isolated restore caused the actual maintenance
operator to reject replay before changing schedules; the review was restored.

Monitoring, seven-day replay planning and bounded independent backups passed
actual first runs. Their Almaz timers are enabled: monitoring every five minutes,
replay at 00:10 UTC and backup at 00:30 UTC. The monitor passed a natural timer
activation. Daily timer next-run times were inspected; a natural daily cycle and
long-term reliability remain unobserved. Checkpoints have a 16 GiB cap, 2 GiB
write allowance and 1 GiB reserve; budget exhaustion fails without deleting
backups. [Maintenance instructions](../public-operations.md) record the separate
operator credentials and restore proof boundary.

Bill Nye and Democracy Now! now have reviewed API-only pin adoption PRs. Their
frontend source, package files and lockfiles remain unchanged. The registry
keeps their legacy combined-engine/component lane separately, and CLI selection
uses the release artifact kind so each future release reaches its correct lane.
The adoption branches remain in review; registry activation and hosted checks
must be tracked separately from the already observed public deployment.

Private deployment receipts live under
`/mnt/spektr/server/data/arkhiv-platform-20261004/qualification` on Almaz.
[The reviewed ingress configuration](../../deploy/public-sites.Caddyfile)
preserves the Anarchist route and legacy document/asset APIs. Ingress rollback
must reverse only the owned Arkhiv/brand blocks after checking current hashes;
never restore an old whole configuration over unrelated later edits. Image
rollback retains previous immutable frontends and core artifacts. Catalog and
access-history rollback use independent checkpoints without deleting originals.
