-- +goose Up
-- Shared group-level default speaking rule. This is not per-account mute state.
ALTER TABLE system_groups
    ADD COLUMN IF NOT EXISTS free_speech_status VARCHAR(16) NOT NULL DEFAULT 'unknown';

-- Membership belongs to tenant/account state, never to the shared group library.
UPDATE system_groups SET status='active' WHERE status='joined';

-- +goose Down
ALTER TABLE system_groups
    DROP COLUMN IF EXISTS free_speech_status;
