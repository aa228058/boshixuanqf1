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

func TestTryAddDailyUsageAmountReservesWholeBatchAtomically(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (tenant_id, username, password_hash, name, api_key_hash, plan_id)
		VALUES ($1, $2, 'x', 'quota batch test', $3, 'free')`,
		tenantID, "quota-batch-"+tenantID.String(), "quota-batch-key-"+tenantID.String())
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	repo := NewPlanLimitRepository(pool)
	const limit = 5
	const amount = 2
	const requests = 12
	var allowed atomic.Int32
	var wg sync.WaitGroup
	errs := make(chan error, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := repo.TryAddDailyUsageAmount(ctx, tenantID, KindBroadcastMsgs, amount, limit)
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
		t.Errorf("TryAddDailyUsageAmount: %v", err)
	}
	if got := allowed.Load(); got != 2 {
		t.Fatalf("allowed batches = %d, want 2 complete batches", got)
	}
	used, err := repo.GetDailyUsage(ctx, tenantID, KindBroadcastMsgs)
	if err != nil {
		t.Fatalf("GetDailyUsage: %v", err)
	}
	if used != 4 {
		t.Fatalf("stored usage = %d, want 4 (never exceed limit %d)", used, limit)
	}

	// A single batch larger than the entire cap must not create an over-limit row.
	ok, err := repo.TryAddDailyUsageAmount(ctx, tenantID, KindBroadcastMsgs, 6, limit)
	if err != nil || ok {
		t.Fatalf("oversized batch reservation = (%v, %v), want (false, nil)", ok, err)
	}
	used, err = repo.GetDailyUsage(ctx, tenantID, KindBroadcastMsgs)
	if err != nil || used != 4 {
		t.Fatalf("usage after rejected oversized batch = (%d, %v), want (4, nil)", used, err)
	}
}
