CREATE TABLE representation_transport_reviews (
 item_id text NOT NULL,
 representation_id text NOT NULL,
 collection_id text NOT NULL,
 review jsonb NOT NULL,
 preservation jsonb NOT NULL DEFAULT '{}',
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(item_id,representation_id,collection_id),
 FOREIGN KEY(item_id,representation_id,collection_id) REFERENCES catalog_representation_memberships(item_id,representation_id,collection_id)
);
CREATE TABLE representation_transport_history (
 id bigserial PRIMARY KEY,
 item_id text NOT NULL,
 representation_id text NOT NULL,
 collection_id text NOT NULL,
 review jsonb NOT NULL,
 preservation jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(item_id,representation_id,collection_id) REFERENCES catalog_representation_memberships(item_id,representation_id,collection_id)
);
