-- +goose Up
-- Tenant-controlled, opt-in decorative suffix for broadcast messages.
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS auto_message_decoration_enabled BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE tenants
    DROP COLUMN IF EXISTS auto_message_decoration_enabled;
