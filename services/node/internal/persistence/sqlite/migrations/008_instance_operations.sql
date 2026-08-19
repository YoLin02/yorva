CREATE UNIQUE INDEX operations_one_active_instance_mutation
    ON operations(target_type, target_id)
    WHERE status IN ('PENDING', 'RUNNING')
      AND operation_type IN ('instance.create', 'instance.delete');
