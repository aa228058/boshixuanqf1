package repository

import (
	"testing"

	"github.com/google/uuid"
	"tg-sender/internal/domain"
)

func TestCopyFromLogsDoesNotConsumeFallbackBatch(t *testing.T) {
	first := &domain.TaskLog{LogID: uuid.New()}
	second := &domain.TaskLog{LogID: uuid.New()}
	original := []*domain.TaskLog{first, second}
	copyRows := append([]*domain.TaskLog(nil), original...)

	source := copyFromLogs(copyRows)
	if !source.Next() {
		t.Fatal("expected first COPY row")
	}
	if _, err := source.Values(); err != nil {
		t.Fatalf("read first COPY row: %v", err)
	}

	if len(original) != 2 {
		t.Fatalf("fallback batch length=%d, want 2", len(original))
	}
	if original[0] != first || original[1] != second {
		t.Fatal("COPY iteration mutated the original fallback batch")
	}
}
