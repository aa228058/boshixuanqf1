package grouplib

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ImportJob struct {
	JobID uuid.UUID `json:"job_id"`
	Status string `json:"status"`
	TotalCount int `json:"total_count"`
	ProcessedCount int `json:"processed_count"`
	ImportedCount int `json:"imported_count"`
	FailedCount int `json:"failed_count"`
	SkippedCount int `json:"skipped_count"`
	LastError string `json:"last_error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

func (e *Engine) CreateImportJob(ctx context.Context, tenantID uuid.UUID, fileName, payload string) (*ImportJob, error) {
	lines := splitImportLines(payload)
	if len(lines) == 0 { return nil, fmt.Errorf("没有找到可导入的链接") }
	var job ImportJob
	err := e.pool.QueryRow(ctx, `
		INSERT INTO group_library_import_jobs (tenant_id, file_name, payload, total_count)
		VALUES ($1, $2, $3, $4)
		RETURNING job_id, status, total_count, processed_count, imported_count,
		          failed_count, skipped_count, created_at, finished_at`,
		tenantID, fileName, payload, len(lines)).Scan(
		&job.JobID, &job.Status, &job.TotalCount, &job.ProcessedCount,
		&job.ImportedCount, &job.FailedCount, &job.SkippedCount, &job.CreatedAt, &job.FinishedAt)
	if err != nil { return nil, err }
	return &job, nil
}

func (e *Engine) GetImportJob(ctx context.Context, tenantID, jobID uuid.UUID) (*ImportJob, error) {
	var job ImportJob
	err := e.pool.QueryRow(ctx, `
		SELECT job_id, status, total_count, processed_count, imported_count,
		       failed_count, skipped_count, last_error, created_at, finished_at
		FROM group_library_import_jobs WHERE tenant_id=$1 AND job_id=$2`,
		tenantID, jobID).Scan(
		&job.JobID, &job.Status, &job.TotalCount, &job.ProcessedCount,
		&job.ImportedCount, &job.FailedCount, &job.SkippedCount,
		&job.LastError, &job.CreatedAt, &job.FinishedAt)
	if err != nil { return nil, err }
	return &job, nil
}

func (e *Engine) processImportJobs(ctx context.Context) {
	for batch := 0; batch < 10; batch++ {
		var jobID, tenantID uuid.UUID
		var payload string
		var offset, total int
		err := e.pool.QueryRow(ctx, `
			UPDATE group_library_import_jobs
			SET status='processing', started_at=COALESCE(started_at, NOW()), updated_at=NOW()
			WHERE job_id = (
				SELECT job_id FROM group_library_import_jobs
				WHERE status='pending'
				   OR (status='processing' AND updated_at < NOW() - INTERVAL '2 minutes')
				ORDER BY created_at
				LIMIT 1
				FOR UPDATE SKIP LOCKED
			)
			RETURNING job_id, tenant_id, payload, processed_count, total_count`).Scan(
			&jobID, &tenantID, &payload, &offset, &total)
		if err != nil { return }
		lines := splitImportLines(payload)
		if offset >= len(lines) {
			_, _ = e.pool.Exec(ctx, `UPDATE group_library_import_jobs SET status='completed', finished_at=NOW(), updated_at=NOW() WHERE job_id=$1`, jobID)
			continue
		}
		end := offset + 200
		if end > len(lines) { end = len(lines) }
		result, importErr := e.ImportByLinks(ctx, tenantID, uuid.Nil, lines[offset:end])
		imported, failed, skipped := 0, 0, 0
		lastError := ""
		if result != nil { imported, failed, skipped = len(result.Imported), len(result.Failed), result.Skipped }
		if importErr != nil { lastError = importErr.Error() }
		nextStatus := "processing"
		var finished any
		if end >= len(lines) { nextStatus = "completed"; finished = time.Now() }
		_, updateErr := e.pool.Exec(ctx, `
			UPDATE group_library_import_jobs
			SET processed_count=$2, imported_count=imported_count+$3,
			    failed_count=failed_count+$4, skipped_count=skipped_count+$5,
			    last_error=CASE WHEN $6='' THEN last_error ELSE $6 END,
			    status=$7, finished_at=$8, updated_at=NOW()
			WHERE job_id=$1`, jobID, end, imported, failed, skipped, lastError, nextStatus, finished)
		if updateErr != nil { return }
	}
}

func splitImportLines(payload string) []string {
	return strings.FieldsFunc(payload, func(r rune) bool {
		switch r {
		case 10, 13, 32, 9, ',', '；', ';':
			return true
		default:
			return false
		}
	})
}
