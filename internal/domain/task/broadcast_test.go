package task

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestShouldSendBroadcastTarget(t *testing.T) {
	tests := []struct {
		name   string
		status string
		chatID int64
		want   bool
	}{
		{name: "valid target", status: "success", chatID: -1001, want: true},
		{name: "account already restricted", status: "failed", chatID: -1001, want: false},
		{name: "peer resolution failed", status: "failed", chatID: 0, want: false},
		{name: "missing chat ID", status: "success", chatID: 0, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldSendBroadcastTarget(tt.status, tt.chatID); got != tt.want {
				t.Fatalf("shouldSendBroadcastTarget(%q, %d) = %v, want %v", tt.status, tt.chatID, got, tt.want)
			}
		})
	}
}

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

func TestParseBroadcastMappingsMergesRepeatedAccountEntries(t *testing.T) {
	accountID := uuid.New()
	got, err := parseBroadcastMappings([]any{
		map[string]any{"account_id": accountID.String(), "targets": []any{"-1001", "-1001"}},
		map[string]any{"account_id": accountID.String(), "targets": []any{"-1002"}},
	}, []uuid.UUID{accountID})
	if err != nil {
		t.Fatalf("parseBroadcastMappings returned error: %v", err)
	}
	if want := []string{"-1001", "-1002"}; !reflect.DeepEqual(got[accountID], want) {
		t.Fatalf("targets=%v, want %v", got[accountID], want)
	}
}

func TestParseBroadcastMappingsRejectsInvalidInput(t *testing.T) {
	selected := uuid.New()
	unselected := uuid.New()
	tests := []struct {
		name string
		raw  any
	}{
		{name: "missing mappings", raw: nil},
		{name: "wrong mapping entry", raw: []any{"bad"}},
		{name: "invalid account", raw: []any{map[string]any{"account_id": "not-a-uuid", "targets": []any{"-1001"}}}},
		{name: "unselected account", raw: []any{map[string]any{"account_id": unselected.String(), "targets": []any{"-1001"}}}},
		{name: "empty targets", raw: []any{map[string]any{"account_id": selected.String(), "targets": []any{}}}},
		{name: "empty target", raw: []any{map[string]any{"account_id": selected.String(), "targets": []any{" "}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseBroadcastMappings(tt.raw, []uuid.UUID{selected}); err == nil {
				t.Fatal("expected invalid mapping to be rejected")
			}
		})
	}
}

func TestDedupeAccountIDsRemovesDuplicatesAndNil(t *testing.T) {
	first := uuid.New()
	second := uuid.New()
	got := dedupeAccountIDs([]uuid.UUID{first, first, uuid.Nil, second, second})
	if want := []uuid.UUID{first, second}; !reflect.DeepEqual(got, want) {
		t.Fatalf("account IDs=%v, want %v", got, want)
	}
}

func TestDedupeAccountTargetsTrimsWhitespace(t *testing.T) {
	accountID := uuid.New()
	got := dedupeAccountTargets(map[uuid.UUID][]string{
		accountID: {" -1001 ", "-1001", "   ", "-1002"},
	})
	if want := []string{"-1001", "-1002"}; !reflect.DeepEqual(got[accountID], want) {
		t.Fatalf("targets=%v, want %v", got[accountID], want)
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

func TestWaitBroadcastDelayCanBeCancelledWhileWaiting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(10*time.Millisecond, cancel)
	defer timer.Stop()
	if waitBroadcastDelay(ctx, time.Second) {
		t.Fatal("expected in-flight delay to be cancelled")
	}
}

func TestCountBroadcastTargetProgress(t *testing.T) {
	accountID := uuid.New()
	targets := []string{"one", "two", "three"}
	successes := map[string]bool{
		accountID.String() + "|one":   true,
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

func TestReconcileBroadcastTimeoutProgress(t *testing.T) {
	tests := []struct {
		name                string
		total, done, failed int
		timedOut            bool
		wantFailed          int
		wantTimedOut        bool
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
