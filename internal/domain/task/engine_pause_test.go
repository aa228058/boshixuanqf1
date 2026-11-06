package task

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"tg-sender/internal/domain"
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

func TestTaskLeaseIDUsesClaimToken(t *testing.T) {
	engine := &Engine{workerID: "engine-worker"}
	claimToken := "engine-worker:claim-unique"
	task := &domain.Task{WorkerID: &claimToken}

	if got := engine.taskLeaseID(task); got != claimToken {
		t.Fatalf("taskLeaseID() = %q, want claim token %q", got, claimToken)
	}
	if got := engine.taskLeaseID(&domain.Task{}); got != "engine-worker" {
		t.Fatalf("taskLeaseID() fallback = %q, want engine worker ID", got)
	}
}
