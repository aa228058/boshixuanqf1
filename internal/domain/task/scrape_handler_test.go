package task

import (
	"testing"
	"time"
)

func TestScrapeHandlerEntryStopsAdmissionAndDrains(t *testing.T) {
	entry := &scrapeHandlerEntry{accepting: true}
	if !entry.enter() {
		t.Fatal("expected initial callback admission")
	}

	drained := make(chan struct{})
	go func() {
		entry.stopAndWait()
		close(drained)
	}()

	deadline := time.Now().Add(time.Second)
	for entry.enter() {
		entry.leave()
		if time.Now().After(deadline) {
			t.Fatal("stopAndWait did not close callback admission")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case <-drained:
		t.Fatal("stopAndWait returned before in-flight callback completed")
	default:
	}

	entry.leave()
	select {
	case <-drained:
	case <-time.After(time.Second):
		t.Fatal("stopAndWait did not drain callback")
	}
	if entry.enter() {
		entry.leave()
		t.Fatal("callback admitted after shutdown")
	}
}
