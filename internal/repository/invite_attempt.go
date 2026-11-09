package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type InviteAttemptStatus string

const (
	InviteAttemptInProgress    InviteAttemptStatus = "in_progress"
	InviteAttemptSucceeded     InviteAttemptStatus = "succeeded"
	InviteAttemptFailedDefinite InviteAttemptStatus = "failed_definite"
	InviteAttemptOutcomeUnknown InviteAttemptStatus = "outcome_unknown"
)

type InviteAttemptKey struct {
	TenantID  uuid.UUID
	TaskID    uuid.UUID
	AccountID uuid.UUID
	TGUserID  int64
}

// ClaimInviteAttempt atomically claims a member for this task/account.
// Only a prior, explicitly definite failure can be retried. Succeeded,
// in-progress and outcome-unknown rows remain tombstones and are never claimed.
func (r *TaskRepository) ClaimInviteAttempt(ctx context.Context, key InviteAttemptKey) (bool, InviteAttemptStatus, error) {
	if key.TenantID == uuid.Nil || key.TaskID == uuid.Nil || key.AccountID == uuid.Nil || key.TGUserID <= 0 {
		return false, "", errors.New("invalid invite attempt key")
	}

	var status InviteAttemptStatus
	err := r.db.QueryRow(ctx, `
		INSERT INTO invite_member_attempts
			(tenant_id, task_id, account_id, tg_user_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'in_progress', NOW(), NOW())
		ON CONFLICT (tenant_id, task_id, account_id, tg_user_id) DO UPDATE
		SET status = 'in_progress', last_error = NULL, updated_at = NOW()
		WHERE invite_member_attempts.status = 'failed_definite'
		RETURNING status
	`, key.TenantID, key.TaskID, key.AccountID, key.TGUserID).Scan(&status)
	if err == nil {
		return true, status, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, "", fmt.Errorf("claim invite attempt: %w", err)
	}

	err = r.db.QueryRow(ctx, `
		SELECT status FROM invite_member_attempts
		WHERE tenant_id=$1 AND task_id=$2 AND account_id=$3 AND tg_user_id=$4
	`, key.TenantID, key.TaskID, key.AccountID, key.TGUserID).Scan(&status)
	if err != nil {
		return false, "", fmt.Errorf("read existing invite attempt: %w", err)
	}
	return false, status, nil
}

// FinishInviteAttempt only transitions a currently claimed attempt. Callers
// must use outcome_unknown whenever Telegram's result cannot be proven.
func (r *TaskRepository) FinishInviteAttempt(ctx context.Context, key InviteAttemptKey, status InviteAttemptStatus, lastError string) error {
	if status != InviteAttemptSucceeded && status != InviteAttemptFailedDefinite && status != InviteAttemptOutcomeUnknown {
		return fmt.Errorf("invalid terminal invite attempt status %q", status)
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE invite_member_attempts
		SET status=$5, last_error=NULLIF($6, ''), updated_at=NOW()
		WHERE tenant_id=$1 AND task_id=$2 AND account_id=$3 AND tg_user_id=$4
		  AND status='in_progress'
	`, key.TenantID, key.TaskID, key.AccountID, key.TGUserID, status, lastError)
	if err != nil {
		return fmt.Errorf("finish invite attempt: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("finish invite attempt: expected one in-progress row, affected %d", tag.RowsAffected())
	}
	return nil
}

// ResolveStaleInviteAttempts converts abandoned in-progress rows to unknown.
// It deliberately does not make them claimable: a process may have crashed
// after Telegram accepted the invite but before the result was persisted.
func (r *TaskRepository) ResolveStaleInviteAttempts(ctx context.Context, tenantID, taskID uuid.UUID, olderThan time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE invite_member_attempts
		SET status='outcome_unknown',
		    last_error=COALESCE(last_error, 'worker stopped before invite outcome was persisted'),
		    updated_at=NOW()
		WHERE tenant_id=$1 AND task_id=$2
		  AND status='in_progress' AND updated_at < $3
	`, tenantID, taskID, olderThan)
	if err != nil {
		return 0, fmt.Errorf("resolve stale invite attempts: %w", err)
	}
	return tag.RowsAffected(), nil
}
