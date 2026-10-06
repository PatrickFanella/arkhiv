# Optional accounts and channel discussion

The archive runs without these modules. Configure accounts separately for each
product; provider selection and public activation require the deployment owner's
decision. Consumer repositories own their interfaces and same-origin routing.
The shared SDK exports `usePersonal` and `useChannelChat` for that composition.

## Service boundaries

Run `vhness serve-session` as the consumer's optional session service. Route
`/api/account/v1`, `/api/personal/v1` and `/api/chat/v1` to it on the consumer's
origin. Keep catalog and playback routes on the archive service. The session
service never forwards browser cookies or bearer tokens to the archive. It
creates short-lived Ed25519 assertions scoped to one product and session.

Sign-in uses authorization code flow, PKCE, state, nonce and an exact callback
URL at `/api/account/v1/callback`. Provider discovery is lazy and bounded;
provider failure leaves anonymous catalog viewing available. Provider tokens
are validated in memory and are not stored or exposed to the browser. Sessions
expire at the earlier of eight hours or the validated identity token's expiry.
Mutations require the exact configured origin and the session's CSRF token.
Account deletion requires authentication within the last five minutes.

Configure `ARKHIV_SESSION_CONFIG_FILE` outside Git with these fields:

```json
{
  "origin": "https://consumer.example",
  "issuer": "https://identity.example/realm",
  "client_id": "owner-registered-client",
  "client_secret_file": "/run/secrets/oidc-client",
  "private_key_file": "/run/secrets/assertion-private.pem",
  "database_url_file": "/run/secrets/personal-runtime-url",
  "core_url": "http://archive:8080",
  "deployment": "consumer-product",
  "assertion_issuer": "consumer-session"
}
```

Generate a dedicated Ed25519 key pair. The private key must be PKCS#8 PEM with
owner-only permissions; the public key is PKIX PEM. Mount only the public key
in the archive process. Keep secrets and connection URLs in private files.
`allow_loopback_http` is available only for isolated loopback qualification.
Never use it for public sign-in.

Set `ARKHIV_COMMUNITY_CONFIG_FILE` on the archive process to enable its optional
module:

```json
{
  "deployment": "consumer-product",
  "assertion_issuer": "consumer-session",
  "public_key_file": "/run/config/assertion-public.pem",
  "database_url_file": "/run/secrets/personal-runtime-url",
  "recovery_directory": "/independent-recovery/personal-events",
  "chat_read": false,
  "chat_post": false,
  "moderators": [],
  "moderation_contact": ""
}
```

Posting requires reads, an explicit moderator roster and a visible moderation
contact. Roster entries are SHA-256 hashes of the provider issuer, a NUL byte,
and the opaque provider subject. The core checks this roster independently of
assertion scopes. Disable posting with `chat_post`; disable all room reads with
`chat_read`. Restart with the reviewed configuration. Playback is independent.

## Personal database

Provision a dedicated database and an operator owner. Apply
`migrations/community` using `vhness community-migrate` with an operator-only
configuration and connection file. These migrations are separate from archive
migrations. Provision a runtime role without superuser, database/role creation,
replication or bypass-RLS privileges. It must not own tables or schemas and must
have no archive write privileges.

Grant CONNECT on the personal database and USAGE on its schema. Grant
SELECT, INSERT, UPDATE and DELETE only on `personal_*` and `chat_*` tables, and
USAGE/SELECT on their identity sequences. Grant no permissions on
`schema_versions`. Revoke CREATE on the schema from PUBLIC and runtime roles.
Use a separate restricted connection file for serving. Startup checks reject
privileged roles, table owners and roles that can write other tables. Never
share the archive connection file with this module.

Favorites use stable collection/item IDs, revisions and removal tombstones.
`If-Match` supplies the acknowledged revision. Conflicting changes return 409;
idempotent retries retain their original result. Adding a favorite checks
current public availability; an unavailable saved reference remains removable.
Imports and conflict rebasing require user review of the count and destination.
Signing out retains browser-local favorites and pending account changes.

Chat uses ten-second polling with backoff and bounded cursor pages. Bodies are
plain text, limited to 1,000 Unicode characters and 4 KiB. Posts carry an
idempotency key and the current schedule revision, program start and episode.
The server checks current schedule context and rejects stale, paused or
on-demand submissions. Playback mode is a client declaration, not proof that a
person is physically watching. It grants no archive or schedule permissions.
Blocking filters the signed-in user's reads. Reports and audit reasons remain
private. Restrictions and withdrawals require independently assigned moderator
privileges. Video playback continues when the optional service is unavailable.

## Retention and recovery

Run `vhness community-prune` daily using the dedicated operator configuration.
It expires chat, reports, request receipts and moderation evidence after 30 days,
plus expired sessions, flows and rate windows. Its result reports unresolved
reports reaching expiry; operators must review those before the deadline.
Opaque deletion events last 90 days. Archive metadata, source inventories,
originals, decisions and schedules are outside this policy.

The recovery directory must be absolute, private (0700), durable and independent
of both personal dump rotation and the restored database. Each accepted account
or message deletion, withdrawal or restriction writes an opaque journal event
with file and directory fsync before the database commit is acknowledged.
A failed database commit can leave a durable deletion intent; recovery applies
it conservatively. Journal loss is a recovery failure, never a reason to reopen
serving from an old dump. Replicate this directory to the independent recovery
target and verify its integrity and completeness.

Use `scripts/personal_backup.py backup` with `--database-url-file`, `--deployment`,
`--destination`, `--checkpoint-directory` and `--binary`. Its operator config
must name that exact personal database. It writes a custom PostgreSQL dump and
SHA-256 receipt, refreshes the independent recovery checkpoint and prunes only
validated personal dumps older than 35 days. Unknown, modified and symlinked
entries are preserved for review. The tool refuses archive or unknown tables.

Restore into a newly provisioned, empty personal database while reads and posts
are disabled. Use the matching destination operator configuration and
`personal_backup.py restore --database-url-file ... --deployment ... --backup
... --checkpoint ... --binary ...`. The tool never drops or resets a database.
It verifies age and checksum, restores the dump and reapplies the current
independent journal/checkpoint. Recovery revokes every restored session and
login flow, suppresses deleted accounts and bodies, hides withdrawals and
reapplies current restrictions. Regrant the restricted runtime permissions.
Verify counts, deletions, bans, access boundaries and a fresh sign-in from an
independent process before reopening service. A successful dump, schema-copy
test or local unit test alone is not independent restore proof.
