//go:build integration

package repository

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCallCooldownIsDurableAtomicAndScoped(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	repo := NewCallCooldownRepository(pool)
	tenantID, accountID := uuid.New(), uuid.New()
	const calleeID int64 = 99887766

	var winners atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, err := repo.Claim(ctx, tenantID, accountID, calleeID)
			if err != nil {
				t.Errorf("concurrent cooldown claim: %v", err)
				return
			}
			if claimed {
				winners.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := winners.Load(); got != 1 {
		t.Fatalf("concurrent claims = %d, want exactly one", got)
	}

	// A new repository instance models an application restart.
	restarted := NewCallCooldownRepository(pool)
	if claimed, err := restarted.Claim(ctx, tenantID, accountID, calleeID); err != nil || claimed {
		t.Fatalf("claim after restart = (%t, %v), want (false, nil)", claimed, err)
	}

	// Cooldowns are isolated by tenant, monitoring account and callee.
	if claimed, err := restarted.Claim(ctx, uuid.New(), accountID, calleeID); err != nil || !claimed {
		t.Fatalf("other tenant claim = (%t, %v), want (true, nil)", claimed, err)
	}
	if claimed, err := restarted.Claim(ctx, tenantID, uuid.New(), calleeID); err != nil || !claimed {
		t.Fatalf("other account claim = (%t, %v), want (true, nil)", claimed, err)
	}
	if claimed, err := restarted.Claim(ctx, tenantID, accountID, calleeID+1); err != nil || !claimed {
		t.Fatalf("other callee claim = (%t, %v), want (true, nil)", claimed, err)
	}

	// After the ten-minute window expires, the same key can reserve a new call.
	if _, err := pool.Exec(ctx, `
		UPDATE monitor_call_cooldowns
		SET last_called_at = NOW() - INTERVAL '11 minutes'
		WHERE tenant_id=$1 AND account_id=$2 AND callee_id=$3
	`, tenantID, accountID, calleeID); err != nil {
		t.Fatalf("expire cooldown: %v", err)
	}
	if claimed, err := restarted.Claim(ctx, tenantID, accountID, calleeID); err != nil || !claimed {
		t.Fatalf("claim after expiry = (%t, %v), want (true, nil)", claimed, err)
	}

	var lastCalled time.Time
	if err := pool.QueryRow(ctx, `
		SELECT last_called_at FROM monitor_call_cooldowns
		WHERE tenant_id=$1 AND account_id=$2 AND callee_id=$3
	`, tenantID, accountID, calleeID).Scan(&lastCalled); err != nil {
		t.Fatalf("read persisted cooldown: %v", err)
	}
	if time.Since(lastCalled) > time.Minute {
		t.Fatalf("last_called_at was not refreshed: %s", lastCalled)
	}
}
