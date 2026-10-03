ALTER TABLE collections ADD COLUMN source_item text NOT NULL DEFAULT '';
ALTER TABLE collections ADD CONSTRAINT selected_item_policy CHECK (source_item='' OR (policy='local_preserve' AND id<>'democracy_now_vid' AND source_item<>'democracy_now_vid'));
