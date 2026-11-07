//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGroupLibraryImportLeaseFencingAndProgressCAS(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO tenants (tenant_id, username, password_hash, name, api_key_hash, plan_id) VALUES ($1, $2, 'x', 'Import Lease Test', $3, 'free')`, tenantID, "lease_"+tenantID.String()[:8], "lease-key-"+tenantID.String())
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	jobID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO group_library_import_jobs (job_id, tenant_id, file_name, payload, status, total_count) VALUES ($1, $2, 'groups.txt', 'one\ntwo\nthree', 'pending', 3)`, jobID, tenantID)
	if err != nil {
		t.Fatalf("seed import job: %v", err)
	}

	claim := func() (uuid.UUID, bool) {
		var token uuid.UUID
		err := pool.QueryRow(ctx, `UPDATE group_library_import_jobs SET status='processing', lease_token=gen_random_uuid(), lease_expires_at=NOW()+INTERVAL '2 minutes', updated_at=NOW() WHERE job_id=(SELECT job_id FROM group_library_import_jobs WHERE status='pending' OR (status='processing' AND (lease_expires_at IS NULL OR lease_expires_at<NOW())) ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED) RETURNING lease_token`).Scan(&token)
		return token, err == nil
	}
	first, ok := claim()
	if !ok {
		t.Fatal("first worker failed to claim pending job")
	}
	if _, ok := claim(); ok {
		t.Fatal("second worker claimed active lease")
	}
	_, err = pool.Exec(ctx, `UPDATE group_library_import_jobs SET lease_expires_at=NOW()-INTERVAL '1 second' WHERE job_id=$1 AND lease_token=$2`, jobID, first)
	if err != nil {
		t.Fatalf("expire first lease: %v", err)
	}
	second, ok := claim()
	if !ok || second == first {
		t.Fatal("replacement worker did not receive a fresh fencing token")
	}

	stale, err := pool.Exec(ctx, `UPDATE group_library_import_jobs SET processed_count=processed_count+1 WHERE job_id=$1 AND status='processing' AND lease_token=$2 AND processed_count=0`, jobID, first)
	if err != nil || stale.RowsAffected() != 0 {
		t.Fatalf("stale worker affected %d rows, err=%v; want 0", stale.RowsAffected(), err)
	}
	current, err := pool.Exec(ctx, `UPDATE group_library_import_jobs SET processed_count=processed_count+1, lease_expires_at=NOW()+INTERVAL '2 minutes' WHERE job_id=$1 AND status='processing' AND lease_token=$2 AND processed_count=0`, jobID, second)
	if err != nil || current.RowsAffected() != 1 {
		t.Fatalf("current worker affected %d rows, err=%v; want 1", current.RowsAffected(), err)
	}
	replay, err := pool.Exec(ctx, `UPDATE group_library_import_jobs SET processed_count=processed_count+1 WHERE job_id=$1 AND status='processing' AND lease_token=$2 AND processed_count=0`, jobID, second)
	if err != nil || replay.RowsAffected() != 0 {
		t.Fatalf("replayed update affected %d rows, err=%v; want 0", replay.RowsAffected(), err)
	}

	var processed int
	var expires time.Time
	if err := pool.QueryRow(ctx, `SELECT processed_count, lease_expires_at FROM group_library_import_jobs WHERE job_id=$1`, jobID).Scan(&processed, &expires); err != nil {
		t.Fatalf("read import job: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed_count=%d, want 1", processed)
	}
	if time.Until(expires) < time.Minute {
		t.Fatalf("lease was not renewed: expires in %s", time.Until(expires))
	}
}
