-- +goose Up
-- Preserve per-link failures so users can retry only failed entries after a completed batch.
CREATE TABLE IF NOT EXISTS group_library_import_job_failures (
    job_id UUID NOT NULL REFERENCES group_library_import_jobs(job_id) ON DELETE CASCADE,
    original_link TEXT NOT NULL,
    error_text TEXT NOT NULL DEFAULT '',
    retry_job_id UUID REFERENCES group_library_import_jobs(job_id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (job_id, original_link)
);
CREATE INDEX IF NOT EXISTS idx_group_library_import_job_failures_retry
    ON group_library_import_job_failures(job_id, retry_job_id);

-- +goose Down
DROP TABLE IF EXISTS group_library_import_job_failures;
