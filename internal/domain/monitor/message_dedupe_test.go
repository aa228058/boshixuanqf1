package monitor

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMarkMessageSeenDeduplicatesWithinAccount(t *testing.T) {
	engine := &Engine{seenMessages: make(map[messageDedupKey]time.Time)}
	tenantID := uuid.New()
	accountID := uuid.New()

	if !engine.markMessageSeen(tenantID, accountID, -1001, 42) {
		t.Fatal("first delivery should be accepted")
	}
	if engine.markMessageSeen(tenantID, accountID, -1001, 42) {
		t.Fatal("duplicate delivery should be rejected")
	}
}

func TestMarkMessageSeenScopesByTenantAndAccount(t *testing.T) {
	engine := &Engine{seenMessages: make(map[messageDedupKey]time.Time)}
	tenantID := uuid.New()
	accountA := uuid.New()
	accountB := uuid.New()

	if !engine.markMessageSeen(tenantID, accountA, -1001, 42) {
		t.Fatal("first account delivery should be accepted")
	}
	if !engine.markMessageSeen(tenantID, accountB, -1001, 42) {
		t.Fatal("another account must not be suppressed")
	}
	if !engine.markMessageSeen(uuid.New(), accountA, -1001, 42) {
		t.Fatal("another tenant must not be suppressed")
	}
}

func TestMarkMessageSeenExpiresOldEntries(t *testing.T) {
	engine := &Engine{seenMessages: make(map[messageDedupKey]time.Time)}
	tenantID := uuid.New()
	accountID := uuid.New()
	key := messageDedupKey{
		tenantID:  tenantID,
		accountID: accountID,
		chatID:    -1001,
		messageID: 42,
	}
	engine.seenMessages[key] = time.Now().Add(-messageDedupTTL - time.Second)

	if !engine.markMessageSeen(tenantID, accountID, -1001, 42) {
		t.Fatal("expired message key should be accepted again")
	}
}
