package monitor

import (
	"github.com/google/uuid"
)

type dmActionKey struct {
	tenantID        uuid.UUID
	accountID       uuid.UUID
	ruleID          uuid.UUID
	targetID        int64
	sourceChatID    int64
	sourceMessageID int
	action          string
}

type dmActionStatus uint8

const (
	dmActionInProgress dmActionStatus = iota + 1
	dmActionSucceeded
	dmActionFailedDefinite
	dmActionOutcomeUnknown
)

// claimDMAction atomically grants one execution attempt for an individual DM
// action. Terminal states are intentionally not retried automatically: a
// transport error may mean Telegram accepted the request but its response was
// lost.
func (e *Engine) claimDMAction(ctx context.Context, key dmActionKey) bool {\n\tif e.dmActionStore != nil {\n\t\tclaimed, err := e.dmActionStore.Claim(ctx, key.tenantID, key.accountID, key.ruleID, key.targetID, key.sourceChatID, key.sourceMessageID, key.action)\n\t\tif err != nil {\n\t\t\tlog.Printf("[monitor] DM 动作持久化领取失败，按 fail-closed 跳过: %v", err)\n\t\t\treturn false\n\t\t}\n\t\treturn claimed\n\t}
	e.dmActionMu.Lock()
	defer e.dmActionMu.Unlock()
	if e.dmActions == nil {
		e.dmActions = make(map[dmActionKey]dmActionStatus)
	}
	if _, exists := e.dmActions[key]; exists {
		return false
	}
	// Fail closed at the cap: evicting old keys could resend an action that
	// was already delivered, while an unbounded map would leak memory.
	if len(e.dmActions) >= 100000 {
		return false
	}
	e.dmActions[key] = dmActionInProgress
	return true
}

func (e *Engine) finishDMAction(ctx context.Context, key dmActionKey, status dmActionStatus) {\n\tif e.dmActionStore != nil {\n\t\tif err := e.dmActionStore.Finish(ctx, key.tenantID, key.accountID, key.ruleID, key.targetID, key.sourceChatID, key.sourceMessageID, key.action, dmActionStatusName(status), ""); err != nil {\n\t\t\tlog.Printf("[monitor] DM 动作持久化完成状态失败: %v", err)\n\t\t}\n\t\treturn\n\t}
	e.dmActionMu.Lock()
	defer e.dmActionMu.Unlock()
	if e.dmActions == nil {
		e.dmActions = make(map[dmActionKey]dmActionStatus)
	}
	if e.dmActions[key] == dmActionInProgress {
		e.dmActions[key] = status
	}
}

func (e *Engine) dmActionKey(tenantID, accountID, ruleID uuid.UUID, targetID, sourceChatID int64, sourceMessageID int, action string) dmActionKey {
	return dmActionKey{
		tenantID: tenantID, accountID: accountID, ruleID: ruleID,
		targetID: targetID, sourceChatID: sourceChatID, sourceMessageID: sourceMessageID,
		action: action,
	}
}

func dmActionStatusName(status dmActionStatus) string {
	switch status {
	case dmActionSucceeded:
		return "succeeded"
	case dmActionFailedDefinite:
		return "failed_definite"
	case dmActionOutcomeUnknown:
		return "outcome_unknown"
	default:
		return "in_progress"
	}
}
