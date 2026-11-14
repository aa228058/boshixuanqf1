package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlanLimitRepository 套餐日额度: 免费体验版限额, 付费档 0=不限。
type PlanLimitRepository struct {
	db *pgxpool.Pool
}

func NewPlanLimitRepository(db *pgxpool.Pool) *PlanLimitRepository {
	return &PlanLimitRepository{db: db}
}

// GetPlanLimits 返回租户套餐的日额度(0=不限)。
// 群发日限用既有列 max_daily_broadcasts(与 can_* 开关同期的老配额体系)。
func (r *PlanLimitRepository) GetPlanLimits(ctx context.Context, tenantID uuid.UUID) (dailyBroadcast, dailySearch int, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(p.max_daily_broadcasts, 0), COALESCE(p.daily_search_limit, 0)
		FROM tenants t JOIN plans p ON p.plan_id = t.plan_id
		WHERE t.tenant_id = $1`, tenantID).Scan(&dailyBroadcast, &dailySearch)
	return
}

// TryAddDailyUsageAmount 原子预留指定数量的额度；单次任务超出上限或并发累计超限均不会写入。
func (r *PlanLimitRepository) TryAddDailyUsageAmount(ctx context.Context, tenantID uuid.UUID, kind string, amount, limit int) (bool, error) {
	if amount <= 0 || limit <= 0 || amount > limit {
		return false, nil
	}
	var used int
	err := r.db.QueryRow(ctx, `
		INSERT INTO plan_daily_counters (tenant_id, day, kind, used)
		SELECT $1, CURRENT_DATE, $2, $3::integer
		WHERE $3::integer <= $4::integer
		ON CONFLICT (tenant_id, day, kind) DO UPDATE
		SET used = plan_daily_counters.used + EXCLUDED.used
		WHERE plan_daily_counters.used + EXCLUDED.used <= $4
		RETURNING used`, tenantID, kind, amount, limit).Scan(&used)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ReleaseDailyUsageAmount 仅在已预留额度但对应业务记录未创建时回滚预留。
func (r *PlanLimitRepository) ReleaseDailyUsageAmount(ctx context.Context, tenantID uuid.UUID, kind string, amount int) error {
	if amount <= 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		UPDATE plan_daily_counters
		SET used = GREATEST(used - $3, 0)
		WHERE tenant_id = $1 AND day = CURRENT_DATE AND kind = $2`, tenantID, kind, amount)
	return err
}

// TryAddDailyUsage 在数据库内原子执行“未达上限才加一”，并发请求也无法突破日额度。
func (r *PlanLimitRepository) TryAddDailyUsage(ctx context.Context, tenantID uuid.UUID, kind string, limit int) (bool, error) {
	var used int
	err := r.db.QueryRow(ctx, `
		INSERT INTO plan_daily_counters (tenant_id, day, kind, used)
		VALUES ($1, CURRENT_DATE, $2, 1)
		ON CONFLICT (tenant_id, day, kind) DO UPDATE
		SET used = plan_daily_counters.used + 1
		WHERE plan_daily_counters.used < $3
		RETURNING used`, tenantID, kind, limit).Scan(&used)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetDailyUsage 返回某租户今天某类用量的当前值（0 表示今天还没用过）。
// 集成测试用它断言配额扣减，线上也可用于展示"今日已用"。
func (r *PlanLimitRepository) GetDailyUsage(ctx context.Context, tenantID uuid.UUID, kind string) (int, error) {
	var used int
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(used, 0) FROM plan_daily_counters
		WHERE tenant_id = $1 AND day = CURRENT_DATE AND kind = $2`, tenantID, kind).Scan(&used)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil // 今日还没用过
	}
	return used, err
}

// Kind 常量
const (
	KindBroadcastMsgs = "broadcast_msgs"
	KindGroupSearch   = "group_search"
)
