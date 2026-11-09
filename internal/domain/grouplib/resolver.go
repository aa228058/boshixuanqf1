package grouplib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

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
	var pending []row
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.sgID, &item.link, &item.username, &item.gtype, &item.title); err == nil {
			pending = append(pending, item)
		}
	}
	rows.Close()
	if len(pending) == 0 {
		return
	}

	for _, p := range pending {
		isInviteLink := inviteHashFromLink(p.link) != ""
		// Defensive cleanup for bot links inserted by older import/aggregation paths.
		if !isInviteLink && strings.HasSuffix(strings.ToLower(strings.TrimPrefix(p.username, "@")), "bot") {
			if err := e.repo.UpdateLinkValidityByLink(ctx, p.link, "invalid"); err != nil {
				log.Printf("[grouplib] 标记机器人链接无效失败 %s: %v", p.link, err)
			} else {
				e.finishResolution(ctx, p.sgID, "bot link")
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
			// Move this row out of the hot pending set before pausing. Otherwise the
			// same first row is selected on every scheduler pass and starves the tail.
			e.deferResolutionRetry(ctx, p.sgID, "Telegram rate limit")
			log.Printf("[grouplib] Telegram 网页限流，暂停本轮解析队列 %s: %v", rateLimitErr.RetryAfter, rateLimitErr)
			timer := time.NewTimer(rateLimitErr.RetryAfter)
			select {
			case <-ctx.Done():
				timer.Stop()
			case <-timer.C:
			}
			return
		}
		if resolveErr != nil {
			permanent := isPermanentResolutionError(resolveErr, isInviteLink)
			if permanent {
				if err := e.repo.UpdateLinkValidityByLink(ctx, p.link, "invalid"); err != nil {
					log.Printf("[grouplib] 标记永久失效链接失败 %s: %v", p.link, err)
				} else {
					e.finishResolution(ctx, p.sgID, "permanently invalid link")
				}
			} else {
				e.deferResolutionRetry(ctx, p.sgID, "resolution failed")
			}
		} else if pageType == "user" {
			// Valid Telegram account/bot URLs are not valid group-library entries.
			// Do not mark the candidate resolved if link validity persistence failed;
			// otherwise the failed write would be hidden from every future retry.
			if err := e.repo.UpdateLinkValidityByLink(ctx, p.link, "invalid"); err != nil {
				log.Printf("[grouplib] 标记非群组链接无效失败 %s: %v", p.link, err)
				e.deferResolutionRetry(ctx, p.sgID, "non-group link validity persistence failed")
			} else {
				e.finishResolution(ctx, p.sgID, "not a group")
			}
		} else if title != "" || count > 0 {
			if pageType == "" || pageType == "unknown" {
				pageType = p.gtype
			}
			if err := e.repo.UpdateResolvedInfo(ctx, p.sgID, title, count, pageType, 0); err != nil {
				log.Printf("[grouplib] 回写群组资料失败 %s: %v", p.sgID, err)
				e.deferResolutionRetry(ctx, p.sgID, "metadata persistence failed")
			} else if err := e.repo.UpdateLinkValidityByLink(ctx, p.link, "valid"); err != nil {
				log.Printf("[grouplib] 标记有效链接失败 %s: %v", p.link, err)
			}
		}
	}

	log.Printf("[grouplib] 本轮解析补名 %d 条(仅使用公开页面解析)", len(pending))
}

// deferResolutionRetry moves a failed candidate out of the hot pending queue.
// identity_pending rows become eligible again after the existing 30-minute
// cooldown, allowing later rows to make progress instead of being starved by
// the same first resolveBatchSize failures on every aggregation cycle.
func (e *Engine) deferResolutionRetry(ctx context.Context, sgID uuid.UUID, reason string) {
	tag, err := e.pool.Exec(ctx, `
		UPDATE system_groups
		SET resolution_status='identity_pending', last_seen_at=NOW()
		WHERE sg_id=$1 AND status='active'
	`, sgID)
	if err != nil {
		log.Printf("[grouplib] 延后群组解析重试失败 %s (%s): %v", sgID, reason, err)
		return
	}
	if tag.RowsAffected() != 1 {
		log.Printf("[grouplib] 延后群组解析未更新记录 %s (%s)", sgID, reason)
	}
}

// finishResolution removes permanently invalid candidates from the retry queue.
func (e *Engine) finishResolution(ctx context.Context, sgID uuid.UUID, reason string) {
	tag, err := e.pool.Exec(ctx, `
		UPDATE system_groups SET resolution_status='resolved'
		WHERE sg_id=$1 AND status='active'
	`, sgID)
	if err != nil {
		log.Printf("[grouplib] 结束群组解析失败 %s (%s): %v", sgID, reason, err)
		return
	}
	if tag.RowsAffected() != 1 {
		log.Printf("[grouplib] 结束群组解析未更新记录 %s (%s)", sgID, reason)
	}
}

// isPermanentResolutionError separates terminal link failures from transient
// network/server errors, which must remain eligible for delayed retry.
func isPermanentResolutionError(err error, inviteLink bool) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "返回 404") || strings.Contains(message, "返回 410") {
		return true
	}
	return inviteLink && (strings.Contains(message, "过期") || strings.Contains(message, "无效"))
}
