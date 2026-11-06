-- Typed topic and tag labels for catalog items. Machine and editor proposals
-- stay private until an operator accepts them; this migration accepts nothing.
CREATE TABLE catalog_labels (
 id text PRIMARY KEY CHECK(id ~ '^[a-z0-9][a-z0-9._-]{0,99}$'),
 kind text NOT NULL CHECK(kind IN ('topic','place','organization','person','event')),
 label text NOT NULL CHECK(length(trim(label)) BETWEEN 1 AND 200),
 aliases jsonb NOT NULL DEFAULT '[]'::jsonb CHECK(jsonb_typeof(aliases)='array'),
 description text NOT NULL DEFAULT '',
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE catalog_label_assignments (
 item_id text NOT NULL REFERENCES catalog_items(id),
 label_id text NOT NULL REFERENCES catalog_labels(id),
 state text NOT NULL CHECK(state IN ('proposed','accepted','rejected')),
 method text NOT NULL CHECK(length(trim(method)) > 0),
 score real NOT NULL DEFAULT 0,
 evidence jsonb NOT NULL DEFAULT '{}'::jsonb,
 notes text NOT NULL DEFAULT '',
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(item_id,label_id)
);
CREATE INDEX catalog_label_assignments_accepted ON catalog_label_assignments(label_id,item_id) WHERE state='accepted';
CREATE TABLE catalog_label_history (
 id bigserial PRIMARY KEY,
 item_id text NOT NULL,
 label_id text NOT NULL,
 state text NOT NULL,
 method text NOT NULL,
 score real NOT NULL,
 evidence jsonb NOT NULL,
 notes text NOT NULL CHECK(length(trim(notes)) > 0),
 created_at timestamptz NOT NULL DEFAULT now()
);
