CREATE TABLE asset_companions (
 asset_sha text NOT NULL REFERENCES assets(sha256),
 companion_sha text NOT NULL REFERENCES assets(sha256),
 filename text NOT NULL,
 source_snapshot text NOT NULL REFERENCES source_snapshots(id),
 PRIMARY KEY(asset_sha,companion_sha)
);
