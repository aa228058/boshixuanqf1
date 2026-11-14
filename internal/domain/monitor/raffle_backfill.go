package monitor

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
)

// 停机补拉参数：服务重启期间发生的开奖/中奖消息不会主动推送，只能回扫补收。
// 上限是硬约束——12 小时的历史在活跃账号上可能有几万条，全量扫会打满 TG 请求。
const (
	// RaffleBackfillDefaultWindow 首次补拉（本进程内还没有水位线）覆盖的时长。
	RaffleBackfillDefaultWindow = 12 * time.Hour
	// RaffleBackfillMinWindow 断档窗口下限：掉线几十秒不值得扫 50 个会话。
	RaffleBackfillMinWindow = 5 * time.Minute

	raffleBackfillMaxChats = 50
	raffleBackfillMaxMsgs  = 200
	raffleBackfillChatGap  = 200 * time.Millisecond
)

// RaffleBackfillWindow 计算本次补拉应覆盖的时间窗：
// 还没有水位线（首次）→ 默认 12 小时；否则补"距上次"的断档，夹在 [5m, 12h]。
// 这样服务不重启但账号中途掉线时，恢复后能补上那一段，而不是只补一次。
func RaffleBackfillWindow(last, now time.Time) time.Duration {
	if last.IsZero() {
		return RaffleBackfillDefaultWindow
	}
	gap := now.Sub(last)
	if gap > RaffleBackfillDefaultWindow {
		gap = RaffleBackfillDefaultWindow
	}
	if gap < RaffleBackfillMinWindow {
		gap = RaffleBackfillMinWindow
	}
	return gap
}

// BackfillRaffleHistory 在会话建立后回扫最近 12 小时的消息，补收停机期间漏掉的
// 开奖/中奖事件。
//
// 只把消息喂给结果追踪(watchRaffleMessage)：绝不重放"自动参与"——对 12 小时前
// 的旧活动去抢参与等于给群里刷广告。
// 去重沿用 markMessageSeen，与实时通道共用同一张表，重复执行安全。
// 返回值表示这一轮是否真的扫完了（false = 取不到客户端/会话列表，调用方应尽快重试）。
func (e *Engine) BackfillRaffleHistory(ctx context.Context, tenantID, accountID uuid.UUID, window time.Duration) bool {
	if e.raffleWatch == nil || e.raffleWatch.onResult == nil {
		return true // 没有追踪需求，视作"无需补拉"，避免调用方反复重试
	}
	if e.accEngine == nil {
		return true
	}
	if window <= 0 {
		window = RaffleBackfillDefaultWindow
	}
	client, _, err := e.accEngine.GetClientByAccount(ctx, tenantID, accountID)
	if err != nil || client == nil {
		log.Printf("[raffle-backfill] 取客户端失败 tenant=%s account=%s: %v", tenantID, accountID, err)
		return false
	}
	defer e.accEngine.ReleaseClient(tenantID, accountID, client)

	cutoff := time.Now().Add(-window)
	dialogs, err := client.GetDialogs(ctx, raffleBackfillMaxChats, 0)
	if err != nil {
		log.Printf("[raffle-backfill] 拉会话列表失败 tenant=%s account=%s: %v", tenantID, accountID, err)
		return false
	}

	budget := raffleBackfillMaxMsgs
	scanned := 0
	replayed := 0
	for _, d := range dialogs {
		if budget <= 0 || ctx.Err() != nil {
			break
		}
		if d == nil || d.ID == 0 {
			continue
		}
		limit := 100
		if limit > budget {
			limit = budget
		}
		msgs, err := client.GetChatHistory(ctx, d.ID, limit, 0)
		if err != nil {
			continue // 单个会话取不到（已退群/无权限）不影响其它会话
		}
		scanned++
		for _, m := range msgs {
			if m == nil {
				continue
			}
			budget--
			// 历史按时间倒序返回，越过窗口就可以停了这个会话。
			if m.Date.Before(cutoff) {
				break
			}
			if !e.markMessageSeen(tenantID, accountID, m.ChatID, m.ID) {
				continue
			}
			snapshot := *m // 不跨 goroutine 持有客户端管理的对象
			e.watchRaffleMessage(ctx, tenantID, accountID, m.ChatID, &snapshot, client)
			replayed++
		}
		select {
		case <-ctx.Done():
			return true
		case <-time.After(raffleBackfillChatGap):
		}
	}
	log.Printf("[raffle-backfill] tenant=%s account=%s 回扫 %d 个会话 / %d 条窗口内消息 (窗口 %s)",
		tenantID, accountID, scanned, replayed, window)
	return true
}
