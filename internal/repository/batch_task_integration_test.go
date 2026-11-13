//go:build integration

package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"tg-sender/internal/domain"
)

func TestBatchTaskUpdateSubTaskIsIdempotentAndTenantScoped(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()
	otherTenantID := uuid.New()
	for i, id := range []uuid.UUID{tenantID, otherTenantID} {
		_, err := pool.Exec(ctx, `
			INSERT INTO tenants (tenant_id, username, password_hash, name, api_key_hash, plan_id)
			VALUES ($1, $2, 'x', 'batch linkage test', $3, 'free')`,
			id, "batch-link-"+id.String(), "batch-link-key-"+id.String())
		if err != nil {
			t.Fatalf("seed tenant %d: %v", i, err)
		}
	}

	repo := NewBatchTaskRepository(pool)
	batchID := uuid.New()
	subTaskID := uuid.New()
	batch := &domain.BatchTask{
		BatchID:   batchID,
		TenantID:  tenantID,
		Message:   "batch linkage",
		TotalMsgs: 1,
		SubTasks:  []uuid.UUID{},
		CreatedAt: time.Now(),
	}
	if err := repo.Create(ctx, batch); err != nil {
		t.Fatalf("create batch: %v", err)
	}
	if err := repo.UpdateSubTask(ctx, tenantID, batchID, subTaskID); err != nil {
		t.Fatalf("first link: %v", err)
	}
	if err := repo.UpdateSubTask(ctx, tenantID, batchID, subTaskID); err != nil {
		t.Fatalf("idempotent repeated link: %v", err)
	}
	got, err := repo.GetByID(ctx, tenantID, batchID)
	if err != nil {
		t.Fatalf("get batch: %v", err)
	}
	if len(got.SubTasks) != 1 || got.SubTasks[0] != subTaskID {
		t.Fatalf("sub_tasks = %v, want exactly [%s]", got.SubTasks, subTaskID)
	}

	err = repo.UpdateSubTask(ctx, otherTenantID, batchID, uuid.New())
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("foreign-tenant batch link error = %v, want not-found error", err)
	}
}
