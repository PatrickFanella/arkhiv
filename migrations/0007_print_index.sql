CREATE TABLE print_indexes (
 item_id text NOT NULL,
 representation_id text NOT NULL,
 file text NOT NULL,
 snapshot_sha256 text NOT NULL REFERENCES snapshots(sha256),
 original_sha256 text NOT NULL,
 pages integer NOT NULL CHECK(pages BETWEEN 0 AND 10000),
 text_status text NOT NULL CHECK(text_status IN ('page-text','document-text','no-text','failed')),
 tool text NOT NULL,
 text_file text NOT NULL DEFAULT '',
 text_sha256 text NOT NULL DEFAULT '',
 cover_path text NOT NULL DEFAULT '',
 cover_sha256 text NOT NULL DEFAULT '',
 cover_bytes bigint NOT NULL DEFAULT 0,
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(item_id,representation_id,file),
 FOREIGN KEY(item_id,representation_id) REFERENCES catalog_representations(item_id,id)
);
CREATE TABLE print_page_text (
 item_id text NOT NULL,
 representation_id text NOT NULL,
 file text NOT NULL,
 page integer NOT NULL CHECK(page BETWEEN 0 AND 10000),
 text text NOT NULL,
 search tsvector GENERATED ALWAYS AS (to_tsvector('simple',text)) STORED,
 PRIMARY KEY(item_id,representation_id,file,page),
 FOREIGN KEY(item_id,representation_id,file) REFERENCES print_indexes(item_id,representation_id,file)
);
CREATE INDEX print_page_text_search ON print_page_text USING gin(search);
