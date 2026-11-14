// 账号列表的展示型纯函数：从 accounts/List.vue 抽出，便于单测与复用。

/** 冷却剩余时间文案（空串表示不在冷却）。now 由调用方传入，保持纯函数可测。 */
export function cooldownText(blockedUntil: string | null | undefined, now: number): string {
  if (!blockedUntil) return ''
  const untilTs = new Date(blockedUntil).getTime()
  if (!Number.isFinite(untilTs)) return ''
  const ms = untilTs - now
  if (ms <= 0) return ''
  const seconds = Math.ceil(ms / 1000)
  if (seconds < 60) return `${seconds}秒`
  const minutes = Math.ceil(ms / 60000)
  return minutes >= 60
    ? `${Math.floor(minutes / 60)}小时${minutes % 60 ? (minutes % 60) + '分' : ''}`
    : `${minutes}分钟`
}

/** 在线状态圆点配色（入参判空：上游可能传 undefined）。 */
export function statusDotClass(status?: string): string {
  switch (status) {
    case 'online': return 'bg-[var(--t-ok)]'
    case 'offline': return 'bg-text-muted'
    case 'banned': return 'bg-[var(--t-err)]'
    case 'session_revoked': return 'bg-[var(--t-err)] animate-pulse'
    case 'restricted': return 'bg-[var(--t-warn)]'
    case 'pending_auth': return 'bg-[var(--t-warn)] animate-pulse'
    default: return 'bg-text-muted'
  }
}
