package grouplib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
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
	// Process at most one chunk per job per scheduler pass. Re-queuing a partial
	// job must not let the oldest large import monopolize all ten claim slots.
	claimedJobIDs := make([]uuid.UUID, 0, 10)
	for batch := 0; batch < 10; batch++ {
		var jobID, tenantID uuid.UUID
		var payload string
		var offset, total int
		leaseToken := uuid.New()
		err := e.pool.QueryRow(ctx, `
			UPDATE group_library_import_jobs
			SET status='processing', started_at=COALESCE(started_at, NOW()), updated_at=NOW(), lease_token=$1
			WHERE job_id = (
				SELECT job_id FROM group_library_import_jobs
				WHERE (status='pending'
				   OR (status='processing' AND updated_at < NOW() - INTERVAL '2 minutes'))
				  AND NOT (job_id = ANY($2::uuid[]))
				ORDER BY created_at
				LIMIT 1
				FOR UPDATE SKIP LOCKED
			)
			RETURNING job_id, tenant_id, payload, processed_count, total_count`, leaseToken, claimedJobIDs).Scan(
			&jobID, &tenantID, &payload, &offset, &total)
		if err != nil {
			return
		}
		claimedJobIDs = append(claimedJobIDs, jobID)
		lines := splitImportLines(payload)
		if offset >= len(lines) {
			_, _ = e.pool.Exec(ctx, `UPDATE group_library_import_jobs SET status='completed', finished_at=NOW(), updated_at=NOW(), lease_token=NULL WHERE job_id=$1 AND lease_token=$2`, jobID, leaseToken)
			continue
		}
		end := offset + 30
		if end > len(lines) {
			end = len(lines)
		}
		// Keep a per-claim lease alive while the external page resolver processes
		// this chunk. Without a heartbeat, a 30-link chunk can exceed the 2-minute
		// stale threshold and another process may execute the same chunk concurrently.
		batchCtx, cancelBatch := context.WithCancel(ctx)
		var leaseLost atomic.Bool
		heartbeatStop := make(chan struct{})
		heartbeatDone := make(chan struct{})
		go func() {
			defer close(heartbeatDone)
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-heartbeatStop:
					return
				case <-ticker.C:
					heartbeatCtx, heartbeatCancel := context.WithTimeout(context.Background(), 5*time.Second)
					tag, heartbeatErr := e.pool.Exec(heartbeatCtx, `
						UPDATE group_library_import_jobs SET updated_at=NOW()
						WHERE job_id=$1 AND lease_token=$2 AND status='processing'`, jobID, leaseToken)
					heartbeatCancel()
					if heartbeatErr != nil || tag.RowsAffected() != 1 {
						leaseLost.Store(true)
						cancelBatch()
						if heartbeatErr != nil {
							log.Printf("[grouplib] 导入任务租约续期失败 job=%s: %v", jobID, heartbeatErr)
						} else {
							log.Printf("[grouplib] 导入任务租约已失效 job=%s", jobID)
						}
					return
					}
				}
			}
		}()
		// 群组库导入默认走无账号的共享 HTTP 页面解析器，不读取、不占用租户 Telegram 会话。
		// 公开页面无法提供资料时记录失败原因；不自动切换账号，避免后台导入隐式消耗账号额度。
		result, importErr := e.ImportByLinks(batchCtx, tenantID, uuid.Nil, lines[offset:end])
		close(heartbeatStop)
		<-heartbeatDone
		cancelBatch()
		if leaseLost.Load() {
			// Another worker owns the row now (or the lease store is unavailable).
			// Never persist progress from this stale worker.
			continue
		}
		var rateLimitErr *TelegramRateLimitError
		if errors.As(importErr, &rateLimitErr) {
			// 429 不计为失败；只提交限流前已处理的前缀，offset 停在被限流链接。
			partialImported, partialFailed, partialSkipped := 0, 0, 0
			if result != nil {
				partialImported = len(result.Imported)
				partialFailed = len(result.Failed)
				partialSkipped = result.Skipped
			}
			if partialFailed > 0 {
				partialFailed-- // 最后一条失败就是当前 429 链接，必须留待重试。
			}
			processedPrefix := 0
			if result != nil {
				processedPrefix = result.Processed
			}
			if processedPrefix > 0 {
				var detail string
				if result != nil && partialFailed > 0 {
					detail = strings.Join(result.Failed[:partialFailed], "\n")
				}
				_, _ = e.pool.Exec(ctx, `
					UPDATE group_library_import_jobs
					SET processed_count=processed_count+$2,
					    imported_count=imported_count+$3,
					    failed_count=failed_count+$4,
					    skipped_count=skipped_count+$5,
					    last_error=CASE WHEN $6='' THEN last_error
					                    WHEN last_error='' THEN $6
					                    ELSE LEFT(last_error || E'\\n' || $6, 4000) END,
					    updated_at=$7
					WHERE job_id=$1 AND lease_token=$8 AND status='processing'`, jobID, processedPrefix, partialImported, partialFailed,
					partialSkipped, detail, time.Now().Add(rateLimitErr.RetryAfter), leaseToken)
			} else {
				_, _ = e.pool.Exec(ctx, `UPDATE group_library_import_jobs SET updated_at=$2 WHERE job_id=$1 AND lease_token=$3 AND status='processing'`,
					jobID, time.Now().Add(rateLimitErr.RetryAfter), leaseToken)
			}
			timer := time.NewTimer(rateLimitErr.RetryAfter)
			select {
			case <-ctx.Done():
				timer.Stop()
				persistCtx, persistCancel := context.WithTimeout(context.Background(), 5*time.Second)
				_, _ = e.pool.Exec(persistCtx, `UPDATE group_library_import_jobs SET status='pending', lease_token=NULL, updated_at=NOW() WHERE job_id=$1 AND lease_token=$2 AND status='processing'`, jobID, leaseToken)
				persistCancel()
				return
			case <-timer.C:
			}
			// 冷却结束后释放租约，任务立即回到可领取队列。
			_, _ = e.pool.Exec(ctx, `UPDATE group_library_import_jobs SET status='pending', lease_token=NULL, updated_at=NOW() WHERE job_id=$1 AND lease_token=$2 AND status='processing'`, jobID, leaseToken)
			continue
		}
		if ctx.Err() != nil {
			// A graceful shutdown must not turn the unprocessed suffix into failures.
			// Persist the completed prefix with a short independent context so the job
			// can resume from the exact next input row after the process restarts.
			processed, imported, failed, skipped := 0, 0, 0, 0
			detail := ""
			if result != nil {
				processed = result.Processed
				imported = len(result.Imported)
				failed = len(result.Failed)
				skipped = result.Skipped
				if failed > 0 {
					detail = strings.Join(result.Failed, "\n")
				}
			}
			persistCtx, persistCancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, persistErr := e.pool.Exec(persistCtx, `
				UPDATE group_library_import_jobs
				SET processed_count=$2,
				    imported_count=imported_count+$3,
				    failed_count=failed_count+$4,
				    skipped_count=skipped_count+$5,
				    last_error=CASE WHEN $6='' THEN last_error ELSE LEFT($6, 4000) END,
				    status='pending', lease_token=NULL, updated_at=NOW()
				WHERE job_id=$1 AND status='processing' AND lease_token=$7
			`, jobID, offset+processed, imported, failed, skipped, detail, leaseToken)
			persistCancel()
			if persistErr != nil {
				log.Printf("[grouplib] 保存中断导入进度失败 job=%s: %v", jobID, persistErr)
			}
			return
		}
		imported, failed, skipped := 0, 0, 0
		lastError := ""
		if result != nil {
			imported, failed, skipped = len(result.Imported), len(result.Failed), result.Skipped
			if len(result.Failed) > 0 {
				// 将逐条失败原因持久化，任务中心可以直接展示，避免只看到失败数量。
				lastError = strings.Join(result.Failed, "\n")
				if len(lastError) > 4000 {
					lastError = lastError[:4000] + "…（错误明细已截断）"
				}
			}
		} else if importErr != nil {
			// 整批调用失败时，不能把这一批记成零失败的成功任务。
			failed = end - offset
		}
		if importErr != nil {
			lastError = strings.TrimSpace(lastError + "\n" + importErr.Error())
			if len(lastError) > 4000 {
				lastError = lastError[:4000] + "…（错误明细已截断）"
			}
		}
		nextStatus := "pending"
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
			    status=$7, finished_at=$8, payload=CASE WHEN $7='completed' THEN '' ELSE payload END,
			    lease_token=NULL, updated_at=NOW()
			WHERE job_id=$1 AND lease_token=$9 AND status='processing'`, jobID, end, imported, failed, skipped, lastError, nextStatus, finished, leaseToken)
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
