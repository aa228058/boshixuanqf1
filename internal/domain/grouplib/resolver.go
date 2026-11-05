package grouplib

import (
	"context"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"tg-sender/internal/domain"
)

// resolveBatchSize 每轮(2分钟)最多解析的条数。1 万条私密链接按 30 条/轮 ≈ 11 小时跑完,
// 对 TG 来说是极低频, 不会触发 flood; 也可临时调大加速(撞限流会自动等)。
const resolveBatchSize = 30

// resolvePendingTitles 后台批量解析: 扫出私密且无群名的行, 用 checkChatInvite 补名。
// 挂进 Engine.runOnce, 导入 1 万条后无需任何手工操作, 群名自动逐批补齐。
func (e *Engine) resolvePendingTitles(ctx context.Context) {
	type row struct {
		sgID     uuid.UUID
		link     string
		dedupKey string
	}
	rows, err := e.pool.Query(ctx, `
		SELECT sg_id, tme_link, dedup_key FROM system_groups
		WHERE gtype = 'private' AND COALESCE(title, '') = '' AND status = 'active'
		ORDER BY last_seen_at ASC LIMIT $1`, resolveBatchSize)
	if err != nil {
		log.Printf("[grouplib] 扫描待解析群失败: %v", err)
		return
	}
	var pending []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.sgID, &r.link, &r.dedupKey); err == nil {
			pending = append(pending, r)
		}
	}
	rows.Close()
	if len(pending) == 0 {
		return
	}

	// 随机借一个在线账号做校验(避免单号高频): 每轮重选, 天然轮换
	type accRef struct {
		tenantID  uuid.UUID
		accountID uuid.UUID
	}
	accRows, err := e.pool.Query(ctx, `
		SELECT tenant_id, account_id FROM accounts
		WHERE status = 'online' AND (blocked_until IS NULL OR blocked_until < NOW())
		ORDER BY RANDOM() LIMIT 5`)
	if err != nil {
		log.Printf("[grouplib] 扫描在线账号失败: %v", err)
		return
	}
	var accs []accRef
	for accRows.Next() {
		var a accRef
		if err := accRows.Scan(&a.tenantID, &a.accountID); err == nil {
			accs = append(accs, a)
		}
	}
	accRows.Close()
	if len(accs) == 0 {
		// 没有任何在线账号时仍可尝试 Telegram 网页预览；这不需要加入群组。
		for _, p := range pending {
			hash := strings.TrimPrefix(strings.TrimPrefix(p.link, "https://"), "t.me/+")
			hash = strings.TrimPrefix(hash, "+")
			title, count, werr := resolveInviteInfoViaWeb(ctx, hash)
			if werr == nil && (title != "" || count > 0) {
				_ = e.repo.UpdateResolvedInfo(ctx, p.sgID, title, count, "private")
			}
		}
		return
	}

	for i, p := range pending {
		a := accs[rand.Intn(len(accs))]
		client, err := e.acc.GetClient(ctx, a.tenantID, a.accountID)
		if err != nil {
			continue
		}
		hash := strings.TrimPrefix(strings.TrimPrefix(p.link, "https://"), "t.me/+")
		hash = strings.TrimPrefix(hash, "+")
		title, count, gtype, rerr := resolveInviteInfo(ctx, client, hash)
		e.acc.ReleaseClient(a.tenantID, a.accountID, client)
		if rerr != nil {
			errStr := strings.ToLower(rerr.Error())
			switch {
			case strings.Contains(rerr.Error(), "过期") || strings.Contains(rerr.Error(), "无效"):
				// 过期/无效: 直接标记失效, 不再反复解析
				_ = e.repo.MarkInvalidByDedupKey(ctx, p.dedupKey)
			case strings.Contains(errStr, "channel_private") || strings.Contains(errStr, "chat_private") ||
				strings.Contains(errStr, "kicked") || strings.Contains(errStr, "banned"):
				// 主路径拿不到(被踢/私密不可见): 降级抓 t.me 网页 og:title 兜底补名
				if title, webCount, werr := resolveInviteInfoViaWeb(ctx, hash); werr == nil && (title != "" || webCount > 0) {
					_ = e.repo.UpdateResolvedInfo(ctx, p.sgID, title, webCount, "private")
				}
			}
			continue
		}
		if err := e.repo.UpdateResolvedInfo(ctx, p.sgID, title, count, gtype); err != nil {
			log.Printf("[grouplib] 回写群名失败 %s: %v", p.sgID, err)
		}
		// 间隔 2s: 比导入接口更保守, 批量场景宁可慢不可封
		if i < len(pending)-1 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}
	log.Printf("[grouplib] 本轮解析补名 %d 条(待解析队列滚动处理)", len(pending))
}

// 保留 domain 引用(resolver 返回的群信息最终也以 SystemGroup 形态被查询)
var _ = domain.SystemGroup{}
