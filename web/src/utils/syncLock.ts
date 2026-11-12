// Per-account sync locks prevent duplicate Telegram requests without allowing
// one account's completion to release another account's in-flight sync.
const syncingAccounts = new Map<string, number>()
const SYNC_LOCK_TTL = 120000 // Slightly above the 110s backend sync deadline
const SYNC_INTERVAL = 10 * 60 * 1000 // 10 minutes between syncs

function pruneExpiredLocks() {
  const now = Date.now()
  for (const [accountId, startedAt] of syncingAccounts) {
    if (now - startedAt > SYNC_LOCK_TTL) syncingAccounts.delete(accountId)
  }
}

export const syncLock = {
  isSyncing(accountId?: string): boolean {
    pruneExpiredLocks()
    return accountId ? syncingAccounts.has(accountId) : syncingAccounts.size > 0
  },
  acquire(accountId: string): boolean {
    if (this.isSyncing(accountId)) return false
    syncingAccounts.set(accountId, Date.now())
    return true
  },
  release(accountId?: string) {
    // Keep no-argument release for compatibility with any external callers.
    if (accountId) syncingAccounts.delete(accountId)
    else syncingAccounts.clear()
  },
  getAccountId(): string | null {
    pruneExpiredLocks()
    return syncingAccounts.keys().next().value ?? null
  },
}

// Persistent sync cache: prevents re-syncing within 10 minutes
export function needsSync(accountID: string): boolean {
  const key = `last_sync_${accountID}`
  const last = localStorage.getItem(key)
  if (!last) return true
  // P2-34: 原 parseInt 对脏数据(手写/损坏值)返回 NaN, NaN > interval 恒 false——
  // 同步逻辑永远判断"不需要同步"。NaN 视为需要同步。
  const t = parseInt(last)
  if (Number.isNaN(t)) return true
  return Date.now() - t > SYNC_INTERVAL
}

export function markSyncing(accountID: string): void {
  localStorage.setItem(`last_sync_${accountID}`, Date.now().toString())
}

// R117: clearSyncCache 已删(零外部引用)
export default syncLock
