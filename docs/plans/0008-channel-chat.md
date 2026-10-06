# Channel chat

Proposed 2026-10-05 for [issue #111](https://git.subcult.tv/subculture-collective/arkhiv/issues/111).
Status: scope and proposed retention accepted by the owner on 2026-10-05.
The owner selected "Accept the proposed designs" in this task's review.
No chat service or public posting is enabled.

## Product decision

Start with authenticated discussion of the shared scheduled replay channel.
Use the [account design](0007-accounts-and-personal-data.md). Anonymous users may
read approved chat when the deployment owner enables it, but only signed-in
users can post. On-demand viewers can read the channel discussion in a clearly
labeled panel; posting asks them to join the channel. Episode comment threads
are separate future work.

Each deployment owns its chat data and moderator roster. Arkhiv supplies an
optional chat module and versioned contracts; brands own panels, sign-in prompts,
moderation interfaces and styling. A shared provider login does not share rooms
or messages between brands.

## Program context

Bind a room to a deployment and collection/channel. Each message records server
time and the schedule revision, program start and stable episode identity.
Server validation rejects posts referring to a stale or unavailable program.
The frontend refreshes the channel and lets the user decide whether to resubmit.
Posting never edits the shared schedule.

At a boundary, keep the room but visibly separate the new program. Earlier
messages retain their original program/date context and may be read within the
retention window. Replaying the same episode later creates a new program context;
old conversation is not presented as happening now. Paused viewers see the
channel's current program and must rejoin before posting to that program.

## Transport and data

The first API uses bounded cursor pagination and ten-second polling, with
backoff and an explicit unavailable state. A messaging service outage must not
interrupt video. A later push transport can replace polling after measured need.
No persistent sockets are required for the first version.

Proposed routes under `/api/chat/v1` provide room capabilities/current context,
message pages, authenticated message submission with idempotency keys, own-message
deletion, reports and separate moderator actions. Every read checks room access;
every write checks subject, scope, posting eligibility and server program context.
Keep chat serving roles separate from source/editorial/schedule roles.

Store opaque message/user/room IDs, a public alias, plain-text body, program
context, timestamps and moderation state. Do not publish email, provider subject,
IP address, reports or private moderation notes. Limit messages to 1,000 Unicode
characters and 4 KiB of UTF-8. Render text without HTML, embeds or automatic
media fetching. Aliases must not imply verified staff or official affiliation.

## Moderation and abuse handling

The deployment owner appoints moderators and remains responsible for reports.
Activation requires a named moderator roster and a visible reporting/contact
procedure. Moderators can hide messages, mute users temporarily, ban posting
and reopen reviewed reports. Authentication alone never grants those powers.

Proposed initial limits are five messages per minute per subject and a separate
bounded room-wide flood limit. Reports are deduplicated by subject/message and
limited to three per hour. Rate counters expire and do not become permanent
IP logs. Measure false positives before enabling stricter network limits.

Users can block other users locally, report a message and delete their own
messages. A hidden message is removed from public responses immediately;
moderation evidence follows the bounded private retention below. Moderator
audit records contain actor, action, target, reason and time. Clients never
receive reporter identity. Define an operator emergency switch to disable posts
or the whole chat module without stopping playback.

## Retention, deletion and recovery

Proposed retention is 30 days for messages and resolved report evidence, with
unresolved reports reviewed before expiry. Deleted bodies disappear from the
serving store immediately. A 35-day personal-backup window and 90-day opaque
deletion ledger follow the account proposal. The owner accepted these periods
on 2026-10-05; public activation remains separate.
They never authorize pruning archival source records or media.

An account export includes that user's messages and report submissions with
third-party private fields removed. Account deletion removes profile linkage
and message bodies and revokes posting. Minimal moderation action records may
remain for the approved abuse-evidence period with opaque subject IDs. Explain
that period before sign-up; no indefinite hidden body retention is proposed.

Restores reapply deletions, bans and moderation withdrawals before reopening
reads or posts. Prove that a backup cannot resurrect removed bodies or bypass
a ban. Public chat activation requires the owner to approve hosting, capacity,
moderation coverage and the final retention policy.

## Implementation and qualification

Arkhiv implements isolated room/message storage, context checks, idempotent
submission, pagination, reporting, rate limits, moderation and deletion-aware
recovery. Each consumer adds its own accessible panel and controls after opting
in. Validate stale boundaries, paused/on-demand users, disabled rooms, banned
subjects, blocked identities, hostile text, flood limits, retries, service failure,
program replay context and independent restore.

Local/hosted checks and browser behavior are separate evidence. The owner
accepted this design on 2026-10-05. Shared implementation is tracked in
[#113](https://git.subcult.tv/subculture-collective/arkhiv/issues/113), with
consumer interfaces in
[Bill Nye #11](https://git.subcult.tv/subculture-collective/bill-nye-tv/issues/11),
[Democracy Now! #11](https://git.subcult.tv/subculture-collective/democracy-now-tv/issues/11)
and [template #10](https://git.subcult.tv/subculture-collective/video-archive-template/issues/10).
Implementation evidence and activation approval remain distinct.
