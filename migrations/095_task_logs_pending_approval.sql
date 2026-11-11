-- +goose Up
-- +goose StatementBegin
-- Join requests can remain pending until Telegram approves the account.
-- The executor persists this state in task_logs, so it must be accepted by
-- the database constraint as well as by the application.
ALTER TABLE task_logs DROP CONSTRAINT IF EXISTS task_logs_status_check;
ALTER TABLE task_logs ADD CONSTRAINT task_logs_status_check
  CHECK (status IN ('success','failed','skipped','info','pending','new',
    'not_supported','running','paused','wait','cancelled','pending_approval'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Do not remove pending_approval while rows using it may still exist.
-- The down migration intentionally refuses to narrow the constraint.
-- +goose StatementEnd
