package repository

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"tg-sender/internal/domain"
)

// TaskRepository handles task data access.
type TaskRepository struct {
	db *pgxpool.Pool
}

// NewTaskRepository creates a new task repository.
func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{db: db}
}

// List retrieves tasks for a tenant, optionally filtered by status and account_id.
// When accountID is provided, only tasks that include this account in account_ids are returned.
func (r *TaskRepository) List(ctx context.Context, tenantID uuid.UUID, status *domain.TaskStatus, accountID *uuid.UUID, taskType *domain.TaskType, hideLoops bool, since *time.Time, limit, offset int) ([]*domain.Task, error) {
	// 动态拼装 WHERE, 所有过滤条件(状态/账号/类型)都在 SQL 层生效, 避免翻页后再过滤导致"筛选没反应"
	query := `
		SELECT task_id, tenant_id, type, status, priority, account_ids, target_ids, payload, progress,
		       locked_at, worker_id, retry_count, scheduled_at, started_at, completed_at, created_at,
		       is_loop, loop_interval_minutes, min_interval_seconds, max_interval_seconds, next_run_at
		FROM tasks WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	idx := 2
	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, *status)
		idx++
	}
	if accountID != nil {
		query += fmt.Sprintf(" AND account_ids @> $%d::jsonb", idx)
		args = append(args, fmt.Sprintf(`["%s"]`, accountID.String()))
		idx++
	}
	if taskType != nil && *taskType != "" {
		query += fmt.Sprintf(" AND type = $%d", idx)
		args = append(args, *taskType)
		idx++
	}
	if hideLoops {
		// 任务列表页隐藏循环包装任务, 只显示实际执行记录
		query += " AND is_loop = FALSE"
	}
	if since != nil {
		// 今日/近7日筛选(任务列表页时间范围Tab)
		query += fmt.Sprintf(" AND created_at >= $%d", idx)
		args = append(args, *since)
		idx++
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", idx, idx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	return r.scanTasks(rows)
}

// GetByID retrieves a task by ID.
// GetByTaskIDAnyTenant 仅用于取消落库等系统级操作(普通业务必须用带tenant的GetByID)。
func (r *TaskRepository) GetByTaskIDAnyTenant(ctx context.Context, taskID uuid.UUID) (*domain.Task, error) {
	var t domain.Task
	var lockedAt pgtype.Timestamptz
	var workerID pgtype.Text
	var scheduledAt, startedAt, completedAt, nextRunAt pgtype.Timestamptz
	err := r.db.QueryRow(ctx, `
		SELECT task_id, tenant_id, type, status, priority, account_ids, target_ids, payload, progress,
		       locked_at, worker_id, retry_count, scheduled_at, started_at, completed_at, created_at,
		       is_loop, loop_interval_minutes, min_interval_seconds, max_interval_seconds, next_run_at
		FROM tasks WHERE task_id = $1`, taskID,
	).Scan(&t.TaskID, &t.TenantID, &t.Type, &t.Status, &t.Priority, &t.AccountIDs, &t.TargetIDs, &t.Payload, &t.Progress,
		&lockedAt, &workerID, &t.RetryCount, &scheduledAt, &startedAt, &completedAt, &t.CreatedAt,
		&t.IsLoop, &t.LoopIntervalMinutes, &t.MinIntervalSeconds, &t.MaxIntervalSeconds, &nextRunAt)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	// P3修复(精读发现): 查了 next_run_at 但漏赋值, 与 GetByID/scanTasks 行为不一致
	if nextRunAt.Valid {
		tt := nextRunAt.Time
		t.NextRunAt = &tt
	}
	if lockedAt.Valid {
		tt := lockedAt.Time
		t.LockedAt = &tt
	}
	if workerID.Valid {
		t.WorkerID = &workerID.String
	}
	if scheduledAt.Valid {
		tt := scheduledAt.Time
		t.ScheduledAt = &tt
	}
	if startedAt.Valid {
		tt := startedAt.Time
		t.StartedAt = &tt
	}
	if completedAt.Valid {
		tt := completedAt.Time
		t.CompletedAt = &tt
	}
	return &t, nil
}

func (r *TaskRepository) GetByID(ctx context.Context, tenantID, taskID uuid.UUID) (*domain.Task, error) {
	var t domain.Task
	var lockedAt pgtype.Timestamptz
	var workerID pgtype.Text
	var scheduledAt pgtype.Timestamptz
	var startedAt pgtype.Timestamptz
	var completedAt pgtype.Timestamptz
	var nextRunAt pgtype.Timestamptz

	err := r.db.QueryRow(ctx, `
		SELECT task_id, tenant_id, type, status, priority, account_ids, target_ids, payload, progress,
		       locked_at, worker_id, retry_count, scheduled_at, started_at, completed_at, created_at,
					       is_loop, loop_interval_minutes, min_interval_seconds, max_interval_seconds, next_run_at
		FROM tasks WHERE tenant_id = $1 AND task_id = $2`, tenantID, taskID,
	).Scan(&t.TaskID, &t.TenantID, &t.Type, &t.Status, &t.Priority, &t.AccountIDs, &t.TargetIDs, &t.Payload, &t.Progress,
		&lockedAt, &workerID, &t.RetryCount, &scheduledAt, &startedAt, &completedAt, &t.CreatedAt,
		&t.IsLoop, &t.LoopIntervalMinutes, &t.MinIntervalSeconds, &t.MaxIntervalSeconds, &nextRunAt)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if lockedAt.Valid {
		tt := lockedAt.Time
		t.LockedAt = &tt
	}
	if workerID.Valid {
		t.WorkerID = &workerID.String
	}
	if scheduledAt.Valid {
		tt := scheduledAt.Time
		t.ScheduledAt = &tt
	}
	if startedAt.Valid {
		tt := startedAt.Time
		t.StartedAt = &tt
	}
	if completedAt.Valid {
		tt := completedAt.Time
		t.CompletedAt = &tt
	}
	if nextRunAt.Valid {
		tt := nextRunAt.Time
		t.NextRunAt = &tt
	}
	return &t, nil
}

// Create inserts a new task and notifies workers.
func (r *TaskRepository) Create(ctx context.Context, t *domain.Task) error {
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO tasks (task_id, tenant_id, type, status, priority, account_ids, target_ids, payload, progress, scheduled_at,
			                   is_loop, loop_interval_minutes, min_interval_seconds, max_interval_seconds, next_run_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
			t.TaskID, t.TenantID, t.Type, t.Status, t.Priority, t.AccountIDs, t.TargetIDs, t.Payload, t.Progress, t.ScheduledAt,
			t.IsLoop, t.LoopIntervalMinutes, t.MinIntervalSeconds, t.MaxIntervalSeconds, t.NextRunAt)
		return err
	})
	if err != nil {
		return err
	}
	// Notify workers that a new task is available
	if _, notifyErr := r.db.Exec(ctx, "NOTIFY new_task"); notifyErr != nil {
		// Non-fatal: workers will still poll
		log.Printf("[task] failed to notify new task: %v", notifyErr)
	}
	return nil
}

// ClaimTasks atomically claims pending/queued tasks for a worker using row-level locking.
// lockTimeoutMin 与 running 锁过期一致：超过该分钟数未更新的锁视为 worker 崩溃可回收。
func (r *TaskRepository) ClaimTasks(ctx context.Context, workerID string, limit int, lockTimeoutMin int) ([]*domain.Task, error) {
	return r.claimTasksWithStatus(ctx, workerID, limit, "queued", lockTimeoutMin)
}

// ClaimTimeoutTasks claims tasks whose lock has expired.
func (r *TaskRepository) ClaimTimeoutTasks(ctx context.Context, workerID string, limit int, lockTimeoutMin int) ([]*domain.Task, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE tasks AS claimed SET status = 'running', locked_at = NOW(), worker_id = $1, started_at = COALESCE(claimed.started_at, NOW())
		WHERE claimed.task_id IN (
			SELECT t.task_id FROM tasks t
			JOIN tenants tenant ON tenant.tenant_id = t.tenant_id
			WHERE t.status = 'running'
			  AND (t.locked_at IS NULL OR t.locked_at < NOW() - ($3 * INTERVAL '1 minute'))
			  AND tenant.status = 'active'
			  AND (tenant.expires_at IS NULL OR tenant.expires_at > NOW())
			ORDER BY t.priority DESC, t.created_at ASC
			FOR UPDATE OF t SKIP LOCKED
			LIMIT $2
		)
		RETURNING task_id, tenant_id, type, status, priority, account_ids, target_ids, payload, progress,
		          locked_at, worker_id, retry_count, scheduled_at, started_at, completed_at, created_at,
					       is_loop, loop_interval_minutes, min_interval_seconds, max_interval_seconds, next_run_at`,
		workerID, limit, lockTimeoutMin)
	if err != nil {
		return nil, fmt.Errorf("claim timeout tasks: %w", err)
	}
	defer rows.Close()
	tasks, err := r.scanTasks(rows)
	if err != nil {
		return nil, err
	}
	sortClaimedTasks(tasks)
	return tasks, nil
}

// CountRunningGlobal 全局 running 任务总数(内存自适应的全局并发上限检查用)
func (r *TaskRepository) CountRunningGlobal(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM tasks t
		JOIN tenants tenant ON tenant.tenant_id = t.tenant_id
		WHERE t.status = 'running' AND tenant.status = 'active'
		  AND (tenant.expires_at IS NULL OR tenant.expires_at > NOW())`).Scan(&n)
	return n, err
}

func (r *TaskRepository) claimTasksWithStatus(ctx context.Context, workerID string, limit int, status domain.TaskStatus, lockTimeoutMin int) ([]*domain.Task, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE tasks AS claimed SET status = 'running', locked_at = NOW(), worker_id = $1, started_at = COALESCE(claimed.started_at, NOW())
		WHERE claimed.task_id IN (
			SELECT t.task_id FROM tasks t
			JOIN tenants tenant ON tenant.tenant_id = t.tenant_id
			WHERE t.status = $2
			  AND (t.locked_at IS NULL OR t.locked_at < NOW() - ($4 * INTERVAL '1 minute'))
			  AND (t.scheduled_at IS NULL OR t.scheduled_at <= NOW())
			  AND tenant.status = 'active'
			  AND (tenant.expires_at IS NULL OR tenant.expires_at > NOW())
			ORDER BY t.priority DESC, t.created_at ASC
			FOR UPDATE OF t SKIP LOCKED
			LIMIT $3
		)
		RETURNING task_id, tenant_id, type, status, priority, account_ids, target_ids, payload, progress,
		          locked_at, worker_id, retry_count, scheduled_at, started_at, completed_at, created_at,
					       is_loop, loop_interval_minutes, min_interval_seconds, max_interval_seconds, next_run_at`,
		workerID, status, limit, lockTimeoutMin)
	if err != nil {
		return nil, fmt.Errorf("claim tasks: %w", err)
	}
	defer rows.Close()
	tasks, err := r.scanTasks(rows)
	if err != nil {
		return nil, err
	}
	sortClaimedTasks(tasks)
	return tasks, nil
}

// sortClaimedTasks makes claim results deterministic because PostgreSQL does not
// guarantee the order of rows returned by UPDATE ... RETURNING.
func sortClaimedTasks(tasks []*domain.Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].Priority != tasks[j].Priority {
			return tasks[i].Priority > tasks[j].Priority
		}
		if !tasks[i].CreatedAt.Equal(tasks[j].CreatedAt) {
			return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
		}
		return tasks[i].TaskID.String() < tasks[j].TaskID.String()
	})
}

// ActivatePendingTasks converts pending tasks whose scheduled_at has passed to queued.
func (r *TaskRepository) ActivatePendingTasks(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `
		UPDATE tasks AS t SET status = 'queued', updated_at = NOW()
		WHERE t.status = 'pending' AND t.scheduled_at IS NOT NULL AND t.scheduled_at <= NOW()
		  AND EXISTS (
			SELECT 1 FROM tenants tenant
			WHERE tenant.tenant_id = t.tenant_id AND tenant.status = 'active'
			  AND (tenant.expires_at IS NULL OR tenant.expires_at > NOW())
		  )`)
	return err
}

// GetNextPendingTaskTime returns the scheduled_at of the earliest pending task.
// Returns nil if no pending tasks exist.
func (r *TaskRepository) GetNextPendingTaskTime(ctx context.Context) (*time.Time, error) {
	var scheduledAt pgtype.Timestamptz
	err := r.db.QueryRow(ctx, `
		SELECT t.scheduled_at FROM tasks t
		JOIN tenants tenant ON tenant.tenant_id = t.tenant_id
		WHERE t.status = 'pending' AND t.scheduled_at IS NOT NULL AND t.scheduled_at > NOW()
		  AND tenant.status = 'active'
		  AND (tenant.expires_at IS NULL OR tenant.expires_at > NOW())
		ORDER BY t.scheduled_at ASC LIMIT 1`).Scan(&scheduledAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get next pending task: %w", err)
	}
	if scheduledAt.Valid {
		t := scheduledAt.Time
		return &t, nil
	}
	return nil, nil
}

// CountOwnedAccounts 校验这些账号都属于该租户(M-02: 防任务Update写入跨租户账号)。
func (r *TaskRepository) CountOwnedAccounts(ctx context.Context, tenantID uuid.UUID, accountIDs []uuid.UUID) (int, error) {
	if len(accountIDs) == 0 {
		return 0, nil
	}
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM accounts WHERE tenant_id=$1 AND account_id = ANY($2)`, tenantID, accountIDs,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count accounts owned by tenant: %w", err)
	}
	return n, nil
}

// UpdateProgress atomically increments task progress counters.
// Uses row-level locking to prevent concurrent updates from overwriting each other.
func (r *TaskRepository) UpdateProgress(ctx context.Context, tenantID, taskID uuid.UUID, progress domain.TaskProgress) error {
	_, err := r.db.Exec(ctx, `
		UPDATE tasks
		SET progress = jsonb_set(
			jsonb_set(
				jsonb_set(progress, '{done}', to_jsonb(COALESCE((progress->>'done')::int, 0) + $3)),
				'{failed}', to_jsonb(COALESCE((progress->>'failed')::int, 0) + $4)
			),
			'{current_target}', to_jsonb($5::text)
		)
		WHERE tenant_id = $1 AND task_id = $2`,
		tenantID, taskID, progress.Done, progress.Failed, progress.CurrentTarget)
	return err
}

// UpdateProgressExact sets progress to exact values (use only for final status).
func (r *TaskRepository) UpdateProgressExact(ctx context.Context, tenantID, taskID uuid.UUID, progress domain.TaskProgress) error {
	_, err := r.db.Exec(ctx, `
		UPDATE tasks SET progress = $3 WHERE tenant_id = $1 AND task_id = $2`,
		tenantID, taskID, progress)
	return err
}

// RefreshTaskLock extends the lease only while this worker still owns a running task.
// A stale worker must not refresh a lease that has already been reclaimed.
func (r *TaskRepository) RefreshTaskLock(ctx context.Context, tenantID, taskID uuid.UUID, workerID string) (bool, error) {
	res, err := r.db.Exec(ctx, `
		UPDATE tasks SET locked_at = NOW()
		WHERE tenant_id = $1 AND task_id = $2
		  AND status = 'running' AND worker_id = $3
	`, tenantID, taskID, workerID)
	if err != nil {
		return false, fmt.Errorf("refresh task lock: %w", err)
	}
	return res.RowsAffected() == 1, nil
}

// UpdateStatusOwned updates a running task only while the caller still owns its lease.
// This prevents a stale worker from overwriting cancellation or a newer worker's state.
func (r *TaskRepository) UpdateStatusOwned(ctx context.Context, tenantID, taskID uuid.UUID, workerID string, status domain.TaskStatus) error {
	if workerID == "" {
		return fmt.Errorf("worker ID is required")
	}
	var completedAt interface{}
	if status == domain.TaskStatusCompleted || status == domain.TaskStatusFailed || status == domain.TaskStatusCancelled {
		completedAt = time.Now()
	}
	res, err := r.db.Exec(ctx, `
		UPDATE tasks SET status = $4, completed_at = $5,
			locked_at = CASE WHEN $4 = 'running' THEN locked_at ELSE NULL END,
			worker_id = CASE WHEN $4 = 'running' THEN worker_id ELSE NULL END
		WHERE tenant_id = $1 AND task_id = $2
		  AND status = 'running' AND worker_id = $3
		  AND $4 IN ('running', 'paused', 'completed', 'failed', 'cancelled', 'queued')
	`, tenantID, taskID, workerID, status, completedAt)
	if err != nil {
		return fmt.Errorf("update owned task status: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("task lease lost or task is no longer running")
	}
	return nil
}

// PauseIfActive atomically pauses a task only while it is still queued, running, or scheduled.
// Clearing the worker lease prevents an in-flight worker from writing a later terminal state.
func (r *TaskRepository) PauseIfActive(ctx context.Context, tenantID, taskID uuid.UUID) (bool, error) {
	res, err := r.db.Exec(ctx, `
		UPDATE tasks SET status = 'paused', completed_at = NULL, locked_at = NULL, worker_id = NULL
		WHERE tenant_id = $1 AND task_id = $2
		  AND status IN ('pending', 'queued', 'running', 'scheduled')
	`, tenantID, taskID)
	if err != nil {
		return false, fmt.Errorf("pause task: %w", err)
	}
	return res.RowsAffected() == 1, nil
}

// UpdateStatus updates task status.
func (r *TaskRepository) UpdateStatus(ctx context.Context, tenantID, taskID uuid.UUID, status domain.TaskStatus) error {
	var completedAt interface{}
	if status == domain.TaskStatusCompleted || status == domain.TaskStatusFailed || status == domain.TaskStatusCancelled {
		completedAt = time.Now()
	}
	// F-27: 迁移守卫——WHERE 内嵌合法迁移矩阵，非法迁移命中 0 行即报错。
	// 终态（failed/cancelled）与恢复（queued）为全来源白名单（引擎 crash 恢复、
	// 管理操作、任务失败路径都需要）；其余按状态机矩阵校验。
	res, err := r.db.Exec(ctx, `
		UPDATE tasks SET status = $3, completed_at = $4,
			locked_at = CASE WHEN $3 = 'running' THEN locked_at ELSE NULL END,
			worker_id = CASE WHEN $3 = 'running' THEN worker_id ELSE NULL END
		WHERE tenant_id = $1 AND task_id = $2 AND (
			$3 IN ('failed', 'cancelled', 'queued')
			OR (status = 'scheduled' AND $3 IN ('queued', 'cancelled'))
			OR (status = 'queued'   AND $3 IN ('running', 'cancelled'))
			OR (status = 'running'  AND $3 IN ('paused', 'completed', 'failed', 'cancelled'))
			OR (status = 'paused'   AND $3 IN ('running', 'cancelled'))
		)`,
		tenantID, taskID, status, completedAt)
	if err != nil {
		return err
	}
	if n := res.RowsAffected(); n == 0 {
		return fmt.Errorf("invalid task transition to %q (or task not found)", status)
	}
	return nil
}

// CancelIfActive atomically cancels only non-terminal tasks. A late cancel request
// must never overwrite a task that has already completed or failed.
func (r *TaskRepository) CancelIfActive(ctx context.Context, tenantID, taskID uuid.UUID) (bool, error) {
	res, err := r.db.Exec(ctx, `
		UPDATE tasks SET status = 'cancelled', completed_at = NOW(), locked_at = NULL, worker_id = NULL
		WHERE tenant_id = $1 AND task_id = $2
		  AND status NOT IN ('completed', 'failed', 'cancelled')
	`, tenantID, taskID)
	if err != nil {
		return false, fmt.Errorf("cancel task: %w", err)
	}
	return res.RowsAffected() == 1, nil
}

// UpdateStatusUnlessCancelled is reserved for asynchronous approval resolution.
// Approval polling may finish after a user cancellation; cancellation wins that race.
func (r *TaskRepository) UpdateStatusUnlessCancelled(ctx context.Context, tenantID, taskID uuid.UUID, status domain.TaskStatus) (bool, error) {
	if status != domain.TaskStatusCompleted && status != domain.TaskStatusFailed {
		return false, fmt.Errorf("invalid approval task status %q", status)
	}
	var completedAt interface{} = time.Now()
	res, err := r.db.Exec(ctx, `
		UPDATE tasks SET status = $3, completed_at = $4, locked_at = NULL, worker_id = NULL
		WHERE tenant_id = $1 AND task_id = $2 AND status <> 'cancelled'
	`, tenantID, taskID, status, completedAt)
	if err != nil {
		return false, fmt.Errorf("update task approval status: %w", err)
	}
	return res.RowsAffected() == 1, nil
}

// FinalizeTaskAfterApprovals updates the task only after every approval record is resolved.
// Any timed-out approval makes the overall join task fail; cancellation always wins.
func (r *TaskRepository) FinalizeTaskAfterApprovals(ctx context.Context, tenantID, taskID uuid.UUID) (bool, error) {
	res, err := r.db.Exec(ctx, `
		UPDATE tasks AS t
		SET status = CASE WHEN t.status = 'failed' OR EXISTS (
			SELECT 1 FROM pending_approvals pa
			WHERE pa.tenant_id = t.tenant_id AND pa.task_id = t.task_id AND pa.status = 'timeout'
		) THEN 'failed' ELSE 'completed' END,
		completed_at = NOW(), locked_at = NULL, worker_id = NULL
		WHERE t.tenant_id = $1 AND t.task_id = $2
		  AND t.status IN ('completed', 'failed')
		  AND NOT EXISTS (
			SELECT 1 FROM pending_approvals pa
			WHERE pa.tenant_id = t.tenant_id AND pa.task_id = t.task_id AND pa.status = 'pending'
		  )
	`, tenantID, taskID)
	if err != nil {
		return false, fmt.Errorf("finalize task approvals: %w", err)
	}
	return res.RowsAffected() == 1, nil
}

// ResetToQueued resets a failed task to queued for retry.
func (r *TaskRepository) ResetToQueued(ctx context.Context, tenantID, taskID uuid.UUID) error {
	res, err := r.db.Exec(ctx, `
		UPDATE tasks SET status = 'queued', locked_at = NULL, worker_id = NULL,
		                 retry_count = retry_count + CASE WHEN status = 'failed' THEN 1 ELSE 0 END,
		                 completed_at = NULL
		WHERE tenant_id = $1 AND task_id = $2 AND status IN ('failed', 'paused')`, tenantID, taskID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("task is no longer failed or paused")
	}
	return nil
}

// ResumeLoop restores a paused recurring task without discarding its next scheduled run.
// If the scheduled time has passed (or the task was paused mid-run), it is queued immediately.
func (r *TaskRepository) ResumeLoop(ctx context.Context, tenantID, taskID uuid.UUID) error {
	res, err := r.db.Exec(ctx, `
		UPDATE tasks
		SET status = CASE WHEN next_run_at IS NOT NULL AND next_run_at > NOW() THEN 'pending' ELSE 'queued' END,
		    scheduled_at = CASE WHEN next_run_at IS NOT NULL AND next_run_at > NOW() THEN next_run_at ELSE NULL END,
		    locked_at = NULL, worker_id = NULL, completed_at = NULL
		WHERE tenant_id = $1 AND task_id = $2 AND status = 'paused' AND is_loop = TRUE
	`, tenantID, taskID)
	if err != nil {
		return fmt.Errorf("resume loop task: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("loop task is no longer paused")
	}
	if _, err := r.db.Exec(ctx, "NOTIFY new_task"); err != nil {
		log.Printf("[task] failed to notify new task after loop resume: %v", err)
	}
	return nil
}

// Delete removes a task.
func (r *TaskRepository) Delete(ctx context.Context, tenantID, taskID uuid.UUID) error {
	res, err := r.db.Exec(ctx, `
		DELETE FROM tasks
		WHERE tenant_id = $1 AND task_id = $2
		  AND status IN ('pending', 'scheduled', 'paused', 'completed', 'failed', 'cancelled')
	`, tenantID, taskID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("task is active or no longer exists")
	}
	return nil
}

// CountStats returns task statistics for a tenant.
func (r *TaskRepository) CountStats(ctx context.Context, tenantID uuid.UUID) (total, running, failed int, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE status = 'running'),
		       COUNT(*) FILTER (WHERE status = 'failed')
		FROM tasks WHERE tenant_id = $1`, tenantID).Scan(&total, &running, &failed)
	return
}

// Count returns total tasks for a tenant (optionally filtered by status and account_id).
func (r *TaskRepository) Count(ctx context.Context, tenantID uuid.UUID, status *domain.TaskStatus, accountID *uuid.UUID, taskType *domain.TaskType, since *time.Time) (int, error) {
	query := `SELECT COUNT(*) FROM tasks WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	idx := 2
	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, *status)
		idx++
	}
	if accountID != nil {
		// Use parameterised JSONB array to prevent injection
		args = append(args, []string{accountID.String()})
		query += fmt.Sprintf(" AND account_ids @> $%d::jsonb", idx)
		idx++
	}
	if taskType != nil && *taskType != "" {
		query += fmt.Sprintf(" AND type = $%d", idx)
		args = append(args, *taskType)
		idx++
	}
	if since != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", idx)
		args = append(args, *since)
		idx++
	}
	var count int
	if err := r.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count tasks: %w", err)
	}
	return count, nil
}

func (r *TaskRepository) scanTasks(rows pgx.Rows) ([]*domain.Task, error) {
	var tasks []*domain.Task
	for rows.Next() {
		var t domain.Task
		var lockedAt pgtype.Timestamptz
		var workerID pgtype.Text
		var scheduledAt pgtype.Timestamptz
		var startedAt pgtype.Timestamptz
		var completedAt pgtype.Timestamptz
		var nextRunAt pgtype.Timestamptz

		err := rows.Scan(&t.TaskID, &t.TenantID, &t.Type, &t.Status, &t.Priority, &t.AccountIDs, &t.TargetIDs, &t.Payload, &t.Progress,
			&lockedAt, &workerID, &t.RetryCount, &scheduledAt, &startedAt, &completedAt, &t.CreatedAt,
			&t.IsLoop, &t.LoopIntervalMinutes, &t.MinIntervalSeconds, &t.MaxIntervalSeconds, &nextRunAt)
		if err != nil {
			return nil, err
		}
		if nextRunAt.Valid {
			tt := nextRunAt.Time
			t.NextRunAt = &tt
		}
		if lockedAt.Valid {
			tt := lockedAt.Time
			t.LockedAt = &tt
		}
		if workerID.Valid {
			t.WorkerID = &workerID.String
		}
		if scheduledAt.Valid {
			tt := scheduledAt.Time
			t.ScheduledAt = &tt
		}
		if startedAt.Valid {
			tt := startedAt.Time
			t.StartedAt = &tt
		}
		if completedAt.Valid {
			tt := completedAt.Time
			t.CompletedAt = &tt
		}
		tasks = append(tasks, &t)
	}
	return tasks, rows.Err()
}

// CountByStatus counts tasks with a specific status for a tenant.
// P1-46 审计修复: 原来 tenantID 为 Nil 时报错返回 0——container 注入的 getQueueDepth
// 正是传 uuid.Nil(想要全平台队列深度), 结果恒 0, 自适应升档永不触发。
// 现在 Nil = 全租户统计。
func (r *TaskRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status string) (int, error) {
	var count int
	var err error
	if tenantID == uuid.Nil {
		err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE status = $1`, status).Scan(&count)
	} else {
		err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE tenant_id = $1 AND status = $2`, tenantID, status).Scan(&count)
	}
	if err != nil {
		return 0, fmt.Errorf("count tasks by status: %w", err)
	}
	return count, nil
}

// CountCompletedToday counts tasks completed today for a tenant.
func (r *TaskRepository) CountCompletedToday(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM tasks
		WHERE tenant_id = $1 AND status = 'completed'
		AND completed_at >= CURRENT_DATE`, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count completed tasks today: %w", err)
	}
	return count, nil
}

// CountCompletedYesterday counts tasks completed yesterday for a tenant.
func (r *TaskRepository) CountCompletedYesterday(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM tasks
		WHERE tenant_id = $1 AND status = 'completed'
		AND completed_at >= CURRENT_DATE - INTERVAL '1 day'
		AND completed_at < CURRENT_DATE`, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count completed tasks yesterday: %w", err)
	}
	return count, nil
}

// Update updates task fields (only allowed for pending/paused tasks).
func (r *TaskRepository) Update(ctx context.Context, tenantID, taskID uuid.UUID, accountIDs []uuid.UUID, targetIDs []string, payload map[string]any) error {
	cmd, err := r.db.Exec(ctx, `
		UPDATE tasks
		SET account_ids = $1, target_ids = $2, payload = $3, updated_at = NOW()
		WHERE tenant_id = $4 AND task_id = $5 AND status IN ('pending', 'paused')`,
		accountIDs, targetIDs, payload, tenantID, taskID)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("task not found or cannot be updated (only pending/paused tasks can be updated)")
	}
	return nil
}

// CountCompletedLast7Days returns completed task counts for the last 7 days (oldest first).
func (r *TaskRepository) CountCompletedLast7Days(ctx context.Context, tenantID uuid.UUID) ([]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT 
			COALESCE(COUNT(*) FILTER (WHERE completed_at >= CURRENT_DATE - INTERVAL '6 days' AND completed_at < CURRENT_DATE - INTERVAL '5 days'), 0) as d6,
			COALESCE(COUNT(*) FILTER (WHERE completed_at >= CURRENT_DATE - INTERVAL '5 days' AND completed_at < CURRENT_DATE - INTERVAL '4 days'), 0) as d5,
			COALESCE(COUNT(*) FILTER (WHERE completed_at >= CURRENT_DATE - INTERVAL '4 days' AND completed_at < CURRENT_DATE - INTERVAL '3 days'), 0) as d4,
			COALESCE(COUNT(*) FILTER (WHERE completed_at >= CURRENT_DATE - INTERVAL '3 days' AND completed_at < CURRENT_DATE - INTERVAL '2 days'), 0) as d3,
			COALESCE(COUNT(*) FILTER (WHERE completed_at >= CURRENT_DATE - INTERVAL '2 days' AND completed_at < CURRENT_DATE - INTERVAL '1 day'), 0) as d2,
			COALESCE(COUNT(*) FILTER (WHERE completed_at >= CURRENT_DATE - INTERVAL '1 day' AND completed_at < CURRENT_DATE), 0) as d1,
			COALESCE(COUNT(*) FILTER (WHERE completed_at >= CURRENT_DATE), 0) as d0
		FROM tasks
		WHERE tenant_id = $1 AND status = 'completed'
		  AND completed_at >= CURRENT_DATE - INTERVAL '6 days'`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("count completed last 7 days: %w", err)
	}
	defer rows.Close()

	var result []int
	if rows.Next() {
		var d0, d1, d2, d3, d4, d5, d6 int
		if err := rows.Scan(&d6, &d5, &d4, &d3, &d2, &d1, &d0); err != nil {
			return nil, fmt.Errorf("scan task trend: %w", err)
		}
		result = []int{d6, d5, d4, d3, d2, d1, d0}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task trend: %w", err)
	}
	return result, nil
}

// SetLoopConfig marks a task as a scheduled broadcast loop task.
func (r *TaskRepository) SetLoopConfig(ctx context.Context, tenantID, taskID uuid.UUID, isLoop bool, loopInterval, minInt, maxInt int, taskType domain.TaskType) error {
	_, err := r.db.Exec(ctx, `
		UPDATE tasks SET is_loop = $3, loop_interval_minutes = $4, min_interval_seconds = $5,
		                 max_interval_seconds = $6, type = $7, next_run_at = NOW()
		WHERE tenant_id = $1 AND task_id = $2`,
		tenantID, taskID, isLoop, loopInterval, minInt, maxInt, taskType)
	return err
}

// StopLoop 停止循环任务: 关闭循环标记并置为暂停, 调度器不再激活它。
func (r *TaskRepository) StopLoop(ctx context.Context, tenantID, taskID uuid.UUID) error {
	res, err := r.db.Exec(ctx, `
		UPDATE tasks SET is_loop = FALSE, status = 'paused'
		WHERE tenant_id = $1 AND task_id = $2 AND is_loop = TRUE
		  AND status IN ('pending', 'queued', 'scheduled', 'paused', 'completed')
	`, tenantID, taskID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("loop task is no longer stoppable")
	}
	return nil
}

// RescheduleLoopTask resets a finished loop task back to scheduled with a new run time.
func (r *TaskRepository) RescheduleLoopTask(ctx context.Context, tenantID, taskID uuid.UUID, nextRunAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		-- 轮数在完成本轮时递增；达到最大轮数或结束时间后保留 completed 并关闭循环。
		WITH limits AS (
			SELECT task_id,
			       COALESCE(NULLIF(payload->>'round_count', '')::int, 0) + 1 AS next_round,
			       COALESCE(NULLIF(payload->>'max_rounds', '')::int, 0) AS max_rounds,
			       NULLIF(payload->>'end_at', '')::timestamptz AS end_at
			FROM tasks WHERE tenant_id = $1 AND task_id = $2
		)
		UPDATE tasks t
		SET status = CASE WHEN
		        (l.max_rounds > 0 AND l.next_round >= l.max_rounds)
		        OR (l.end_at IS NOT NULL AND l.end_at <= GREATEST(NOW(), $3))
		    THEN 'completed' ELSE 'pending' END,
		    is_loop = CASE WHEN
		        (l.max_rounds > 0 AND l.next_round >= l.max_rounds)
		        OR (l.end_at IS NOT NULL AND l.end_at <= GREATEST(NOW(), $3))
		    THEN FALSE ELSE TRUE END,
		    next_run_at = CASE WHEN
		        (l.max_rounds > 0 AND l.next_round >= l.max_rounds)
		        OR (l.end_at IS NOT NULL AND l.end_at <= GREATEST(NOW(), $3))
		    THEN NULL ELSE $3 END,
		    scheduled_at = CASE WHEN
		        (l.max_rounds > 0 AND l.next_round >= l.max_rounds)
		        OR (l.end_at IS NOT NULL AND l.end_at <= GREATEST(NOW(), $3))
		    THEN NULL ELSE $3 END,
		    completed_at = CASE WHEN
		        (l.max_rounds > 0 AND l.next_round >= l.max_rounds)
		        OR (l.end_at IS NOT NULL AND l.end_at <= GREATEST(NOW(), $3))
		    THEN NOW() ELSE NULL END,
		    locked_at = NULL, worker_id = NULL,
		    progress = CASE WHEN
		        (l.max_rounds > 0 AND l.next_round >= l.max_rounds)
		        OR (l.end_at IS NOT NULL AND l.end_at <= GREATEST(NOW(), $3))
		    THEN t.progress ELSE '{"total":0,"done":0,"failed":0}'::jsonb END,
		    payload = jsonb_set(COALESCE(t.payload, '{}'::jsonb), '{round_count}', to_jsonb(l.next_round), TRUE)
		FROM limits l
		WHERE t.tenant_id = $1 AND t.task_id = l.task_id
		  AND t.status = 'completed' AND t.is_loop = TRUE`, tenantID, taskID, nextRunAt)
	return err
}

// AccountStats 返回单个账号维度的仪表盘统计(账号切换后只看该账号的数据)。
// 查询都走 account_ids @> 账号 的索引条件。
func (r *TaskRepository) AccountStats(ctx context.Context, tenantID, accountID uuid.UUID) (map[string]any, error) {
	// 近 7 天每天完成的任务数
	rows, err := r.db.Query(ctx, `
		SELECT to_char(d, 'MM-DD') AS day, COUNT(t.task_id) AS cnt
		FROM generate_series(CURRENT_DATE - INTERVAL '6 days', CURRENT_DATE, INTERVAL '1 day') d
		LEFT JOIN tasks t ON t.tenant_id = $1 AND t.account_ids @> $2::jsonb
			AND t.status = 'completed' AND date(t.completed_at) = d
		GROUP BY d ORDER BY d`, tenantID, fmt.Sprintf(`["%s"]`, accountID.String()))
	if err != nil {
		return nil, fmt.Errorf("account task trend: %w", err)
	}
	defer rows.Close()
	trend := make([]int, 0, 7)
	for rows.Next() {
		var day string
		var cnt int
		if err := rows.Scan(&day, &cnt); err != nil {
			return nil, err
		}
		trend = append(trend, cnt)
	}
	if len(trend) == 0 {
		trend = []int{0, 0, 0, 0, 0, 0, 0}
	}

	out := map[string]any{"task_trend": trend}
	// 注意: pgx 严格校验参数个数, 每条 SQL 按实际占位符数量传参
	counts := []struct {
		key  string
		sql  string
		args []interface{}
	}{
		{"completed_today", `SELECT COUNT(*) FROM tasks WHERE tenant_id=$1 AND account_ids @> $2::jsonb AND status='completed' AND completed_at::date = CURRENT_DATE`,
			[]interface{}{tenantID, fmt.Sprintf(`["%s"]`, accountID.String())}},
		{"completed_yesterday", `SELECT COUNT(*) FROM tasks WHERE tenant_id=$1 AND account_ids @> $2::jsonb AND status='completed' AND completed_at::date = CURRENT_DATE - 1`,
			[]interface{}{tenantID, fmt.Sprintf(`["%s"]`, accountID.String())}},
		// V32 审计修复: 原来不查 is_joined——已退出的群也计入"总群数", 仪表盘数字虚高
		{"total_groups", `SELECT COUNT(*) FROM groups WHERE tenant_id=$1 AND group_id IN (SELECT group_id FROM account_groups WHERE account_id=$2 AND is_joined=true)`,
			[]interface{}{tenantID, accountID}},
		{"risks_24h", `SELECT COUNT(*) FROM risk_events WHERE tenant_id=$1 AND account_id=$2 AND created_at >= NOW() - INTERVAL '24 hours'`,
			[]interface{}{tenantID, accountID}},
	}
	for _, cq := range counts {
		var n int
		if err := r.db.QueryRow(ctx, cq.sql, cq.args...).Scan(&n); err != nil {
			return nil, fmt.Errorf("account stats %s: %w", cq.key, err)
		}
		out[cq.key] = n
	}
	return out, nil
}
