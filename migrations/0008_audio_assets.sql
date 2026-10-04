CREATE TABLE audio_assets (
 item_id text NOT NULL,
 representation_id text NOT NULL,
 collection_id text NOT NULL,
 receipt jsonb NOT NULL,
 notes text NOT NULL CHECK(length(trim(notes)) > 0),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(item_id,representation_id,collection_id),
 FOREIGN KEY(item_id,representation_id,collection_id) REFERENCES catalog_representation_memberships(item_id,representation_id,collection_id)
);
CREATE TABLE audio_asset_history (
 id bigserial PRIMARY KEY,
 item_id text NOT NULL,
 representation_id text NOT NULL,
 collection_id text NOT NULL,
 receipt jsonb NOT NULL,
 notes text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE audio_schedules (
 collection_id text NOT NULL REFERENCES collections(id),
 revision text NOT NULL,
 schedule jsonb NOT NULL,
 notes text NOT NULL CHECK(length(trim(notes)) > 0),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(collection_id,revision)
);
