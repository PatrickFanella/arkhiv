CREATE TABLE source_catalog_audio_reviews (
 collection_id text NOT NULL,
 item_id text NOT NULL,
 source_hash text NOT NULL REFERENCES snapshots(sha256),
 state text NOT NULL CHECK(state IN ('reviewed','unsupported','held','retry')),
 review jsonb,
 error text NOT NULL DEFAULT '',
 attempts integer NOT NULL DEFAULT 1,
 next_attempt timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(collection_id,item_id),
 FOREIGN KEY(collection_id,item_id) REFERENCES source_catalog_items(collection_id,identifier)
);
CREATE INDEX source_catalog_audio_reviews_ready ON source_catalog_audio_reviews(collection_id,state,next_attempt);
CREATE INDEX catalog_representations_source_audio ON catalog_representations(source_item_id,source_hash) WHERE media='audio';
CREATE INDEX source_catalog_items_import_order ON source_catalog_items(collection_id,imported_at,identifier);
