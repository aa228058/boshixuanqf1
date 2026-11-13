export interface JoinLinkFileLike { name: string; size: number; type?: string }

export function validateJoinLinkFile(file: JoinLinkFileLike, maxBytes = 5 * 1024 * 1024): string | null {
  const extension = file.name.split('.').pop()?.toLowerCase()
  const supported = extension === 'txt' || extension === 'csv' || file.type === 'text/plain' || file.type === 'text/csv'
  if (!supported) return '仅支持 TXT 或 CSV 文件'
  if (!Number.isFinite(file.size) || file.size <= 0) return '文件不能为空'
  if (file.size > maxBytes) return '文件不能超过 5MB'
  return null
}

function finiteCount(value: unknown): number {
  const n = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(n) && n > 0 ? Math.floor(n) : 0
}

export function normalizeTaskProgress(value: unknown): { total: number; done: number; failed: number } {
  let source: any = value
  if (typeof source === 'string') {
    try { source = JSON.parse(source) } catch { source = null }
  }
  if (!source || typeof source !== 'object' || Array.isArray(source)) return { total: 0, done: 0, failed: 0 }
  const total = finiteCount(source.total)
  return {
    total,
    done: Math.min(total, finiteCount(source.done)),
    failed: Math.min(total, finiteCount(source.failed)),
  }
}

export function safeProgressPercent(progress: { total: number; done: number }): number {
  if (!Number.isFinite(progress.total) || progress.total <= 0) return 0
  if (!Number.isFinite(progress.done) || progress.done <= 0) return 0
  return Math.min(100, Math.round(progress.done / progress.total * 100))
}

export function safeTaskDate(value: string | null | undefined, fallback = '—'): string {
  if (!value) return fallback
  const date = new Date(value)
  return Number.isFinite(date.getTime()) ? date.toLocaleString('zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }) : fallback
}

export function relativeTaskTime(value: string | null | undefined, now = Date.now()): string {
  if (!value) return '—'
  const timestamp = new Date(value).getTime()
  if (!Number.isFinite(timestamp)) return '—'
  const seconds = Math.max(0, Math.floor((now - timestamp) / 1000))
  if (seconds < 60) return `${seconds}秒前`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟前`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}小时前`
  return `${Math.floor(seconds / 86400)}天前`
}

export function uniqueTaskIds(values: unknown[]): string[] {
  return [...new Set(values.filter((value): value is string => typeof value === 'string').map(value => value.trim()).filter(Boolean))]
}
