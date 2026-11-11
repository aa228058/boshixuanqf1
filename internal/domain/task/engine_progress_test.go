package task

import (
	"testing"

	"github.com/google/uuid"
	"tg-sender/internal/domain"
)

func TestPublishProgressDoesNotRegressDone(t *testing.T) {
	engine := &Engine{}
	taskID := uuid.New()
	tenantID := uuid.New()

	engine.publishProgress(tenantID, taskID, domain.TaskProgress{Total: 10, Done: 7})
	engine.publishProgress(tenantID, taskID, domain.TaskProgress{Total: 10, Done: 3, CurrentTarget: "late-callback"})

	got, ok := engine.GetProgress(taskID)
	if !ok {
		t.Fatal("expected cached task progress")
	}
	if got.Done != 7 {
		t.Fatalf("progress regressed: got Done=%d, want 7", got.Done)
	}
}
