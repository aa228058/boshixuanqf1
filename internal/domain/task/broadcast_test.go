package task

import (
	"context"
	"reflect"
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
