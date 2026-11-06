package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"tg-sender/internal/domain"
)

type PendingApprovalRepository struct {
	pool *pgxpool.Pool
}

func NewPendingApprovalRepository(pool *pgxpool.Pool) *PendingApprovalRepository {
	return &PendingApprovalRepository{pool: pool}
}

func (r *PendingApprovalRepository) Create(ctx context.Context, pa *domain.PendingApproval) error {
	// ON CONFLICT DO NOTHING is intentionally idempotent: a concurrent worker may
	// have inserted the same task/account/target approval after the caller's lookup.
	// PostgreSQL reports actual insert/constraint failures through err; zero affected
	// rows therefore means the unique approval already exists and is not an error.
	_, err := r.pool.Exec(ctx, `
		INSERT INTO pending_approvals (task_id, tenant_id, account_id, chat_id, access_hash, target_link, status, max_checks)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (task_id, account_id, target_link) DO NOTHING
	`, pa.TaskID, pa.TenantID, pa.AccountID, pa.ChatID, pa.AccessHash, pa.TargetLink, pa.Status, pa.MaxChecks)
	return err
}

func (r *PendingApprovalRepository) GetPending(ctx context.Context, tenantID uuid.UUID, limit int) ([]*domain.PendingApproval, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, task_id, tenant_id, account_id, chat_id, access_hash, target_link, status, check_count, max_checks, created_at, checked_at
		FROM pending_approvals
		WHERE status = 'pending' AND tenant_id = $1
		ORDER BY created_at
		LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domain.PendingApproval
	for rows.Next() {
		pa := &domain.PendingApproval{}
		if err := rows.Scan(&pa.ID, &pa.TaskID, &pa.TenantID, &pa.AccountID, &pa.ChatID, &pa.AccessHash, &pa.TargetLink, &pa.Status, &pa.CheckCount, &pa.MaxChecks, &pa.CreatedAt, &pa.CheckedAt); err != nil {
			return nil, fmt.Errorf("scan pending approval: %w", err)
		}
		results = append(results, pa)
	}
	return results, rows.Err()
}

// GetPendingAll retrieves pending approvals across all tenants (for background worker).
func (r *PendingApprovalRepository) GetPendingAll(ctx context.Context, limit int) ([]*domain.PendingApproval, error) {
	// G-03: 跳过非活跃/过期租户的待审批（其任务认领已被拦截，审批轮询同步跳过）。
	rows, err := r.pool.Query(ctx, `
		SELECT pa.id, pa.task_id, pa.tenant_id, pa.account_id, pa.chat_id, pa.access_hash, pa.target_link,
		       pa.status, pa.check_count, pa.max_checks, pa.created_at, pa.checked_at
		FROM pending_approvals pa
		JOIN tenants t ON t.tenant_id = pa.tenant_id
		JOIN tasks task ON task.tenant_id = pa.tenant_id AND task.task_id = pa.task_id
		WHERE pa.status = 'pending' AND task.status <> 'cancelled' AND t.status = 'active'
		  AND (t.expires_at IS NULL OR t.expires_at > NOW())
		ORDER BY pa.created_at
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domain.PendingApproval
	for rows.Next() {
		pa := &domain.PendingApproval{}
		if err := rows.Scan(&pa.ID, &pa.TaskID, &pa.TenantID, &pa.AccountID, &pa.ChatID, &pa.AccessHash, &pa.TargetLink, &pa.Status, &pa.CheckCount, &pa.MaxChecks, &pa.CreatedAt, &pa.CheckedAt); err != nil {
			return nil, fmt.Errorf("scan pending approval: %w", err)
		}
		results = append(results, pa)
	}
	return results, rows.Err()
}

// GetByAccountAndLink checks if there's already a pending approval for the same account + link.
func (r *PendingApprovalRepository) GetByAccountAndLink(ctx context.Context, accountID uuid.UUID, targetLink string) (*domain.PendingApproval, error) {
	pa := &domain.PendingApproval{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, task_id, tenant_id, account_id, chat_id, access_hash, target_link, status, check_count, max_checks, created_at, checked_at
		FROM pending_approvals
		WHERE account_id = $1 AND target_link = $2 AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT 1
	`, accountID, targetLink).Scan(&pa.ID, &pa.TaskID, &pa.TenantID, &pa.AccountID, &pa.ChatID, &pa.AccessHash, &pa.TargetLink, &pa.Status, &pa.CheckCount, &pa.MaxChecks, &pa.CreatedAt, &pa.CheckedAt)
	if err != nil {
		return nil, err
	}
	return pa, nil
}

// ResolvePending performs a compare-and-set so timeout and approval cannot win simultaneously.
func (r *PendingApprovalRepository) ResolvePending(ctx context.Context, tenantID uuid.UUID, approvalID int, status string) (bool, error) {
	if status != "approved" && status != "timeout" {
		return false, fmt.Errorf("invalid pending approval resolution status %q", status)
	}
	res, err := r.pool.Exec(ctx, `
		UPDATE pending_approvals SET status = $1, checked_at = NOW()
		WHERE tenant_id = $2 AND id = $3 AND status = 'pending'
	`, status, tenantID, approvalID)
	if err != nil {
		return false, fmt.Errorf("resolve pending approval: %w", err)
	}
	return res.RowsAffected() == 1, nil
}

// CancelPendingForTask stops background polling after its task is cancelled.
func (r *PendingApprovalRepository) CancelPendingForTask(ctx context.Context, tenantID, taskID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE pending_approvals
		SET status = 'cancelled', checked_at = NOW()
		WHERE tenant_id = $1 AND task_id = $2 AND status = 'pending'
		  AND EXISTS (
			SELECT 1 FROM tasks t
			WHERE t.tenant_id = $1 AND t.task_id = $2 AND t.status = 'cancelled'
		  )
	`, tenantID, taskID)
	return err
}

func (r *PendingApprovalRepository) IncrementCheck(ctx context.Context, tenantID uuid.UUID, approvalID int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE pending_approvals
		SET check_count = check_count + 1, checked_at = NOW()
		WHERE tenant_id = $1 AND id = $2 AND status = 'pending'
	`, tenantID, approvalID)
	return err
}
