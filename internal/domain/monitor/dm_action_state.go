package monitor

import (
	"context"
	"log"

	"github.com/google/uuid"
)

// DMActionStore is the durable claim ledger. Claim must be atomic across processes.
type DMActionStore interface {
	Claim(ctx context.Context, tenantID, accountID, ruleID uuid.UUID, targetID, sourceChatID int64, sourceMessageID int, action string) (bool, error)
	Finish(ctx context.Context, tenantID, accountID, ruleID uuid.UUID, targetID, sourceChatID int64, sourceMessageID int, action, status, lastError string) error
}

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

// claimDMAction fails closed when the durable ledger is unavailable.
func (e *Engine) claimDMAction(ctx context.Context, key dmActionKey) bool {
	if e.dmActionStore != nil {
		claimed, err := e.dmActionStore.Claim(ctx, key.tenantID, key.accountID, key.ruleID, key.targetID, key.sourceChatID, key.sourceMessageID, key.action)
		if err != nil {
			log.Printf("[monitor] DM action claim failed; skipping safely: %v", err)
			return false
		}
		return claimed
	}
	// In-memory fallback keeps isolated unit tests and explicitly unconfigured
	// Engine instances usable. Production wiring always injects the DB store.
	e.dmActionMu.Lock()
	defer e.dmActionMu.Unlock()
	if e.dmActions == nil {
		e.dmActions = make(map[dmActionKey]dmActionStatus)
	}
	if _, exists := e.dmActions[key]; exists {
		return false
	}
	if len(e.dmActions) >= 100000 {
		return false
	}
	e.dmActions[key] = dmActionInProgress
	return true
}

func (e *Engine) finishDMAction(ctx context.Context, key dmActionKey, status dmActionStatus) {
	if e.dmActionStore != nil {
		if err := e.dmActionStore.Finish(ctx, key.tenantID, key.accountID, key.ruleID, key.targetID, key.sourceChatID, key.sourceMessageID, key.action, dmActionStatusName(status), ""); err != nil {
			log.Printf("[monitor] DM action finish persistence failed: %v", err)
		}
		return
	}
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
