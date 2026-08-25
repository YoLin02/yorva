DROP INDEX operations_one_active_instance_mutation;

CREATE UNIQUE INDEX operations_one_active_instance_mutation
    ON operations(target_type, target_id)
    WHERE status IN ('PENDING', 'RUNNING')
      AND target_type = 'runtime-installation'
      AND operation_type IN (
          'instance.create', 'instance.delete',
          'backup.create', 'backup.delete', 'backup.restore',
          'runtime.upgrade', 'runtime.rollback'
      );

DROP INDEX operations_one_active_instance_runtime_mutation;

CREATE UNIQUE INDEX operations_one_active_instance_runtime_mutation
    ON operations(target_type, target_id)
    WHERE status IN ('PENDING', 'RUNNING')
      AND target_type = 'instance'
      AND operation_type IN (
          'instance.start', 'instance.stop', 'instance.restart',
          'channel.connect', 'channel.disconnect',
          'skill.install', 'skill.update', 'skill.enable',
          'skill.disable', 'skill.remove',
          'mcp.install', 'mcp.authenticate', 'mcp.test',
          'mcp.configure', 'mcp.remove'
      );

CREATE TRIGGER operations_block_runtime_wide_mutation_when_instance_active
BEFORE INSERT ON operations
WHEN NEW.status IN ('PENDING', 'RUNNING')
  AND NEW.target_type = 'runtime-installation'
  AND NEW.operation_type IN ('backup.create', 'backup.restore', 'runtime.upgrade', 'runtime.rollback')
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
          AND active.operation_type IN ('backup.create', 'backup.restore', 'runtime.upgrade', 'runtime.rollback')
    );
END;
