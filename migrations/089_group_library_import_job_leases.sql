-- +goose Up
-- Extend the lease-token fencing introduced in 087 with an explicit expiry.
-- lease_token is owned by migration 087 and must never be dropped here.
ALTER TABLE group_library_import_jobs
    ADD COLUMN IF NOT EXISTS lease_expires_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_group_library_import_jobs_lease
    ON group_library_import_jobs(status, lease_expires_at, created_at)
    WHERE status IN ('pending', 'processing', 'cancelling');

-- +goose Down
DROP INDEX IF EXISTS idx_group_library_import_jobs_lease;
ALTER TABLE group_library_import_jobs
    DROP COLUMN IF EXISTS lease_expires_at;
