# Standalone products and shared engine

Confirmed by the owner on 2026-10-03: Democracy Now! and Bill Nye will use
separate repositories with a shared engine. Each app owns its design system,
frontend, deployment, and release cycle. VHNESS's eventual name and visual
identity remain undecided.

This extends the [original video plan](0001-video-archive-and-channel.md).
It records a delivery plan; the standalone apps have not been implemented.
Existing pilot qualification continues. Public deployment remains a separate
owner decision.

## Repository and runtime boundaries

| Repository | Ownership |
| --- | --- |
| Existing VHNESS repository | Versioned Go engine, migrations, ingestion, complete metadata/history, editorial decisions, playback resolution, schedules, preservation, recovery tools, API contracts, and qualification fixtures. Retain the existing frontend as a development/reference client. |
| Democracy Now! app; final repository slug pending | Independent React frontend, supplied assets, news-archive design system, dated library, episode pages, replay guide, browser checks, and deployment configuration. |
| Bill Nye app; final repository slug pending | Independent React frontend, science-TV design system, episode guide, CRT shell, browser checks, and deployment configuration. |

Release the engine as a versioned Go binary/container. Do not fork the backend
into the app repositories or require cross-repository source checkouts to
build an app. Share typed API contracts and playback behavior where needed;
visual components and tokens belong to each app. Pin exact engine/client
versions and verify compatibility, migrations, and rollback.

Start with one engine instance and database per product. Each owns its worker
checkpoints, editorial decisions, shared schedule, configuration, credentials,
logs, and backups. Sharing code does not mean sharing mutable channel state.
Enforce the configured collection at API, playback, schedule, and operator
boundaries; removing a frontend collection selector is insufficient.

Both products retain on-demand viewing and a scheduled replay channel. Prefer
same-origin API routing. Viewer pause/seek/episode selection never changes the
shared schedule. Archived programs retain their original dates and replay
labels.

The engine is the white-label foundation: stable content/playback contracts,
collection policy, runtime configuration, and product-owned presentation.
A future VHNESS app can use these contracts once its identity is decided.
A universal theme editor or tenant platform is outside this first split.

## Democracy Now! design

Use the supplied logo variants and Amy Goodman photography. Build a news
archive with a white or warm-white reading surface, black/charcoal text, red
emphasis from the assets, clear rules, strong headlines, readable descriptions,
and prominent original broadcast dates. Inspect each logo on its intended
background before finalizing tokens; the supplied SVG and raster treatments
have different colors.

The library leads with dated broadcasts, search, year navigation, and episode
descriptions. The channel leads with the current replay, original date,
Tune in/Rejoin, and upcoming programs. Episode pages retain source links and
credits. Distinguish starting from the beginning from joining the schedule.

Democracy Now! remains permanently source-streaming only. Store complete
metadata, every file entry, history, decisions, and schedules. Never acquire,
proxy, persistently cache, or transcode its video. No service-worker video
cache or silent local fallback is introduced. Browser playback buffering
remains normal source streaming.

## Bill Nye design

Use a 1990s science-TV direction based on the series logo, colorful title
graphics, and supplied CRT references. Build a charcoal/plastic cabinet around
the player, tactile controls, speaker-grille detail, a restrained phosphor-style
status display, and science graphics such as orbit lines and lab markings.
Draw blue, yellow, red, purple, and pink accents from the series references.

The channel is the main television experience. Keep the episode guide easy
to reach, with clear titles, descriptions, and eligible playback actions.
Power/Tune/Rejoin controls must describe their actual behavior and support
keyboard use and accessible labels.

Create the cabinet with CSS/SVG rather than overlaying controls on a reference
photograph. Preserve the original picture proportions with `object-fit:
contain`; do not stretch or crop the video to fit a CRT frame. Apply decorative
scanline/glow treatment to the frame and surrounding UI by default. Any optional
picture effect must be switchable and keep captions and controls readable.
Respect reduced motion; watching must not require flashing static, animated
noise, or startup sounds. On narrow screens, prioritize picture and controls
over cabinet ornament. Use readable UI text alongside decorative typography.

Loading, network interruption, unavailable episodes, schedule gaps, and
exhaustion must stay understandable within the TV design.

The preservation authorization remains exactly `01 Flight.mp4` and
`02 Earth's Crust.mp4`: 274,780,857 bytes, a 512 MiB collection budget, and a
1 GiB free-space reserve. Originals stay under
`~/.local/state/vhness/private-pilot/media/bill-nye`; independent backups stay
under `/home/onnwee/Backups/vhness-private-pilot` on NUC. This split does not
expand the manifest or move originals. Complete catalog metadata may describe
other episodes, but playback eligibility reflects approved assets. Unknown
episode dates remain unknown.

## Asset intake

Read-only inspection on 2026-10-03 found four Democracy Now! assets and fifteen
Bill Nye reference images in the main checkout at
`/home/onnwee/Work/Subcult/vhness/democracy-now/` and `bill-nye/`. They are
currently untracked there; fetched remote `main` does not contain them.
Preserve those originals and unrelated checkout state.

Record copied assets' original filenames, SHA-256, supplied sources, intended
use, and known provenance. Unknown provenance/rights remain unknown. Copy
selected Democracy Now! identity assets to its app repository. Keep Bill Nye
inspiration separate from shipped assets; reference photos need not enter the
frontend bundle. Supplied assets alone do not establish public-release
clearance or official affiliation.

## Delivery order and exit gates

1. **Finish existing private qualification.** Complete the collection-wide
   metadata inventory and final independent restore. Retain prior engine,
   playback, channel, preservation, and owner audiovisual evidence.
2. **Establish the engine release boundary.** Version the API/client and
   artifacts, add instance-specific configuration and collection enforcement,
   and prove that another product's collection cannot be browsed, resolved,
   scheduled, or acquired. Verify upgrade and rollback compatibility.
3. **Build Democracy Now! first.** Create its separate repository, intake
   selected assets with provenance, implement the news-archive system, and
   connect both modes to its own engine instance. Qualify dates/replay labels,
   search, source playback, failures, keyboard use, narrow-screen layout, and
   absence of application-managed video storage.
4. **Build Bill Nye.** Create its separate repository, establish science-TV
   tokens and the CRT shell, and connect both modes to its own engine instance.
   Qualify the approved originals, controls, proportions, eligible captions,
   reduced motion, keyboard use, and narrow-screen layout. Repeat independent
   restore within the existing manifest and budget.
5. **Release independently when authorized.** Pin tested engine versions and
   record local checks, hosted CI, browser/device proof, audiovisual review,
   restore, and deployment separately. Decide public audience/source use
   before rollout. Leave VHNESS naming and visual design deferred.

Earlier engine/browser evidence does not automatically qualify the new
frontends or deployments. Track the engine boundary and each product as
separate Gitea tasks with native prerequisites and reviewable PRs.

## Tracking

- [#28: shared engine release boundary](https://git.subcult.tv/subculture-collective/VHNESS/issues/28)
- [#29: Democracy Now! app](https://git.subcult.tv/subculture-collective/VHNESS/issues/29)
- [#30: Bill Nye app](https://git.subcult.tv/subculture-collective/VHNESS/issues/30)

These tasks are planned and open. The engine task precedes both product
integrations; each product task will link its implementation issues and PRs
after its repository is created.
