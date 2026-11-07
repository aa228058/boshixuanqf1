package monitor

import (
	"sync"
	"testing"

	"github.com/google/uuid"
)

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

func TestDMActionTerminalStatesAreNotRetried(t *testing.T) {
	for _, status := range []dmActionStatus{
		dmActionSucceeded,
		dmActionFailedDefinite,
		dmActionOutcomeUnknown,
	} {
		t.Run(map[dmActionStatus]string{
			dmActionSucceeded:      "succeeded",
			dmActionFailedDefinite: "failed_definite",
			dmActionOutcomeUnknown: "outcome_unknown",
		}[status], func(t *testing.T) {
			e := &Engine{dmActions: make(map[dmActionKey]dmActionStatus)}
			key := dmActionKey{tenantID: uuid.New(), accountID: uuid.New(), ruleID: uuid.New(), targetID: 123, action: "text"}
			if !e.claimDMAction(key) {
				t.Fatal("initial claim should succeed")
			}
			e.finishDMAction(key, status)
			if e.claimDMAction(key) {
				t.Fatalf("%v state must not be retried automatically", status)
			}
			if got := e.dmActions[key]; got != status {
				t.Fatalf("terminal state changed: got %v, want %v", got, status)
			}
		})
	}
}

func TestDMActionFinishDoesNotOverwriteTerminalState(t *testing.T) {
	e := &Engine{dmActions: make(map[dmActionKey]dmActionStatus)}
	key := dmActionKey{tenantID: uuid.New(), accountID: uuid.New(), ruleID: uuid.New(), targetID: 123, action: "contact"}
	if !e.claimDMAction(key) {
		t.Fatal("initial claim should succeed")
	}
	e.finishDMAction(key, dmActionSucceeded)
	e.finishDMAction(key, dmActionOutcomeUnknown)
	if got := e.dmActions[key]; got != dmActionSucceeded {
		t.Fatalf("terminal state overwritten: got %v, want %v", got, dmActionSucceeded)
	}
}
