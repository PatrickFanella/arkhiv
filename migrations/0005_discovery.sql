ALTER TABLE source_inventories ADD COLUMN scope text NOT NULL DEFAULT '';
UPDATE source_inventories SET scope='item:' || source_item;
CREATE TABLE source_discoveries (
 collection_id text NOT NULL REFERENCES collections(id),
 scope text NOT NULL,
 raw jsonb NOT NULL,
 checked_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(collection_id,scope)
);
