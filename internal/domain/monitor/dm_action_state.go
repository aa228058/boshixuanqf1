package monitor

import (
	"sync"
	"testing"

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

func TestDMActionClaimIsAtomic(t *testing.T) {
	e := &Engine{dmActions: make(map[dmActionKey]dmActionStatus)}
	key := dmActionKey{tenantID: uuid.New(), accountID: uuid.New(), ruleID: uuid.New(), targetID: 123, action: "text"}
	var wg sync.WaitGroup
	var mu sync.Mutex
	claims := 0
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e.claimDMAction(key) {
				mu.Lock()
				claims++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if claims != 1 {
		t.Fatalf("expected exactly one claim, got %d", claims)
	}
}

func TestDMActionsAreIndependentAndScoped(t *testing.T) {
	e := &Engine{dmActions: make(map[dmActionKey]dmActionStatus)}
	tenantID, accountID, ruleID := uuid.New(), uuid.New(), uuid.New()
	textKey := e.dmActionKey(tenantID, accountID, ruleID, 123, "text")
	contactKey := e.dmActionKey(tenantID, accountID, ruleID, 123, "contact")
	if !e.claimDMAction(textKey) || !e.claimDMAction(contactKey) {
		t.Fatal("different actions should be independently claimable")
	}
	if e.claimDMAction(textKey) {
		t.Fatal("same action must not be claimed twice")
	}
	otherTenant := e.dmActionKey(uuid.New(), accountID, ruleID, 123, "text")
	if !e.claimDMAction(otherTenant) {
		t.Fatal("another tenant must have an independent action key")
	}
}

func TestDMActionTerminalStateIsNotRetried(t *testing.T) {
	e := &Engine{dmActions: make(map[dmActionKey]dmActionStatus)}
	key := dmActionKey{tenantID: uuid.New(), accountID: uuid.New(), ruleID: uuid.New(), targetID: 123, action: "text"}
	if !e.claimDMAction(key) {
		t.Fatal("initial claim should succeed")
	}
	e.finishDMAction(key, dmActionOutcomeUnknown)
	if e.claimDMAction(key) {
		t.Fatal("unknown outcome must not be retried automatically")
	}
}
