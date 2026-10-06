# Accounts and personal data

Proposed 2026-10-05 for [issue #110](https://git.subcult.tv/subculture-collective/arkhiv/issues/110).
Status: scope and proposed retention accepted by the owner on 2026-10-05.
The owner selected "Accept the proposed designs" in this task's review.
This document defines implementation scope;
it does not enable accounts or choose a live identity provider.

## Product decision

Use optional sign-in to synchronize favorites between devices within one
product. Anonymous browsing, viewing and browser-local favorites continue to
work. One sign-in provider may serve several brands, but each deployment keeps
its own personal data. Cross-product favorites and account linking are outside
the first version.

Arkhiv owns an optional personal-data API module and its additive migrations.
Each consumer owns the sign-in interface, same-origin session service and
identity-provider configuration. Identity administration belongs to the
deployment owner. Arkhiv does not store passwords or run an identity provider.
The catalog engine remains usable without the personal-data module.

## Authentication and authorization

The consumer's server performs OpenID Connect authorization-code sign-in with
PKCE, validates the configured issuer, audience, signature, expiry and nonce,
and uses exact registered redirect URLs. Identity is the issuer/subject pair,
never an email address or display name. These choices follow
[OpenID Connect Core](https://openid.net/specs/openid-connect-core-1_0.html)
and the [OAuth security BCP](https://www.rfc-editor.org/rfc/rfc9700.html).

Keep provider tokens on the consumer server. Give the browser a revocable,
opaque session cookie with Secure, HttpOnly and SameSite=Lax attributes.
State-changing requests require CSRF protection and origin validation. Sign-out
revokes the local session; revocation of all devices is a separate account
action. Expired or unavailable identity services leave anonymous viewing usable.

The personal API accepts only verified, deployment-scoped subjects with its
own audience and limited scopes. A browser-supplied user ID or forwarding header
is never sufficient. Requests may read or change only that subject's personal
records. Moderator privileges are separately assigned; signing in never grants
operator, editorial, public-review, acquisition or schedule permissions.

## Storage and proposed API

Use separately named personal tables and a serving role that cannot mutate
archive/source/schedule tables. Keep deployment identity, internal user UUID,
provider issuer/subject mapping and minimal account preferences. Store favorites
by user, collection and stable item/episode identity, with revision/tombstone
state. A unique key makes repeated additions idempotent.

Proposed optional routes under `/api/personal/v1`:

| Route | Behavior |
| --- | --- |
| `GET /me` | Minimal local profile and enabled capabilities |
| `GET /favorites` | Current user's favorites, including explicit unavailable references |
| `PUT /favorites/{collection}/{item}` | Idempotent add after collection/item validation |
| `DELETE /favorites/{collection}/{item}` | Idempotent removal and sync tombstone |
| `GET /export` | Versioned JSON export of the user's personal records |
| `DELETE /me` | Reauthenticated local account deletion and session revocation |

This introduces a new optional writable personal API. Existing catalog and
playback APIs remain read-only. Favorites never grant access; every playback
request uses current source/public reviews. Withdrawn items can remain as
unavailable favorites without exposing their private metadata.

## Browser-local migration and sync

After sign-in, ask whether to import browser-local favorites. Show the count and
destination product. Default to keeping the local copy until an acknowledged
import and a verified account export. Merge by stable collection/item identity,
retain unavailable references, and reject foreign deployment scope. Signing
out does not silently delete either local favorites or account data.

Use server revisions and explicit removal tombstones rather than replacing an
entire favorites list. Retried adds/removes are idempotent. Clients acknowledge
new revisions before retiring their pending changes; show conflicts or offline
pending actions honestly.

## Deletion, recovery and loss of account

Export covers favorites, local profile and pending sync state. Deleting a local
account removes its favorites/profile and revokes sessions. It never deletes
source metadata, originals, editorial history or schedules, and does not imply
deletion of the external provider account.

Proposed personal-backup policy is a 35-day retention window, with a deletion
ledger retained for 90 days so restoration can reapply deletion requests before
serving data. The ledger holds opaque internal IDs and timestamps, not message
bodies or favorites. These are proposed personal-data limits, not archive
retention changes. Implementation must prove that restores cannot resurrect
deleted accounts. The owner accepted these periods on 2026-10-05; public
activation remains separate.

Provider account recovery remains the provider's responsibility. Do not relink
a lost account by matching email. A provider migration needs a reviewed mapping,
proof of both identities or an operator recovery procedure, and a recoverable
checkpoint. Users with no account access can still use exported/local favorites.

## Implementation and qualification

First implement the optional module, role restrictions, issuer/subject tests,
idempotent favorites, export/deletion and restore suppression in Arkhiv. Then
add sign-in/session/import interfaces in each opting-in consumer. Test two-device
sync, offline retries, changed access decisions, foreign subject/scope attempts,
sign-out, provider failure and restored deletion state. No real provider or
public sign-in is activated by these source changes.

Before activation, resolve the actual OIDC provider/client registrations and
which products enable accounts first, and verify the accepted retention periods.
Separate implementation issues
must carry these decisions and their evidence. Shared implementation is tracked
in [#112](https://git.subcult.tv/subculture-collective/arkhiv/issues/112).
Consumer account work is tracked in
[Bill Nye #10](https://git.subcult.tv/subculture-collective/bill-nye-tv/issues/10),
[Democracy Now! #10](https://git.subcult.tv/subculture-collective/democracy-now-tv/issues/10)
and [template #9](https://git.subcult.tv/subculture-collective/video-archive-template/issues/9).
