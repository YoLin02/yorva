CREATE TABLE database_migration_history (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    source_version    INTEGER NOT NULL CHECK (source_version >= 0),
    target_version    INTEGER NOT NULL CHECK (target_version > source_version),
    protection_sha256 TEXT NOT NULL CHECK (length(protection_sha256) IN (0, 64)),
    state             TEXT NOT NULL CHECK (state = 'SUCCEEDED'),
    started_at        TEXT NOT NULL,
    completed_at      TEXT NOT NULL
);
