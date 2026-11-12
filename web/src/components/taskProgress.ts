export interface TaskProgress {
  total: number
  done: number
  failed: number
}

const EMPTY_PROGRESS: TaskProgress = { total: 0, done: 0, failed: 0 }

function finiteCount(value: unknown): number {
  const parsed = typeof value === 'number' || typeof value === 'string' ? Number(value) : NaN
  return Number.isFinite(parsed) ? Math.max(0, Math.floor(parsed)) : 0
}

/** Normalize API/SSE progress payloads before arithmetic or rendering. */
export function normalizeTaskProgress(value: unknown): TaskProgress {
  let source: unknown = value
  if (typeof source === 'string') {
    try {
      source = JSON.parse(source)
    } catch {
      return { ...EMPTY_PROGRESS }
    }
  }
  if (!source || typeof source !== 'object' || Array.isArray(source)) {
    return { ...EMPTY_PROGRESS }
  }

  const record = source as Record<string, unknown>
  return {
    total: finiteCount(record.total),
    done: finiteCount(record.done),
    failed: finiteCount(record.failed),
  }
}


/** Merge a newest-first live SSE buffer with an older API snapshot without dropping events
 * that arrived while the snapshot request was in flight. */
export function mergeRecentTaskLogs<T extends { log_id?: string | number | null }>(
  apiLogs: T[],
  liveLogs: T[],
  limit = 6,
): T[] {
  if (limit <= 0) return []
  const seen = new Set<string | number>()
  const merged: T[] = []
  for (const log of [...liveLogs, ...apiLogs.slice().reverse()]) {
    const id = log.log_id
    if (id !== undefined && id !== null) {
      if (seen.has(id)) continue
      seen.add(id)
    }
    merged.push(log)
    if (merged.length >= Math.max(0, limit)) break
  }
  return merged
}
