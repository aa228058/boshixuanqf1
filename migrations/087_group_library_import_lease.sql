-- +goose Up
-- A claim token prevents a stale worker from committing progress after another
-- worker has reclaimed an expired processing job.
ALTER TABLE group_library_import_jobs
    ADD COLUMN IF NOT EXISTS lease_token UUID;

CREATE INDEX IF NOT EXISTS idx_group_library_import_jobs_processing_lease
    ON group_library_import_jobs(updated_at)
    WHERE status = 'processing';

-- +goose Down
DROP INDEX IF EXISTS idx_group_library_import_jobs_processing_lease;
ALTER TABLE group_library_import_jobs
    DROP COLUMN IF EXISTS lease_token;
