CREATE TABLE IF NOT EXISTS schema_versions (name text PRIMARY KEY, sha256 text NOT NULL);
CREATE TABLE collections (
 id text PRIMARY KEY,
 policy text NOT NULL CHECK(policy IN ('source_stream','local_preserve')),
 budget bigint NOT NULL DEFAULT 0,
 reserve bigint NOT NULL DEFAULT 0,
 CHECK(id <> 'democracy_now_vid' OR policy = 'source_stream'),
 CHECK(policy <> 'local_preserve' OR (budget > 0 AND reserve > 0))
);
CREATE TABLE snapshots (
 sha256 text PRIMARY KEY,
 item_id text NOT NULL,
 raw bytea NOT NULL,
 parsed jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE observations (
 id bigserial PRIMARY KEY,
 item_id text NOT NULL,
 snapshot_hash text REFERENCES snapshots(sha256),
 status integer NOT NULL,
 etag text NOT NULL DEFAULT '',
 modified text NOT NULL DEFAULT '',
 error text NOT NULL DEFAULT '',
 observed_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE episodes (
 id text PRIMARY KEY,
 collection_id text NOT NULL REFERENCES collections(id),
 source_hash text NOT NULL REFERENCES snapshots(sha256),
 source jsonb NOT NULL,
 decision jsonb NOT NULL DEFAULT '{}',
 unavailable boolean NOT NULL DEFAULT false,
 local_path text NOT NULL DEFAULT '',
 local_sha256 text NOT NULL DEFAULT '',
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE decision_history (
 id bigserial PRIMARY KEY,
 episode_id text NOT NULL REFERENCES episodes(id),
 decision jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE schedules (
 revision text PRIMARY KEY,
 payload jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE current_schedule (id integer PRIMARY KEY CHECK(id=1), revision text NOT NULL REFERENCES schedules(revision));
CREATE TABLE worker_state (
 collection_id text PRIMARY KEY REFERENCES collections(id),
 page integer NOT NULL DEFAULT 1,
 discovery_complete boolean NOT NULL DEFAULT false,
 source_count bigint NOT NULL DEFAULT 0,
 retry_at timestamptz,
 last_success timestamptz,
 last_error text NOT NULL DEFAULT '',
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE metadata_queue (
 collection_id text NOT NULL REFERENCES collections(id),
 item_id text NOT NULL,
 state text NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','imported','retry','unavailable','rejected')),
 attempts integer NOT NULL DEFAULT 0,
 next_attempt timestamptz NOT NULL DEFAULT now(),
 error text NOT NULL DEFAULT '',
 discovered_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(collection_id,item_id)
);
CREATE INDEX metadata_queue_ready ON metadata_queue(collection_id,next_attempt);
