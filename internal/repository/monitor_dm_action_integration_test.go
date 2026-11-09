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

	// A process crash leaves a durable in_progress claim. Startup recovery marks
	// it outcome_unknown rather than risking a duplicate external side effect.
	if claimed, err := repo.Claim(ctx, tenantID, accountID, ruleID, 22345, -10001, 78, "contact"); err != nil || !claimed {
		t.Fatalf("create stale claim = (%t, %v), want (true, nil)", claimed, err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE monitor_dm_actions SET updated_at=NOW()-INTERVAL '6 minutes'
		WHERE tenant_id=$1 AND account_id=$2 AND rule_id=$3
		  AND target_id=22345 AND source_chat_id=-10001 AND source_message_id=78 AND action='contact'
	`, tenantID, accountID, ruleID); err != nil {
		t.Fatalf("age stale claim: %v", err)
	}
	resolved, err := repo.ResolveStale(ctx, 5*time.Minute)
	if err != nil {
		t.Fatalf("resolve stale claims: %v", err)
	}
	if resolved != 1 {
		t.Fatalf("resolved stale claims = %d, want 1", resolved)
	}
	// A fresh in_progress action must not be swept.
	if claimed, err := repo.Claim(ctx, tenantID, accountID, ruleID, 42345, -10001, 80, "text"); err != nil || !claimed {
		t.Fatalf("create fresh claim = (%t, %v), want (true, nil)", claimed, err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT status FROM monitor_dm_actions
		WHERE tenant_id=$1 AND account_id=$2 AND rule_id=$3
		  AND target_id=42345 AND source_chat_id=-10001 AND source_message_id=80 AND action='text'
	`, tenantID, accountID, ruleID).Scan(&status); err != nil {
		t.Fatalf("read fresh action: %v", err)
	}
	if status != "in_progress" {
		t.Fatalf("fresh action state = %q, want in_progress", status)
	}
	if claimed, err := repo.Claim(ctx, tenantID, accountID, ruleID, 22345, -10001, 78, "contact"); err != nil || claimed {
		t.Fatalf("stale claim retry = (%t, %v), want (false, nil)", claimed, err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT status, last_error FROM monitor_dm_actions
		WHERE tenant_id=$1 AND account_id=$2 AND rule_id=$3
		  AND target_id=22345 AND source_chat_id=-10001 AND source_message_id=78 AND action='contact'
	`, tenantID, accountID, ruleID).Scan(&status, &lastError); err != nil {
		t.Fatalf("read recovered action: %v", err)
	}
	if status != "outcome_unknown" || lastError == "" {
		t.Fatalf("recovered state = %q / %q, want outcome_unknown with explanation", status, lastError)
	}

	// Duplicate delivery can also close one exact stale claim without reclaiming it.
	if claimed, err := repo.Claim(ctx, tenantID, accountID, ruleID, 32345, -10001, 79, "call"); err != nil || !claimed {
		t.Fatalf("create stale call claim = (%t, %v), want (true, nil)", claimed, err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE monitor_dm_actions SET updated_at=NOW()-INTERVAL '6 minutes'
		WHERE tenant_id=$1 AND account_id=$2 AND rule_id=$3
		  AND target_id=32345 AND source_chat_id=-10001 AND source_message_id=79 AND action='call'
	`, tenantID, accountID, ruleID); err != nil {
		t.Fatalf("age stale call claim: %v", err)
	}
	if claimed, err := repo.Claim(ctx, tenantID, accountID, ruleID, 32345, -10001, 79, "call"); err != nil || claimed {
		t.Fatalf("duplicate stale call claim = (%t, %v), want (false, nil)", claimed, err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT status FROM monitor_dm_actions
		WHERE tenant_id=$1 AND account_id=$2 AND rule_id=$3
		  AND target_id=32345 AND source_chat_id=-10001 AND source_message_id=79 AND action='call'
	`, tenantID, accountID, ruleID).Scan(&status); err != nil {
		t.Fatalf("read stale call state: %v", err)
	}
	if status != "outcome_unknown" {
		t.Fatalf("duplicate delivery changed stale call state to %q", status)
	}
}
