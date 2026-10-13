CREATE TABLE source_catalog_items (
 collection_id text NOT NULL REFERENCES collections(id),
 identifier text NOT NULL,
 PRIMARY KEY(collection_id,identifier),
 source_hash text NOT NULL REFERENCES snapshots(sha256),
 normalized jsonb NOT NULL,
 normalization jsonb NOT NULL,
 editorial jsonb NOT NULL DEFAULT '{}',
 effective jsonb NOT NULL,
 unavailable boolean NOT NULL DEFAULT false,
 imported_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 search tsvector NOT NULL
);
CREATE FUNCTION project_source_catalog() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 NEW.effective := NEW.normalized || NEW.editorial;
 NEW.search := to_tsvector('simple',
  COALESCE(NEW.effective->>'title','') || ' ' || COALESCE(NEW.effective->>'creator','') || ' ' ||
  COALESCE(NEW.effective->>'venue','') || ' ' || COALESCE(NEW.effective->>'coverage','') || ' ' ||
  COALESCE(NEW.effective->>'date','') || ' ' || COALESCE(NEW.effective->>'description','') || ' ' ||
  COALESCE(NEW.effective->>'taper','') || ' ' || COALESCE(NEW.effective->>'transferer','') || ' ' ||
  COALESCE(NEW.effective->>'source','') || ' ' || COALESCE(NEW.effective->>'lineage','') || ' ' ||
  COALESCE(NEW.effective->>'subject','') || ' ' || COALESCE(NEW.effective->>'songs',''));
 RETURN NEW;
END $$;
CREATE TRIGGER source_catalog_items_projection BEFORE INSERT OR UPDATE ON source_catalog_items
 FOR EACH ROW EXECUTE FUNCTION project_source_catalog();
CREATE INDEX source_catalog_items_search ON source_catalog_items USING GIN(search);
CREATE INDEX source_catalog_items_artist ON source_catalog_items((effective->>'creator'),identifier);
CREATE INDEX source_catalog_items_year ON source_catalog_items((left(effective->>'date',4)),identifier);
CREATE INDEX source_catalog_items_venue ON source_catalog_items((effective->>'venue'),identifier);
CREATE INDEX source_catalog_items_date ON source_catalog_items((effective->>'date'),identifier);

CREATE TABLE source_catalog_history (
 id bigserial PRIMARY KEY,
 collection_id text NOT NULL,
 item_id text NOT NULL,
 patch jsonb NOT NULL,
 notes text NOT NULL CHECK(length(trim(notes))>0),
 created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(collection_id,item_id) REFERENCES source_catalog_items(collection_id,identifier)
);
CREATE TABLE source_catalog_worker_state (
 collection_id text PRIMARY KEY REFERENCES collections(id),
 cursor text NOT NULL DEFAULT '',
 complete boolean NOT NULL DEFAULT false,
 discovered bigint NOT NULL DEFAULT 0,
 retry_at timestamptz,
 last_error text NOT NULL DEFAULT '',
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE source_catalog_queue (
 collection_id text NOT NULL REFERENCES collections(id),
 item_id text NOT NULL,
 state text NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','imported','retry','unavailable','rejected')),
 attempts integer NOT NULL DEFAULT 0,
 next_attempt timestamptz NOT NULL DEFAULT now(),
 error text NOT NULL DEFAULT '',
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(collection_id,item_id)
);
CREATE INDEX source_catalog_queue_ready ON source_catalog_queue(collection_id,state,next_attempt,item_id);
-- Invalid source dates remain visible but must not pass complete-date filters.
CREATE FUNCTION source_catalog_complete_date(value text) RETURNS boolean
LANGUAGE plpgsql IMMUTABLE AS $$
BEGIN
 IF value IS NULL OR value !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN RETURN false; END IF;
 RETURN to_char(value::date, 'YYYY-MM-DD') = value;
EXCEPTION WHEN invalid_datetime_format OR datetime_field_overflow THEN RETURN false;
END $$;
