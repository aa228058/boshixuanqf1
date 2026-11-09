package monitor

import (
	"context"
	"sync"

	"tg-sender/internal/safego"
)

// monitorWorkTracker closes task admission before waiting, preventing
// WaitGroup.Add from racing with Wait during session shutdown.
type monitorWorkTracker struct {
	mu sync.Mutex
	accepting bool
	wg sync.WaitGroup
}

func newMonitorWorkTracker() *monitorWorkTracker {
	return &monitorWorkTracker{accepting: true}
}

func (t *monitorWorkTracker) enter() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.accepting {
		return false
	}
	t.wg.Add(1)
	return true
}

func (t *monitorWorkTracker) leave() { t.wg.Done() }

func (t *monitorWorkTracker) stopAndWait() {
	t.mu.Lock()
	t.accepting = false
	t.mu.Unlock()
	t.wg.Wait()
}

type monitorWorkTrackerKey struct{}

func monitorGo(ctx context.Context, fn func()) {
	if ctx == nil {
		return
	}
	t, _ := ctx.Value(monitorWorkTrackerKey{}).(*monitorWorkTracker)
	if t == nil {
		safego.Go(fn)
		return
	}
	if !t.enter() {
		return
	}
	safego.Go(func() {
		defer t.leave()
		fn()
	})
}
