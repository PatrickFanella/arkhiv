# Video viewer feedback

Recorded 2026-10-05 from the owner's Bill Nye TV notes, written before the
headless change. This is a feedback intake and proposed implementation order.
It does not claim that reported bugs were reproduced on the current deployed
apps, or that the requested features are implemented.

The [headless design](0003-multimedia-platform.md) still applies. Arkhiv owns
shared API contracts and optional viewer behavior. Each consumer owns its
interface and deployment. Shared changes need independent adoption and browser
verification in Bill Nye TV, Democracy Now! TV, and the video archive template.

## Current source evidence

Inspection of this worktree on 2026-10-05 found:

- `packages/viewer/src/Viewer.tsx` renders text artwork rather than thumbnails.
  `model.ts` has no thumbnail field. The Bill Nye thumbnail report still needs
  reproduction in its consumer frontend.
- `Player.tsx` labels viewing modes as SHARED REPLAY and ON DEMAND and shows the
  original broadcast date. `ChannelPanel` offers Tune in and a paused rejoin
  action. The requested stronger distinction is a usability improvement over
  existing labels, rather than a wholly missing viewing-mode model.
- The player uses native video controls in both modes. Channel clock updates
  correct playback position while playing. Seeking therefore needs an explicit
  product rule instead of competing with channel synchronization.
- `Screen` keys the player by asset/session identity. The video player does not
  retain volume or mute preferences across those remounts.
- Video library controls provide title/description search and a year filter.
  Cards show title, date and description; episode details add creator. Playback
  data already includes selected file duration, dimensions and byte size, while
  the backend episode contract also includes source identity, rights, license
  URL and file inventory. Complete stored metadata is broader than the current
  viewer projection and is not automatically safe to expose publicly.

These findings apply to the optional shared viewer at the inspected revision.
They do not establish the current behavior of every consumer or public route.

## Feedback, ownership and acceptance

| Owner note | Proposed work and owner | Evidence needed |
| --- | --- | --- |
| Thumbnails not working | Reproduce in Bill Nye TV, then add or repair an optional reviewed artwork reference in the shared API/client where needed. Render an explicit fallback for missing or failed images. | Real source artwork loads on cards and player posters; failures leave usable titles and actions. Bill Nye episodes share a parent source item, so parent artwork must not be presented as an episode-specific frame. |
| Make broadcast versus on-demand obvious | Shared viewing state with consumer-owned prominent labels near the player. Use "On the channel · Scheduled replay" and "On demand"; retain original dates. | Labels stay correct after choosing an episode, watching from the beginning, pausing, rejoining, and crossing program boundaries. Archived content is never described as a live original broadcast. |
| Maintain volume across videos | Optional shared video preference helper, adopted by each frontend. Save volume and mute per browser origin, validate stored values, and tolerate unavailable storage. | Preferences survive episode changes, channel boundaries, mode switches and reloads. Failed storage does not prevent playback. |
| Channel scrub, pause and rejoin | Use the channel behavior proposed below in optional components and consumer interfaces. | Pause/resume and rejoin land at the current program; on-demand seeking remains independent; viewer actions never write shared schedules. |
| Accounts | Separate identity and personal-data design, with an explicit backend owner. Accounts remain deferred in the current roadmap pending that design. | Agreed account purpose, authentication, authorization, data ownership, export/deletion and recovery behavior before implementation. |
| Chat | Separate product design for channel-scoped chat. Decide whether messages refer to the shared channel or individual episodes. | Defined moderation, reporting, abuse controls, retention and identity requirements; clear treatment of on-demand viewers. |
| Favorites | Start with browser-local favorites using stable collection and episode IDs. Add account synchronization only after the account design. | Favorites remain usable across searches and reloads, show unavailable items honestly, and do not grant playback access or alter editorial data. |
| Easily share social links | Consumer-owned canonical episode URLs, Copy link, and native sharing where available. Episode sharing defaults to on-demand; channel sharing explicitly joins the current channel. | Opening copied links in a fresh session selects the intended episode or channel. Bill Nye links distinguish episodes that share the same Archive source page. Timestamp links are an optional explicit action. |
| Theater mode | Consumer layout toggle that enlarges the existing player and collapses surrounding content. | No second player or playback restart; volume and viewing state survive; keyboard and small-screen controls remain usable. |
| Bill Nye power icon and logo spacing | Reproduce and fix in `bill-nye-tv` frontend source. | Power icon centered in its control, logo separated from the top edge, checked at desktop and mobile sizes. An engine release must not replace the brand's interface. |
| More search filters | Inventory available reviewed metadata first. Extend collection-scoped API filters only where data supports them; consumers own filter layout. Candidates include date range, topic, language, creator and duration. | Filters apply to the full catalog before pagination, combine predictably, expose missing data honestly, and preserve public access restrictions. Do not invent seasons, episode numbers or dates from unrelated parent metadata. |
| Grid and table view | Consumer-owned grid/table toggle over the same query and pagination, with a browser-local preference. | Both views retain search, filters, results and selection; table columns adapt to smaller screens and preserve accessible actions. |
| More metadata if available | Add a public-safe episode detail projection as needed, then consumer rendering for credits, rights/license, duration, format, dimensions, source links and available topics/languages. | Only available values appear, unknown dates remain unknown, source values and editorial corrections remain distinguishable, and private review notes/paths are excluded. |

## Recommended channel controls

Keep pause, volume, mute, captions and fullscreen. Remove the channel scrub bar
and seeking shortcuts. Build accessible controls rather than merely hiding
native controls and losing their other functions. Display elapsed/remaining
program time as read-only information.

While following the schedule, show "On the channel · Scheduled replay". On
pause, show "Paused · Channel continues" and keep a prominent "Rejoin channel"
action. Resume and Rejoin both request current channel state and join the
current program at its current offset, even if a boundary passed while paused.
Choosing "Watch from beginning" changes to on-demand and exposes normal seeking.
On-demand playback keeps a "Join channel" action nearby.

Buffering, autoplay denial, gaps and unavailable sources are distinct states.
Only claim the viewer is following the channel after playback has started and
its position is within the qualified synchronization tolerance. Existing
`paused` state alone does not prove that the picture is at the scheduled
position. The public channel remains an archival replay, even when the viewer
has joined its current scheduled position.

## Proposed delivery order

1. Reproduce thumbnail and Bill Nye spacing reports; fix confirmed defects.
   Add volume/mute persistence and prominent viewing-mode labels.
2. Implement the channel controls and state transitions, canonical sharing,
   and theater mode. Verify joins, boundaries, browser autoplay and keyboard
   behavior in each adopting consumer.
3. Add public-safe metadata, supported filters, grid/table views and local
   favorites. Share API contracts where useful while preserving brand layouts.
4. Design accounts and chat as separate work. Local favorites and link sharing
   can ship independently of those services.

## Issue plan

[Arkhiv #105](https://git.subcult.tv/subculture-collective/arkhiv/issues/105)
tracks the complete plan. Gitea issues own task state and acceptance evidence.
Existing broad adoption and qualification issues remain separate and open.

| Repository | Issues |
| --- | --- |
| Arkhiv | [Artwork #106](https://git.subcult.tv/subculture-collective/arkhiv/issues/106), [volume/mute #107](https://git.subcult.tv/subculture-collective/arkhiv/issues/107), [channel controls/state #108](https://git.subcult.tv/subculture-collective/arkhiv/issues/108), [metadata/filters #109](https://git.subcult.tv/subculture-collective/arkhiv/issues/109), [account design #110](https://git.subcult.tv/subculture-collective/arkhiv/issues/110), [chat design #111](https://git.subcult.tv/subculture-collective/arkhiv/issues/111) |
| Bill Nye TV | [Playback/sharing/theater #7](https://git.subcult.tv/subculture-collective/bill-nye-tv/issues/7), [browsing/favorites #8](https://git.subcult.tv/subculture-collective/bill-nye-tv/issues/8), [power icon/logo spacing #9](https://git.subcult.tv/subculture-collective/bill-nye-tv/issues/9) |
| Democracy Now! TV | [Playback/sharing/theater #8](https://git.subcult.tv/subculture-collective/democracy-now-tv/issues/8), [browsing/favorites #9](https://git.subcult.tv/subculture-collective/democracy-now-tv/issues/9) |
| Video archive template | [Playback/sharing/theater #7](https://git.subcult.tv/subculture-collective/video-archive-template/issues/7), [browsing/favorites #8](https://git.subcult.tv/subculture-collective/video-archive-template/issues/8) |

Consumer playback issues depend on the shared artwork, volume and channel
tasks; consumer browsing issues depend on the metadata/filter task. The tracker
depends on all child outcomes. Accounts and chat are design tasks and do not
block the earlier improvements. Print/audio consumers are outside this video
interface scope. All issues remain open pending their required evidence.

Required implementation gates remain `make verify` and `make verify-curation`,
plus each changed consumer's checks. Browser behavior, actual audiovisual
playback, hosted CI and consumer adoption require their own evidence. Democracy
Now! remains source-streaming only; thumbnail support does not authorize video
acquisition, caching or frame extraction by metadata workers.
