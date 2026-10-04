-- Publication is a separate, snapshot-bound operator decision. Existing private
-- reviews remain private; this additive migration approves no content.
CREATE TABLE episode_public_reviews (
 episode_id text PRIMARY KEY REFERENCES episodes(id),
 source_hash text NOT NULL REFERENCES snapshots(sha256),
 file text NOT NULL,
 approved boolean NOT NULL DEFAULT false,
 notes text NOT NULL CHECK(length(trim(notes)) > 0),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE episode_public_history (
 id bigserial PRIMARY KEY,
 episode_id text NOT NULL REFERENCES episodes(id),
 source_hash text NOT NULL REFERENCES snapshots(sha256),
 file text NOT NULL,
 approved boolean NOT NULL,
 notes text NOT NULL CHECK(length(trim(notes)) > 0),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE VIEW public_catalog_representations AS
 SELECT v.* FROM visible_catalog_representations v
 WHERE (v.legacy_episode_id<>'' AND EXISTS(
  SELECT 1 FROM episode_public_reviews p
  WHERE p.episode_id=v.legacy_episode_id AND p.approved
   AND p.source_hash=v.source_hash AND p.file=v.file
 )) OR (v.legacy_episode_id='' AND EXISTS(
  SELECT 1 FROM representation_transport_reviews t
  WHERE t.item_id=v.item_id AND t.representation_id=v.representation_id
   AND t.collection_id=v.collection_id AND t.review @> '{"public":true}'::jsonb
   AND t.review->>'snapshot_sha256'=v.source_hash
   AND t.review->'files' ? v.file
 ));
