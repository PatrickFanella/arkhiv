ALTER TABLE source_artist_profiles ALTER COLUMN musicbrainz_id DROP NOT NULL;
ALTER TABLE source_artist_profiles ADD COLUMN archive_collection text;
ALTER TABLE source_artist_profiles ADD CONSTRAINT source_artist_profiles_identity_check
 CHECK(musicbrainz_id IS NOT NULL OR archive_collection IS NOT NULL);
ALTER TABLE source_artist_profile_history ALTER COLUMN musicbrainz_id DROP NOT NULL;
ALTER TABLE source_artist_profile_history ADD COLUMN archive_collection text;
CREATE TABLE source_artist_media (
 sha256 text PRIMARY KEY,
 content_type text NOT NULL CHECK(content_type IN ('image/jpeg','image/png')),
 width integer NOT NULL CHECK(width>0),
 height integer NOT NULL CHECK(height>0),
 data bytea NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
