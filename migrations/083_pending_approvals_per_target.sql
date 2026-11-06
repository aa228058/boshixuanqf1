-- +goose Up
-- A join task can have multiple accounts requesting approval for different targets.
-- Keep each account/target request independently pollable and resolvable.
ALTER TABLE pending_approvals
    DROP CONSTRAINT IF EXISTS pending_approvals_task_id_key;

ALTER TABLE pending_approvals
    ADD CONSTRAINT pending_approvals_task_account_target_key
    UNIQUE (task_id, account_id, target_link);

-- +goose Down
ALTER TABLE pending_approvals
    DROP CONSTRAINT IF EXISTS pending_approvals_task_account_target_key;

-- The previous schema allowed only one approval per task. A rollback may fail if
-- multiple approval rows now exist for a task; remove older duplicates first.
DELETE FROM pending_approvals a
USING pending_approvals b
WHERE a.task_id = b.task_id AND a.id > b.id;

ALTER TABLE pending_approvals
    ADD CONSTRAINT pending_approvals_task_id_key UNIQUE (task_id);
