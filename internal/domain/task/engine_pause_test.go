package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestPauseTaskCancelsActiveExecution(t *testing.T) {
	engine := &Engine{}
	taskID := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.taskCtxs.Store(taskID.String(), context.CancelFunc(cancel))
	engine.PauseTask(taskID)

	if ctx.Err() == nil {
		t.Fatal("PauseTask did not cancel the active task context")
	}
}
