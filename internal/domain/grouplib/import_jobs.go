package grouplib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ImportJob struct {
	JobID          uuid.UUID  `json:"job_id"`
	TaskNumber     int64      `json:"task_number"`
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
	return e.CreateImportPreviewWithAccount(ctx, tenantID, uuid.Nil, fileName, payload)
}

// CreateImportPreviewWithAccount stores an optional tenant-owned account for MTProto-assisted resolution.
func (e *Engine) CreateImportPreviewWithAccount(ctx context.Context, tenantID, accountID uuid.UUID, fileName, payload string) (*ImportJob, error) {
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
			(tenant_id, account_id, file_name, payload, status, total_count, skipped_count, raw_count, duplicate_count, invalid_count, last_error)
		VALUES ($1, NULLIF($2, '00000000-0000-0000-0000-000000000000'::uuid), $3, $4, 'review', $5, $6, $7, $8, $9, $10)
		RETURNING job_id, file_name, status, total_count, processed_count, imported_count,
		          failed_count, skipped_count, raw_count, duplicate_count, invalid_count, last_error, created_at, finished_at`,
		tenantID, accountID, fileName, strings.Join(filtered.Links, "\n"), len(filtered.Links), skipped,
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

// CancelImportJob requests cancellation while preserving the active worker's fencing token.
// The worker observes cancelling, stops its resolver context, and finalizes the job.
func (e *Engine) CancelImportJob(ctx context.Context, tenantID, jobID uuid.UUID) (*ImportJob, error) {
	var job ImportJob
	err := e.pool.QueryRow(ctx, `
		UPDATE group_library_import_jobs
		SET status=CASE WHEN status='processing' THEN 'cancelling' ELSE 'cancelled' END,
		    lease_expires_at=CASE WHEN status='processing' THEN NOW() ELSE lease_expires_at END,
		    finished_at=CASE WHEN status='processing' THEN finished_at ELSE NOW() END,
		    updated_at=NOW()
		WHERE tenant_id=$1 AND job_id=$2 AND status IN ('review','pending','processing')
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

// RetryFailedImportJob creates a new job containing only failures that have not already been retried.
func (e *Engine) RetryFailedImportJob(ctx context.Context, tenantID, jobID uuid.UUID) (*ImportJob, error) {
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	// Serialize retries for the same source job so concurrent clicks cannot enqueue duplicate retry batches.
	var sourceJobID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT job_id FROM group_library_import_jobs WHERE tenant_id=$1 AND job_id=$2 AND status IN ('completed','failed','cancelled') FOR UPDATE`, tenantID, jobID).Scan(&sourceJobID); err != nil {
		return nil, fmt.Errorf("原导入任务不存在: %w", err)
	}
	var job ImportJob
	err = tx.QueryRow(ctx, `
		WITH failed AS (
			SELECT string_agg(original_link, E'\n' ORDER BY created_at) AS payload, COUNT(*)::int AS total
			FROM group_library_import_job_failures WHERE job_id=$2 AND retry_job_id IS NULL
		), source AS (
			SELECT tenant_id, account_id, file_name FROM group_library_import_jobs WHERE tenant_id=$1 AND job_id=$2
		)
		INSERT INTO group_library_import_jobs (tenant_id,account_id,file_name,payload,status,total_count,raw_count)
		SELECT source.tenant_id, source.account_id, source.file_name, failed.payload, 'pending', failed.total, failed.total
		FROM source CROSS JOIN failed WHERE failed.total > 0
		RETURNING job_id,file_name,status,total_count,processed_count,imported_count,failed_count,
		          skipped_count,raw_count,duplicate_count,invalid_count,last_error,created_at,finished_at
	`, tenantID, jobID).Scan(
		&job.JobID, &job.FileName, &job.Status, &job.TotalCount, &job.ProcessedCount,
		&job.ImportedCount, &job.FailedCount, &job.SkippedCount, &job.RawCount,
		&job.DuplicateCount, &job.InvalidCount, &job.LastError, &job.CreatedAt, &job.FinishedAt)
	if err != nil {
		return nil, fmt.Errorf("没有可重试的失败链接或原任务不存在: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE group_library_import_job_failures SET retry_job_id=$3, updated_at=NOW() WHERE job_id=$1 AND retry_job_id IS NULL AND EXISTS (SELECT 1 FROM group_library_import_jobs WHERE job_id=$3 AND tenant_id=$2)`, jobID, tenantID, job.JobID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	job.ValidCount = job.TotalCount
	return &job, nil
}

func importFailureLink(detail string) string {
	closeBracket := strings.Index(detail, "] ")
	if closeBracket < 0 {
		return ""
	}
	value := detail[closeBracket+2:]
	separator := strings.Index(value, ": ")
	if separator < 0 {
		return ""
	}
	link := strings.TrimSpace(value[:separator])
	if strings.HasPrefix(link, "https://t.me/") || strings.HasPrefix(link, "http://t.me/") || strings.HasPrefix(link, "tg://") {
		return link
	}
	return ""
}

func (e *Engine) persistImportFailures(ctx context.Context, jobID uuid.UUID, failures []string) error {
	if len(failures) == 0 {
		return nil
	}
	links := make([]string, 0, len(failures))
	details := make([]string, 0, len(failures))
	for _, detail := range failures {
		link := importFailureLink(detail)
		if link == "" {
			continue
		}
		links = append(links, link)
		details = append(details, detail)
	}
	if len(links) == 0 {
		return nil
	}
	// Persist a whole worker chunk in one round trip instead of issuing one
	// INSERT per failed link. UNNEST keeps the two arrays positionally paired.
	_, err := e.pool.Exec(ctx, `
		INSERT INTO group_library_import_job_failures (job_id, original_link, error_text)
		SELECT $1, item.original_link, item.error_text
		FROM UNNEST($2::text[], $3::text[]) AS item(original_link, error_text)
		ON CONFLICT (job_id, original_link) DO UPDATE
		SET error_text=EXCLUDED.error_text, updated_at=NOW()
	`, jobID, links, details)
	return err
}

func (e *Engine) CreateImportJob(ctx context.Context, tenantID uuid.UUID, fileName, payload string) (*ImportJob, error) {
	// Keep the legacy one-step API on the same normalized payload path as the
	// preview/confirm flow. Worker offsets are candidate-line offsets, so raw
	// invalid, duplicate, and bot lines must never be stored in the job payload.
	filtered := filterImportLinks(payload)
	if len(filtered.Links) == 0 {
		return nil, fmt.Errorf("没有筛选出有效的 Telegram 群组链接")
	}
	skipped := filtered.Duplicate + filtered.Invalid
	var job ImportJob
	err := e.pool.QueryRow(ctx, `
		INSERT INTO group_library_import_jobs
			(tenant_id, file_name, payload, total_count, skipped_count, raw_count, duplicate_count, invalid_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING job_id, file_name, status, total_count, processed_count, imported_count,
		          failed_count, skipped_count, raw_count, duplicate_count, invalid_count, created_at, finished_at`,
		tenantID, fileName, strings.Join(filtered.Links, "\n"), len(filtered.Links), skipped,
		filtered.RawCount, filtered.Duplicate, filtered.Invalid).Scan(
		&job.JobID, &job.FileName, &job.Status, &job.TotalCount, &job.ProcessedCount,
		&job.ImportedCount, &job.FailedCount, &job.SkippedCount, &job.RawCount,
		&job.DuplicateCount, &job.InvalidCount, &job.CreatedAt, &job.FinishedAt)
	if err != nil {
		return nil, err
	}
	job.ValidCount = job.TotalCount
	return &job, nil
}

// DeleteImportJob removes only an inactive import-job history row owned by the tenant.
// Active or leased jobs cannot be deleted; failure details are removed by the table's FK cascade.
func (e *Engine) DeleteImportJob(ctx context.Context, tenantID, jobID uuid.UUID) error {
	tag, err := e.pool.Exec(ctx, `
		DELETE FROM group_library_import_jobs
		WHERE tenant_id=$1 AND job_id=$2
		  AND status IN ('review','completed','failed','cancelled')
		  AND (lease_token IS NULL OR lease_expires_at < NOW())`,
		tenantID, jobID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("导入任务不存在、仍在运行或租约尚未结束")
	}
	return nil
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

type importBatchProgress struct {
	processed int
	imported  int
	failed    int
	skipped   int
	detail    string
}

// summarizeImportBatch converts ImportByLinks results into durable job counters.
// Import jobs store only pre-filtered candidate links. On 429, ImportByLinks'
// Skipped value may describe the whole parsed slice (including lines after the
// rate-limited item), so only completed-prefix counters are safe to persist.
func summarizeImportBatch(result *ImportResult, rateLimited bool) importBatchProgress {
	var progress importBatchProgress
	if result == nil {
		return progress
	}
	progress.processed = result.Processed
	progress.imported = len(result.Imported)
	progress.failed = len(result.Failed)
	progress.skipped = result.Skipped
	if rateLimited {
		progress.skipped = 0
		if progress.failed > 0 {
			progress.failed-- // the final failure is the current, unprocessed 429 item
		}
	}
	detailCount := progress.failed
	if detailCount > len(result.Failed) {
		detailCount = len(result.Failed)
	}
	if detailCount > 0 {
		progress.detail = strings.Join(result.Failed[:detailCount], "\n")
	}
	return progress
}

func (e *Engine) importWorkerBatch(ctx context.Context, tenantID, accountID uuid.UUID, lines []string) (*ImportResult, error) {
	if e.workerImportOverride != nil {
		return e.workerImportOverride(ctx, tenantID, accountID, lines)
	}
	return e.importByLinks(ctx, tenantID, accountID, lines, true)
}

const importJobHeartbeatInterval = 30 * time.Second

// processImportJobs 使用数据库租约领取任务。租约令牌同时作为 fencing token，
// 所有进度/统计写入都必须匹配当前令牌和预期 offset，防止过期 Worker 覆盖新 Worker。
func (e *Engine) processImportJobs(ctx context.Context) {
	// A worker may die after a cancellation request. Finalize abandoned cancelling leases after expiry.
	_, _ = e.pool.Exec(ctx, `UPDATE group_library_import_jobs SET status='cancelled', finished_at=NOW(), lease_token=NULL, lease_expires_at=NULL, updated_at=NOW() WHERE status='cancelling' AND (lease_expires_at IS NULL OR lease_expires_at < NOW())`)
	claimedJobIDs := make([]uuid.UUID, 0, 10)
	for jobBatch := 0; jobBatch < 10; jobBatch++ {
		var jobID, tenantID, accountID, leaseToken uuid.UUID
		var payload string
		var offset, total int
		err := e.pool.QueryRow(ctx, `
			UPDATE group_library_import_jobs
			SET status='processing', lease_token=gen_random_uuid(),
			    lease_expires_at=NOW() + INTERVAL '2 minutes',
			    started_at=COALESCE(started_at, NOW()), updated_at=NOW()
			WHERE job_id = (
				SELECT job_id FROM group_library_import_jobs
				WHERE (status='pending'
				   OR (status='processing' AND
				       (lease_expires_at IS NULL OR lease_expires_at < NOW())))
				  AND NOT (job_id = ANY($1::uuid[]))
				ORDER BY created_at
				LIMIT 1
				FOR UPDATE SKIP LOCKED
			)
			RETURNING job_id, tenant_id, COALESCE(account_id, '00000000-0000-0000-0000-000000000000'::uuid), payload, processed_count, total_count, lease_token`, claimedJobIDs).Scan(
			&jobID, &tenantID, &accountID, &payload, &offset, &total, &leaseToken)
		if err != nil {
			return
		}

		claimedJobIDs = append(claimedJobIDs, jobID)
		leaseCtx, stopLease := context.WithCancel(ctx)
		leaseLost := make(chan struct{})
		heartbeatDone := make(chan struct{})
		go func() {
			defer close(heartbeatDone)
			ticker := time.NewTicker(importJobHeartbeatInterval)
			defer ticker.Stop()
			for {
				select {
				case <-leaseCtx.Done():
					return
				case <-ticker.C:
					renewCtx, cancel := context.WithTimeout(leaseCtx, 5*time.Second)
					tag, renewErr := e.pool.Exec(renewCtx, `
						UPDATE group_library_import_jobs
						SET lease_expires_at=NOW() + INTERVAL '2 minutes', updated_at=NOW()
						WHERE job_id=$1 AND status IN ('processing','cancelling') AND lease_token=$2`,
						jobID, leaseToken)
					cancel()
					if renewErr != nil || tag.RowsAffected() != 1 {
						close(leaseLost)
						stopLease()
						return
					}
				}
			}
		}()

		cancelWatchDone := make(chan struct{})
		go func() {
			defer close(cancelWatchDone)
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-leaseCtx.Done():
					return
				case <-ticker.C:
					var status, token string
					checkCtx, cancel := context.WithTimeout(leaseCtx, 3*time.Second)
					err := e.pool.QueryRow(checkCtx, `SELECT status, COALESCE(lease_token::text,'') FROM group_library_import_jobs WHERE job_id=$1`, jobID).Scan(&status, &token)
					if err == nil && status == "cancelling" && token == leaseToken.String() {
						cancel()
						stopLease()
						return
					}
					if err == nil && (status != "processing" || token != leaseToken.String()) {
						cancel()
						stopLease()
						return
					}
					cancel()
				}
			}
		}()
		lines := splitImportLines(payload)
		if offset >= len(lines) {
			_, _ = e.pool.Exec(leaseCtx, `
				UPDATE group_library_import_jobs
				SET status=CASE WHEN status='cancelling' THEN 'cancelled' ELSE 'completed' END,
				    finished_at=NOW(), payload='', lease_token=NULL,
				    lease_expires_at=NULL, updated_at=NOW()
				WHERE job_id=$1 AND status IN ('processing','cancelling') AND lease_token=$2 AND processed_count=$3`,
				jobID, leaseToken, offset)
			stopLease()
			<-heartbeatDone
			<-cancelWatchDone
			continue
		}

		leaseValid := func() bool {
			select {
			case <-leaseLost:
				return false
			default:
				return true
			}
		}
		deferUntilLeaseExpiry := false
		for chunk := 0; chunk < 1 && offset < len(lines); chunk++ {
			if leaseCtx.Err() != nil {
				break
			}
			end := offset + 30
			if end > len(lines) {
				end = len(lines)
			}
			// 公开链接优先共享网页解析；选择账号时，私密邀请及网页失败项由 MTProto 补充解析。
			result, importErr := e.importWorkerBatch(leaseCtx, tenantID, accountID, lines[offset:end])
			if importErr != nil && accountID != uuid.Nil && strings.Contains(importErr.Error(), "取账号连接失败") && leaseCtx.Err() == nil {
				log.Printf("[grouplib] 导入账号暂不可用，任务 %s 本批降级为网页解析: %v", jobID, importErr)
				result, importErr = e.importWorkerBatch(leaseCtx, tenantID, uuid.Nil, lines[offset:end])
			}

			var rateLimitErr *TelegramRateLimitError
			if errors.As(importErr, &rateLimitErr) {
				progress := summarizeImportBatch(result, true)
				// A rate-limit result is a durable checkpoint. The resolver lease context
				// may already be cancelled during shutdown; persist with a short detached
				// context while retaining the fencing-token and expected-offset checks.
				persistCtx, persistCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
				if result != nil && progress.failed > 0 {
					if err := e.persistImportFailures(persistCtx, jobID, result.Failed[:progress.failed]); err != nil {
						persistCancel()
						log.Printf("[grouplib] 保存失败链接失败 job=%s: %v", jobID, err)
						break
					}
				}
				partialImported, partialFailed, partialSkipped := progress.imported, progress.failed, progress.skipped
				processedPrefix, detail := progress.processed, progress.detail
				if processedPrefix > 0 {
					tag, saveErr := e.pool.Exec(persistCtx, `
						UPDATE group_library_import_jobs
						SET processed_count=processed_count+$3,
							imported_count=imported_count+$4,
							failed_count=failed_count+$5,
							skipped_count=skipped_count+$6,
							last_error=CASE WHEN $7='' THEN last_error
							                WHEN last_error='' THEN $7
							                ELSE LEFT(last_error || E'\\n' || $7, 4000) END,
							updated_at=NOW()
						WHERE job_id=$1 AND status IN ('processing','cancelling') AND lease_token=$2
						  AND processed_count=$8`,
						jobID, leaseToken, processedPrefix, partialImported, partialFailed,
						partialSkipped, detail, offset)
					if saveErr != nil || tag.RowsAffected() != 1 {
						persistCancel()
						break
					}
					offset += processedPrefix
				}
				persistCancel()
				// Do not block the shared scheduler while waiting for Telegram's RetryAfter.
				// Keep the fenced lease until the cooldown expires; other jobs can progress.
				deferUntilLeaseExpiry = true
				deferCtx, deferCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				_, deferErr := e.pool.Exec(deferCtx, `
					UPDATE group_library_import_jobs
					SET lease_expires_at=NOW()+($3 * INTERVAL '1 second'), updated_at=NOW()
					WHERE job_id=$1 AND status='processing' AND lease_token=$2`,
					jobID, leaseToken, rateLimitErr.RetryAfter.Seconds())
				deferCancel()
				if deferErr != nil {
					log.Printf("[grouplib] 延迟限流任务租约失败 job=%s: %v", jobID, deferErr)
				}
				break
			}

			if leaseCtx.Err() != nil {
				if !leaseValid() {
					break
				}
				// 优雅停机：用独立短上下文保存已完成前缀；CAS 防止重复累计。
				progress := summarizeImportBatch(result, false)
				processed, imported, failed, skipped, detail := progress.processed, progress.imported, progress.failed, progress.skipped, progress.detail
				persistCtx, persistCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
				if result != nil && progress.failed > 0 {
					if err := e.persistImportFailures(persistCtx, jobID, result.Failed[:progress.failed]); err != nil {
						log.Printf("[grouplib] 保存中断任务失败链接失败 job=%s: %v", jobID, err)
					}
				}
				_, persistErr := e.pool.Exec(persistCtx, `
					UPDATE group_library_import_jobs
					SET processed_count=processed_count+$3,
					    imported_count=imported_count+$4,
					    failed_count=failed_count+$5,
					    skipped_count=skipped_count+$6,
					    last_error=CASE WHEN $7='' THEN last_error ELSE LEFT($7, 4000) END,
					    status=CASE WHEN status='cancelling' THEN 'cancelled' ELSE 'pending' END,
					    finished_at=CASE WHEN status='cancelling' THEN NOW() ELSE finished_at END,
					    lease_token=NULL, lease_expires_at=NULL, updated_at=NOW()
					WHERE job_id=$1 AND status IN ('processing','cancelling') AND lease_token=$2
					  AND processed_count=$8`,
					jobID, leaseToken, processed, imported, failed, skipped, detail, offset)
				persistCancel()
				if persistErr != nil {
					log.Printf("[grouplib] 保存中断导入进度失败 job=%s: %v", jobID, persistErr)
				}
				stopLease()
				<-heartbeatDone
				<-cancelWatchDone
				return
			}

			if !leaseValid() {
				break
			}
			progress := summarizeImportBatch(result, false)
			if result != nil && progress.failed > 0 {
				if err := e.persistImportFailures(leaseCtx, jobID, result.Failed); err != nil {
					log.Printf("[grouplib] 保存失败链接失败 job=%s: %v", jobID, err)
					break
				}
			}
			imported, failed, skipped := progress.imported, progress.failed, progress.skipped
			lastError := progress.detail
			if result == nil && importErr != nil {
				failed = end - offset
				syntheticFailures := make([]string, 0, failed)
				for _, link := range lines[offset:end] {
					syntheticFailures = append(syntheticFailures, fmt.Sprintf("[transient] %s: %v", link, importErr))
				}
				if err := e.persistImportFailures(leaseCtx, jobID, syntheticFailures); err != nil {
					log.Printf("[grouplib] 保存批次级失败链接失败 job=%s: %v", jobID, err)
					break
				}
			}
			if len(lastError) > 4000 {
				lastError = lastError[:4000] + "…（错误明细已截断）"
			}
			if importErr != nil {
				lastError = strings.TrimSpace(lastError + "\n" + importErr.Error())
				if len(lastError) > 4000 {
					lastError = lastError[:4000] + "…（错误明细已截断）"
				}
			}
			nextStatus := "processing"
			var finished any
			if end >= len(lines) {
				nextStatus = "completed"
				finished = time.Now()
			}
			tag, updateErr := e.pool.Exec(leaseCtx, `
				UPDATE group_library_import_jobs
				SET processed_count=$3, imported_count=imported_count+$4,
				    failed_count=failed_count+$5, skipped_count=skipped_count+$6,
				    last_error=CASE WHEN $7='' THEN last_error ELSE $7 END,
				    status=CASE WHEN status='cancelling' THEN 'cancelled' ELSE $8 END,
				    finished_at=CASE WHEN status='cancelling' THEN NOW() ELSE $9 END,
				    payload=CASE WHEN status='cancelling' OR $8='completed' THEN '' ELSE payload END,
				    lease_token=CASE WHEN status='cancelling' OR $8='completed' THEN NULL ELSE lease_token END,
				    lease_expires_at=CASE WHEN status='cancelling' OR $8='completed' THEN NULL ELSE lease_expires_at END,
				    updated_at=NOW()
				WHERE job_id=$1 AND status IN ('processing','cancelling') AND lease_token=$2
				  AND processed_count=$10`,
				jobID, leaseToken, end, imported, failed, skipped, lastError,
				nextStatus, finished, offset)
			if updateErr != nil || tag.RowsAffected() != 1 {
				break
			}
			offset = end
			if nextStatus == "completed" {
				break
			}
		}

		if leaseCtx.Err() != nil && leaseValid() {
			persistCtx, persistCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			_, _ = e.pool.Exec(persistCtx, `UPDATE group_library_import_jobs SET status='cancelled', finished_at=NOW(), lease_token=NULL, lease_expires_at=NULL, updated_at=NOW() WHERE job_id=$1 AND status='cancelling' AND lease_token=$2`, jobID, leaseToken)
			persistCancel()
		}
		// 每个任务每轮只处理一个分片，未完成任务立即释放回 pending。
		if leaseValid() && leaseCtx.Err() == nil && !deferUntilLeaseExpiry {
			_, _ = e.pool.Exec(leaseCtx, `
				UPDATE group_library_import_jobs
				SET status='pending', lease_token=NULL, lease_expires_at=NULL, updated_at=NOW()
				WHERE job_id=$1 AND status='processing' AND lease_token=$2`,
				jobID, leaseToken)
		}
		stopLease()
		<-heartbeatDone
		<-cancelWatchDone
		// The next claimed job gets a turn in this scheduler pass; this job resumes later.
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
