package task

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAcquireBroadcastSlotRespectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	semaphore := make(chan struct{}, 1)

	if acquireBroadcastSlot(ctx, semaphore) {
		t.Fatal("expected cancelled context not to acquire a slot")
	}
	if got := len(semaphore); got != 0 {
		t.Fatalf("semaphore length=%d, want 0", got)
	}
}

func TestAcquireBroadcastSlotAcquiresAvailableSlot(t *testing.T) {
	semaphore := make(chan struct{}, 1)
	if !acquireBroadcastSlot(context.Background(), semaphore) {
		t.Fatal("expected available slot to be acquired")
	}
	if got := len(semaphore); got != 1 {
		t.Fatalf("semaphore length=%d, want 1", got)
	}
}

func TestDedupeBroadcastTargetsPreservesTargetsAcrossAccounts(t *testing.T) {
	accountA := uuid.New()
	accountB := uuid.New()
	targets := map[uuid.UUID][]string{
		accountA: {"-1001", "-1002", "-1001"},
		accountB: {"-1001", "-1003"},
	}

	if got := dedupeBroadcastTargets(targets); got != 4 {
		t.Fatalf("deduplicated total=%d, want 4 account-target sends", got)
	}
	if got := len(targets[accountA]); got != 2 {
		t.Fatalf("account A targets=%d, want 2", got)
	}
	if got := len(targets[accountB]); got != 2 {
		t.Fatalf("account B targets=%d, want 2; shared target must remain assigned", got)
	}
	if targets[accountA][0] != "-1001" || targets[accountB][0] != "-1001" {
		t.Fatal("shared target was not preserved for both accounts")
	}
}

func TestDedupeBroadcastTargetsEmpty(t *testing.T) {
	if got := dedupeBroadcastTargets(map[uuid.UUID][]string{}); got != 0 {
		t.Fatalf("empty target total=%d, want 0", got)
	}
}

func TestCountBroadcastTargetProgress(t *testing.T) {
	accountID := uuid.New()
	targets := []string{"one", "two", "three"}
	successes := map[string]bool{
		accountID.String() + "|one": true,
		accountID.String() + "|three": true,
	}
	done, pending := countBroadcastTargetProgress(accountID, targets, successes)
	if done != 2 || pending != 1 {
		t.Fatalf("got done=%d pending=%d, want done=2 pending=1", done, pending)
	}
	if done+pending != len(targets) {
		t.Fatalf("progress does not reconcile: done+pending=%d total=%d", done+pending, len(targets))
	}
}
