-- Keep processed-video listing separate from source/original permissions.
-- This migration changes only projections and approves no asset or public use.
CREATE OR REPLACE VIEW visible_catalog_representations AS
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
 FROM episodes e WHERE e.decision @> '{"published":true}'::jsonb OR e.decision->'processed_video' @> '{"published":true}'::jsonb;

CREATE OR REPLACE VIEW public_catalog_representations AS
 SELECT v.* FROM visible_catalog_representations v
 WHERE (v.legacy_episode_id<>'' AND EXISTS(
  SELECT 1 FROM episode_public_reviews p JOIN episodes e ON e.id=p.episode_id
  WHERE p.episode_id=v.legacy_episode_id AND p.approved AND e.decision->'processed_video' IS NULL
   AND p.source_hash=v.source_hash AND p.file=v.file
 )) OR (v.legacy_episode_id='' AND EXISTS(
  SELECT 1 FROM representation_transport_reviews t
  WHERE t.item_id=v.item_id AND t.representation_id=v.representation_id
   AND t.collection_id=v.collection_id AND t.review @> '{"public":true}'::jsonb
   AND t.review->>'snapshot_sha256'=v.source_hash
   AND t.review->'files' ? v.file
 ));
