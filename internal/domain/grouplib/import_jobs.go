package grouplib

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ImportJob struct {
	JobID          uuid.UUID  `json:"job_id"`
	RawCount       int        `json:"raw_count"`
	ValidCount     int        `json:"valid_count"`
	DuplicateCount int        `json:"duplicate_count"`
	InvalidCount   int        `json:"invalid_count"`
	FileName       string     `json:"file_name"`
	Status         string     `json:"status"`
	TotalCount     int        `json:"total_count"`
	ProcessedCount int        `json:"processed_count"`
	ImportedCount  int        `json:"imported_count"`
	FailedCount    int        `json:"failed_count"`
	SkippedCount   int        `json:"skipped_count"`
	LastError      string     `json:"last_error,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
}

// CreateImportPreview 先做纯格式筛选并保存为待确认任务；此阶段不会访问 Telegram 或写入群库。
func (e *Engine) CreateImportPreview(ctx context.Context, tenantID uuid.UUID, fileName, payload string) (*ImportJob, error) {
	filtered := filterImportLinks(payload)
	if len(filtered.Links) == 0 {
		return nil, fmt.Errorf("没有筛选出有效的 Telegram 群组链接")
	}
	skipped := filtered.Duplicate + filtered.Invalid
	summary := fmt.Sprintf("筛选完成：原始 %d 条，格式有效 %d 条，重复 %d 条，无效 %d 条。确认后才开始解析。",
		filtered.RawCount, len(filtered.Links), filtered.Duplicate, filtered.Invalid)
	var job ImportJob
	err := e.pool.QueryRow(ctx, `
		INSERT INTO group_library_import_jobs
			(tenant_id, file_name, payload, status, total_count, skipped_count, raw_count, duplicate_count, invalid_count, last_error)
		VALUES ($1, $2, $3, 'review', $4, $5, $6, $7, $8, $9)
		RETURNING job_id, file_name, status, total_count, processed_count, imported_count,
		          failed_count, skipped_count, raw_count, duplicate_count, invalid_count, last_error, created_at, finished_at`,
		tenantID, fileName, strings.Join(filtered.Links, "\n"), len(filtered.Links), skipped,
		filtered.RawCount, filtered.Duplicate, filtered.Invalid, summary).Scan(
		&job.JobID, &job.FileName, &job.Status, &job.TotalCount, &job.ProcessedCount,
		&job.ImportedCount, &job.FailedCount, &job.SkippedCount, &job.RawCount,
		&job.DuplicateCount, &job.InvalidCount, &job.LastError, &job.CreatedAt, &job.FinishedAt)
	if err != nil {
		return nil, err
	}
	job.ValidCount = job.TotalCount
	return &job, nil
}

// ConfirmImportJob 只有用户确认筛选结果后，任务才进入后台解析队列。
func (e *Engine) ConfirmImportJob(ctx context.Context, tenantID, jobID uuid.UUID) (*ImportJob, error) {
	var job ImportJob
	err := e.pool.QueryRow(ctx, `
		UPDATE group_library_import_jobs
		SET status='pending', last_error='', updated_at=NOW()
		WHERE tenant_id=$1 AND job_id=$2 AND status='review'
		RETURNING job_id,file_name,status,total_count,processed_count,imported_count,failed_count,
		          skipped_count,raw_count,duplicate_count,invalid_count,last_error,created_at,finished_at`,
		tenantID, jobID).Scan(
		&job.JobID, &job.FileName, &job.Status, &job.TotalCount, &job.ProcessedCount,
		&job.ImportedCount, &job.FailedCount, &job.SkippedCount, &job.RawCount,
		&job.DuplicateCount, &job.InvalidCount, &job.LastError, &job.CreatedAt, &job.FinishedAt)
	if err != nil {
		return nil, err
	}
	job.ValidCount = job.TotalCount
	return &job, nil
}

func (e *Engine) CreateImportJob(ctx context.Context, tenantID uuid.UUID, fileName, payload string) (*ImportJob, error) {
	lines := splitImportLines(payload)
	if len(lines) == 0 {
		return nil, fmt.Errorf("没有找到可导入的链接")
	}
	var job ImportJob
	err := e.pool.QueryRow(ctx, `
		INSERT INTO group_library_import_jobs (tenant_id, file_name, payload, total_count)
		VALUES ($1, $2, $3, $4)
		RETURNING job_id, file_name, status, total_count, processed_count, imported_count,
		          failed_count, skipped_count, created_at, finished_at`,
		tenantID, fileName, payload, len(lines)).Scan(
		&job.JobID, &job.FileName, &job.Status, &job.TotalCount, &job.ProcessedCount,
		&job.ImportedCount, &job.FailedCount, &job.SkippedCount, &job.CreatedAt, &job.FinishedAt)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (e *Engine) ListImportJobs(ctx context.Context, tenantID uuid.UUID, limit int) ([]ImportJob, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := e.pool.Query(ctx, `SELECT job_id,file_name,status,total_count,processed_count,imported_count,failed_count,skipped_count,raw_count,duplicate_count,invalid_count,last_error,created_at,finished_at FROM group_library_import_jobs WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := make([]ImportJob, 0)
	for rows.Next() {
		var job ImportJob
		if err := rows.Scan(&job.JobID, &job.FileName, &job.Status, &job.TotalCount, &job.ProcessedCount, &job.ImportedCount, &job.FailedCount, &job.SkippedCount, &job.RawCount, &job.DuplicateCount, &job.InvalidCount, &job.LastError, &job.CreatedAt, &job.FinishedAt); err != nil {
			return nil, err
		}
		job.ValidCount = job.TotalCount
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (e *Engine) GetImportJob(ctx context.Context, tenantID, jobID uuid.UUID) (*ImportJob, error) {
	var job ImportJob
	err := e.pool.QueryRow(ctx, `
		SELECT job_id, file_name, status, total_count, processed_count, imported_count,
		       failed_count, skipped_count, raw_count, duplicate_count, invalid_count, last_error, created_at, finished_at
		FROM group_library_import_jobs WHERE tenant_id=$1 AND job_id=$2`,
		tenantID, jobID).Scan(
		&job.JobID, &job.FileName, &job.Status, &job.TotalCount, &job.ProcessedCount,
		&job.ImportedCount, &job.FailedCount, &job.SkippedCount, &job.RawCount, &job.DuplicateCount, &job.InvalidCount,
		&job.LastError, &job.CreatedAt, &job.FinishedAt)
	if err != nil {
		return nil, err
	}
	job.ValidCount = job.TotalCount
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
		if err != nil {
			return
		}
		lines := splitImportLines(payload)
		if offset >= len(lines) {
			_, _ = e.pool.Exec(ctx, `UPDATE group_library_import_jobs SET status='completed', finished_at=NOW(), updated_at=NOW() WHERE job_id=$1`, jobID)
			continue
		}
		end := offset + 30
		if end > len(lines) {
			end = len(lines)
		}
		result, importErr := e.ImportByLinks(ctx, tenantID, uuid.Nil, lines[offset:end])
		imported, failed, skipped := 0, 0, 0
		lastError := ""
		if result != nil {
			imported, failed, skipped = len(result.Imported), len(result.Failed), result.Skipped
		}
		if importErr != nil {
			lastError = importErr.Error()
		}
		nextStatus := "processing"
		var finished any
		if end >= len(lines) {
			nextStatus = "completed"
			finished = time.Now()
		}
		_, updateErr := e.pool.Exec(ctx, `
			UPDATE group_library_import_jobs
			SET processed_count=$2, imported_count=imported_count+$3,
			    failed_count=failed_count+$4, skipped_count=skipped_count+$5,
			    last_error=CASE WHEN $6='' THEN last_error ELSE $6 END,
			    status=$7, finished_at=$8, payload=CASE WHEN $7='completed' THEN '' ELSE payload END, updated_at=NOW()
			WHERE job_id=$1`, jobID, end, imported, failed, skipped, lastError, nextStatus, finished)
		if updateErr != nil {
			return
		}
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
