package grouplib

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
)

const resolveBatchSize = 30

// randomResolveDelay adds bounded jitter to keep background Telegram lookups
// from creating a fixed, burst-like request pattern. It is rate control, not
// a mechanism for bypassing Telegram access restrictions.
func randomResolveDelay() time.Duration {
	return 5*time.Second + time.Duration(rand.Int63n(int64(10*time.Second)+1))
}

func (e *Engine) resolvePendingTitles(ctx context.Context) {
	type row struct {
		sgID     uuid.UUID
		link     string
		dedupKey string
		username string
		gtype    string
		title    string
	}
	rows, err := e.pool.Query(ctx, "SELECT sg_id,tme_link,dedup_key,COALESCE(username,''),gtype,COALESCE(title,'') FROM system_groups WHERE gtype IN ('private','unknown','public') AND (resolution_status='pending' OR (resolution_status='identity_pending' AND last_seen_at<NOW()-INTERVAL '30 minutes')) AND status='active' ORDER BY last_seen_at ASC LIMIT $1", resolveBatchSize)
	if err != nil {
		log.Printf("[grouplib] 扫描待解析群失败: %v", err)
		return
	}
	var pending []row
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.sgID, &item.link, &item.dedupKey, &item.username, &item.gtype, &item.title); err == nil {
			pending = append(pending, item)
		}
	}
	rows.Close()
	if len(pending) == 0 {
		return
	}

	type accRef struct {
		tenantID  uuid.UUID
		accountID uuid.UUID
	}
	accRows, err := e.pool.Query(ctx, "SELECT tenant_id,account_id FROM accounts WHERE status='online' AND (blocked_until IS NULL OR blocked_until<NOW()) ORDER BY RANDOM() LIMIT 5")
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

	for i, p := range pending {
		if len(accs) == 0 && p.title != "" {
			continue
		}
		var title string
		var count int
		var gtype string
		var chatID int64
		var resolveErr error
		if len(accs) > 0 {
			a := accs[rand.Intn(len(accs))]
			client, clientErr := e.acc.GetClient(ctx, a.tenantID, a.accountID)
			if clientErr == nil {
				if p.gtype == "private" {
					hash := inviteHashFromLink(p.link)
					title, count, gtype, chatID, resolveErr = resolveInviteInfo(ctx, client, hash)
				} else {
					title, count, gtype, chatID, resolveErr = resolveGroupInfo(ctx, client, p.username)
				}
				e.acc.ReleaseClient(a.tenantID, a.accountID, client)
			} else {
				resolveErr = clientErr
			}
		}
		if resolveErr != nil || len(accs) == 0 {
			mtprotoErr := resolveErr
			var webTitle string
			var webCount int
			var webErr error
			if p.gtype == "private" {
				hash := inviteHashFromLink(p.link)
				webTitle, webCount, webErr = resolveInviteInfoViaWeb(ctx, hash)
			} else {
				webTitle, webCount, webErr = resolvePublicGroupInfoViaWeb(ctx, p.username)
			}
			if webErr == nil && (webTitle != "" || webCount > 0) {
				title, count, chatID = webTitle, webCount, 0
				gtype = p.gtype
				if gtype == "unknown" {
					gtype = "public"
				}
				resolveErr = nil
			} else {
				var rateLimitErr *TelegramRateLimitError
				if errors.As(webErr, &rateLimitErr) {
					resolveErr = webErr
				} else if mtprotoErr != nil {
					resolveErr = mtprotoErr
				} else {
					resolveErr = webErr
				}
			}
		}
		var rateLimitErr *TelegramRateLimitError
		if errors.As(resolveErr, &rateLimitErr) {
			log.Printf("[grouplib] Telegram 网页限流，暂停本轮解析队列 %s: %v", rateLimitErr.RetryAfter, rateLimitErr)
			timer := time.NewTimer(rateLimitErr.RetryAfter)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
			case <-timer.C:
			}
			return
		}
		if resolveErr != nil {
			errText := strings.ToLower(resolveErr.Error())
			if (p.gtype == "private" && (strings.Contains(errText, "过期") || strings.Contains(errText, "无效"))) || (p.gtype == "unknown" && strings.Contains(errText, "返回 404")) {
				_ = e.repo.MarkInvalidByDedupKey(ctx, p.dedupKey)
				_ = e.repo.UpdateLinkValidity(ctx, p.sgID, 0, "invalid")
			}
		} else if title != "" || count > 0 {
			if gtype == "" {
				gtype = p.gtype
			}
			if err := e.repo.UpdateResolvedInfo(ctx, p.sgID, title, count, gtype, chatID); err != nil {
				log.Printf("[grouplib] 回写群组资料失败 %s: %v", p.sgID, err)
			} else {
				_ = e.repo.UpdateLinkValidity(ctx, p.sgID, chatID, "valid")
			}
		}
		if i < len(pending)-1 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(randomResolveDelay()):
			}
		}
	}

	log.Printf("[grouplib] 本轮解析补名 %d 条(待解析队列滚动处理)", len(pending))
}
