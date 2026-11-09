package task

import (
	"context"
	"sync"
	"testing"
	"time"
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
