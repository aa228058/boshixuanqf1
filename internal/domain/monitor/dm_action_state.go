package monitor

import (
	"github.com/google/uuid"
)

type dmActionKey struct {
	tenantID  uuid.UUID
	accountID uuid.UUID
	ruleID    uuid.UUID
	targetID  int64
	action    string
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
func (e *Engine) claimDMAction(key dmActionKey) bool {
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

func (e *Engine) finishDMAction(key dmActionKey, status dmActionStatus) {
	e.dmActionMu.Lock()
	defer e.dmActionMu.Unlock()
	if e.dmActions == nil {
		e.dmActions = make(map[dmActionKey]dmActionStatus)
	}
	if e.dmActions[key] == dmActionInProgress {
		e.dmActions[key] = status
	}
}

func (e *Engine) dmActionKey(tenantID, accountID uuid.UUID, ruleID uuid.UUID, targetID int64, action string) dmActionKey {
	return dmActionKey{tenantID: tenantID, accountID: accountID, ruleID: ruleID, targetID: targetID, action: action}
}

