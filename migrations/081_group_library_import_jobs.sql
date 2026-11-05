-- +goose Up
-- 持久化群组库批量导入任务；导入只完善公共群组库，不创建加群任务。
CREATE TABLE IF NOT EXISTS group_library_import_jobs (
    job_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL DEFAULT '',
    payload TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    total_count INTEGER NOT NULL DEFAULT 0,
    processed_count INTEGER NOT NULL DEFAULT 0,
    imported_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    skipped_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_group_library_import_jobs_pending
    ON group_library_import_jobs(status, created_at)
    WHERE status IN ('pending', 'processing');
CREATE INDEX IF NOT EXISTS idx_group_library_import_jobs_tenant
    ON group_library_import_jobs(tenant_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS group_library_import_jobs;
