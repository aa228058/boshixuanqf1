package repository

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"tg-sender/internal/domain"
)

// connectTestDB 从环境变量连接测试数据库。未设置 DB_HOST 时 skip，
// 使本地无库也能跑单测；CI 里有 Postgres 服务会真实执行。
func connectTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	host := os.Getenv("DB_HOST")
	if host == "" {
		t.Skip("DB_HOST not set; skipping integration test")
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), host, port, os.Getenv("DB_NAME"))
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// ensureTasksTable 建立任务仓库集成测试所需的最小表结构。
// 测试 fixture 使用与正式迁移相同的全局 sequence 默认值。
func ensureTasksTable(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `CREATE SEQUENCE IF NOT EXISTS global_task_number_seq AS BIGINT`); err != nil {
		t.Fatalf("create global task number sequence: %v", err)
	}
	_, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS tenants (
			tenant_id UUID PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			username TEXT NOT NULL DEFAULT '',
			password_hash TEXT NOT NULL DEFAULT '',
			api_key_hash TEXT NOT NULL DEFAULT '',
			plan_id TEXT NOT NULL DEFAULT 'free',
			status TEXT NOT NULL DEFAULT 'active',
			expires_at TIMESTAMPTZ
		)`)
	if err != nil {
		t.Fatalf("create tenants table: %v", err)
	}
	_, err = pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS tasks (
			task_id UUID PRIMARY KEY,
			task_number BIGINT NOT NULL DEFAULT nextval('global_task_number_seq'),
			tenant_id UUID NOT NULL,
			type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'queued',
			priority INT DEFAULT 5,
			account_ids JSONB DEFAULT '[]',
			target_ids JSONB DEFAULT '[]',
			payload JSONB DEFAULT '{}',
			progress JSONB DEFAULT '{}',
			locked_at TIMESTAMPTZ,
			worker_id TEXT,
			retry_count INT DEFAULT 0,
			scheduled_at TIMESTAMPTZ,
			started_at TIMESTAMPTZ,
			completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			is_loop BOOLEAN DEFAULT FALSE,
			loop_interval_minutes INT DEFAULT 60,
			min_interval_seconds INT DEFAULT 5,
			max_interval_seconds INT DEFAULT 15,
			next_run_at TIMESTAMPTZ
		)`)
	if err != nil {
		t.Fatalf("create tasks table: %v", err)
	}
	// Keep reused minimal test databases compatible with TaskRepository.scanTasks,
	// which reads task_number in the same position as the production schema.
	if _, err := pool.Exec(context.Background(), `ALTER TABLE tasks ADD COLUMN IF NOT EXISTS task_number BIGINT NOT NULL DEFAULT nextval('global_task_number_seq')`); err != nil {
		t.Fatalf("ensure tasks.task_number: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `ALTER TABLE tasks ALTER COLUMN task_number SET DEFAULT nextval('global_task_number_seq')`); err != nil {
		t.Fatalf("set tasks.task_number default: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_task_number_test ON tasks(task_number)`); err != nil {
		t.Fatalf("create tasks.task_number test index: %v", err)
	}
	_, err = pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS group_library_import_jobs (
			job_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			file_name VARCHAR(255) NOT NULL DEFAULT '',
			payload TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			total_count INTEGER NOT NULL DEFAULT 0,
			processed_count INTEGER NOT NULL DEFAULT 0,
			imported_count INTEGER NOT NULL DEFAULT 0,
			failed_count INTEGER NOT NULL DEFAULT 0,
			skipped_count INTEGER NOT NULL DEFAULT 0,
			last_error TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			started_at TIMESTAMPTZ,
			finished_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			account_id UUID,
			task_number BIGINT NOT NULL DEFAULT nextval('global_task_number_seq')
		)
	`)
	if err != nil {
		t.Fatalf("create group import jobs fixture: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `ALTER TABLE group_library_import_jobs ADD COLUMN IF NOT EXISTS task_number BIGINT NOT NULL DEFAULT nextval('global_task_number_seq')`); err != nil {
		t.Fatalf("ensure import job task_number: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `ALTER TABLE group_library_import_jobs ALTER COLUMN task_number SET DEFAULT nextval('global_task_number_seq')`); err != nil {
		t.Fatalf("set import job task_number default: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `CREATE UNIQUE INDEX IF NOT EXISTS idx_group_library_import_jobs_task_number_test ON group_library_import_jobs(task_number)`); err != nil {
		t.Fatalf("create import task_number test index: %v", err)
	}
	// The task-log cursor integration test uses a deliberately minimal fixture
	// instead of running the full migration chain. Keep the columns consumed by
	// TaskLogRepository available in that isolated test database.
	_, err = pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS task_logs (
			log_id UUID PRIMARY KEY,
			task_id UUID NOT NULL,
			tenant_id UUID NOT NULL,
			account_id UUID,
			target_id TEXT,
			target_name TEXT NOT NULL DEFAULT '',
			action TEXT NOT NULL,
			status TEXT NOT NULL,
			tg_response TEXT,
			error_code TEXT,
			message TEXT NOT NULL DEFAULT '',
			tg_message_id BIGINT,
			executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`)
	if err != nil {
		t.Fatalf("create task_logs table: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `DELETE FROM task_logs`); err != nil {
		t.Fatalf("truncate task_logs: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `DELETE FROM tasks`); err != nil {
		t.Fatalf("truncate tasks: %v", err)
	}
}

func seedTask(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, taskType string, priority int) uuid.UUID {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO tenants (tenant_id, name, username, password_hash, api_key_hash, plan_id, status, expires_at)
		VALUES ($1, $2, $2, 'test-password-hash', $3, 'free', 'active', NULL)
		ON CONFLICT (tenant_id) DO UPDATE SET status = 'active', expires_at = NULL
	`, tenantID, tenantID.String(), tenantID.String())
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	id := uuid.New()
	_, err = pool.Exec(context.Background(),
		`INSERT INTO tasks (task_id, tenant_id, type, status, priority) VALUES ($1,$2,$3,'queued',$4)`,
		id, tenantID, taskType, priority)
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}
	return id
}

func TestGlobalTaskNumbersSharedAcrossTasksAndImportJobs(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	ctx := context.Background()
	tenantID := uuid.New()
	_ = seedTask(t, pool, tenantID, string(domain.TaskTypeBroadcast), 5)

	var firstTaskNumber int64
	if err := pool.QueryRow(ctx, `SELECT task_number FROM tasks WHERE tenant_id=$1 ORDER BY task_number DESC LIMIT 1`, tenantID).Scan(&firstTaskNumber); err != nil {
		t.Fatalf("read first task number: %v", err)
	}
	var importJobID uuid.UUID
	var importNumber int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO group_library_import_jobs (tenant_id, payload) VALUES ($1, 'https://t.me/example')
		RETURNING job_id, task_number
	`, tenantID).Scan(&importJobID, &importNumber); err != nil {
		t.Fatalf("create import job: %v", err)
	}
	if importNumber <= firstTaskNumber {
		t.Fatalf("shared sequence did not advance across task types: task=%d import=%d", firstTaskNumber, importNumber)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM tasks WHERE tenant_id=$1`, tenantID); err != nil {
		t.Fatalf("delete task fixture: %v", err)
	}
	var nextTaskNumber int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO tasks (task_id, tenant_id, type, status, priority)
		VALUES ($1, $2, 'broadcast', 'queued', 5) RETURNING task_number
	`, uuid.New(), tenantID).Scan(&nextTaskNumber); err != nil {
		t.Fatalf("create task after deletion: %v", err)
	}
	if nextTaskNumber <= importNumber {
		t.Fatalf("task number was reused after deleting a task: import=%d next_task=%d", importNumber, nextTaskNumber)
	}

	var duplicates int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) - COUNT(DISTINCT task_number)
		FROM (
			SELECT task_number FROM tasks WHERE tenant_id=$1
			UNION ALL
			SELECT task_number FROM group_library_import_jobs WHERE tenant_id=$1
		) numbered
	`, tenantID).Scan(&duplicates); err != nil {
		t.Fatalf("check duplicate global task numbers: %v", err)
	}
	if duplicates != 0 {
		t.Fatalf("found %d duplicate task numbers across task types", duplicates)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM group_library_import_jobs WHERE job_id=$1`, importJobID); err != nil {
		t.Fatalf("cleanup import job fixture: %v", err)
	}
}

func TestTaskLogKeysetPaginationDoesNotShiftWhenNewRowsArrive(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	tenantID := uuid.New()
	taskID := seedTask(t, pool, tenantID, string(domain.TaskTypeBroadcast), 5)
	logRepo := NewTaskLogRepository(pool)
	base := time.Now().UTC().Truncate(time.Microsecond)

	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for i, offset := range []time.Duration{0, time.Second, 2 * time.Second} {
		entry := &domain.TaskLog{
			LogID: ids[i], TaskID: taskID, TenantID: tenantID,
			TargetID: fmt.Sprintf("target-%d", i), Action: "send_message",
			Status: "success", ExecutedAt: base.Add(offset),
		}
		if err := logRepo.Create(context.Background(), entry); err != nil {
			t.Fatalf("seed task log %d: %v", i, err)
		}
	}

	firstPage, err := logRepo.ListByTask(context.Background(), tenantID, taskID, 2, 0)
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(firstPage) != 2 || firstPage[0].LogID != ids[2] || firstPage[1].LogID != ids[1] {
		t.Fatalf("first page order = %v, want newest two [%s %s]", logIDs(firstPage), ids[2], ids[1])
	}

	// A new live log arrives after page 1 was read. OFFSET pagination would
	// shift page 2; the keyset cursor must continue strictly before the old row.
	newest := &domain.TaskLog{
		LogID: uuid.New(), TaskID: taskID, TenantID: tenantID,
		TargetID: "late-target", Action: "send_message",
		Status: "success", ExecutedAt: base.Add(3 * time.Second),
	}
	if err := logRepo.Create(context.Background(), newest); err != nil {
		t.Fatalf("insert concurrent task log: %v", err)
	}

	nextPage, err := logRepo.ListBeforeByTask(context.Background(), tenantID, taskID, firstPage[1].LogID, 2)
	if err != nil {
		t.Fatalf("cursor page: %v", err)
	}
	if len(nextPage) != 1 || nextPage[0].LogID != ids[0] {
		t.Fatalf("cursor page = %v, want only oldest row %s", logIDs(nextPage), ids[0])
	}
}

func logIDs(logs []*domain.TaskLog) []uuid.UUID {
	ids := make([]uuid.UUID, len(logs))
	for i, entry := range logs {
		ids[i] = entry.LogID
	}
	return ids
}

func TestClaimTasks_ClaimsQueuedInPriorityOrder(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	repo := NewTaskRepository(pool)
	tenant := uuid.New()

	low := seedTask(t, pool, tenant, string(domain.TaskTypeBroadcast), 1)
	mid := seedTask(t, pool, tenant, string(domain.TaskTypeBroadcast), 5)
	high := seedTask(t, pool, tenant, string(domain.TaskTypeBroadcast), 9)

	claimed, err := repo.ClaimTasks(context.Background(), "worker-A", 10, 5)
	if err != nil {
		t.Fatalf("ClaimTasks: %v", err)
	}
	if len(claimed) != 3 {
		t.Fatalf("claimed %d tasks, want 3", len(claimed))
	}
	// 优先级降序：high → mid → low
	if claimed[0].TaskID != high || claimed[1].TaskID != mid || claimed[2].TaskID != low {
		t.Errorf("claim order wrong: got %v,%v,%v want %v,%v,%v",
			claimed[0].TaskID, claimed[1].TaskID, claimed[2].TaskID, high, mid, low)
	}
	// 认领后状态必须 running 且带 worker_id（WorkerID 是 *string）
	for _, task := range claimed {
		if task.Status != "running" {
			t.Errorf("task %s status = %q, want running", task.TaskID, task.Status)
		}
		if task.WorkerID == nil || *task.WorkerID != "worker-A" {
			t.Errorf("task %s worker = %v, want worker-A", task.TaskID, task.WorkerID)
		}
	}
	// 再认领应为空（无 queued 剩余）
	again, err := repo.ClaimTasks(context.Background(), "worker-B", 10, 5)
	if err != nil {
		t.Fatalf("second ClaimTasks: %v", err)
	}
	if len(again) != 0 {
		t.Errorf("second claim got %d tasks, want 0 (all already running)", len(again))
	}
}

func TestClaimTasks_ConcurrentClaimersGetDisjointSets(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	repo := NewTaskRepository(pool)
	tenant := uuid.New()

	const total = 20
	ids := make([]uuid.UUID, total)
	for i := 0; i < total; i++ {
		ids[i] = seedTask(t, pool, tenant, string(domain.TaskTypeBroadcast), 5)
	}

	// 两个并发 worker 各认领 12（> total/2，必然竞争同一批行）
	var wg sync.WaitGroup
	results := make([][]*domain.Task, 2)
	errs := make([]error, 2)
	for w := 0; w < 2; w++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// 轻微错峰让两个事务真正重叠在 SKIP LOCKED 上
			time.Sleep(time.Duration(idx) * 50 * time.Millisecond)
			tasks, err := repo.ClaimTasks(context.Background(), fmt.Sprintf("worker-%d", idx), 12, 5)
			results[idx] = tasks
			errs[idx] = err
		}(w)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("worker-%d claim: %v", i, err)
		}
	}

	got := len(results[0]) + len(results[1])
	if got != total {
		t.Errorf("total claimed = %d, want %d (double-claim or lost task!)", got, total)
	}

	// 两批必须不相交（SKIP LOCKED 核心保证）
	seen := map[uuid.UUID]int{}
	for _, batch := range results {
		for _, task := range batch {
			seen[task.TaskID]++
		}
	}
	for id, n := range seen {
		if n > 1 {
			t.Errorf("task %s claimed %d times by different workers", id, n)
		}
	}
}

func TestClaimTimeoutTasks_ReclaimsExpiredLock(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	repo := NewTaskRepository(pool)
	tenant := uuid.New()

	id := seedTask(t, pool, tenant, string(domain.TaskTypeBroadcast), 5)
	// 模拟崩溃的 worker：状态 running 但 locked_at 已过期 6 分钟
	_, err := pool.Exec(context.Background(),
		`UPDATE tasks SET status='running', worker_id='crashed-worker', locked_at = NOW() - INTERVAL '6 minutes' WHERE task_id=$1`, id)
	if err != nil {
		t.Fatalf("simulate crashed worker: %v", err)
	}

	claimed, err := repo.ClaimTimeoutTasks(context.Background(), "rescue-worker", 5, 5)
	if err != nil {
		t.Fatalf("ClaimTimeoutTasks: %v", err)
	}
	if len(claimed) != 1 || claimed[0].TaskID != id {
		t.Fatalf("reclaimed %d tasks, want the crashed one", len(claimed))
	}
	if claimed[0].WorkerID == nil || *claimed[0].WorkerID != "rescue-worker" {
		t.Errorf("new worker = %v, want rescue-worker", claimed[0].WorkerID)
	}
}

func TestClaimTimeoutTasks_ReclaimsRunningTaskWithoutLease(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	repo := NewTaskRepository(pool)
	tenant := uuid.New()

	id := seedTask(t, pool, tenant, string(domain.TaskTypeBroadcast), 5)
	_, err := pool.Exec(context.Background(),
		`UPDATE tasks SET status='running', worker_id=NULL, locked_at=NULL WHERE task_id=$1`, id)
	if err != nil {
		t.Fatalf("simulate running task without lease: %v", err)
	}

	claimed, err := repo.ClaimTimeoutTasks(context.Background(), "recovery-worker", 5, 5)
	if err != nil {
		t.Fatalf("ClaimTimeoutTasks: %v", err)
	}
	if len(claimed) != 1 || claimed[0].TaskID != id {
		t.Fatalf("reclaimed %d tasks, want task with missing lease", len(claimed))
	}
	if claimed[0].WorkerID == nil || *claimed[0].WorkerID != "recovery-worker" {
		t.Errorf("new worker = %v, want recovery-worker", claimed[0].WorkerID)
	}
}

func TestClaimTasks_SkipsInactiveAndExpiredTenants(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	repo := NewTaskRepository(pool)

	activeTenant := uuid.New()
	inactiveTenant := uuid.New()
	expiredTenant := uuid.New()
	activeTask := seedTask(t, pool, activeTenant, string(domain.TaskTypeBroadcast), 5)
	inactiveTask := seedTask(t, pool, inactiveTenant, string(domain.TaskTypeBroadcast), 9)
	expiredTask := seedTask(t, pool, expiredTenant, string(domain.TaskTypeBroadcast), 8)

	if _, err := pool.Exec(context.Background(),
		`UPDATE tenants SET status = 'suspended' WHERE tenant_id = $1`, inactiveTenant); err != nil {
		t.Fatalf("suspend test tenant: %v", err)
	}
	if _, err := pool.Exec(context.Background(),
		`UPDATE tenants SET expires_at = NOW() - INTERVAL '1 minute' WHERE tenant_id = $1`, expiredTenant); err != nil {
		t.Fatalf("expire test tenant: %v", err)
	}

	claimed, err := repo.ClaimTasks(context.Background(), "active-worker", 10, 5)
	if err != nil {
		t.Fatalf("ClaimTasks: %v", err)
	}
	if len(claimed) != 1 || claimed[0].TaskID != activeTask {
		t.Fatalf("claimed tasks = %v, want only active tenant task %s", taskIDs(claimed), activeTask)
	}
	for _, id := range []uuid.UUID{inactiveTask, expiredTask} {
		var status string
		if err := pool.QueryRow(context.Background(), `SELECT status FROM tasks WHERE task_id = $1`, id).Scan(&status); err != nil {
			t.Fatalf("read task %s: %v", id, err)
		}
		if status != "queued" {
			t.Errorf("inactive tenant task %s status = %q, want queued", id, status)
		}
	}
}

func taskIDs(tasks []*domain.Task) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.TaskID)
	}
	return ids
}

func TestTaskLifecycleUpdatesRespectCurrentStatus(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	repo := NewTaskRepository(pool)
	ctx := context.Background()
	tenant := uuid.New()

	loopID := seedTask(t, pool, tenant, string(domain.TaskTypeScheduledBroadcast), 5)
	if _, err := pool.Exec(ctx, `UPDATE tasks SET is_loop = TRUE WHERE task_id = $1`, loopID); err != nil {
		t.Fatalf("mark loop task: %v", err)
	}
	if err := repo.StopLoop(ctx, tenant, loopID); err != nil {
		t.Fatalf("StopLoop: %v", err)
	}
	if err := repo.ResetToQueued(ctx, tenant, loopID); err != nil {
		t.Fatalf("resume paused loop task: %v", err)
	}
	var status string
	var isLoop bool
	var retryCount int
	if err := pool.QueryRow(ctx, `SELECT status, is_loop, retry_count FROM tasks WHERE task_id = $1`, loopID).Scan(&status, &isLoop, &retryCount); err != nil {
		t.Fatalf("read resumed task: %v", err)
	}
	if status != "queued" || isLoop || retryCount != 0 {
		t.Fatalf("resumed loop state = status:%s loop:%t retries:%d; want queued/false/0", status, isLoop, retryCount)
	}

	if _, err := pool.Exec(ctx, `UPDATE tasks SET status = 'cancelled' WHERE task_id = $1`, loopID); err != nil {
		t.Fatalf("cancel task fixture: %v", err)
	}
	if err := repo.ResetToQueued(ctx, tenant, loopID); err == nil {
		t.Fatal("ResetToQueued must not revive a cancelled task")
	}

	completedID := seedTask(t, pool, tenant, string(domain.TaskTypeBroadcast), 5)
	if _, err := pool.Exec(ctx, `UPDATE tasks SET status = 'completed' WHERE task_id = $1`, completedID); err != nil {
		t.Fatalf("complete task fixture: %v", err)
	}
	cancelled, err := repo.CancelIfActive(ctx, tenant, completedID)
	if err != nil {
		t.Fatalf("CancelIfActive: %v", err)
	}
	if cancelled {
		t.Fatal("CancelIfActive must not overwrite a completed task")
	}
	if err := pool.QueryRow(ctx, `SELECT status FROM tasks WHERE task_id = $1`, completedID).Scan(&status); err != nil {
		t.Fatalf("read completed task: %v", err)
	}
	if status != "completed" {
		t.Fatalf("completed task status = %q, want completed", status)
	}

	finishedLoopID := seedTask(t, pool, tenant, string(domain.TaskTypeScheduledBroadcast), 5)
	if _, err := pool.Exec(ctx, `UPDATE tasks SET is_loop = TRUE, status = 'completed' WHERE task_id = $1`, finishedLoopID); err != nil {
		t.Fatalf("mark completed loop task: %v", err)
	}
	if err := repo.StopLoop(ctx, tenant, finishedLoopID); err != nil {
		t.Fatalf("StopLoop on completed cycle: %v", err)
	}
	if err := repo.RescheduleLoopTask(ctx, tenant, finishedLoopID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("RescheduleLoopTask after stop: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT status, is_loop FROM tasks WHERE task_id = $1`, finishedLoopID).Scan(&status, &isLoop); err != nil {
		t.Fatalf("read stopped loop task: %v", err)
	}
	if status != "paused" || isLoop {
		t.Fatalf("stopped loop state = status:%s loop:%t; want paused/false", status, isLoop)
	}
}

func TestPauseIfActiveRespectsCurrentTaskState(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	repo := NewTaskRepository(pool)
	ctx := context.Background()
	tenant := uuid.New()

	queuedID := seedTask(t, pool, tenant, string(domain.TaskTypeBroadcast), 5)
	paused, err := repo.PauseIfActive(ctx, tenant, queuedID)
	if err != nil || !paused {
		t.Fatalf("PauseIfActive(queued) = %t, %v; want true, nil", paused, err)
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM tasks WHERE task_id = $1`, queuedID).Scan(&status); err != nil {
		t.Fatalf("read queued task: %v", err)
	}
	if status != "paused" {
		t.Fatalf("queued task status = %q, want paused", status)
	}
	if paused, err := repo.PauseIfActive(ctx, tenant, queuedID); err != nil || paused {
		t.Fatalf("PauseIfActive(paused) = %t, %v; want false, nil", paused, err)
	}

	runningID := seedTask(t, pool, tenant, string(domain.TaskTypeBroadcast), 5)
	if _, err := pool.Exec(ctx, `UPDATE tasks SET status='running', worker_id='worker-a', locked_at=NOW() WHERE task_id=$1`, runningID); err != nil {
		t.Fatalf("mark task running: %v", err)
	}
	paused, err = repo.PauseIfActive(ctx, tenant, runningID)
	if err != nil || !paused {
		t.Fatalf("PauseIfActive(running) = %t, %v; want true, nil", paused, err)
	}
	var workerID *string
	if err := pool.QueryRow(ctx, `SELECT status, worker_id FROM tasks WHERE task_id = $1`, runningID).Scan(&status, &workerID); err != nil {
		t.Fatalf("read paused running task: %v", err)
	}
	if status != "paused" || workerID != nil {
		t.Fatalf("running task after pause = status:%s worker:%v; want paused/nil", status, workerID)
	}

	completedID := seedTask(t, pool, tenant, string(domain.TaskTypeBroadcast), 5)
	if _, err := pool.Exec(ctx, `UPDATE tasks SET status='completed' WHERE task_id=$1`, completedID); err != nil {
		t.Fatalf("mark task completed: %v", err)
	}
	if paused, err := repo.PauseIfActive(ctx, tenant, completedID); err != nil || paused {
		t.Fatalf("PauseIfActive(completed) = %t, %v; want false, nil", paused, err)
	}
	if err := pool.QueryRow(ctx, `SELECT status FROM tasks WHERE task_id = $1`, completedID).Scan(&status); err != nil {
		t.Fatalf("read completed task: %v", err)
	}
	if status != "completed" {
		t.Fatalf("completed task status = %q, want completed", status)
	}
}

func TestRescheduleLoopTaskStoresRFC3339RoundCheckpoint(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	repo := NewTaskRepository(pool)
	tenantID := uuid.New()
	taskID := seedTask(t, pool, tenantID, string(domain.TaskTypeScheduledBroadcast), 5)

	if _, err := pool.Exec(context.Background(), `
		UPDATE tasks
		SET status = 'completed', is_loop = TRUE,
		    scheduled_at = NOW() - INTERVAL '1 minute',
		    progress = '{"total":10,"done":8,"failed":2}'::jsonb,
		    payload = jsonb_build_object(
		        'round_count', 0, 'max_rounds', 3,
		        'round_started_at', to_char(NOW() - INTERVAL '2 minutes', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
		    )
		WHERE tenant_id = $1 AND task_id = $2
	`, tenantID, taskID); err != nil {
		t.Fatalf("prepare completed loop task: %v", err)
	}

	nextRun := time.Now().UTC().Add(time.Hour)
	if err := repo.RescheduleLoopTask(context.Background(), tenantID, taskID, nextRun); err != nil {
		t.Fatalf("RescheduleLoopTask: %v", err)
	}

	var checkpoint string
	var status string
	var isLoop bool
	if err := pool.QueryRow(context.Background(), `
		SELECT payload->>'round_started_at', status, is_loop
		FROM tasks WHERE tenant_id = $1 AND task_id = $2
	`, tenantID, taskID).Scan(&checkpoint, &status, &isLoop); err != nil {
		t.Fatalf("read rescheduled loop task: %v", err)
	}
	if _, err := time.Parse(time.RFC3339Nano, checkpoint); err != nil {
		t.Fatalf("round_started_at %q is not RFC3339Nano: %v", checkpoint, err)
	}
	if status != "pending" || !isLoop {
		t.Fatalf("rescheduled state = status:%s loop:%t; want pending/true", status, isLoop)
	}

	var round, total, success, failed int
	var startedAt, completedAt, nextRunText, roundStatus string
	if err := pool.QueryRow(context.Background(), `
		SELECT
			(payload->'round_history'->0->>'round')::int,
			(payload->'round_history'->0->>'total')::int,
			(payload->'round_history'->0->>'success')::int,
			(payload->'round_history'->0->>'failed')::int,
			payload->'round_history'->0->>'started_at',
			payload->'round_history'->0->>'completed_at',
			payload->'round_history'->0->>'next_run_at',
			payload->'round_history'->0->>'status'
		FROM tasks WHERE tenant_id = $1 AND task_id = $2
	`, tenantID, taskID).Scan(&round, &total, &success, &failed, &startedAt, &completedAt, &nextRunText, &roundStatus); err != nil {
		t.Fatalf("read round history: %v", err)
	}
	if round != 1 || total != 10 || success != 8 || failed != 2 {
		t.Fatalf("round summary = round:%d total:%d success:%d failed:%d", round, total, success, failed)
	}
	if startedAt == "" || completedAt == "" || nextRunText == "" || roundStatus != "scheduled" {
		t.Fatalf("incomplete round summary: started=%q completed=%q next=%q status=%q", startedAt, completedAt, nextRunText, roundStatus)
	}
}

func TestSetLoopConfigPreservesFirstScheduledRun(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	repo := NewTaskRepository(pool)
	tenantID := uuid.New()
	taskID := seedTask(t, pool, tenantID, string(domain.TaskTypeScheduledBroadcast), 5)
	firstRun := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Microsecond)
	if _, err := pool.Exec(context.Background(), `
		UPDATE tasks SET scheduled_at = $3, status = 'pending'
		WHERE tenant_id = $1 AND task_id = $2
	`, tenantID, taskID, firstRun); err != nil {
		t.Fatalf("set first scheduled run: %v", err)
	}
	if err := repo.SetLoopConfig(context.Background(), tenantID, taskID, true, 60, 5, 12, domain.TaskTypeScheduledBroadcast); err != nil {
		t.Fatalf("SetLoopConfig: %v", err)
	}
	var nextRun time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT next_run_at FROM tasks WHERE tenant_id = $1 AND task_id = $2
	`, tenantID, taskID).Scan(&nextRun); err != nil {
		t.Fatalf("read next run: %v", err)
	}
	if !nextRun.Equal(firstRun) {
		t.Fatalf("next_run_at = %s, want first scheduled run %s", nextRun, firstRun)
	}
}

func TestRescheduleLoopTaskArchivesFinalRoundAndCapsHistory(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	repo := NewTaskRepository(pool)
	tenantID := uuid.New()
	taskID := seedTask(t, pool, tenantID, string(domain.TaskTypeScheduledBroadcast), 5)

	if _, err := pool.Exec(context.Background(), `
		UPDATE tasks
		SET status = 'completed', is_loop = TRUE,
		    scheduled_at = NOW() - INTERVAL '1 minute',
		    progress = '{"total":4,"done":3,"failed":1}'::jsonb,
		    payload = jsonb_build_object(
		        'round_count', 2, 'max_rounds', 3,
		        'round_started_at', to_char(NOW() - INTERVAL '2 minutes', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
		        'round_history', (SELECT jsonb_agg(jsonb_build_object('round', n)) FROM generate_series(1, 55) n)
		    )
		WHERE tenant_id = $1 AND task_id = $2
	`, tenantID, taskID); err != nil {
		t.Fatalf("prepare final loop round: %v", err)
	}

	if err := repo.RescheduleLoopTask(context.Background(), tenantID, taskID, time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("RescheduleLoopTask: %v", err)
	}

	var status string
	var isLoop bool
	var historyCount int
	var round, success, failed int
	var summaryStatus string
	var nextRun *time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT status, is_loop, next_run_at,
		       jsonb_array_length(payload->'round_history'),
		       (payload->'round_history'->49->>'round')::int,
		       (payload->'round_history'->49->>'success')::int,
		       (payload->'round_history'->49->>'failed')::int,
		       payload->'round_history'->49->>'status'
		FROM tasks WHERE tenant_id = $1 AND task_id = $2
	`, tenantID, taskID).Scan(&status, &isLoop, &nextRun, &historyCount, &round, &success, &failed, &summaryStatus); err != nil {
		t.Fatalf("read final loop state: %v", err)
	}
	if status != "completed" || isLoop || nextRun != nil {
		t.Fatalf("final loop state = status:%s loop:%t next:%v; want completed/false/nil", status, isLoop, nextRun)
	}
	if historyCount != 50 || round != 3 || success != 3 || failed != 1 || summaryStatus != "finished" {
		t.Fatalf("final round archive = count:%d round:%d success:%d failed:%d status:%s", historyCount, round, success, failed, summaryStatus)
	}
}

func TestResumeLoopPreservesFutureCalendarRun(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	repo := NewTaskRepository(pool)
	tenantID := uuid.New()
	taskID := seedTask(t, pool, tenantID, string(domain.TaskTypeScheduledBroadcast), 5)
	nextRun := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Microsecond)
	if _, err := pool.Exec(context.Background(), `
		UPDATE tasks SET status = 'paused', is_loop = TRUE,
		    next_run_at = $3, scheduled_at = $3
		WHERE tenant_id = $1 AND task_id = $2
	`, tenantID, taskID, nextRun); err != nil {
		t.Fatalf("prepare paused calendar task: %v", err)
	}
	if err := repo.ResumeLoop(context.Background(), tenantID, taskID); err != nil {
		t.Fatalf("ResumeLoop: %v", err)
	}
	var status string
	var scheduledAt *time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT status, scheduled_at FROM tasks WHERE tenant_id = $1 AND task_id = $2
	`, tenantID, taskID).Scan(&status, &scheduledAt); err != nil {
		t.Fatalf("read resumed calendar task: %v", err)
	}
	if status != "pending" || scheduledAt == nil || !scheduledAt.Equal(nextRun) {
		t.Fatalf("resumed future task = status:%s scheduled_at:%v; want pending at %s", status, scheduledAt, nextRun)
	}
}

func TestResumeLoopQueuesOverdueCalendarRunImmediately(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	repo := NewTaskRepository(pool)
	tenantID := uuid.New()
	taskID := seedTask(t, pool, tenantID, string(domain.TaskTypeScheduledBroadcast), 5)
	pastRun := time.Now().UTC().Add(-time.Hour)
	if _, err := pool.Exec(context.Background(), `
		UPDATE tasks SET status = 'paused', is_loop = TRUE,
		    next_run_at = $3, scheduled_at = $3
		WHERE tenant_id = $1 AND task_id = $2
	`, tenantID, taskID, pastRun); err != nil {
		t.Fatalf("prepare overdue paused calendar task: %v", err)
	}
	if err := repo.ResumeLoop(context.Background(), tenantID, taskID); err != nil {
		t.Fatalf("ResumeLoop: %v", err)
	}
	var status string
	var scheduledAt *time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT status, scheduled_at FROM tasks WHERE tenant_id = $1 AND task_id = $2
	`, tenantID, taskID).Scan(&status, &scheduledAt); err != nil {
		t.Fatalf("read resumed overdue task: %v", err)
	}
	if status != "queued" || scheduledAt != nil {
		t.Fatalf("resumed overdue task = status:%s scheduled_at:%v; want queued/nil", status, scheduledAt)
	}
}

func TestTaskLogsCanBePagedWithinOneRoundWindow(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	tenantID := uuid.New()
	taskID := seedTask(t, pool, tenantID, string(domain.TaskTypeScheduledBroadcast), 5)
	logRepo := NewTaskLogRepository(pool)
	start := time.Now().UTC().Truncate(time.Second)
	logIDs := make([]uuid.UUID, 3)
	for i := range logIDs {
		logIDs[i] = uuid.New()
		entry := &domain.TaskLog{
			LogID: logIDs[i], TaskID: taskID, TenantID: tenantID,
			TargetID: fmt.Sprintf("target-%d", i+1), Action: "broadcast",
			Status: "success", ExecutedAt: start.Add(time.Duration(i+1) * time.Minute),
		}
		if err := logRepo.Create(context.Background(), entry); err != nil {
			t.Fatalf("insert round log %d: %v", i+1, err)
		}
	}

	from, to := start.Add(30*time.Second), start.Add(4*time.Minute)
	firstPage, err := logRepo.ListByTaskRange(context.Background(), tenantID, taskID, &from, &to, 2, 0)
	if err != nil {
		t.Fatalf("ListByTaskRange: %v", err)
	}
	if len(firstPage) != 2 || firstPage[0].LogID != logIDs[2] || firstPage[1].LogID != logIDs[1] {
		t.Fatalf("first round page = %#v; want newest two logs in descending order", firstPage)
	}

	nextPage, err := logRepo.ListBeforeByTaskRange(context.Background(), tenantID, taskID, firstPage[1].LogID, &from, &to, 2)
	if err != nil {
		t.Fatalf("ListBeforeByTaskRange: %v", err)
	}
	if len(nextPage) != 1 || nextPage[0].LogID != logIDs[0] {
		t.Fatalf("cursor round page = %#v; want oldest remaining log", nextPage)
	}

	excludedFrom, excludedTo := start.Add(2*time.Minute), start.Add(3*time.Minute)
	exactWindow, err := logRepo.ListByTaskRange(context.Background(), tenantID, taskID, &excludedFrom, &excludedTo, 10, 0)
	if err != nil {
		t.Fatalf("ListByTaskRange exact window: %v", err)
	}
	if len(exactWindow) != 1 || exactWindow[0].LogID != logIDs[1] {
		t.Fatalf("half-open round window = %#v; want only log at lower bound", exactWindow)
	}
}

func TestClaimTimeoutTasks_PreservesRecurringRoundCheckpoint(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	repo := NewTaskRepository(pool)
	tenantID := uuid.New()
	taskID := seedTask(t, pool, tenantID, string(domain.TaskTypeScheduledBroadcast), 5)
	checkpoint := time.Now().UTC().Add(-2 * time.Minute).Format(time.RFC3339Nano)

	_, err := pool.Exec(context.Background(), `
		UPDATE tasks
		SET status = 'running', is_loop = TRUE, worker_id = 'stopped-worker',
		    locked_at = NOW() - INTERVAL '6 minutes',
		    payload = jsonb_build_object('round_count', 2, 'round_started_at', $2::text)
		WHERE tenant_id = $1 AND task_id = $3
	`, tenantID, checkpoint, taskID)
	if err != nil {
		t.Fatalf("prepare interrupted recurring broadcast: %v", err)
	}

	claimed, err := repo.ClaimTimeoutTasks(context.Background(), "recovery-worker", 5, 5)
	if err != nil {
		t.Fatalf("ClaimTimeoutTasks: %v", err)
	}
	if len(claimed) != 1 || claimed[0].TaskID != taskID {
		t.Fatalf("reclaimed tasks=%d, want the interrupted recurring broadcast", len(claimed))
	}

	var storedCheckpoint string
	var roundCount int
	var workerID string
	if err := pool.QueryRow(context.Background(), `
		SELECT payload->>'round_started_at', (payload->>'round_count')::int, worker_id
		FROM tasks WHERE tenant_id = $1 AND task_id = $2
	`, tenantID, taskID).Scan(&storedCheckpoint, &roundCount, &workerID); err != nil {
		t.Fatalf("read recovered task: %v", err)
	}
	if storedCheckpoint != checkpoint || roundCount != 2 {
		t.Fatalf("recovery changed round checkpoint: checkpoint=%q round=%d", storedCheckpoint, roundCount)
	}
	if workerID != "recovery-worker" {
		t.Fatalf("worker after recovery=%q, want recovery-worker", workerID)
	}
}


func TestDeleteOnlyAllowsTerminalTasks(t *testing.T) {
	pool := connectTestDB(t)
	ensureTasksTable(t, pool)
	ctx := context.Background()
	tenantID := uuid.New()
	repo := NewTaskRepository(pool)

	// seedTask also creates the tenant row required by the foreign key.
	activeID := seedTask(t, pool, tenantID, string(domain.TaskTypeBroadcast), 5)
	if err := repo.Delete(ctx, tenantID, activeID); err == nil {
		t.Fatal("queued task deletion unexpectedly succeeded")
	}

	for _, status := range []string{"pending", "scheduled", "paused", "running"} {
		id := uuid.New()
		if _, err := pool.Exec(ctx, `
			INSERT INTO tasks (task_id, tenant_id, type, status, priority)
			VALUES ($1, $2, 'broadcast', $3, 5)
		`, id, tenantID, status); err != nil {
			t.Fatalf("insert %s task: %v", status, err)
		}
		if err := repo.Delete(ctx, tenantID, id); err == nil {
			t.Errorf("%s task deletion unexpectedly succeeded", status)
		}
	}

	for _, status := range []string{"completed", "failed", "cancelled"} {
		id := uuid.New()
		if _, err := pool.Exec(ctx, `
			INSERT INTO tasks (task_id, tenant_id, type, status, priority)
			VALUES ($1, $2, 'broadcast', $3, 5)
		`, id, tenantID, status); err != nil {
			t.Fatalf("insert %s task: %v", status, err)
		}
		if err := repo.Delete(ctx, tenantID, id); err != nil {
			t.Errorf("%s task deletion failed: %v", status, err)
		}
	}
}
