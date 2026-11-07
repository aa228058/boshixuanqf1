-- +goose Up
-- 群组库导入任务使用可续租租约与 fencing token，避免慢解析任务被其他 Worker 重复领取。
ALTER TABLE group_library_import_jobs
    ADD COLUMN IF NOT EXISTS lease_token UUID,
    ADD COLUMN IF NOT EXISTS lease_expires_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_group_library_import_jobs_lease
    ON group_library_import_jobs(status, lease_expires_at, created_at)
    WHERE status IN ('pending', 'processing');

-- +goose Down
DROP INDEX IF EXISTS idx_group_library_import_jobs_lease;
ALTER TABLE group_library_import_jobs
    DROP COLUMN IF EXISTS lease_expires_at,
    DROP COLUMN IF EXISTS lease_token;
