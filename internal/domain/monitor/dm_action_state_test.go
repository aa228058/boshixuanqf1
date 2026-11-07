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
			if e.claimDMAction(context.Background(), key) {
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
	textKey := e.dmActionKey(tenantID, accountID, ruleID, 123, -1001, 55, "text")
	contactKey := e.dmActionKey(tenantID, accountID, ruleID, 123, -1001, 55, "contact")
	if !e.claimDMAction(context.Background(), textKey) || !e.claimDMAction(context.Background(), contactKey) {
		t.Fatal("different actions should be independently claimable")
	}
	if e.claimDMAction(context.Background(), textKey) {
		t.Fatal("same action must not be claimed twice")
	}
	otherTenant := e.dmActionKey(uuid.New(), accountID, ruleID, 123, -1001, 55, "text")
	if !e.claimDMAction(context.Background(), otherTenant) {
		t.Fatal("another tenant must have an independent action key")
	}
}

func TestDMActionsAreScopedToTriggeringMessage(t *testing.T) {
	e := &Engine{dmActions: make(map[dmActionKey]dmActionStatus)}
	tenantID, accountID, ruleID := uuid.New(), uuid.New(), uuid.New()
	first := e.dmActionKey(tenantID, accountID, ruleID, 123, -1001, 55, "text")
	second := e.dmActionKey(tenantID, accountID, ruleID, 123, -1001, 56, "text")
	otherChat := e.dmActionKey(tenantID, accountID, ruleID, 123, -1002, 55, "text")
	if !e.claimDMAction(context.Background(), first) {
		t.Fatal("first triggering message should be claimable")
	}
	if e.claimDMAction(context.Background(), first) {
		t.Fatal("duplicate delivery of the same message must be suppressed")
	}
	if !e.claimDMAction(context.Background(), second) {
		t.Fatal("a later message from the same user should trigger independently")
	}
	if !e.claimDMAction(context.Background(), otherChat) {
		t.Fatal("the same message ID in another chat should be independent")
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
			if !e.claimDMAction(context.Background(), key) {
				t.Fatal("initial claim should succeed")
			}
			e.finishDMAction(context.Background(), key, status)
			if e.claimDMAction(context.Background(), key) {
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
	if !e.claimDMAction(context.Background(), key) {
		t.Fatal("initial claim should succeed")
	}
	e.finishDMAction(context.Background(), key, dmActionSucceeded)
	e.finishDMAction(context.Background(), key, dmActionOutcomeUnknown)
	if got := e.dmActions[key]; got != dmActionSucceeded {
		t.Fatalf("terminal state overwritten: got %v, want %v", got, dmActionSucceeded)
	}
}
