-- Rebuildable lookup over the immutable independently recovered donor export.
-- Current transport reviews remain the authority for all derivative serving.
CREATE TABLE retained_print_indexes (
 collection_id text NOT NULL REFERENCES collections(id),
 recovery_sha256 text NOT NULL CHECK (recovery_sha256 ~ '^[0-9a-f]{64}$'),
 asset_sha256 text NOT NULL CHECK (asset_sha256 ~ '^[0-9a-f]{64}$'),
 derivative jsonb NOT NULL,
 sections jsonb NOT NULL,
 variants jsonb NOT NULL,
 PRIMARY KEY(collection_id,recovery_sha256,asset_sha256)
);
