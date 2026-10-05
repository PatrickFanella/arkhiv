-- Imported Kollektor selections remain operator drafts. This table is absent
-- from listing and transport views and cannot grant any access decision.
CREATE TABLE curation_drafts (
 sha256 text PRIMARY KEY CHECK(sha256 ~ '^[0-9a-f]{64}$'),
 collection_id text NOT NULL REFERENCES collections(id),
 manifest jsonb NOT NULL,
 notes text NOT NULL CHECK(length(trim(notes)) > 0),
 created_at timestamptz NOT NULL DEFAULT now()
);
