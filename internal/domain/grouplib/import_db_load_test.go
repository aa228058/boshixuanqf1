//go:build importbench

package grouplib

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPersistImportFailuresLoad is an opt-in real-PostgreSQL load test.
// It is intentionally separate from normal unit tests and runs only after
// the CI migration step has created the production schema.
func TestPersistImportFailuresLoad(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	host := envOr("DB_HOST", "localhost")
	port := envOr("DB_PORT", "5432")
	user := envOr("DB_USER", "postgres")
	password := envOr("DB_PASSWORD", "postgres")
	name := envOr("DB_NAME", "tg_sender_test")
	sslmode := envOr("DB_SSLMODE", "disable")
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, name, sslmode)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}

	tenantID := uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO tenants (tenant_id, username, password_hash, name, api_key_hash, plan_id)
		 VALUES ($1, $2, 'x', 'Import DB Load Test', $3, 'free')`,
		tenantID, "importbench_"+tenantID.String()[:8], "importbench-key-"+tenantID.String())
	if err != nil {
		t.Fatalf("create isolated benchmark tenant: %v", err)
	}
	defer func() { _, _ = pool.Exec(context.Background(), "DELETE FROM tenants WHERE tenant_id=$1", tenantID) }()

	engine := &Engine{pool: pool}
	for _, size := range []int{10_000, 30_000, 50_000} {
		jobID := uuid.New()
		_, err := pool.Exec(ctx,
			`INSERT INTO group_library_import_jobs (job_id, tenant_id, file_name, payload, status, total_count)
			 VALUES ($1, $2, 'import-db-benchmark.txt', '', 'completed', $3)`,
			jobID, tenantID, size)
		if err != nil {
			t.Fatalf("create benchmark job for %d rows: %v", size, err)
		}

		failures := make([]string, size)
		for i := range failures {
			failures[i] = fmt.Sprintf("[not_found] https://t.me/BenchGroup%08d: not found", i)
		}

		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		started := time.Now()
		if err := engine.persistImportFailures(ctx, jobID, failures); err != nil {
			t.Fatalf("persist %d failure rows: %v", size, err)
		}
		elapsed := time.Since(started)
		runtime.ReadMemStats(&after)

		var persisted int
		if err := pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM group_library_import_job_failures WHERE job_id=$1", jobID).Scan(&persisted); err != nil {
			t.Fatalf("verify %d persisted rows: %v", size, err)
		}
		if persisted != size {
			t.Fatalf("persisted %d rows for size %d, want exact count", persisted, size)
		}

		report := map[string]any{
			"scenario":                "postgres_batch_failure_upsert",
			"rows":                    size,
			"elapsed_ms":              float64(elapsed.Microseconds()) / 1000,
			"rows_per_second":         float64(size) / elapsed.Seconds(),
			"heap_alloc_delta_bytes":  int64(after.HeapAlloc) - int64(before.HeapAlloc),
			"total_alloc_delta_bytes": int64(after.TotalAlloc) - int64(before.TotalAlloc),
			"verified_rows":           persisted,
			"database":                "PostgreSQL",
		}
		encoded, _ := json.Marshal(report)
		t.Logf("IMPORT_DB_BENCH %s", encoded)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
