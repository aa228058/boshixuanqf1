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

func TestCreateSplitBatchIsAtomic(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (tenant_id, username, password_hash, name, api_key_hash, plan_id)
		VALUES ($1, $2, 'x', 'atomic batch test', $3, 'free')`,
		tenantID, "atomic-batch-"+tenantID.String(), "atomic-batch-key-"+tenantID.String())
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	repo := NewTaskRepository(pool)
	now := time.Now()
	firstID, secondID := uuid.New(), uuid.New()
	batch := &domain.BatchTask{
		BatchID: uuid.New(), TenantID: tenantID, Message: "atomic",
		TotalMsgs: 2, SubTasks: []uuid.UUID{firstID, secondID}, CreatedAt: now,
	}
	makeTask := func(id uuid.UUID) *domain.Task {
		return &domain.Task{
			TaskID: id, TenantID: tenantID, Type: domain.TaskTypeBroadcast,
			Status: domain.TaskStatusQueued, Priority: 5,
			AccountIDs: []uuid.UUID{}, TargetIDs: []string{"target"},
			Payload:  map[string]any{"batch_id": batch.BatchID.String()},
			Progress: domain.TaskProgress{Total: 1}, CreatedAt: now,
		}
	}
	if err := repo.CreateSplitBatch(ctx, batch, []*domain.Task{makeTask(firstID), makeTask(secondID)}); err != nil {
		t.Fatalf("create atomic split batch: %v", err)
	}
	storedBatch, err := NewBatchTaskRepository(pool).GetByID(ctx, tenantID, batch.BatchID)
	if err != nil {
		t.Fatalf("get committed batch: %v", err)
	}
	if len(storedBatch.SubTasks) != 2 || storedBatch.SubTasks[0] != firstID || storedBatch.SubTasks[1] != secondID {
		t.Fatalf("stored sub_tasks = %v, want both child IDs in order", storedBatch.SubTasks)
	}
	var committed int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE tenant_id=$1 AND task_id = ANY($2::uuid[])`, tenantID, batch.SubTasks).Scan(&committed); err != nil {
		t.Fatalf("count committed children: %v", err)
	}
	if committed != 2 {
		t.Fatalf("committed child count = %d, want 2", committed)
	}

	// A duplicate child ID forces the second insert to fail. The transaction
	// must roll back the parent and the first child instead of leaving a
	// partially-created batch that could be retried and sent twice.
	rollbackBatchID := uuid.New()
	duplicateID := uuid.New()
	rollbackBatch := &domain.BatchTask{
		BatchID: rollbackBatchID, TenantID: tenantID, Message: "rollback",
		TotalMsgs: 2, SubTasks: []uuid.UUID{duplicateID, duplicateID}, CreatedAt: now,
	}
	err = repo.CreateSplitBatch(ctx, rollbackBatch, []*domain.Task{makeTask(duplicateID), makeTask(duplicateID)})
	if err == nil {
		t.Fatal("expected duplicate child insert to fail")
	}
	var parentCount, childCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM batch_tasks WHERE tenant_id=$1 AND batch_id=$2`, tenantID, rollbackBatchID).Scan(&parentCount); err != nil {
		t.Fatalf("count rolled-back parent: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE tenant_id=$1 AND task_id=$2`, tenantID, duplicateID).Scan(&childCount); err != nil {
		t.Fatalf("count rolled-back child: %v", err)
	}
	if parentCount != 0 || childCount != 0 {
		t.Fatalf("partial batch survived rollback: parent=%d child=%d", parentCount, childCount)
	}
}
