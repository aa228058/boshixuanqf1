import { afterEach, describe, expect, it, vi } from 'vitest'
import { syncLock } from './syncLock'

describe('syncLock', () => {
  afterEach(() => {
    syncLock.release()
    vi.useRealTimers()
  })

  it('prevents duplicate syncs for the same account', () => {
    expect(syncLock.acquire('account-a')).toBe(true)
    expect(syncLock.acquire('account-a')).toBe(false)
    expect(syncLock.isSyncing('account-a')).toBe(true)
  })

  it('allows independent accounts to sync without overwriting each other', () => {
    expect(syncLock.acquire('account-a')).toBe(true)
    expect(syncLock.acquire('account-b')).toBe(true)
    expect(syncLock.isSyncing()).toBe(true)

    syncLock.release('account-a')
    expect(syncLock.isSyncing('account-a')).toBe(false)
    expect(syncLock.isSyncing('account-b')).toBe(true)
    expect(syncLock.getAccountId()).toBe('account-b')
  })

  it('expires stale account locks independently', () => {
    vi.useFakeTimers()
    expect(syncLock.acquire('account-a')).toBe(true)
    vi.advanceTimersByTime(120001)
    expect(syncLock.isSyncing('account-a')).toBe(false)
  })
})
