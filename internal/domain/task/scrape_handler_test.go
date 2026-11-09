package task

import (
	"sync/atomic"
	"testing"
)

func TestScrapeHandlerEntryActiveFlag(t *testing.T) {
	entry := &scrapeHandlerEntry{active: 1}
	if got := atomic.LoadInt32(&entry.active); got != 1 {
		t.Fatalf("new handler active flag = %d, want 1", got)
	}

	atomic.StoreInt32(&entry.active, 0)
	if got := atomic.LoadInt32(&entry.active); got != 0 {
		t.Fatalf("stopped handler active flag = %d, want 0", got)
	}
}
