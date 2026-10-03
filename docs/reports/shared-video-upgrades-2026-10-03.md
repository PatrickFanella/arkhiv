# Shared video release and upgrade qualification

Observed on Kvant and the isolated Kuznya test VM, 2026-10-03. This qualifies
registry publication and hosted upgrade proposals. Remaining browser checks are
listed separately; no production cutover is claimed.

The published `@subcult/archive-viewer` releases are 0.1.0 and 0.1.1.
Hosted release runs [12470](https://git.subcult.tv/subculture-collective/VHNESS/actions/runs/12470)
and [12509](https://git.subcult.tv/subculture-collective/VHNESS/actions/runs/12509)
completed successfully, including publication and artifact upload. The retained
[0.1.0 receipt](../releases/release-0.1.0.json) and
[0.1.1 receipt](../releases/release-0.1.1.json) identify source commits, exact
package checksums, compatibility and immutable engine digests. Downloaded registry
tarballs match the hosted receipts. Consumer installs use a separate package-read
credential; proposal automation uses a separate repository-write credential.

Both consumers adopted the package in merged PR #2. Hosted proposal
[run 12525](https://git.subcult.tv/subculture-collective/VHNESS/actions/runs/12525)
at `2494de967c57084ba4bc170964d8aee9e664280f` checked both consumers and created
[neutral PR #3](https://git.subcult.tv/subculture-collective/video-archive-template/pulls/3)
and [Democracy Now! PR #3](https://git.subcult.tv/subculture-collective/democracy-now-tv/pulls/3).
Repeat [run 12534](https://git.subcult.tv/subculture-collective/VHNESS/actions/runs/12534)
completed successfully and returned the same PRs. Their heads remained
`0e2d07a515a5ac10e44ab9007e847a0125660b18` and
`54292e3f0954d45277debc5f35b30fe5f310a39a`, respectively. These are proposals,
not merged or deployed revisions. Repository-specific queued CI is distinct from
the successful checks executed by the hosted proposer.

Actual private images were built from those exact clean PR heads, with the
released engine and package labels checked. Each runs on its own internal network
against its own restored metadata database. The previous 0.1.0 images remain
available for comparison. No workers or acquisition were started, and neither
test app mounts media storage. Application filesystems are read-only.

## Observable shared fix

In the isolated browser, delaying a search request on 0.1.0 left 24 previous
episode cards visible beneath the new query. In T3's collaborative browser, the
same five-second request delay on each 0.1.1 consumer immediately produced zero
cards, `aria-busy=true`, and the status “Loading episodes…”. After completion,
both returned zero results and `aria-busy=false`. The temporary browser fetch
wrappers were restored. Democracy Now!'s logo and portrait loaded successfully;
the update retained each consumer's page layout and assets.

On-demand playback in both 0.1.1 consumers reached ready state 4, advanced from
about 8–9 seconds to 16–17 seconds, and accepted a seek to 90 seconds. Both used
the direct Archive.org `dn2026-1002_vid.mp4` URL and displayed its source page.
Tune in selected the existing server schedule and direct source asset in each
consumer. Further replay, keyboard, error and narrow-screen checks remain open.
The recorded clips do not establish subjective audiovisual fidelity or a full
episode run.

## Browser automation limitation

After the owner reopened the preview, T3 navigation and inspection worked again.
Viewport resizing then timed out and subsequent snapshot/evaluate calls failed
on the client. Fresh navigation and a new tab did not restore inspection. Both
private `/api/ready` endpoints still succeed. This is retained as a browser
automation limitation, not an application failure or completed mobile proof.

Full isolated source verification passed before the final registry/documentation
update: Go format/vet/race/database checks, 18 Python checks, 13 shared-viewer
tests, two reference tests, builds and a fresh packed consumer. The outstanding
browser matrix prevents closing issue #41.
