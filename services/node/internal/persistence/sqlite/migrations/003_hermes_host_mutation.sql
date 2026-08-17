DROP INDEX IF EXISTS operations_one_active_runtime_install;

CREATE UNIQUE INDEX operations_one_active_hermes_host_mutation
    ON operations(target_type, target_id)
    WHERE status IN ('PENDING', 'RUNNING')
      AND operation_type IN ('runtime.install', 'hermes.prerequisites');
