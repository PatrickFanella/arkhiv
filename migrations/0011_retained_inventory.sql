-- Collection-wide source inventory: one row per Internet Archive item, with
-- normalized show fields and track titles for calendar, date and search views.
-- Raw metadata snapshots stay on disk; only their hashes are recorded here.
CREATE TABLE IF NOT EXISTS retained_audio_items (
  collection_id text NOT NULL REFERENCES collections(id),
  id text NOT NULL,
  PRIMARY KEY(collection_id,id),
  show_date date,
  date_text text NOT NULL DEFAULT '',
  artist text NOT NULL DEFAULT '',
  venue text NOT NULL DEFAULT '',
  location text NOT NULL DEFAULT '',
  title text NOT NULL DEFAULT '',
  kind text NOT NULL DEFAULT 'unknown',
  source text NOT NULL DEFAULT '',
  lineage text NOT NULL DEFAULT '',
  taper text NOT NULL DEFAULT '',
  transferer text NOT NULL DEFAULT '',
  downloads integer NOT NULL DEFAULT 0,
  avg_rating real NOT NULL DEFAULT 0,
  num_reviews integer NOT NULL DEFAULT 0,
  item_size bigint NOT NULL DEFAULT 0,
  added_at timestamptz,
  tracks jsonb NOT NULL DEFAULT '[]',
  songs text NOT NULL DEFAULT '',
  duration double precision NOT NULL DEFAULT 0,
  warnings jsonb NOT NULL DEFAULT '[]',
  summary jsonb NOT NULL DEFAULT '{}',
  snapshot_sha256 text NOT NULL DEFAULT '',
  observed_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  search tsvector GENERATED ALWAYS AS (
    to_tsvector('simple', title || ' ' || venue || ' ' || location || ' ' || date_text || ' ' || taper || ' ' || transferer || ' ' || source || ' ' || songs)
  ) STORED
);
CREATE INDEX IF NOT EXISTS retained_audio_items_show_date ON retained_audio_items (show_date);
CREATE INDEX IF NOT EXISTS retained_audio_items_month_day ON retained_audio_items ((EXTRACT(MONTH FROM show_date)), (EXTRACT(DAY FROM show_date)));
CREATE INDEX IF NOT EXISTS retained_audio_items_search ON retained_audio_items USING GIN (search);

-- Full table exports retain source, editorial, decision and schedule history.
-- Compatibility projections do not rewrite these immutable recovery records.
CREATE TABLE retained_legacy_exports (
 collection_id text NOT NULL REFERENCES collections(id),
 module text NOT NULL CHECK(module IN ('print','audio')),
 sha256 text NOT NULL CHECK(sha256 ~ '^[0-9a-f]{64}$'),
 recovery_sha256 text NOT NULL CHECK(recovery_sha256 ~ '^[0-9a-f]{64}$'),
 payload jsonb NOT NULL,
 notes text NOT NULL CHECK(length(trim(notes))>0),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(collection_id,module,sha256)
);
