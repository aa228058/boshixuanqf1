package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"tg-sender/internal/domain"
)

func TestCopyFromLogsDoesNotConsumeOriginalBatch(t *testing.T) {
	first := &domain.TaskLog{LogID: uuid.New(), ExecutedAt: time.Now()}
	second := &domain.TaskLog{LogID: uuid.New(), ExecutedAt: time.Now()}
	batch := []*domain.TaskLog{first, second}
	source := &copyFromLogs{logs: batch}

	if !source.Next() {
		t.Fatal("expected first row")
	}
	values, err := source.Values()
	if err != nil {
		t.Fatalf("read first row: %v", err)
	}
	if len(values) != 13 {
		t.Fatalf("expected 13 COPY values, got %d", len(values))
	}
	if len(batch) != 2 || batch[0] != first || batch[1] != second {
		t.Fatal("COPY source mutated the original fallback batch")
	}

	if !source.Next() {
		t.Fatal("expected second row")
	}
	if _, err := source.Values(); err != nil {
		t.Fatalf("read second row: %v", err)
	}
	if source.Next() {
		t.Fatal("expected source to be exhausted")
	}
	if len(batch) != 2 {
		t.Fatal("original batch must remain intact after COPY source exhaustion")
	}
}
