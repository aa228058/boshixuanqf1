package monitor

import (
    "context"
    "github.com/google/uuid"
)

// CallCooldownStore atomically reserves the ten-minute cooldown for a callee.
type CallCooldownStore interface {
    Claim(ctx context.Context, tenantID, accountID uuid.UUID, calleeID int64) (bool, error)
}

// SetCallCooldownStore configures the durable voice-call cooldown ledger.
func (e *Engine) SetCallCooldownStore(store CallCooldownStore) {
    e.callCooldownStore = store
}
