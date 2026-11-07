//go:build integration

package repository

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
)

func TestDMActionClaimIsDurableAndAtomic(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	repo := NewDMActionRepository(pool)
	tenantID, accountID, ruleID := uuid.New(), uuid.New(), uuid.New()

	var winners atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, err := repo.Claim(ctx, tenantID, accountID, ruleID, 12345, -10001, 77, "text")
			if err != nil {
				t.Errorf("claim: %v", err)
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

	// A fresh repository instance models process restart; the database remains authoritative.
	restartedRepo := NewDMActionRepository(pool)
	if claimed, err := restartedRepo.Claim(ctx, tenantID, accountID, ruleID, 12345, -10001, 77, "text"); err != nil || claimed {
		t.Fatalf("claim after restart = (%t, %v), want (false, nil)", claimed, err)
	}

	// Same triggering message under another tenant remains isolated.
	if claimed, err := restartedRepo.Claim(ctx, uuid.New(), accountID, ruleID, 12345, -10001, 77, "text"); err != nil || !claimed {
		t.Fatalf("cross-tenant claim = (%t, %v), want (true, nil)", claimed, err)
	}

	if err := repo.Finish(ctx, tenantID, accountID, ruleID, 12345, -10001, 77, "text", "outcome_unknown", "telegram response timed out"); err != nil {
		t.Fatalf("finish outcome_unknown: %v", err)
	}
	// Terminal state is immutable: another finish cannot overwrite it.
	if err := repo.Finish(ctx, tenantID, accountID, ruleID, 12345, -10001, 77, "text", "succeeded", ""); err == nil {
		t.Fatal("second finish should not update a terminal action")
	}
	var status, lastError string
	if err := pool.QueryRow(ctx, `
		SELECT status, last_error FROM monitor_dm_actions
		WHERE tenant_id=$1 AND account_id=$2 AND rule_id=$3
		  AND target_id=12345 AND source_chat_id=-10001 AND source_message_id=77 AND action='text'
	`, tenantID, accountID, ruleID).Scan(&status, &lastError); err != nil {
		t.Fatalf("read persisted action: %v", err)
	}
	if status != "outcome_unknown" || lastError != "telegram response timed out" {
		t.Fatalf("persisted state = %q / %q", status, lastError)
	}
}
