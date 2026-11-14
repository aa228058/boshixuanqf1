package account

import (
	"context"
	"log"
	"regexp"
	"strconv"

	"tg-sender/internal/domain"

	"github.com/google/uuid"
)

var floodWaitRe = regexp.MustCompile(`(?i)flood_wait_(\d+)`)

func (e *Engine) recordRiskEvent(ctx context.Context, tenantID, accountID uuid.UUID, eventType, severity, detail string) {
	if e.riskRepo == nil {
		log.Printf("[RISK] %s | 租户=%s 账号=%s | %s\n", eventType, tenantID, accountID, detail)
		return
	}
	if err := e.riskRepo.Create(ctx, &domain.RiskEvent{
		EventID:   uuid.New(), // 不设则零值UUID, 第二次插入必撞主键
		TenantID:  tenantID,
		AccountID: accountID,
		EventType: eventType,
		Severity:  severity,
		Details:   map[string]any{"detail": detail},
	}); err != nil {
		log.Printf("[RISK] 记录账号 %s 的事件 %s 失败: %v\n", eventType, accountID, err)
	}
}

func extractFloodSeconds(errStr string) int {
	// P2-55 核实: floodWaitRe 本身带 (?i) 不区分大小写, 此函数无大小写漏洞。
	matches := floodWaitRe.FindStringSubmatch(errStr)
	if len(matches) > 1 {
		if sec, err := strconv.Atoi(matches[1]); err == nil {
			return sec
		}
	}
	return 0
}
