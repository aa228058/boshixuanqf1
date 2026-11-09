package task

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestLinkScraperStartIsIdempotent(t *testing.T) {
	ls := NewLinkScraper(nil, nil, nil, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	ls.Start(ctx)
	ls.Start(ctx)

	ls.handlersMu.Lock()
	started := ls.started
	ls.handlersMu.Unlock()
	if !started {
		t.Fatal("Start should mark the scraper as started")
	}

	cancel()
	done := make(chan struct{})
	go func() {
		ls.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not drain scraper goroutines")
	}
}

func TestLinkScraperDoesNotStartAfterStop(t *testing.T) {
	ls := NewLinkScraper(nil, nil, nil, nil, nil)
	ls.Stop()
	ls.Start(context.Background())

	ls.handlersMu.Lock()
	started, stopped := ls.started, ls.stopped
	ls.handlersMu.Unlock()
	if started {
		t.Fatal("Start after Stop must not launch background workers")
	}
	if !stopped {
		t.Fatal("Stop should permanently close the scraper lifecycle")
	}
}

func TestLinkScraperConcurrentStartAndStop(t *testing.T) {
	for i := 0; i < 20; i++ {
		ls := NewLinkScraper(nil, nil, nil, nil, nil)
		ctx, cancel := context.WithCancel(context.Background())

		var callers sync.WaitGroup
		callers.Add(2)
		go func() {
			defer callers.Done()
			ls.Start(ctx)
		}()
		go func() {
			defer callers.Done()
			ls.Stop()
		}()
		callers.Wait()
		cancel()

		done := make(chan struct{})
		go func() {
			ls.Stop()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("iteration %d: Stop did not finish", i)
		}
	}
}

func TestScrapedLinkDedupKeyIsTenantScoped(t *testing.T) {
	firstTenant := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	secondTenant := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	linkHash := "public-group-name"

	firstKey := scrapedLinkDedupKey(firstTenant, linkHash)
	if firstKey != scrapedLinkDedupKey(firstTenant, linkHash) {
		t.Fatal("same tenant and link hash should deduplicate")
	}
	if firstKey == scrapedLinkDedupKey(secondTenant, linkHash) {
		t.Fatal("the same Telegram link must not be deduplicated across tenants")
	}
}

func TestLinkScraperDrainsQueuedLinksWithoutNetworkResolution(t *testing.T) {
	ls := NewLinkScraper(nil, nil, nil, nil, nil)
	tenantID := uuid.New()
	accountID := uuid.New()
	ls.linkQueue <- &rawLink{
		TenantID: tenantID,
		AccountID: accountID,
		LinkType: "public",
		LinkHash: "sample-public-group",
		FullURL: "https://t.me/sample-public-group",
	}

	ls.drainQueuedLinks()

	ls.batchMu.Lock()
	defer ls.batchMu.Unlock()
	if len(ls.batchLinks) != 1 {
		t.Fatalf("expected queued link to be retained in final batch, got %d", len(ls.batchLinks))
	}
	if ls.batchLinks[0].Status != "new" {
		t.Fatalf("shutdown drain should preserve unresolved links as new, got %q", ls.batchLinks[0].Status)
	}
	if len(ls.linkQueue) != 0 {
		t.Fatal("shutdown drain should empty the in-memory queue")
	}
}

func TestLinkScraperParentCancellationClosesAdmissionAndDrains(t *testing.T) {
	ls := NewLinkScraper(nil, nil, nil, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	ls.Start(ctx)
	cancel()

	select {
	case <-ls.stopCh:
	case <-time.After(2 * time.Second):
		t.Fatal("parent cancellation did not close scraper callback admission")
	}

	done := make(chan struct{})
	go func() {
		ls.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scraper workers did not drain after parent cancellation")
	}
}
