CREATE TABLE managed_skills (
    id TEXT PRIMARY KEY CHECK (
        length(id) BETWEEN 1 AND 128
        AND id NOT GLOB '*[^A-Za-z0-9._-]*'
        AND id NOT IN ('.', '..')
    ),
    instance_id TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    skill_id TEXT NOT NULL CHECK (
        length(skill_id) BETWEEN 1 AND 128
        AND skill_id NOT GLOB '*[^A-Za-z0-9._-]*'
        AND skill_id NOT IN ('.', '..')
    ),
    source_id TEXT NOT NULL CHECK (
        length(source_id) BETWEEN 1 AND 128
        AND source_id NOT GLOB '*[^A-Za-z0-9._-]*'
        AND source_id NOT IN ('.', '..')
    ),
    source_version TEXT NOT NULL CHECK (length(source_version) BETWEEN 1 AND 4096),
    content_sha256 TEXT NOT NULL CHECK (
        length(content_sha256) = 64
        AND content_sha256 = lower(content_sha256)
        AND content_sha256 NOT GLOB '*[^0-9a-f]*'
    ),
    managed_relative_path TEXT NOT NULL CHECK (
        length(managed_relative_path) BETWEEN 1 AND 4096
        AND substr(managed_relative_path, 1, 1) <> '/'
        AND instr(managed_relative_path, '\') = 0
        AND instr(managed_relative_path, ':') = 0
        AND managed_relative_path NOT LIKE '//%'
        AND managed_relative_path NOT LIKE '%//%'
        AND managed_relative_path <> '.'
        AND managed_relative_path <> '..'
        AND managed_relative_path NOT LIKE './%'
        AND managed_relative_path NOT LIKE '../%'
        AND managed_relative_path NOT LIKE '%/./%'
        AND managed_relative_path NOT LIKE '%/../%'
        AND managed_relative_path NOT LIKE '%/.'
        AND managed_relative_path NOT LIKE '%/..'
    ),
    projection_relative_path TEXT NOT NULL CHECK (
        length(projection_relative_path) BETWEEN 1 AND 4096
        AND substr(projection_relative_path, 1, 1) <> '/'
        AND instr(projection_relative_path, '\') = 0
        AND instr(projection_relative_path, ':') = 0
        AND projection_relative_path NOT LIKE '//%'
        AND projection_relative_path NOT LIKE '%//%'
        AND projection_relative_path <> '.'
        AND projection_relative_path <> '..'
        AND projection_relative_path NOT LIKE './%'
        AND projection_relative_path NOT LIKE '../%'
        AND projection_relative_path NOT LIKE '%/./%'
        AND projection_relative_path NOT LIKE '%/../%'
        AND projection_relative_path NOT LIKE '%/.'
        AND projection_relative_path NOT LIKE '%/..'
    ),
    deployment_id TEXT NOT NULL UNIQUE CHECK (
        length(deployment_id) BETWEEN 1 AND 128
        AND deployment_id NOT GLOB '*[^A-Za-z0-9._-]*'
        AND deployment_id NOT IN ('.', '..')
    ),
    desired_enabled INTEGER NOT NULL CHECK (desired_enabled IN (0, 1)),
    projection_state TEXT NOT NULL CHECK (
        projection_state IN (
            'PROJECTED', 'NOT_PROJECTED', 'DRIFT_MISSING',
            'DRIFT_MODIFIED', 'CONFLICT', 'UNKNOWN'
        )
    ),
    installed_at TEXT NOT NULL CHECK (length(installed_at) BETWEEN 1 AND 64),
    updated_at TEXT NOT NULL CHECK (length(updated_at) BETWEEN 1 AND 64),
    UNIQUE(instance_id, skill_id)
);

CREATE INDEX managed_skills_instance_id
    ON managed_skills(instance_id);

DROP INDEX operations_one_active_instance_runtime_mutation;

CREATE UNIQUE INDEX operations_one_active_instance_runtime_mutation
    ON operations(target_type, target_id)
    WHERE status IN ('PENDING', 'RUNNING')
      AND target_type = 'instance'
      AND operation_type IN (
          'instance.start', 'instance.stop', 'instance.restart',
          'channel.connect', 'channel.disconnect',
          'skill.install', 'skill.update', 'skill.enable',
          'skill.disable', 'skill.remove'
      );
