# Video library qualification — October 6, 2026

Bill Nye TV, Democracy Now TV and the video archive template were exercised as
separately built consumer bundles in the owned Sputnik mixed-media preview.
Their collection constants were changed only in qualification copies to use two
clearly labeled generated episodes. Production collection configuration and
source acquisition permissions remain independent.

## Observed behavior

All three consumers sent the combined topic `Test pattern`, language `en` and
minimum duration `20` to `/api/episodes` with collection scope and page 1 intact.
Both fixture episodes matched. Adding an absent creator produced zero results
and an explicit empty message. Grid and table views use the same result set.
Tables expose captions and column/row headers; keyboard Enter switched the Bill
library view. Table selection survived reload and was restored by each bundle
on this qualification origin. Production origins and collection keys remain
independent.

Original dates, source credit, topics, language, duration, format, dimensions,
file size and source title origin were visible. Unprovided rights/license fields
were omitted. No editorial correction or permission was inferred from an absent
source value. Source metadata, selected-file binding and collection/public scope
are covered separately by the shared API and component gates, including filters
before pagination with enough records to require a second page. The two-episode
browser fixture alone is not evidence of full-catalog pagination.

Saved references survived filters and reload. An unavailable qualification
reference was retained with an explicit unavailable message and a remove action,
then was actually removed through the Bill interface without removing the valid
saved episode. The template's rejected-storage-write check retained the new
favorite on the current page, left the disk copy intact and displayed:
“Browser storage is unavailable. Changes are kept only on this page; export them
before leaving.” A subsequent successful write cleared the temporary state.
Browser favorites remain usable without sign-in and confer no media access.

A held catalog request displayed a loading status in the Democracy Now library.
A controlled HTML 503 exposed raw JSON parser text before the shared fix.
After SDK 0.4.2, all three rebuilt bundles displayed the retryable message
“The archive is temporarily unavailable. Please try again.” Removing the
qualification-only outage and changing the filters recovered the real scoped
catalog. Successful unreadable JSON, missing items and structured source errors
have separate regression checks.

At a 390 CSS-pixel frame width, each consumer restored table view without page
overflow. The usable document width was 375 pixels after the scrollbar. The
Democracy Now table wrapper was 340 pixels wide with 370 pixels of content; the
template wrapper was 336 pixels wide with 362 pixels of content. Their contained
horizontal scrolling retained access to the columns and actions. Narrow padding
and stacked metadata applied. This is a responsive desktop-browser check, not
Android, iOS or a physical-device claim.

## Source and release checks

SDK 0.4.2 was published from
`1619d86a615bf4bd69c7a601bf585ff5b3a6ef89`. The registry package was downloaded
independently and matched SHA-256
`6eaf356a37bfcde823227f61f0fe42a50385dde4502efdd35eab7ecd675bb04d`.
The SDK receipt records its registry and compatibility with Core 0.3.0;
reference engine 0.4.0 remains a separate artifact.

Root `make verify` and `make verify-curation` passed in the disposable `vhness`
environment: 45 SDK checks, packed ESM consumer composition, Go/race/database
checks, 51 operator tests with the two separately qualified real personal-restore
tests skipped in the ordinary gate, four reference-web checks, and 19 frozen
Python 3.12 curation checks. All six consumers passed `make verify` with updated
frozen registry locks and independent receipt checks. Native consumers used
their own `arkhiv`/`dedhed` profiles and explicitly disposable test databases.
Hosted CI remains owner-waived while Kvant is down.

This qualifies the video-library source issues. Playback fidelity, continuous
two-viewer channel behavior, device testing, synchronized accounts, consumer chat
qualification, live migration and public activation retain their own evidence
requirements.
