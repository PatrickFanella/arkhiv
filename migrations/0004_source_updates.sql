CREATE TABLE source_inventories (
  collection_id text NOT NULL REFERENCES collections(id),
  source_item text NOT NULL,
  snapshot_id text NOT NULL REFERENCES source_snapshots(id),
  inventory jsonb NOT NULL,
  checked_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY(collection_id, source_item)
);

CREATE TABLE source_changes (
  id text PRIMARY KEY,
  collection_id text NOT NULL REFERENCES collections(id),
  source_item text NOT NULL,
  filename text NOT NULL,
  kind text NOT NULL CHECK(kind IN ('added','content','metadata','removed','uncertain')),
  snapshot_id text NOT NULL REFERENCES source_snapshots(id),
  detail jsonb NOT NULL,
  decision text NOT NULL DEFAULT 'pending' CHECK(decision IN ('pending','approve','reject','hold','acquired')),
  review_note text NOT NULL DEFAULT '',
  document_id text REFERENCES documents(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  reviewed_at timestamptz
);
CREATE INDEX source_changes_pending ON source_changes(collection_id, decision, created_at);

CREATE TABLE source_refresh_runs (
  id text PRIMARY KEY,
  collection_id text NOT NULL REFERENCES collections(id),
  started_at timestamptz NOT NULL DEFAULT now(),
  finished_at timestamptz,
  status text NOT NULL CHECK(status IN ('running','complete','failed')),
  result jsonb NOT NULL DEFAULT '{}'
);
