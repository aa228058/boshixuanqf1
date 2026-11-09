package task

import (
	"context"
	"reflect"

	"github.com/google/uuid"
	"testing"
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

func TestDedupeAccountTargetsPreservesCrossAccountAssignments(t *testing.T) {
	first := uuid.New()
	second := uuid.New()
	got := dedupeAccountTargets(map[uuid.UUID][]string{
		first:  {"-1001", "-1001", "-1002"},
		second: {"-1001", "-1003"},
	})

	if want := []string{"-1001", "-1002"}; !reflect.DeepEqual(got[first], want) {
		t.Fatalf("first account targets=%v, want %v", got[first], want)
	}
	if want := []string{"-1001", "-1003"}; !reflect.DeepEqual(got[second], want) {
		t.Fatalf("second account targets=%v, want %v", got[second], want)
	}
}
