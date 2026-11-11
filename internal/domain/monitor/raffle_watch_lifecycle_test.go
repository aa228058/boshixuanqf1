package monitor

import (
	"testing"
	"time"
)

func TestRaffleWatcherStopIsIdempotentAndWaits(t *testing.T) {
	watcher := newRaffleWatcher()
	watcher.Stop()
	select {
	case <-watcher.doneCh:
	case <-time.After(time.Second):
		t.Fatal("watcher maintenance goroutine did not stop")
	}
	watcher.Stop()
}
