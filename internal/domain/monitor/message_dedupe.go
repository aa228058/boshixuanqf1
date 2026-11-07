package monitor

import (
	"time"

	"github.com/google/uuid"
)

const (
	messageDedupTTL        = 10 * time.Minute
	messageDedupMaxEntries = 100000
	messageDedupSweepEvery = 256
)

type messageDedupKey struct {
	tenantID  uuid.UUID
	accountID uuid.UUID
	chatID    int64
	messageID int
}

// markMessageSeen returns false when this account has already delivered the
// same Telegram message recently. The key is scoped by tenant, account, chat,
// and message so independent accounts never suppress each other's updates.
func (e *Engine) markMessageSeen(tenantID, accountID uuid.UUID, chatID int64, messageID int) bool {
	now := time.Now()
	key := messageDedupKey{
		tenantID:  tenantID,
		accountID: accountID,
		chatID:    chatID,
		messageID: messageID,
	}

	e.messageDedupMu.Lock()
	defer e.messageDedupMu.Unlock()

	if seenAt, exists := e.seenMessages[key]; exists && now.Sub(seenAt) < messageDedupTTL {
		return false
	}

	e.seenMessages[key] = now
	e.messageDedupOps++

	if e.messageDedupOps%messageDedupSweepEvery == 0 || len(e.seenMessages) > messageDedupMaxEntries {
		cutoff := now.Add(-messageDedupTTL)
		for existingKey, seenAt := range e.seenMessages {
			if seenAt.Before(cutoff) {
				delete(e.seenMessages, existingKey)
			}
		}
	}

	// Keep memory bounded even during a burst of unique messages. Evict the
	// oldest entry only when the hard cap is exceeded.
	if len(e.seenMessages) > messageDedupMaxEntries {
		var oldestKey messageDedupKey
		var oldestAt time.Time
		for existingKey, seenAt := range e.seenMessages {
			if oldestAt.IsZero() || seenAt.Before(oldestAt) {
				oldestKey = existingKey
				oldestAt = seenAt
			}
		}
		delete(e.seenMessages, oldestKey)
	}

	return true
}
