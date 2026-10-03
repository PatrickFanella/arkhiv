-- Private publication does not grant public access. Existing variants start denied.
ALTER TABLE variants ADD COLUMN public_approved boolean NOT NULL DEFAULT false;
CREATE INDEX variants_public_document ON variants(document_id) WHERE public_approved;
CREATE TABLE public_access_events (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 variant_id text NOT NULL REFERENCES variants(id),
 asset_sha text NOT NULL REFERENCES assets(sha256),
 approved boolean NOT NULL,
 review text NOT NULL CHECK(length(trim(review)) > 0),
 manifest_sha256 text NOT NULL DEFAULT '',
 decided_at timestamptz NOT NULL DEFAULT now()
);
