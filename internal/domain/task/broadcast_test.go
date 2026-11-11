package task

import (
	"context"
	"testing"

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

func TestDedupeTargetsPerAccountPreservesCrossAccountAssignments(t *testing.T) {
	accountA := uuid.New()
	accountB := uuid.New()
	got := dedupeTargetsPerAccount(map[uuid.UUID][]string{
		accountA: {"group-1", "group-1", "group-2"},
		accountB: {"group-1"},
	})
	if len(got[accountA]) != 2 || got[accountA][0] != "group-1" || got[accountA][1] != "group-2" {
		t.Fatalf("account A targets = %#v, want [group-1 group-2]", got[accountA])
	}
	if len(got[accountB]) != 1 || got[accountB][0] != "group-1" {
		t.Fatalf("account B targets = %#v, want [group-1]", got[accountB])
	}
}
