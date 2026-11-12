import { afterEach, describe, expect, it, vi } from 'vitest'
import { syncLock } from './syncLock'

afterEach(() => {
  vi.useRealTimers()
})

describe('syncLock', () => {
  it('prevents duplicate syncs for the same account', () => {
    const token = syncLock.acquire('account-a')
    expect(token).not.toBeNull()
    expect(syncLock.acquire('account-a')).toBeNull()
    expect(syncLock.isSyncing('account-a')).toBe(true)
    syncLock.release('account-a', token!)
    expect(syncLock.isSyncing('account-a')).toBe(false)
  })

  it('allows independent accounts to sync without overwriting each other', () => {
    const tokenA = syncLock.acquire('account-a')
    const tokenB = syncLock.acquire('account-b')
    expect(tokenA).not.toBeNull()
    expect(tokenB).not.toBeNull()
    expect(syncLock.isSyncing()).toBe(true)

    syncLock.release('account-a', tokenA!)
    expect(syncLock.isSyncing('account-a')).toBe(false)
    expect(syncLock.isSyncing('account-b')).toBe(true)
    expect(syncLock.getAccountId()).toBe('account-b')
    syncLock.release('account-b', tokenB!)
  })

  it('does not let an expired request release a newer lease for the same account', () => {
    vi.useFakeTimers()
    const staleToken = syncLock.acquire('account-a')
    expect(staleToken).not.toBeNull()

    vi.advanceTimersByTime(120001)
    expect(syncLock.isSyncing('account-a')).toBe(false)

    const freshToken = syncLock.acquire('account-a')
    expect(freshToken).not.toBeNull()
    syncLock.release('account-a', staleToken!)
    expect(syncLock.isSyncing('account-a')).toBe(true)

    syncLock.release('account-a', freshToken!)
    expect(syncLock.isSyncing('account-a')).toBe(false)
  })
})
