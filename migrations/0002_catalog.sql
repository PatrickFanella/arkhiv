CREATE TABLE collections (
 id text PRIMARY KEY, config jsonb NOT NULL, updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE source_snapshots (
 id text PRIMARY KEY, item_id text NOT NULL, raw jsonb NOT NULL, captured_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE assets (
 sha256 text PRIMARY KEY CHECK (sha256 ~ '^[a-f0-9]{64}$'), size bigint NOT NULL CHECK (size > 0), verified_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE documents (
 id text PRIMARY KEY, collection_id text NOT NULL REFERENCES collections(id), source_item text NOT NULL,
 source_key text NOT NULL, metadata jsonb NOT NULL, overrides jsonb NOT NULL DEFAULT '{}',
 published boolean NOT NULL DEFAULT false, review_note text NOT NULL DEFAULT '', rights text NOT NULL DEFAULT 'unknown',
 updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE(collection_id, source_item, source_key)
);
CREATE TABLE variants (
 id text PRIMARY KEY, document_id text NOT NULL REFERENCES documents(id), filename text NOT NULL,
 asset_sha text NOT NULL REFERENCES assets(sha256), source_snapshot text NOT NULL REFERENCES source_snapshots(id),
 kind text NOT NULL DEFAULT 'original', UNIQUE(document_id, filename, asset_sha)
);
CREATE TABLE acquisition_attempts (
 id bigserial PRIMARY KEY, collection_id text NOT NULL REFERENCES collections(id), source_item text NOT NULL,
 filename text NOT NULL, status text NOT NULL CHECK (status IN ('started','verified','failed','reused')),
 detail text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE derivatives (
 asset_sha text PRIMARY KEY REFERENCES assets(sha256), status text NOT NULL,
 pages integer NOT NULL DEFAULT 0, tool text NOT NULL, error text NOT NULL DEFAULT '', updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE page_text (
 asset_sha text NOT NULL REFERENCES assets(sha256), page integer NOT NULL CHECK(page >= 0), text text NOT NULL,
 search tsvector GENERATED ALWAYS AS (to_tsvector('simple',text)) STORED, PRIMARY KEY(asset_sha,page)
);
CREATE INDEX page_text_search ON page_text USING gin(search);
CREATE INDEX documents_collection_public ON documents(collection_id,published,id);
CREATE TABLE review_events (
 id bigserial PRIMARY KEY, document_id text NOT NULL REFERENCES documents(id), action text NOT NULL,
 detail jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
