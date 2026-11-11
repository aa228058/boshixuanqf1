-- +goose Up
-- Durable ten-minute voice-call cooldown across instances and restarts.
CREATE TABLE monitor_call_cooldowns (
    tenant_id UUID NOT NULL,
    account_id UUID NOT NULL,
    callee_id BIGINT NOT NULL,
    last_called_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, account_id, callee_id)
);

CREATE INDEX idx_monitor_call_cooldowns_last_called_at
    ON monitor_call_cooldowns (last_called_at);

-- +goose Down
DROP TABLE IF EXISTS monitor_call_cooldowns;
