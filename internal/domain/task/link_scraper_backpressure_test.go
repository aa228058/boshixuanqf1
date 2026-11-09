package task

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"tg-sender/internal/domain"
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

type flakyScrapedLinkRepository struct {
	domain.ScrapedLinkRepository
	failuresRemaining int
	calls              int
}

func (r *flakyScrapedLinkRepository) CreateBatch(_ context.Context, _ []*domain.ScrapedLink) error {
	r.calls++
	if r.failuresRemaining > 0 {
		r.failuresRemaining--
		return errors.New("temporary database outage")
	}
	return nil
}

func TestLinkScraperRetriesFinalBatchOnShutdown(t *testing.T) {
	repo := &flakyScrapedLinkRepository{failuresRemaining: 1}
	ls := NewLinkScraper(nil, nil, repo, nil, nil)
	ls.batchLinks = []*domain.ScrapedLink{{LinkID: uuid.New(), TenantID: uuid.New()}}
	atomic.StoreInt64(&ls.pendingLinkCount, 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Shutdown writes use a short independent context.
	ls.flushBatchOnShutdown(ctx)

	if repo.calls != 2 {
		t.Fatalf("CreateBatch calls = %d, want 2", repo.calls)
	}
	if got := atomic.LoadInt64(&ls.pendingLinkCount); got != 0 {
		t.Fatalf("pending link count = %d, want 0", got)
	}
	ls.batchMu.Lock()
	defer ls.batchMu.Unlock()
	if len(ls.batchLinks) != 0 {
		t.Fatalf("retained batch length = %d, want 0", len(ls.batchLinks))
	}
}
