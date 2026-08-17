CREATE TABLE operations (
    id TEXT PRIMARY KEY,
    operation_type TEXT NOT NULL,
    target_type TEXT,
    target_id TEXT,
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'RUNNING', 'SUCCEEDED', 'FAILED', 'CANCELLED')),
    stage TEXT,
    progress INTEGER,
    message TEXT,
    error_code TEXT,
    error_message TEXT,
    retryable INTEGER NOT NULL DEFAULT 0 CHECK (retryable IN (0, 1)),
    idempotency_key TEXT NOT NULL,
    correlation_id TEXT,
    created_at TEXT NOT NULL,
    started_at TEXT,
    completed_at TEXT,
    updated_at TEXT NOT NULL,
    CHECK (
        (status IN ('PENDING', 'RUNNING') AND completed_at IS NULL)
        OR (status IN ('SUCCEEDED', 'FAILED', 'CANCELLED') AND completed_at IS NOT NULL)
    )
);

CREATE UNIQUE INDEX operations_idempotency_key ON operations(idempotency_key);

CREATE INDEX operations_status_created_at ON operations(status, created_at);

CREATE INDEX operations_target_created_at ON operations(target_id, created_at);

CREATE UNIQUE INDEX operations_one_active_runtime_install
    ON operations(operation_type, target_type, target_id)
    WHERE status IN ('PENDING', 'RUNNING') AND operation_type = 'runtime.install';

CREATE TABLE runtime_installations (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL REFERENCES nodes(id),
    runtime_kind TEXT NOT NULL,
    install_path TEXT NOT NULL,
    version TEXT NOT NULL,
    support_state TEXT NOT NULL,
    status TEXT NOT NULL,
    metadata_json TEXT,
    last_detected_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(node_id, runtime_kind, install_path)
);
