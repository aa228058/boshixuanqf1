package task

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
)

func TestLinkScraperTracksPendingLinksAndOverloadStats(t *testing.T) {
	ls := NewLinkScraper(nil, nil, nil, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Skip network resolution; this test covers queue-to-batch accounting only.

	ls.processLink(ctx, &rawLink{
		TenantID:  uuid.New(),
		AccountID: uuid.New(),
		LinkType:  "public",
		LinkHash:  "pending-link",
		FullURL:   "https://t.me/pending-link",
	})
	if got := atomic.LoadInt64(&ls.pendingLinkCount); got != 1 {
		t.Fatalf("pending link count = %d, want 1", got)
	}

	atomic.StoreInt64(&ls.queueDropped, 3)
	atomic.StoreInt64(&ls.persistenceFailures, 2)
	stats := ls.GetStats()
	if got := stats["pending_links"]; got != int64(1) {
		t.Fatalf("pending_links = %v, want 1", got)
	}
	if got := stats["queue_dropped_total"]; got != int64(3) {
		t.Fatalf("queue_dropped_total = %v, want 3", got)
	}
	if got := stats["persistence_failures_total"]; got != int64(2) {
		t.Fatalf("persistence_failures_total = %v, want 2", got)
	}
}

func TestLinkScraperPendingLimitIsPositiveAndBounded(t *testing.T) {
	if maxPendingScrapedLinks <= 0 || maxPendingScrapedLinks > 100000 {
		t.Fatalf("unexpected pending link limit: %d", maxPendingScrapedLinks)
	}
}
