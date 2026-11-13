//go:build integration

package repository

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
)

func TestTryAddDailyUsageDoesNotExceedLimitUnderConcurrency(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (tenant_id, username, password_hash, name, api_key_hash, plan_id)
		VALUES ($1, $2, 'x', 'quota test', $3, 'free')`,
		tenantID, "quota-"+tenantID.String(), "quota-key-"+tenantID.String())
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	repo := NewPlanLimitRepository(pool)
	const limit = 3
	const requests = 24
	var allowed atomic.Int32
	var wg sync.WaitGroup
	errs := make(chan error, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := repo.TryAddDailyUsage(ctx, tenantID, KindGroupSearch, limit)
			if err != nil {
				errs <- err
				return
			}
			if ok {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("TryAddDailyUsage: %v", err)
	}
	if got := allowed.Load(); got != limit {
		t.Fatalf("allowed requests = %d, want exactly %d", got, limit)
	}
	used, err := repo.GetDailyUsage(ctx, tenantID, KindGroupSearch)
	if err != nil {
		t.Fatalf("GetDailyUsage: %v", err)
	}
	if used != limit {
		t.Fatalf("stored usage = %d, want %d", used, limit)
	}
}
