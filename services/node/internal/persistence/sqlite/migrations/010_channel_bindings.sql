DROP INDEX operations_one_active_instance_lifecycle;

CREATE UNIQUE INDEX operations_one_active_instance_runtime_mutation
    ON operations(target_type, target_id)
    WHERE status IN ('PENDING', 'RUNNING')
      AND target_type = 'instance'
      AND operation_type IN (
          'instance.start', 'instance.stop', 'instance.restart',
          'channel.connect', 'channel.disconnect'
      );

CREATE TABLE channel_bindings (
    id TEXT PRIMARY KEY,
    instance_id TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    channel_type TEXT NOT NULL CHECK (channel_type IN ('weixin', 'wecom')),
    state TEXT NOT NULL CHECK (state IN ('NOT_CONFIGURED', 'CONNECTING', 'CONNECTED', 'DISCONNECTED', 'FAILED', 'UNKNOWN')),
    account_label TEXT NOT NULL DEFAULT '',
    external_id TEXT NOT NULL DEFAULT '',
    metadata_json TEXT NOT NULL DEFAULT '{}' CHECK (metadata_json = '{}'),
    last_checked_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(instance_id, channel_type)
);

CREATE INDEX channel_bindings_instance_id
    ON channel_bindings(instance_id);
