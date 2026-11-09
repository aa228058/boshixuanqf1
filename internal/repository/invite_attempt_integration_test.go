//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestInviteAttemptLedgerClaimsAndTransitions(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	tenantID, accountID, taskID := uuid.New(), uuid.New(), uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (tenant_id, username, password_hash, name, api_key_hash, plan_id)
		VALUES ($1, $2, 'x', 'Invite ledger', $3, 'free')
	`, tenantID, "invite_"+tenantID.String(), "invite-ledger-"+tenantID.String())
	if err != nil {
		t.Fatalf("seed invite ledger tenant: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO accounts (account_id, tenant_id, phone, session_name, status)
		VALUES ($1, $2, '+10000000001', 'invite-ledger', 'offline')
	`, accountID, tenantID); err != nil {
		t.Fatalf("seed invite ledger account: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO tasks (task_id, tenant_id, type, status)
		VALUES ($1, $2, 'broadcast', 'running')
	`, taskID, tenantID); err != nil {
		t.Fatalf("seed invite ledger task: %v", err)
	}

	repo := NewTaskRepository(pool)
	key := InviteAttemptKey{TenantID: tenantID, TaskID: taskID, AccountID: accountID, TGUserID: 123456}
	claimed, status, err := repo.ClaimInviteAttempt(ctx, key)
	if err != nil || !claimed || status != InviteAttemptInProgress {
		t.Fatalf("first claim = (%v, %q, %v), want claimed/in_progress", claimed, status, err)
	}

	claimed, status, err = repo.ClaimInviteAttempt(ctx, key)
	if err != nil || claimed || status != InviteAttemptInProgress {
		t.Fatalf("duplicate claim = (%v, %q, %v), want blocked/in_progress", claimed, status, err)
	}

	if err := repo.FinishInviteAttempt(ctx, key, InviteAttemptOutcomeUnknown, "connection reset"); err != nil {
		t.Fatalf("finish unknown: %v", err)
	}
	claimed, status, err = repo.ClaimInviteAttempt(ctx, key)
	if err != nil || claimed || status != InviteAttemptOutcomeUnknown {
		t.Fatalf("unknown claim = (%v, %q, %v), want blocked/outcome_unknown", claimed, status, err)
	}

	// A definite failure is the only state eligible for an explicit retry.
	key2 := key
	key2.TGUserID++
	if claimed, _, err := repo.ClaimInviteAttempt(ctx, key2); err != nil || !claimed {
		t.Fatalf("claim second member: claimed=%v err=%v", claimed, err)
	}
	if err := repo.FinishInviteAttempt(ctx, key2, InviteAttemptFailedDefinite, "USER_PRIVACY_RESTRICTED"); err != nil {
		t.Fatalf("finish definite failure: %v", err)
	}
	claimed, status, err = repo.ClaimInviteAttempt(ctx, key2)
	if err != nil || !claimed || status != InviteAttemptInProgress {
		t.Fatalf("retry definite failure = (%v, %q, %v), want claimed/in_progress", claimed, status, err)
	}
	if err := repo.FinishInviteAttempt(ctx, key2, InviteAttemptSucceeded, ""); err != nil {
		t.Fatalf("finish success: %v", err)
	}
	claimed, status, err = repo.ClaimInviteAttempt(ctx, key2)
	if err != nil || claimed || status != InviteAttemptSucceeded {
		t.Fatalf("successful claim = (%v, %q, %v), want blocked/succeeded", claimed, status, err)
	}
}

func TestResolveStaleInviteAttemptsNeverMakesThemRetryable(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	tenantID, accountID, taskID := uuid.New(), uuid.New(), uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (tenant_id, username, password_hash, name, api_key_hash, plan_id)
		VALUES ($1, $2, 'x', 'Invite stale', $3, 'free')
	`, tenantID, "stale_"+tenantID.String(), "invite-stale-"+tenantID.String())
	if err != nil {
		t.Fatalf("seed stale tenant: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO accounts (account_id, tenant_id, phone, session_name, status)
		VALUES ($1, $2, '+10000000002', 'invite-stale', 'offline')
	`, accountID, tenantID); err != nil {
		t.Fatalf("seed stale account: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO tasks (task_id, tenant_id, type, status)
		VALUES ($1, $2, 'broadcast', 'running')
	`, taskID, tenantID); err != nil {
		t.Fatalf("seed stale task: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO invite_member_attempts (tenant_id, task_id, account_id, tg_user_id, status, updated_at)
		VALUES ($1, $2, $3, 654321, 'in_progress', NOW() - INTERVAL '10 minutes')
	`, tenantID, taskID, accountID); err != nil {
		t.Fatalf("seed stale attempt: %v", err)
	}

	repo := NewTaskRepository(pool)
	n, err := repo.ResolveStaleInviteAttempts(ctx, time.Now().Add(-5*time.Minute))
	if err != nil || n != 1 {
		t.Fatalf("resolve stale = (%d, %v), want (1, nil)", n, err)
	}
	key := InviteAttemptKey{TenantID: tenantID, TaskID: taskID, AccountID: accountID, TGUserID: 654321}
	claimed, status, err := repo.ClaimInviteAttempt(ctx, key)
	if err != nil || claimed || status != InviteAttemptOutcomeUnknown {
		t.Fatalf("stale claim = (%v, %q, %v), want blocked/outcome_unknown", claimed, status, err)
	}
}
