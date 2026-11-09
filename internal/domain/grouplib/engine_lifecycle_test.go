package grouplib

import (
	"testing"
)

func TestEngineStopBeforeStartIsIdempotent(t *testing.T) {
	engine := NewEngine(nil, nil, nil)

	engine.Stop()
	engine.Stop()
	engine.Start()

	engine.lifecycleMu.Lock()
	started, stopped := engine.started, engine.stopped
	engine.lifecycleMu.Unlock()

	if started {
		t.Fatal("Start after Stop must not launch background workers")
	}
	if !stopped {
		t.Fatal("Stop must permanently close the engine lifecycle")
	}
}
