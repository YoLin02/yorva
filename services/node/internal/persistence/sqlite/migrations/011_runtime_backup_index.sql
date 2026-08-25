-- Phase 7 Runtime-scoped backup index.
--
-- This is deliberately a new table. Any predecessor `backups` table is
-- Instance-scoped and must remain untouched; its rows cannot truthfully be
-- reinterpreted as Runtime-scoped backups.
CREATE TABLE runtime_backups (
    id TEXT PRIMARY KEY,
    runtime_installation_id TEXT NOT NULL REFERENCES runtime_installations(id),
    scope_type TEXT NOT NULL CHECK (scope_type = 'RUNTIME'),
    format_version TEXT NOT NULL CHECK (length(format_version) BETWEEN 1 AND 64),
    runtime_version TEXT NOT NULL CHECK (length(runtime_version) BETWEEN 1 AND 64),
    artifact_path TEXT NOT NULL CHECK (length(artifact_path) BETWEEN 1 AND 32767),
    size_bytes INTEGER NOT NULL CHECK (size_bytes > 0),
    checksum_sha256 TEXT NOT NULL CHECK (
        length(checksum_sha256) = 64
        AND checksum_sha256 = lower(checksum_sha256)
        AND checksum_sha256 NOT GLOB '*[^0-9a-f]*'
    ),
    state TEXT NOT NULL CHECK (
        state IN ('AVAILABLE', 'MISSING', 'CHANGED', 'UNDECRYPTABLE', 'MALFORMED', 'UNKNOWN')
    ),
    key_mode TEXT NOT NULL CHECK (key_mode IN ('DEVICE', 'PASSPHRASE')),
    key_ref TEXT,
    created_at TEXT NOT NULL,
    verified_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK (
        (key_mode = 'DEVICE' AND key_ref IS NOT NULL AND length(key_ref) BETWEEN 1 AND 256)
        OR (key_mode = 'PASSPHRASE' AND key_ref IS NULL)
    ),
    UNIQUE(runtime_installation_id, artifact_path)
);

CREATE INDEX runtime_backups_runtime_created_at
    ON runtime_backups(runtime_installation_id, created_at DESC, id);
