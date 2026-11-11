package monitor

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestMonitorWorkTrackerStopsAdmissionAndDrains(t *testing.T) {
	tracker := newMonitorWorkTracker()
	release := make(chan struct{})
	if !tracker.enter() {
		t.Fatal("expected initial work admission")
	}
	var finished atomic.Bool
	go func() {
		<-release
		finished.Store(true)
		tracker.leave()
	}()

	drained := make(chan struct{})
	go func() {
		tracker.stopAndWait()
		close(drained)
	}()
	deadline := time.Now().Add(time.Second)
	for tracker.enter() {
		tracker.leave()
		if time.Now().After(deadline) {
			t.Fatal("stopAndWait did not close admission")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case <-drained:
		t.Fatal("tracker returned before admitted work finished")
	default:
	}
	close(release)
	select {
	case <-drained:
	case <-time.After(time.Second):
		t.Fatal("tracker did not finish draining")
	}
	if !finished.Load() {
		t.Fatal("admitted work did not finish")
	}
	if tracker.enter() {
		tracker.leave()
		t.Fatal("tracker admitted work after shutdown")
	}
}
