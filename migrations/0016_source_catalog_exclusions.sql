-- Listing decisions are independent of imported source metadata and survive refresh.
ALTER TABLE source_catalog_items ADD COLUMN excluded boolean NOT NULL DEFAULT false;
