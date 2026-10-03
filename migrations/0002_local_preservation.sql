ALTER TABLE episodes ADD COLUMN local_file text NOT NULL DEFAULT '';
ALTER TABLE episodes ADD COLUMN local_snapshot text NOT NULL DEFAULT '';
CREATE TABLE preservation_history (
 id bigserial PRIMARY KEY,
 episode_id text NOT NULL REFERENCES episodes(id),
 receipt jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
