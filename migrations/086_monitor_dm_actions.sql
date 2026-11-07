-- +goose Up
-- Durable idempotency ledger for each DM sub-action triggered by a specific Telegram message.
CREATE TABLE monitor_dm_actions (
    tenant_id UUID NOT NULL,
    account_id UUID NOT NULL,
    rule_id UUID NOT NULL,
    target_id BIGINT NOT NULL,
    source_chat_id BIGINT NOT NULL,
    source_message_id INTEGER NOT NULL CHECK (source_message_id > 0),
    action VARCHAR(16) NOT NULL CHECK (action IN ('text', 'contact', 'call')),
    status VARCHAR(24) NOT NULL CHECK (status IN ('in_progress', 'succeeded', 'failed_definite', 'outcome_unknown')),
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, account_id, rule_id, target_id, source_chat_id, source_message_id, action)
);

CREATE INDEX idx_monitor_dm_actions_created_at
    ON monitor_dm_actions (created_at);

-- +goose Down
DROP TABLE IF EXISTS monitor_dm_actions;
