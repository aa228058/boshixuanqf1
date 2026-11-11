-- +goose Up
-- One monotonically increasing number space shared by regular tasks and group-import jobs.
CREATE SEQUENCE IF NOT EXISTS global_task_number_seq AS BIGINT;

ALTER TABLE tasks ADD COLUMN IF NOT EXISTS task_number BIGINT;
ALTER TABLE group_library_import_jobs ADD COLUMN IF NOT EXISTS task_number BIGINT;

-- Backfill both tables in creation-time order so historical numbers are also globally unique.
CREATE TEMP TABLE global_task_number_backfill ON COMMIT DROP AS
SELECT entity_type, entity_id, ROW_NUMBER() OVER (ORDER BY created_at, entity_type, entity_id)::BIGINT AS task_number
FROM (
    SELECT 1::SMALLINT AS entity_type, task_id AS entity_id, created_at FROM tasks
    UNION ALL
    SELECT 2::SMALLINT AS entity_type, job_id AS entity_id, created_at FROM group_library_import_jobs
) AS all_jobs;

UPDATE tasks AS t
SET task_number = b.task_number
FROM global_task_number_backfill AS b
WHERE b.entity_type = 1 AND b.entity_id = t.task_id;

UPDATE group_library_import_jobs AS j
SET task_number = b.task_number
FROM global_task_number_backfill AS b
WHERE b.entity_type = 2 AND b.entity_id = j.job_id;

SELECT setval(
    'global_task_number_seq',
    GREATEST(COALESCE((
        SELECT MAX(task_number) FROM (
            SELECT task_number FROM tasks
            UNION ALL
            SELECT task_number FROM group_library_import_jobs
        ) AS numbered
    ), 1), 1),
    EXISTS (
        SELECT 1 FROM tasks WHERE task_number IS NOT NULL
        UNION ALL
        SELECT 1 FROM group_library_import_jobs WHERE task_number IS NOT NULL
    )
);

ALTER TABLE tasks ALTER COLUMN task_number SET DEFAULT nextval('global_task_number_seq');
ALTER TABLE group_library_import_jobs ALTER COLUMN task_number SET DEFAULT nextval('global_task_number_seq');
ALTER TABLE tasks ALTER COLUMN task_number SET NOT NULL;
ALTER TABLE group_library_import_jobs ALTER COLUMN task_number SET NOT NULL;

CREATE UNIQUE INDEX idx_tasks_task_number ON tasks(task_number);
CREATE UNIQUE INDEX idx_group_library_import_jobs_task_number ON group_library_import_jobs(task_number);

-- +goose Down
DROP INDEX IF EXISTS idx_group_library_import_jobs_task_number;
DROP INDEX IF EXISTS idx_tasks_task_number;
ALTER TABLE group_library_import_jobs DROP COLUMN IF EXISTS task_number;
ALTER TABLE tasks DROP COLUMN IF EXISTS task_number;
DROP SEQUENCE IF EXISTS global_task_number_seq;
