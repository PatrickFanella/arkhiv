-- Optional module. Apply only to a separately provisioned personal database.
-- No archive/source/editorial/schedule table is referenced or writable here.
CREATE TABLE personal_users (
    id text PRIMARY KEY,
    deployment text NOT NULL,
    issuer text NOT NULL,
    subject text NOT NULL,
    alias text NOT NULL,
    revision bigint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(deployment, issuer, subject)
);
CREATE TABLE personal_sessions (
    hash text PRIMARY KEY,
    deployment text NOT NULL,
    issuer text NOT NULL,
    subject text NOT NULL,
    authenticated_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL
);
CREATE INDEX personal_sessions_identity ON personal_sessions(deployment, issuer, subject);
CREATE TABLE personal_login_flows (
    state_hash text PRIMARY KEY,
    cookie_hash text NOT NULL,
    deployment text NOT NULL,
    nonce text NOT NULL,
    verifier text NOT NULL,
    expires_at timestamptz NOT NULL
);
CREATE TABLE personal_favorites (
    user_id text NOT NULL REFERENCES personal_users(id) ON DELETE CASCADE,
    collection text NOT NULL,
    item text NOT NULL,
    removed boolean NOT NULL DEFAULT false,
    revision bigint NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY(user_id, collection, item)
);
CREATE TABLE personal_deletions (
    deployment text NOT NULL,
    kind text NOT NULL CHECK(kind IN ('account','message')),
    opaque_id text NOT NULL,
    deleted_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY(deployment, kind, opaque_id)
);
CREATE TABLE chat_messages (
    seq bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    id text UNIQUE NOT NULL,
    deployment text NOT NULL,
    collection text NOT NULL,
    user_id text NOT NULL,
    alias text NOT NULL,
    body text NOT NULL CHECK(octet_length(body) <= 4096 AND length(body) <= 1000),
    revision text NOT NULL,
    program_start timestamptz NOT NULL,
    episode text NOT NULL,
    original_date text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    hidden boolean NOT NULL DEFAULT false,
    idempotency_key text NOT NULL,
    UNIQUE(deployment, user_id, idempotency_key)
);
CREATE INDEX chat_messages_room ON chat_messages(deployment, collection, seq);
CREATE TABLE chat_requests (
    deployment text NOT NULL,
    user_id text NOT NULL,
    key text NOT NULL,
    message_id text NOT NULL,
    digest text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY(deployment, user_id, key)
);
CREATE TABLE chat_reports (
    id text PRIMARY KEY,
    deployment text NOT NULL,
    message_id text NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
    user_id text NOT NULL,
    reason text NOT NULL CHECK(octet_length(reason) <= 4096),
    resolved boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(deployment, message_id, user_id)
);
CREATE TABLE chat_blocks (
    user_id text NOT NULL,
    blocked_id text NOT NULL,
    PRIMARY KEY(user_id, blocked_id)
);
CREATE TABLE chat_restrictions (
    deployment text NOT NULL,
    user_id text NOT NULL,
    until_at timestamptz NOT NULL,
    PRIMARY KEY(deployment, user_id)
);
CREATE TABLE chat_moderation (
    id text PRIMARY KEY,
    deployment text NOT NULL,
    actor text NOT NULL,
    action text NOT NULL,
    target text NOT NULL,
    reason text NOT NULL CHECK(octet_length(reason) <= 4096),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE chat_rate_windows (
    deployment text NOT NULL,
    key text NOT NULL,
    bucket bigint NOT NULL,
    count integer NOT NULL,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY(deployment, key, bucket)
);
