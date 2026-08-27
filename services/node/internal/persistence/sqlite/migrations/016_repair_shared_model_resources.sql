CREATE TABLE IF NOT EXISTS model_provider_connections (
    id TEXT PRIMARY KEY CHECK (length(id) BETWEEN 1 AND 64),
    runtime_installation_id TEXT NOT NULL REFERENCES runtime_installations(id) ON DELETE CASCADE,
    provider_preset_id TEXT NOT NULL CHECK (length(provider_preset_id) BETWEEN 1 AND 64),
    display_name TEXT NOT NULL CHECK (length(display_name) BETWEEN 1 AND 80),
    secret_ref TEXT NOT NULL UNIQUE CHECK (length(secret_ref) BETWEEN 1 AND 128),
    status TEXT NOT NULL CHECK (status IN ('CONFIGURED', 'UNAVAILABLE')),
    revision INTEGER NOT NULL CHECK (revision > 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS model_profiles (
    id TEXT PRIMARY KEY CHECK (length(id) BETWEEN 1 AND 64),
    runtime_installation_id TEXT NOT NULL REFERENCES runtime_installations(id) ON DELETE CASCADE,
    provider_connection_id TEXT NOT NULL REFERENCES model_provider_connections(id) ON DELETE RESTRICT,
    display_name TEXT NOT NULL CHECK (length(display_name) BETWEEN 1 AND 80),
    selected_model_ids_json TEXT NOT NULL,
    default_model_id TEXT NOT NULL CHECK (length(default_model_id) BETWEEN 1 AND 200),
    revision INTEGER NOT NULL CHECK (revision > 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_model_defaults (
    runtime_installation_id TEXT PRIMARY KEY REFERENCES runtime_installations(id) ON DELETE CASCADE,
    model_profile_id TEXT NOT NULL REFERENCES model_profiles(id) ON DELETE RESTRICT,
    applied_revision INTEGER NOT NULL CHECK (applied_revision > 0),
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS instance_model_bindings (
    instance_id TEXT PRIMARY KEY REFERENCES instances(id) ON DELETE CASCADE,
    model_profile_id TEXT NOT NULL REFERENCES model_profiles(id) ON DELETE RESTRICT,
    mode TEXT NOT NULL CHECK (mode IN ('INHERIT', 'OVERRIDE')),
    applied_revision INTEGER NOT NULL CHECK (applied_revision > 0),
    state TEXT NOT NULL CHECK (state IN ('PENDING', 'SUCCEEDED', 'FAILED', 'SKIPPED')),
    error_code TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL
);

DROP TRIGGER IF EXISTS operations_block_runtime_wide_mutation_when_instance_active;
DROP TRIGGER IF EXISTS operations_block_instance_mutation_when_runtime_wide_active;
DROP INDEX IF EXISTS operations_one_active_instance_mutation;

CREATE UNIQUE INDEX operations_one_active_instance_mutation
    ON operations(target_type, target_id)
    WHERE status IN ('PENDING', 'RUNNING')
      AND target_type = 'runtime-installation'
      AND operation_type IN (
          'instance.create', 'instance.delete', 'model.profile.apply',
          'backup.create', 'backup.delete', 'backup.restore',
          'runtime.upgrade', 'runtime.rollback'
      );

CREATE TRIGGER operations_block_runtime_wide_mutation_when_instance_active
BEFORE INSERT ON operations
WHEN NEW.status IN ('PENDING', 'RUNNING')
  AND NEW.target_type = 'runtime-installation'
  AND NEW.operation_type IN ('model.profile.apply', 'backup.create', 'backup.restore', 'runtime.upgrade', 'runtime.rollback')
BEGIN
    SELECT RAISE(ABORT, 'operations_one_active_instance_runtime_mutation')
    WHERE EXISTS (
        SELECT 1
        FROM operations active
        JOIN instances instance_row ON instance_row.id = active.target_id
        WHERE active.target_type = 'instance'
          AND instance_row.runtime_installation_id = NEW.target_id
          AND active.status IN ('PENDING', 'RUNNING')
          AND active.operation_type IN (
              'instance.start', 'instance.stop', 'instance.restart',
              'channel.connect', 'channel.disconnect',
              'skill.install', 'skill.update', 'skill.enable',
              'skill.disable', 'skill.remove',
              'mcp.install', 'mcp.authenticate', 'mcp.test',
              'mcp.configure', 'mcp.remove'
          )
    );
END;

CREATE TRIGGER operations_block_instance_mutation_when_runtime_wide_active
BEFORE INSERT ON operations
WHEN NEW.status IN ('PENDING', 'RUNNING')
  AND NEW.target_type = 'instance'
  AND NEW.operation_type IN (
      'instance.start', 'instance.stop', 'instance.restart',
      'channel.connect', 'channel.disconnect',
      'skill.install', 'skill.update', 'skill.enable',
      'skill.disable', 'skill.remove',
      'mcp.install', 'mcp.authenticate', 'mcp.test',
      'mcp.configure', 'mcp.remove'
  )
BEGIN
    SELECT RAISE(ABORT, 'operations_one_active_instance_runtime_mutation')
    WHERE EXISTS (
        SELECT 1
        FROM instances instance_row
        JOIN operations active
          ON active.target_type = 'runtime-installation'
         AND active.target_id = instance_row.runtime_installation_id
        WHERE instance_row.id = NEW.target_id
          AND active.status IN ('PENDING', 'RUNNING')
          AND active.operation_type IN ('model.profile.apply', 'backup.create', 'backup.restore', 'runtime.upgrade', 'runtime.rollback')
    );
END;
