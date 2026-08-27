CREATE TABLE managed_mcp_bindings (
    instance_id TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    server_id TEXT NOT NULL CHECK (length(server_id) BETWEEN 1 AND 128),
    preset_id TEXT NOT NULL CHECK (length(preset_id) BETWEEN 1 AND 128),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (instance_id, server_id)
);
