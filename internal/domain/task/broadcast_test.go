package task

import (
	"context"
	"testing"
	"time"
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

func TestWaitBroadcastDelayRespectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if waitBroadcastDelay(ctx, time.Second) {
		t.Fatal("expected cancelled delay to stop")
	}
}

func TestWaitBroadcastDelayCompletes(t *testing.T) {
	if !waitBroadcastDelay(context.Background(), time.Millisecond) {
		t.Fatal("expected delay to complete")
	}
}

func TestWaitBroadcastDelayZeroDuration(t *testing.T) {
	if !waitBroadcastDelay(context.Background(), 0) {
		t.Fatal("expected zero delay to complete")
	}
}
