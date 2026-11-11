//go:build importworker

package grouplib

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func importWorkerTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	host := envWorker("DB_HOST", "localhost")
	port := envWorker("DB_PORT", "5432")
	user := envWorker("DB_USER", "postgres")
	password := envWorker("DB_PASSWORD", "postgres")
	name := envWorker("DB_NAME", "tg_sender_test")
	sslmode := envWorker("DB_SSLMODE", "disable")
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, name, sslmode)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping PostgreSQL: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func envWorker(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func createImportWorkerFixture(t *testing.T, pool *pgxpool.Pool, links int, status string, processed int) (uuid.UUID, uuid.UUID, []string) {
	t.Helper()
	ctx := context.Background()
	tenantID, jobID := uuid.New(), uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO tenants (tenant_id, name, api_key_hash) VALUES ($1, 'Import Worker Test', $2)`,
		tenantID, "import-worker-"+tenantID.String())
	if err != nil {
		t.Fatalf("create tenant fixture: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM tenants WHERE tenant_id=$1", tenantID)
	})
	all := make([]string, links)
	for i := range all {
		all[i] = fmt.Sprintf("https://t.me/WorkerGroup%08d", i)
	}
	payload := strings.Join(all, "\n")
	_, err = pool.Exec(ctx, `
		INSERT INTO group_library_import_jobs
			(job_id, tenant_id, file_name, payload, status, total_count, processed_count, lease_expires_at)
		VALUES ($1, $2, 'worker-test.txt', $3, $4, $5, $6,
		        CASE WHEN $4::varchar='processing' THEN NOW()-INTERVAL '1 minute' ELSE NULL END)
	`, jobID, tenantID, payload, status, links, processed)
	if err != nil {
		t.Fatalf("create import job fixture: %v", err)
	}
	return tenantID, jobID, all
}

func readImportWorkerState(t *testing.T, pool *pgxpool.Pool, jobID uuid.UUID) (status string, processed, imported, failed, skipped int, token *uuid.UUID) {
	t.Helper()
	err := pool.QueryRow(context.Background(), `
		SELECT status, processed_count, imported_count, failed_count, skipped_count, lease_token
		FROM group_library_import_jobs WHERE job_id=$1
	`, jobID).Scan(&status, &processed, &imported, &failed, &skipped, &token)
	if err != nil {
		t.Fatalf("read worker job state: %v", err)
	}
	return
}

func TestImportWorkerFairSchedulingProcessesOneChunkPerJob(t *testing.T) {
	pool := importWorkerTestPool(t)
	_, largeJobID, _ := createImportWorkerFixture(t, pool, 301, "pending", 0)
	_, smallJobID, _ := createImportWorkerFixture(t, pool, 45, "pending", 0)
	var calls atomic.Int32
	engine := &Engine{pool: pool}
	engine.workerImportOverride = func(_ context.Context, _, _ uuid.UUID, lines []string) (*ImportResult, error) {
		if len(lines) > 30 {
			t.Errorf("worker sent %d links in one batch; maximum is 30", len(lines))
		}
		calls.Add(1)
		return &ImportResult{Processed: len(lines)}, nil
	}
	engine.processImportJobs(context.Background())
	largeStatus, largeProcessed, _, _, _, _ := readImportWorkerState(t, pool, largeJobID)
	smallStatus, smallProcessed, _, _, _, _ := readImportWorkerState(t, pool, smallJobID)
	if largeStatus != "pending" || largeProcessed != 30 || smallStatus != "pending" || smallProcessed != 30 || calls.Load() != 2 {
		t.Fatalf("one pass large=%s/%d small=%s/%d calls=%d; want pending/30 for both and 2 calls",
			largeStatus, largeProcessed, smallStatus, smallProcessed, calls.Load())
	}
}

func TestImportWorkerResumesExpiredLeaseFromSavedOffset(t *testing.T) {
	pool := importWorkerTestPool(t)
	_, jobID, _ := createImportWorkerFixture(t, pool, 35, "processing", 30)
	var received atomic.Int32
	engine := &Engine{pool: pool}
	engine.workerImportOverride = func(_ context.Context, _, _ uuid.UUID, lines []string) (*ImportResult, error) {
		received.Store(int32(len(lines)))
		return &ImportResult{Processed: len(lines)}, nil
	}
	engine.processImportJobs(context.Background())
	status, processed, _, _, _, token := readImportWorkerState(t, pool, jobID)
	if status != "completed" || processed != 35 || received.Load() != 5 || token != nil {
		t.Fatalf("resume result status=%s processed=%d received=%d token=%v; want completed/35/5/nil", status, processed, received.Load(), token)
	}
}

func TestImportWorkerFencingRejectsStaleWorkerProgress(t *testing.T) {
	pool := importWorkerTestPool(t)
	_, jobID, _ := createImportWorkerFixture(t, pool, 1, "pending", 0)
	replacementToken := uuid.New()
	engine := &Engine{pool: pool}
	engine.workerImportOverride = func(_ context.Context, _, _ uuid.UUID, _ []string) (*ImportResult, error) {
		_, err := pool.Exec(context.Background(), `
			UPDATE group_library_import_jobs SET lease_token=$2
			WHERE job_id=$1 AND status='processing'
		`, jobID, replacementToken)
		if err != nil {
			return nil, err
		}
		return &ImportResult{Processed: 1}, nil
	}
	engine.processImportJobs(context.Background())
	status, processed, _, _, _, token := readImportWorkerState(t, pool, jobID)
	if status != "processing" || processed != 0 || token == nil || *token != replacementToken {
		t.Fatalf("stale worker overwrote fenced job: status=%s processed=%d token=%v", status, processed, token)
	}
}

func TestImportWorkerCancellationPersistsCompletedPrefix(t *testing.T) {
	pool := importWorkerTestPool(t)
	tenantID, jobID, _ := createImportWorkerFixture(t, pool, 2, "pending", 0)
	started := make(chan struct{})
	engine := &Engine{pool: pool}
	engine.workerImportOverride = func(ctx context.Context, _, _ uuid.UUID, _ []string) (*ImportResult, error) {
		close(started)
		<-ctx.Done()
		return &ImportResult{Processed: 1, Skipped: 1}, ctx.Err()
	}
	done := make(chan struct{})
	go func() {
		engine.processImportJobs(context.Background())
		close(done)
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not start resolving")
	}
	if _, err := engine.CancelImportJob(context.Background(), tenantID, jobID); err != nil {
		t.Fatalf("request cancellation: %v", err)
	}
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		t.Fatal("worker did not stop after cancellation")
	}
	status, processed, _, _, _, token := readImportWorkerState(t, pool, jobID)
	if status != "cancelled" || processed != 1 || token != nil {
		t.Fatalf("cancel result status=%s processed=%d token=%v; want cancelled/1/nil", status, processed, token)
	}
}

func TestImportWorkerRateLimitDefersLeaseWithoutBlockingScheduler(t *testing.T) {
	pool := importWorkerTestPool(t)
	_, jobID, links := createImportWorkerFixture(t, pool, 3, "pending", 0)
	engine := &Engine{pool: pool}
	engine.workerImportOverride = func(_ context.Context, _, _ uuid.UUID, _ []string) (*ImportResult, error) {
		return &ImportResult{
			Processed: 1,
			Failed: []string{
				"[not_found] " + links[0] + ": missing",
				"[rate_limited] " + links[1] + ": FLOOD_WAIT",
			},
		}, &TelegramRateLimitError{RetryAfter: 10 * time.Minute, Endpoint: "test"}
	}
	started := time.Now()
	engine.processImportJobs(context.Background())
	elapsed := time.Since(started)
	status, processed, _, failed, _, token := readImportWorkerState(t, pool, jobID)
	var failureRows int
	if err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM group_library_import_job_failures WHERE job_id=$1", jobID).Scan(&failureRows); err != nil {
		t.Fatalf("count persisted prefix failures: %v", err)
	}
	var leaseExpiry time.Time
	if err := pool.QueryRow(context.Background(), "SELECT lease_expires_at FROM group_library_import_jobs WHERE job_id=$1", jobID).Scan(&leaseExpiry); err != nil {
		t.Fatalf("read deferred lease expiry: %v", err)
	}
	if elapsed > 5*time.Second || status != "processing" || processed != 1 || failed != 1 || failureRows != 1 || token == nil || !leaseExpiry.After(time.Now().Add(9*time.Minute)) {
		t.Fatalf("rate-limit deferral elapsed=%s status=%s processed=%d failed=%d failureRows=%d token=%v expiry=%s",
			elapsed, status, processed, failed, failureRows, token, leaseExpiry)
	}
}
func TestImportWorkerFailedLinksRetryExactlyOnce(t *testing.T) {
	pool := importWorkerTestPool(t)
	tenantID, jobID, _ := createImportWorkerFixture(t, pool, 2, "pending", 0)
	engine := &Engine{pool: pool}
	engine.workerImportOverride = func(_ context.Context, _, _ uuid.UUID, batch []string) (*ImportResult, error) {
		failures := make([]string, 0, len(batch))
		for _, link := range batch {
			failures = append(failures, "[not_found] "+link+": not found")
		}
		return &ImportResult{Processed: len(batch), Failed: failures}, nil
	}
	engine.processImportJobs(context.Background())
	status, processed, _, failed, _, _ := readImportWorkerState(t, pool, jobID)
	if status != "completed" || processed != 2 || failed != 2 {
		t.Fatalf("failed import state status=%s processed=%d failed=%d", status, processed, failed)
	}
	retry, err := engine.RetryFailedImportJob(context.Background(), tenantID, jobID)
	if err != nil {
		t.Fatalf("create retry job: %v", err)
	}
	if retry.TotalCount != 2 || retry.Status != "pending" {
		t.Fatalf("retry job total/status=%d/%s; want 2/pending", retry.TotalCount, retry.Status)
	}
	var linked, unlinked int
	if err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FILTER (WHERE retry_job_id=$2), COUNT(*) FILTER (WHERE retry_job_id IS NULL) FROM group_library_import_job_failures WHERE job_id=$1", jobID, retry.JobID).Scan(&linked, &unlinked); err != nil {
		t.Fatalf("verify retry links: %v", err)
	}
	if linked != 2 || unlinked != 0 {
		t.Fatalf("retry linkage linked=%d unlinked=%d; want 2/0", linked, unlinked)
	}
	if _, err := engine.RetryFailedImportJob(context.Background(), tenantID, jobID); err == nil {
		t.Fatal("second retry should not duplicate already retried failures")
	}
}

func TestImportWorkerConcurrentWorkersClaimJobOnlyOnce(t *testing.T) {
	pool := importWorkerTestPool(t)
	_, jobID, _ := createImportWorkerFixture(t, pool, 1, "pending", 0)
	var calls atomic.Int32
	resolver := func(_ context.Context, _, _ uuid.UUID, batch []string) (*ImportResult, error) {
		calls.Add(1)
		time.Sleep(250 * time.Millisecond)
		return &ImportResult{Processed: len(batch)}, nil
	}
	first, second := &Engine{pool: pool}, &Engine{pool: pool}
	first.workerImportOverride = resolver
	second.workerImportOverride = resolver
	done := make(chan struct{}, 2)
	go func() { first.processImportJobs(context.Background()); done <- struct{}{} }()
	go func() { second.processImportJobs(context.Background()); done <- struct{}{} }()
	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(8 * time.Second):
			t.Fatal("concurrent worker did not finish")
		}
	}
	status, processed, _, _, _, _ := readImportWorkerState(t, pool, jobID)
	if calls.Load() != 1 || status != "completed" || processed != 1 {
		t.Fatalf("concurrent claim calls=%d status=%s processed=%d; want 1/completed/1", calls.Load(), status, processed)
	}
}

func TestDeleteImportJobRequiresTenantAndNoActiveLease(t *testing.T) {
	pool := importWorkerTestPool(t)
	tenantID, jobID, _ := createImportWorkerFixture(t, pool, 1, "cancelled", 0)
	engine := &Engine{pool: pool}

	if err := engine.DeleteImportJob(context.Background(), uuid.New(), jobID); err == nil {
		t.Fatal("delete with a different tenant must be rejected")
	}

	leaseToken := uuid.New()
	if _, err := pool.Exec(context.Background(), `
		UPDATE group_library_import_jobs
		SET lease_token=$2, lease_expires_at=NOW()+INTERVAL '1 minute'
		WHERE job_id=$1
	`, jobID, leaseToken); err != nil {
		t.Fatalf("create active lease fixture: %v", err)
	}
	if err := engine.DeleteImportJob(context.Background(), tenantID, jobID); err == nil {
		t.Fatal("delete with an active lease must be rejected")
	}

	if _, err := pool.Exec(context.Background(), `
		UPDATE group_library_import_jobs
		SET lease_expires_at=NOW()-INTERVAL '1 second'
		WHERE job_id=$1 AND lease_token=$2
	`, jobID, leaseToken); err != nil {
		t.Fatalf("expire lease fixture: %v", err)
	}
	if err := engine.DeleteImportJob(context.Background(), tenantID, jobID); err != nil {
		t.Fatalf("delete inactive job history: %v", err)
	}
	var remaining int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM group_library_import_jobs WHERE job_id=$1
	`, jobID).Scan(&remaining); err != nil {
		t.Fatalf("verify deleted job: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("job rows remaining = %d, want 0", remaining)
	}
}
