-- +goose Up
-- Stable keyset pagination for task detail logs while new rows are being inserted.
CREATE INDEX IF NOT EXISTS idx_task_logs_tenant_task_cursor
    ON task_logs (tenant_id, task_id, executed_at DESC, log_id DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_task_logs_tenant_task_cursor;
