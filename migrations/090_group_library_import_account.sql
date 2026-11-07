-- +goose Up
-- Persist the optional tenant-owned Telegram account selected for MTProto-assisted import resolution.
ALTER TABLE group_library_import_jobs
    ADD COLUMN IF NOT EXISTS account_id UUID REFERENCES accounts(account_id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_group_library_import_jobs_account
    ON group_library_import_jobs(account_id, status)
    WHERE account_id IS NOT NULL;

-- Cross-instance per-account MTProto request pacing for concurrent import workers.
CREATE TABLE IF NOT EXISTS group_library_mtproto_rate_limits (
    account_id UUID PRIMARY KEY REFERENCES accounts(account_id) ON DELETE CASCADE,
    next_allowed_at TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS group_library_mtproto_rate_limits;
DROP INDEX IF EXISTS idx_group_library_import_jobs_account;
ALTER TABLE group_library_import_jobs DROP COLUMN IF EXISTS account_id;
