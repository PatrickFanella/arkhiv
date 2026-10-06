# Cross-media viewer qualification, October 5, 2026

Source: `7f5c2484a77df79d2f17bda12b053724aeb34541`. This follow-up extends
viewer feedback to audio and the mixed print/audio/video reference interface.
Arkhiv PR #117 owns the shared change. The earlier video PRs are merged.

## Published artifacts and local gates

- SDK `@subcult/archive-viewer@0.3.1`; independently downloaded package SHA-256
  matches its release receipt.
- Core `0.2.1`, immutable image digest
  `sha256:c86a95818a313866e7dea5cb9025a744b34fd0f9107eb1e47c60aa021cb63e42`.
- Combined engine digest
  `sha256:162d7d034a8fb4d5b0ac0f2a102b505147da6c4cf878ceaaf84429c3f2883641`.
- Shared `make verify && make verify-curation` passed in the isolated T3
  `vhness` profile: Go vet/race and disposable database integrations, 37 SDK
  tests, Python checks, reference checks/build, and 19 curation checks.
- Bill Nye TV #13, Democracy Now! TV #13 and Video Archive Template #12 adopt
  exact registry SDK/lock and immutable engine/core receipts. Each full
  `make verify` passed, including frozen installation, pin checks, typecheck,
  lint, application tests and build.

Hosted CI remains temporarily owner-waived while Kvant is unavailable.
`[skip ci]` titles and commits suppress new runs. The owner handles queued-run
cancellation in the UI. This is not a hosted-CI pass or a deployment.

## Private browser fixtures

The separately owned cross-media preview uses generated test-pattern video,
PCM tones and a three-page PDF. Complete generated source responses and file
inventories are retained. Audio/PDF originals have explicit hash-bound manifests,
collection quota/reserve and copied, independently rehashed fixture recovery
receipts. These are fixture recovery checks, not off-host source recovery proof.
Democracy Now! remains source-streaming; no source video was requested.

Native T3 browser evaluation was used after snapshots and focused clicks failed.
Screenshots later resumed. Actual measured widths were 1402 and 427 CSS pixels;
the preview's configured dimensions did not always match measured dimensions.

Observed behavior:

- Volume `0.27` and mute persisted between two audio recordings, into video
  playback and after reload. Generated audio decoded with readyState 4 and
  advancing time. Reviewed on-demand set selection began at its second track;
  the recording seek control remained available.
- The station has volume control and disabled track jumps, with no position
  scrubber. Pause displayed “Station continues.” Resume across a program
  boundary loaded the current recording, with measured drift `0.125` seconds.
  The shared schedule revision remained unchanged. Active replay, joining and
  paused labels were distinct, and original dates remained visible.
- Grid/table switching produced a five-row semantic table with caption,
  original date, creator, available formats and actions. A creator filter
  returned the two matching audio items; a combined date bound returned zero.
- Collection/item/representation deep links reopened current scoped metadata.
  Local favorites survived reload and were absent from the other collection.
  A denied clipboard operation showed the selectable exact item link; Bluesky
  and email links remained compose intents.
- Print displayed the original date, three pages, variant and filename.
  Indexed “Knownword” search returned all three pages. Page 2 rendered its
  visible text, the readable-text panel matched it, and 150% zoom rendered
  successfully. Focus mode exited with Escape. No document-level overflow was
  observed; zoomed PDF scrolling stays within its canvas wrapper.

The preview initially produced zero animation frames, leaving PDF rendering
pending. After reopening it, frames resumed and the reader cleared its busy
state. Rendering and text were then checked; the initial stalled observation
is not treated as successful evidence.

## Optional account and chat fixture

A generated RSA OIDC provider, temporary assertion keys, separate personal
database and restricted nonowner serving role were created only in the private
preview. The real session service verified nonce and PKCE through its callback.
No live provider was selected or activated. Runtime state and credentials remain
outside Git.

Catalog account controls demonstrated optional sign-in, explicit unchecked
import review, preparation/application of one browser favorite, retention of
the browser copy, alias update, synchronized audio and print favorites, export,
logout-all and reviewed account deletion. After deletion, database counts were
zero for profiles, favorites, sessions and chat bodies. The recovery journal
retained two fixture records; the browser favorite remained. The opaque session
cookie was not visible to page JavaScript.

The template's `ViewerTools.tsx` was copied unchanged into a private qualification
page, paired with the released SDK and real generated video schedule. Posting
was disabled without a joined current program and while paused. A current
authenticated viewer sent a message; literal `<b>` markup stayed plain text.
The message displayed original date and replay schedule identity. Its delete
control removed it from the serving view. Backend disposable integrations
separately cover authorization, idempotency, scope, rate limits, moderation,
retention and recovery; this browser check does not claim every moderator UI
interaction or every brand's rendered account/chat interface.

Audio chat needs an explicit audio/video room distinction in mixed collections.
Print documents do not acquire a fabricated channel program. Branded print/audio
adoption and any further chat contract remain separate tracked work.

No public deployment, collection-wide qualification, human listening, real
source audiovisual fidelity, other-browser coverage or sustained run is claimed.
