package monitor

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestStartMonitoringRejectsAfterStopAll(t *testing.T) {
	engine := NewEngine(nil, nil, nil, nil)
	engine.StopAll()
	engine.StopAll() // shutdown remains idempotent

	err := engine.StartMonitoring(context.Background(), uuid.New(), uuid.New(), nil)
	if err == nil || !strings.Contains(err.Error(), "monitor engine is stopped") {
		t.Fatalf("expected stopped-engine error, got %v", err)
	}
}
