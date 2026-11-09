package task

import (
	"context"
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

func TestReconcileBroadcastTimeoutProgress(t *testing.T) {
	tests := []struct {
		name                 string
		total, done, failed  int
		timedOut             bool
		wantFailed           int
		wantTimedOut         bool
	}{
		{name: "timeout accounts for unprocessed targets", total: 10, done: 4, failed: 2, timedOut: true, wantFailed: 6, wantTimedOut: true},
		{name: "timeout preserves fully reconciled counters", total: 10, done: 10, failed: 0, timedOut: true, wantFailed: 0, wantTimedOut: true},
		{name: "normal completion leaves counters unchanged", total: 10, done: 8, failed: 2, timedOut: false, wantFailed: 2, wantTimedOut: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFailed, gotTimedOut := reconcileBroadcastTimeoutProgress(tt.total, tt.done, tt.failed, tt.timedOut)
			if gotFailed != tt.wantFailed || gotTimedOut != tt.wantTimedOut {
				t.Fatalf("got failed=%d timedOut=%v, want failed=%d timedOut=%v", gotFailed, gotTimedOut, tt.wantFailed, tt.wantTimedOut)
			}
			if tt.done+gotFailed != tt.total {
				t.Fatalf("reconciled counters done+failed=%d, want total=%d", tt.done+gotFailed, tt.total)
			}
		})
	}
}
