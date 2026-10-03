-- Additive catalog indexes. Original video records, decisions and schedules stay
-- authoritative; the video projection reads them live instead of copying history.
CREATE TABLE catalog_items (
 id text PRIMARY KEY CHECK(id !~ '^legacy-video:'),
 title text NOT NULL CHECK(length(trim(title)) > 0),
 description text NOT NULL DEFAULT '',
 date text NOT NULL DEFAULT '',
 creator text NOT NULL DEFAULT '',
 editorial jsonb NOT NULL DEFAULT '{}',
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE catalog_memberships (
 item_id text NOT NULL REFERENCES catalog_items(id),
 collection_id text NOT NULL REFERENCES collections(id),
 listed boolean NOT NULL DEFAULT false,
 PRIMARY KEY(item_id,collection_id)
);
CREATE TABLE catalog_representations (
 item_id text NOT NULL REFERENCES catalog_items(id),
 id text NOT NULL,
 media text NOT NULL CHECK(media IN ('print','audio','video')),
 source_item_id text NOT NULL,
 source_hash text NOT NULL REFERENCES snapshots(sha256),
 file text NOT NULL,
 extension jsonb NOT NULL,
 PRIMARY KEY(item_id,id)
);
CREATE TABLE catalog_representation_memberships (
 item_id text NOT NULL,
 representation_id text NOT NULL,
 collection_id text NOT NULL,
 listed boolean NOT NULL DEFAULT false,
 PRIMARY KEY(item_id,representation_id,collection_id),
 FOREIGN KEY(item_id,representation_id) REFERENCES catalog_representations(item_id,id),
 FOREIGN KEY(item_id,collection_id) REFERENCES catalog_memberships(item_id,collection_id)
);
CREATE TABLE catalog_review_history (
 id bigserial PRIMARY KEY,
 item_id text NOT NULL REFERENCES catalog_items(id),
 payload jsonb NOT NULL,
 notes text NOT NULL CHECK(length(trim(notes)) > 0),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE VIEW visible_catalog_representations AS
 SELECT i.id AS item_id,i.title,i.description,i.date,i.creator,m.collection_id,
 r.id AS representation_id,r.media,r.source_item_id,r.source_hash,r.file,r.extension,
 ''::text AS legacy_episode_id
 FROM catalog_items i
 JOIN catalog_memberships m ON m.item_id=i.id AND m.listed
 JOIN catalog_representation_memberships rm ON rm.item_id=i.id AND rm.collection_id=m.collection_id AND rm.listed
 JOIN catalog_representations r ON r.item_id=i.id AND r.id=rm.representation_id
 UNION ALL
 SELECT 'legacy-video:'||e.id,
 COALESCE(NULLIF(e.decision->>'title',''),e.source->>'title'),
 COALESCE(e.source->>'description',''),COALESCE(e.source->>'date',''),COALESCE(e.source->>'creator',''),
 e.collection_id,'video','video',COALESCE(NULLIF(e.source->>'source_item_id',''),e.id),
 e.source_hash,COALESCE(e.decision->>'file',''),
 jsonb_build_object('video',jsonb_build_object('episode_id',e.id,'captions',COALESCE(e.decision->'captions','[]'::jsonb))),e.id
 FROM episodes e WHERE e.decision @> '{"published":true}'::jsonb;
CREATE INDEX catalog_memberships_collection ON catalog_memberships(collection_id,item_id) WHERE listed;
