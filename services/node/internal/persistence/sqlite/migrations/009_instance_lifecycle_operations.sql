CREATE UNIQUE INDEX operations_one_active_instance_lifecycle
    ON operations(target_type, target_id)
    WHERE status IN ('PENDING', 'RUNNING')
      AND target_type = 'instance'
      AND operation_type IN ('instance.start', 'instance.stop', 'instance.restart');
