import type { TaskLog } from '@/types'

/** Stable identity across persisted-page responses and live SSE events. */
export function taskLogKey(log: TaskLog): string {
  if (log.log_id) return `id:${log.log_id}`
  return JSON.stringify([
    log.executed_at || '',
    log.action || '',
    log.status || '',
    log.target_id || '',
    log.target_name || '',
    log.tg_message_id ?? '',
    log.error_code || '',
    log.message || '',
  ])
}

/** Merge page results without losing live rows that arrived during the request. */
export function mergeTaskLogs(existing: readonly TaskLog[], incoming: readonly TaskLog[]): TaskLog[] {
  const merged = new Map<string, TaskLog>()
  for (const log of existing) merged.set(taskLogKey(log), log)
  for (const log of incoming) merged.set(taskLogKey(log), log)
  return [...merged.values()].sort((a, b) => {
    const timeDiff = new Date(a.executed_at || 0).getTime() - new Date(b.executed_at || 0).getTime()
    return timeDiff || String(a.log_id || '').localeCompare(String(b.log_id || ''))
  })
}

/** Cursor pages are returned newest-first; the final row is the oldest row in that page. */
export function getTaskLogPageState(rows: readonly TaskLog[], pageSize: number) {
  const cursor = rows.length ? rows[rows.length - 1].log_id || null : null
  return {
    cursor,
    hasMore: rows.length === pageSize && !!cursor,
  }
}
