CREATE TABLE instances (
    id TEXT PRIMARY KEY,
    runtime_installation_id TEXT NOT NULL REFERENCES runtime_installations(id),
    native_id TEXT NOT NULL,
    name TEXT NOT NULL,
    is_default INTEGER NOT NULL DEFAULT 0 CHECK (is_default IN (0, 1)),
    is_protected INTEGER NOT NULL DEFAULT 0 CHECK (is_protected IN (0, 1)),
    availability TEXT NOT NULL CHECK (availability IN ('AVAILABLE', 'MISSING', 'UNKNOWN')),
    last_synced_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(runtime_installation_id, native_id)
);

CREATE INDEX instances_runtime_installation_id
    ON instances(runtime_installation_id);
