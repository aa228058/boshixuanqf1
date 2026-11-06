-- +goose Up
-- 预筛选统计：用户确认前不启动 Telegram 解析。
ALTER TABLE group_library_import_jobs
    ADD COLUMN IF NOT EXISTS raw_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS duplicate_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS invalid_count INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE group_library_import_jobs
    DROP COLUMN IF EXISTS invalid_count,
    DROP COLUMN IF EXISTS duplicate_count,
    DROP COLUMN IF EXISTS raw_count;
