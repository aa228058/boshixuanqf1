package grouplib

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTelegramWebScheduler429OpensSharedCircuit(t *testing.T) {
	s := newTelegramWebScheduler(2, 0, 0)
	s.recordHTTPResult(http.StatusTooManyRequests, time.Now(), "30")

	var rateErr *TelegramRateLimitError
	if err := s.circuitError(time.Now()); !errors.As(err, &rateErr) {
		t.Fatalf("expected shared circuit to open, got %v", err)
	}
	if rateErr.RetryAfter < 29*time.Second {
		t.Fatalf("RetryAfter=%s, want at least 29s", rateErr.RetryAfter)
	}
}

func TestTelegramWebSchedulerOpensAfterTransientFailures(t *testing.T) {
	s := newTelegramWebScheduler(2, 0, 0)
	now := time.Now()
	for i := 0; i < telegramWebFailureLimit-1; i++ {
		s.recordHTTPResult(http.StatusBadGateway, now, "")
	}
	if err := s.circuitError(now); err != nil {
		t.Fatalf("circuit opened too early: %v", err)
	}
	s.recordHTTPResult(http.StatusServiceUnavailable, now, "")
	if err := s.circuitError(now); err == nil {
		t.Fatal("expected circuit to open after failure threshold")
	}
}

func TestTelegramWebSchedulerDoesNotTripOnNotFound(t *testing.T) {
	s := newTelegramWebScheduler(2, 0, 0)
	now := time.Now()
	for i := 0; i < telegramWebFailureLimit-1; i++ {
		s.recordHTTPResult(http.StatusBadGateway, now, "")
	}
	s.recordHTTPResult(http.StatusNotFound, now, "")
	if err := s.circuitError(now); err != nil {
		t.Fatalf("404 must not trip breaker: %v", err)
	}
}

func TestTelegramWebSchedulerEnforcesGlobalConcurrency(t *testing.T) {
	var active, peak atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := active.Add(1)
		for {
			old := peak.Load()
			if current <= old || peak.CompareAndSwap(old, current) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		active.Add(-1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := newTelegramWebScheduler(2, 0, 0)
	client := server.Client()
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
			if err != nil {
				t.Error(err)
				return
			}
			resp, err := s.do(context.Background(), client, req)
			if err != nil {
				t.Error(err)
				return
			}
			resp.Body.Close()
		}()
	}
	wg.Wait()
	if got := peak.Load(); got > 2 {
		t.Fatalf("peak concurrency=%d, want <=2", got)
	}
}
