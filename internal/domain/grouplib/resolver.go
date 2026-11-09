package grouplib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
)

const resolveBatchSize = 30

func (e *Engine) resolvePendingTitles(ctx context.Context) {
	type row struct {
		sgID     uuid.UUID
		link     string
		username string
		gtype    string
		title    string
	}
	rows, err := e.pool.Query(ctx, "SELECT sg_id,tme_link,COALESCE(username,''),gtype,COALESCE(title,'') FROM system_groups WHERE gtype IN ('private','unknown','public','group','supergroup','channel') AND (resolution_status='pending' OR (resolution_status='identity_pending' AND last_seen_at<NOW()-INTERVAL '30 minutes')) AND status='active' ORDER BY last_seen_at ASC LIMIT $1", resolveBatchSize)
	if err != nil {
		log.Printf("[grouplib] 扫描待解析群失败: %v", err)
		return
	}
	defer rows.Close()
	var pending []row
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.sgID, &item.link, &item.username, &item.gtype, &item.title); err != nil {
			log.Printf("[grouplib] 读取待解析群记录失败: %v", err)
			return
		}
		pending = append(pending, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[grouplib] 遍历待解析群记录失败: %v", err)
		return
	}
	if len(pending) == 0 {
		return
	}

	for _, p := range pending {
		isInviteLink := inviteHashFromLink(p.link) != ""
		// Defensive cleanup for bot links inserted by older import/aggregation paths.
		if !isInviteLink && strings.HasSuffix(strings.ToLower(strings.TrimPrefix(p.username, "@")), "bot") {
			if err := e.repo.UpdateLinkValidityByLink(ctx, p.link, "invalid"); err != nil {
				log.Printf("[grouplib] 标记机器人链接失效失败 %s: %v", p.link, err)
			}
			continue
		}

		var title string
		var count int
		var pageType string
		var resolveErr error
		if isInviteLink {
			title, count, pageType, resolveErr = resolveInviteInfoViaWeb(ctx, inviteHashFromLink(p.link))
		} else if p.username != "" {
			title, count, pageType, resolveErr = resolvePublicGroupInfoViaWeb(ctx, p.username)
		} else {
			resolveErr = fmt.Errorf("缺少可用于公开页面解析的群组标识")
		}

		var rateLimitErr *TelegramRateLimitError
		if errors.As(resolveErr, &rateLimitErr) {
			// The shared HTTP scheduler already opens a process-wide circuit until RetryAfter.
			// Do not sleep here: this resolver runs on the aggregation goroutine, and holding it
			// during a Telegram cooldown would stall unrelated aggregation work as well.
			log.Printf("[grouplib] Telegram 网页限流，暂停本轮解析队列 %s: %v", rateLimitErr.RetryAfter, rateLimitErr)
			return
		}
		if resolveErr != nil {
			errText := strings.ToLower(resolveErr.Error())
			if (isInviteLink && (strings.Contains(errText, "过期") || strings.Contains(errText, "无效"))) ||
				(strings.Contains(errText, "返回 404") || strings.Contains(errText, "返回 410")) {
				if err := e.repo.UpdateLinkValidityByLink(ctx, p.link, "invalid"); err != nil {
					log.Printf("[grouplib] 标记失效链接失败 %s: %v", p.link, err)
				}
			}
		} else if pageType == "user" {
			// Valid Telegram account/bot URLs are not valid group-library entries.
			if err := e.repo.UpdateLinkValidityByLink(ctx, p.link, "invalid"); err != nil {
				log.Printf("[grouplib] 标记非群组链接失效失败 %s: %v", p.link, err)
			}
		} else if title != "" || count > 0 {
			if pageType == "" || pageType == "unknown" {
				pageType = p.gtype
			}
			if err := e.repo.UpdateResolvedInfo(ctx, p.sgID, title, count, pageType, 0); err != nil {
				log.Printf("[grouplib] 回写群组资料失败 %s: %v", p.sgID, err)
			} else {
				if err := e.repo.UpdateLinkValidityByLink(ctx, p.link, "valid"); err != nil {
					log.Printf("[grouplib] 标记有效链接失败 %s: %v", p.link, err)
				}
			}
		}
	}

	log.Printf("[grouplib] 本轮解析补名 %d 条(仅使用公开页面解析)", len(pending))
}
