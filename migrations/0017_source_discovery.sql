CREATE TABLE source_artist_snapshots (
 sha256 text PRIMARY KEY,
 url text NOT NULL,
 raw bytea NOT NULL,
 fetched_at timestamptz NOT NULL
);
CREATE TABLE source_artist_profiles (
 collection_id text NOT NULL REFERENCES collections(id),
 source_artist text NOT NULL,
 musicbrainz_id text NOT NULL,
 profile jsonb NOT NULL,
 source_hashes jsonb NOT NULL,
 notes text NOT NULL CHECK(length(trim(notes))>0),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(collection_id,source_artist)
);
CREATE TABLE source_artist_profile_history (
 id bigserial PRIMARY KEY,
 collection_id text NOT NULL,
 source_artist text NOT NULL,
 musicbrainz_id text NOT NULL,
 profile jsonb NOT NULL,
 source_hashes jsonb NOT NULL,
 notes text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX source_catalog_visible_artist_date ON source_catalog_items
 (collection_id,lower(effective->>'creator'),(effective->>'date'),identifier) WHERE NOT excluded;
CREATE INDEX source_catalog_visible_month_day ON source_catalog_items
 (collection_id,(substring(effective->>'date',6,5)),(effective->>'date'),identifier)
 WHERE NOT excluded AND source_catalog_complete_date(effective->>'date');
CREATE INDEX source_catalog_visible_month ON source_catalog_items
 (collection_id,(substring(effective->>'date',6,2)),(effective->>'date'),identifier)
 WHERE NOT excluded AND source_catalog_complete_date(effective->>'date');
