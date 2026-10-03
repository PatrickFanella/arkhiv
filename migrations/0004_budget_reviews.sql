CREATE TABLE collection_budget_reviews (
 id bigserial PRIMARY KEY,
 collection_id text NOT NULL REFERENCES collections(id),
 old_budget bigint NOT NULL,
 old_reserve bigint NOT NULL,
 new_budget bigint NOT NULL,
 new_reserve bigint NOT NULL,
 notes text NOT NULL CHECK(length(notes)>0),
 created_at timestamptz NOT NULL DEFAULT now()
);
