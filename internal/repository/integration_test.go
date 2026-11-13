//go:build integration

// T 工作流：裸 SQL 集成测试——D-00（campaigns 主键列名漂移）证明编译器/vet 均
// 覆盖不到裸 SQL 字符串，必须用真实 PostgreSQL 验证。
// 运行：go test -tags=integration ./internal/repository/  （需 Docker）
package repository

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"tg-sender/internal/domain"
)

// setupTestDB 启动容器化 PG，应用全部迁移，返回连接池。
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	// BasicWaitStrategies 等待 PostgreSQL 两次报告 ready，并确认宿主端口可连接，
	// 避免仅容器启动就绪、数据库尚未接受连接时发生 reset。
	pg, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.BasicWaitStrategies(),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Log("postgres container ping failed; see container logs above")
		logCtx, cancelLogs := context.WithTimeout(ctx, 5*time.Second)
		logs, logErr := pg.Logs(logCtx)
		if logErr == nil {
			if raw, readErr := io.ReadAll(logs); readErr == nil {
				t.Logf("postgres container logs:\\n%s", raw)
			} else {
				t.Logf("read postgres container logs: %v", readErr)
			}
			_ = logs.Close()
		} else {
			t.Logf("read postgres container logs: %v", logErr)
		}
		cancelLogs()
		t.Fatalf("ping postgres: %v", err)
	}

	// 应用全部迁移（与生产同路径）
	migDir, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("migrations path: %v", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose dialect: %v", err)
	}
	if err := goose.Up(stdlib.OpenDBFromPool(pool), migDir); err != nil {
		// 连接在迁移期间被远端重置时，附带 PostgreSQL 容器日志，
		// 区分迁移 SQL 错误、服务进程崩溃和容器被外部终止。
		logCtx, cancelLogs := context.WithTimeout(ctx, 5*time.Second)
		logs, logErr := pg.Logs(logCtx)
		if logErr == nil {
			if raw, readErr := io.ReadAll(logs); readErr == nil {
				t.Logf("postgres container logs:\\n%s", raw)
			} else {
				t.Logf("read postgres container logs: %v", readErr)
			}
			_ = logs.Close()
		} else {
			t.Logf("read postgres container logs: %v", logErr)
		}
		cancelLogs()
		t.Fatalf("apply migrations: %v", err)
	}
	return pool
}

// TestDeleteTenantCascadeQueries：验证 D-00 类列名漂移不回归——
// 软删除/PurgeTenant 的全部子查询必须在真实 Schema 上执行成功。
func TestTenantSoftDeleteAndPurgeQueries(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	// 种子：租户+账号+活动+活动账号（验证 campaign 子查询的 id 列名）
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (tenant_id, username, password_hash, name, api_key_hash, plan_id)
		VALUES ('11111111-1111-1111-1111-111111111111', 'integtest', 'x', 'Integ', 'integ-test-key-hash', 'free');
		INSERT INTO accounts (account_id, tenant_id, phone, session_name, status)
		VALUES ('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', '+10000000000', 'integration-session', 'offline');
		INSERT INTO campaigns (id, tenant_id, name, status, objective)  -- 注意：主键是 id（D-00 教训）
		VALUES ('33333333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', 'c1', 'running', '{}'::jsonb);
		INSERT INTO campaign_accounts (campaign_id, account_id, role)
		VALUES ('33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222222', 'adder');
	`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// 软删除：deleted_at 置位，行保留
	var deletedAt *time.Time
	if err := pool.QueryRow(ctx,
		`UPDATE tenants SET status='suspended', deleted_at=NOW() WHERE tenant_id=$1 RETURNING deleted_at`,
		"11111111-1111-1111-1111-111111111111").Scan(&deletedAt); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	if deletedAt == nil {
		t.Fatal("deleted_at should be set")
	}

	// GetByID 过滤软删
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tenants WHERE tenant_id=$1 AND deleted_at IS NULL`,
		"11111111-1111-1111-1111-111111111111").Scan(&n); err != nil || n != 0 {
		t.Fatalf("GetByID filter broken: n=%d err=%v", n, err)
	}

	// 模拟 PurgeTenant 的三个子查询（与 admin.go 中的裸 SQL 保持同构）
	if _, err := pool.Exec(ctx, `DELETE FROM campaign_accounts WHERE campaign_id IN (SELECT id FROM campaigns WHERE tenant_id = $1)`,
		"11111111-1111-1111-1111-111111111111"); err != nil {
		t.Fatalf("campaign_accounts subquery (D-00 regression): %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM campaigns WHERE tenant_id = $1`,
		"11111111-1111-1111-1111-111111111111"); err != nil {
		t.Fatalf("campaigns delete: %v", err)
	}

	// ListRunning / GetPendingAll 的 JOIN 查询（G-03）在真实 Schema 上成立
	if _, err := pool.Exec(ctx, `
		SELECT campaigns.id FROM campaigns
		JOIN tenants t ON t.tenant_id = campaigns.tenant_id
		WHERE campaigns.status='running' AND t.status='active'
		  AND (t.expires_at IS NULL OR t.expires_at > NOW())`); err != nil {
		t.Fatalf("ListRunning join query (G-03): %v", err)
	}
	if _, err := pool.Exec(ctx, `
		SELECT pa.id FROM pending_approvals pa
		JOIN tenants t ON t.tenant_id = pa.tenant_id
		WHERE pa.status='pending' AND t.status='active'
		  AND (t.expires_at IS NULL OR t.expires_at > NOW())`); err != nil {
		t.Fatalf("GetPendingAll join query (G-03): %v", err)
	}
}

// TestUpdateStatusGuardMatrix：F-27 守卫在真实 DB 上的迁移矩阵。
func TestUpdateStatusGuardMatrix(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (tenant_id, username, password_hash, name, api_key_hash, plan_id)
		VALUES ('11111111-1111-1111-1111-111111111111', 'guardtest', 'x', 'G', 'guard-test-key-hash', 'free');
		INSERT INTO tasks (tenant_id, task_id, type, status)
		VALUES ('11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444444', 'broadcast', 'running');
	`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	// 非法迁移 running->scheduled 应命中 0 行
	res, err := pool.Exec(ctx, `
		UPDATE tasks SET status='scheduled'
		WHERE tenant_id=$1 AND task_id=$2 AND (
			'scheduled' IN ('failed','cancelled','queued')
			OR (status='queued' AND 'scheduled' IN ('running','cancelled'))
			OR (status='running' AND 'scheduled' IN ('paused','completed','failed','cancelled')))`,
		"11111111-1111-1111-1111-111111111111", "44444444-4444-4444-4444-444444444444")
	if err != nil {
		t.Fatalf("guard query: %v", err)
	}
	if res.RowsAffected() != 0 {
		t.Fatal("illegal transition running->scheduled should affect 0 rows")
	}
	// 合法迁移 running->paused 应命中 1 行
	res, err = pool.Exec(ctx, `
		UPDATE tasks SET status='paused'
		WHERE tenant_id=$1 AND task_id=$2 AND (
			'paused' IN ('failed','cancelled','queued')
			OR (status='running' AND 'paused' IN ('paused','completed','failed','cancelled')))`,
		"11111111-1111-1111-1111-111111111111", "44444444-4444-4444-4444-444444444444")
	if err != nil {
		t.Fatalf("legal query: %v", err)
	}
	if res.RowsAffected() != 1 {
		t.Fatal("legal transition running->paused should affect 1 row")
	}
}

func TestPendingApprovalResolutionIsAtomicAndCancellationWins(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	tenantID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	accountID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	secondAccountID := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	approvedTaskID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	cancelledTaskID := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	approvedLink := "https://t.me/example_approval"
	timeoutLink := "https://t.me/example_timeout"
	cancelledLink := "https://t.me/example_cancel"

	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (tenant_id, username, password_hash, name, api_key_hash, plan_id)
		VALUES ($1, 'approvaltest', 'x', 'Approval Test', 'approval-test-key', 'free')
	`, tenantID)
	if err != nil {
		t.Fatalf("seed approval tenant: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO tasks (tenant_id, task_id, type, status)
		VALUES ($1, $2, 'join_groups', 'completed'), ($1, $3, 'join_groups', 'pending')
	`, tenantID, approvedTaskID, cancelledTaskID)
	if err != nil {
		t.Fatalf("seed approval tasks: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO pending_approvals (task_id, tenant_id, account_id, target_link)
		VALUES ($2, $1, $4, $6), ($2, $1, $5, $7), ($3, $1, $4, $8)
	`, tenantID, approvedTaskID, cancelledTaskID, accountID, secondAccountID, approvedLink, timeoutLink, cancelledLink)
	if err != nil {
		t.Fatalf("seed pending approvals: %v", err)
	}

	approvalRepo := NewPendingApprovalRepository(pool)
	taskRepo := NewTaskRepository(pool)
	var approvedID, timeoutID int
	if err := pool.QueryRow(ctx, `SELECT id FROM pending_approvals WHERE task_id = $1 AND target_link = $2`, approvedTaskID, approvedLink).Scan(&approvedID); err != nil {
		t.Fatalf("read approved approval id: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM pending_approvals WHERE task_id = $1 AND target_link = $2`, approvedTaskID, timeoutLink).Scan(&timeoutID); err != nil {
		t.Fatalf("read timeout approval id: %v", err)
	}

	resolved, err := approvalRepo.ResolvePending(ctx, uuid.MustParse(tenantID), approvedID, "approved")
	if err != nil || !resolved {
		t.Fatalf("first approval resolution = (%t, %v), want (true, nil)", resolved, err)
	}
	finalized, err := taskRepo.FinalizeTaskAfterApprovals(ctx, uuid.MustParse(tenantID), uuid.MustParse(approvedTaskID))
	if err != nil || finalized {
		t.Fatalf("task finalized with another approval pending = (%t, %v), want (false, nil)", finalized, err)
	}
	resolved, err = approvalRepo.ResolvePending(ctx, uuid.MustParse(tenantID), approvedID, "timeout")
	if err != nil {
		t.Fatalf("second resolution: %v", err)
	}
	if resolved {
		t.Fatal("second resolution must not overwrite the first result")
	}
	resolved, err = approvalRepo.ResolvePending(ctx, uuid.MustParse(tenantID), timeoutID, "timeout")
	if err != nil || !resolved {
		t.Fatalf("timeout approval resolution = (%t, %v), want (true, nil)", resolved, err)
	}
	finalized, err = taskRepo.FinalizeTaskAfterApprovals(ctx, uuid.MustParse(tenantID), uuid.MustParse(approvedTaskID))
	if err != nil || !finalized {
		t.Fatalf("finalize all resolved approvals = (%t, %v), want (true, nil)", finalized, err)
	}
	var finalTaskStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM tasks WHERE task_id = $1`, approvedTaskID).Scan(&finalTaskStatus); err != nil {
		t.Fatalf("read finalized task status: %v", err)
	}
	if finalTaskStatus != "failed" {
		t.Fatalf("final task status = %q, want failed because one approval timed out", finalTaskStatus)
	}

	cancelled, err := taskRepo.CancelIfActive(ctx, uuid.MustParse(tenantID), uuid.MustParse(cancelledTaskID))
	if err != nil || !cancelled {
		t.Fatalf("cancel pending task = (%t, %v), want (true, nil)", cancelled, err)
	}
	if err := approvalRepo.CancelPendingForTask(ctx, uuid.MustParse(tenantID), uuid.MustParse(cancelledTaskID)); err != nil {
		t.Fatalf("cancel pending approval: %v", err)
	}
	updated, err := taskRepo.UpdateStatusUnlessCancelled(ctx, uuid.MustParse(tenantID), uuid.MustParse(cancelledTaskID), domain.TaskStatusCompleted)
	if err != nil {
		t.Fatalf("late approval task update: %v", err)
	}
	if updated {
		t.Fatal("late approval must not overwrite cancelled task")
	}

	var approvedStatus, timeoutStatus, cancelledStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM pending_approvals WHERE task_id = $1 AND target_link = $2`, approvedTaskID, approvedLink).Scan(&approvedStatus); err != nil {
		t.Fatalf("read approved status: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT status FROM pending_approvals WHERE task_id = $1 AND target_link = $2`, approvedTaskID, timeoutLink).Scan(&timeoutStatus); err != nil {
		t.Fatalf("read timeout status: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT status FROM pending_approvals WHERE task_id = $1`, cancelledTaskID).Scan(&cancelledStatus); err != nil {
		t.Fatalf("read cancelled approval status: %v", err)
	}
	if approvedStatus != "approved" || timeoutStatus != "timeout" || cancelledStatus != "cancelled" {
		t.Fatalf("approval statuses = %q/%q/%q, want approved/timeout/cancelled", approvedStatus, timeoutStatus, cancelledStatus)
	}
}

func TestUpdateLinkValidityByLinkPreservesAlternateLinks(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	groupWithAliases := uuid.New()
	groupWithOneLink := uuid.New()
	if _, err := pool.Exec(ctx, "INSERT INTO system_groups (sg_id, dedup_key, source, gtype, status) VALUES ($1, 'u:alpha_group', 'import', 'unknown', 'active'), ($2, 'u:single_group', 'import', 'unknown', 'active')", groupWithAliases, groupWithOneLink); err != nil {
		t.Fatalf("seed shared groups: %v", err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO system_group_links (sg_id, original_link, normalized_link, link_type, validity_status, source) VALUES ($1, 'https://t.me/alpha_group', 't.me/alpha_group', 'public', 'pending', 'import'), ($1, 'https://t.me/+InviteHashABC', 't.me/+InviteHashABC', 'private_invite', 'pending', 'import'), ($2, 'https://t.me/single_group', 't.me/single_group', 'public', 'pending', 'import')", groupWithAliases, groupWithOneLink); err != nil {
		t.Fatalf("seed shared links: %v", err)
	}
	repo := NewSystemGroupRepository(pool)
	if err := repo.UpdateLinkValidityByLink(ctx, "https://t.me/alpha_group/", "invalid"); err != nil {
		t.Fatalf("invalidate public alias: %v", err)
	}
	var publicStatus, inviteStatus, groupStatus string
	if err := pool.QueryRow(ctx, "SELECT (SELECT validity_status FROM system_group_links WHERE normalized_link='t.me/alpha_group'), (SELECT validity_status FROM system_group_links WHERE normalized_link='t.me/+InviteHashABC'), (SELECT status FROM system_groups WHERE sg_id=$1)", groupWithAliases).Scan(&publicStatus, &inviteStatus, &groupStatus); err != nil {
		t.Fatalf("read alias states: %v", err)
	}
	if publicStatus != "invalid" || inviteStatus != "pending" || groupStatus != "active" {
		t.Fatalf("after invalidating one alias: public=%s invite=%s group=%s; want invalid/pending/active", publicStatus, inviteStatus, groupStatus)
	}
	if err := repo.UpdateLinkValidityByLink(ctx, "@single_group", "invalid"); err != nil {
		t.Fatalf("invalidate only alias: %v", err)
	}
	if err := pool.QueryRow(ctx, "SELECT status FROM system_groups WHERE sg_id=$1", groupWithOneLink).Scan(&groupStatus); err != nil {
		t.Fatalf("read single-link group state: %v", err)
	}
	if groupStatus != "invalid" {
		t.Fatalf("single-link group status = %q, want invalid", groupStatus)
	}
}

func TestAccountCountAndCreateRespectsTenantLimitUnderConcurrency(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (tenant_id, username, password_hash, name, api_key_hash, plan_id)
		VALUES ($1, $2, 'x', 'account limit test', $3, 'free')`,
		tenantID, "account-limit-"+tenantID.String(), "account-limit-key-"+tenantID.String())
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	repo := NewAccountRepository(pool)
	const limit = 2
	const requests = 12
	var created atomic.Int32
	var wg sync.WaitGroup
	errs := make(chan error, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := uuid.New()
			account := &domain.Account{
				AccountID:     id,
				TenantID:      tenantID,
				Phone:         "+1" + id.String()[:10],
				SessionName:   "limit-" + id.String(),
				Status:        domain.AccountStatusOffline,
				Level:         domain.AccountLevelNormal,
				DeviceProfile: map[string]any{},
			}
			if _, _, err := repo.CountAndCreate(ctx, account, limit); err != nil {
				if !strings.Contains(err.Error(), "account limit reached") {
					errs <- err
				}
				return
			}
			created.Add(1)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("CountAndCreate: %v", err)
	}
	if got := created.Load(); got != limit {
		t.Fatalf("created accounts = %d, want exactly %d", got, limit)
	}
	var stored int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM accounts WHERE tenant_id = $1 AND pool_status = 'active'`, tenantID).Scan(&stored); err != nil {
		t.Fatalf("count stored accounts: %v", err)
	}
	if stored != limit {
		t.Fatalf("stored active accounts = %d, want %d", stored, limit)
	}
}
