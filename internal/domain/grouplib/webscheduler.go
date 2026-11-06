package grouplib

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	securerand "tg-sender/internal/secure"
)

const (
	telegramWebMaxConcurrent = 2
	telegramWebMinInterval   = 2 * time.Second
	telegramWebMaxJitter     = time.Second
	telegramWebFailureLimit  = 5
	telegramWebCooldown      = time.Minute
)

// telegramWebScheduler coordinates every public-page lookup in this process.
// Tenants and import jobs deliberately share one request budget.
type telegramWebScheduler struct {
	slots       chan struct{}
	mu          sync.Mutex
	nextStart   time.Time
	openUntil   time.Time
	failures    int
	halfOpen    bool
	minInterval time.Duration
	maxJitter   time.Duration
}

func newTelegramWebScheduler(concurrency int, minInterval, maxJitter time.Duration) *telegramWebScheduler {
	if concurrency < 1 {
		concurrency = 1
	}
	return &telegramWebScheduler{
		slots:       make(chan struct{}, concurrency),
		minInterval: minInterval,
		maxJitter:   maxJitter,
	}
}

var sharedTelegramWebScheduler = newTelegramWebScheduler(
	telegramWebMaxConcurrent,
	telegramWebMinInterval,
	telegramWebMaxJitter,
)

func (s *telegramWebScheduler) do(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if err := s.waitForPermit(ctx); err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		s.recordTransientFailure(time.Now(), 0)
		return nil, err
	}
	s.recordHTTPResult(resp.StatusCode, time.Now(), resp.Header.Get("Retry-After"))
	return resp, nil
}

func (s *telegramWebScheduler) waitForPermit(ctx context.Context) error {
	now := time.Now()
	s.mu.Lock()
	if now.Before(s.openUntil) {
		delay := s.openUntil.Sub(now)
		s.mu.Unlock()
		return &TelegramRateLimitError{RetryAfter: delay, Endpoint: "t.me 全局熔断"}
	}
	if !s.openUntil.IsZero() {
		if s.halfOpen {
			s.mu.Unlock()
			return &TelegramRateLimitError{RetryAfter: time.Second, Endpoint: "t.me 熔断探测中"}
		}
		s.halfOpen = true
	}

	startAt := now
	if s.nextStart.After(startAt) {
		startAt = s.nextStart
	}
	jitter := time.Duration(0)
	if s.maxJitter > 0 {
		jitter = time.Duration(securerand.Int63n(int64(s.maxJitter) + 1))
	}
	s.nextStart = startAt.Add(s.minInterval + jitter)
	s.mu.Unlock()

	if wait := time.Until(startAt); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			s.mu.Lock()
			if s.halfOpen && !time.Now().Before(s.openUntil) {
				s.halfOpen = false
			}
			s.mu.Unlock()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return nil
}

func (s *telegramWebScheduler) recordHTTPResult(status int, now time.Time, retryAfter string) {
	if status == http.StatusTooManyRequests {
		delay := parseTelegramRetryAfter(retryAfter, now)
		s.mu.Lock()
		s.failures = telegramWebFailureLimit
		s.openUntil = now.Add(delay)
		s.halfOpen = false
		s.mu.Unlock()
		return
	}
	if status >= http.StatusInternalServerError {
		s.recordTransientFailure(now, 0)
		return
	}

	// A late 2xx from an already in-flight request must not cancel a newer 429.
	s.mu.Lock()
	if now.Before(s.openUntil) {
		s.mu.Unlock()
		return
	}
	s.failures = 0
	s.openUntil = time.Time{}
	s.halfOpen = false
	s.mu.Unlock()
}

func (s *telegramWebScheduler) recordTransientFailure(now time.Time, retryAfter time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures++
	if s.halfOpen || s.failures >= telegramWebFailureLimit {
		delay := telegramWebCooldown
		if retryAfter > delay {
			delay = retryAfter
		}
		s.openUntil = now.Add(delay)
		s.halfOpen = false
	}
}

func (s *telegramWebScheduler) circuitError(now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if now.Before(s.openUntil) {
		return &TelegramRateLimitError{RetryAfter: s.openUntil.Sub(now), Endpoint: "t.me 全局熔断"}
	}
	return nil
}

func telegramWebDo(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("Telegram 请求不能为空")
	}
	return sharedTelegramWebScheduler.do(req.Context(), telegramWebClient, req)
}
