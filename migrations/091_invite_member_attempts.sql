-- +goose Up
-- Durable per-member invite idempotency ledger.
-- Keep succeeded/unknown tombstones: automatic retries could send duplicate invitations.
CREATE TABLE invite_member_attempts (
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    tg_user_id BIGINT NOT NULL CHECK (tg_user_id > 0),
    status VARCHAR(24) NOT NULL CHECK (status IN ('in_progress', 'succeeded', 'failed_definite', 'outcome_unknown')),
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, task_id, account_id, tg_user_id)
);

CREATE INDEX idx_invite_member_attempts_task_stale
    ON invite_member_attempts (tenant_id, task_id, updated_at)
    WHERE status = 'in_progress';

-- +goose Down
DROP TABLE IF EXISTS invite_member_attempts;
