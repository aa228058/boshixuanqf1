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
