import { describe, expect, it } from 'vitest'
import { mergeRecentTaskLogs, normalizeTaskProgress } from './taskProgress'

describe('normalizeTaskProgress', () => {
  it('normalizes object and JSON-string payloads', () => {
    expect(normalizeTaskProgress({ total: '12', done: 7, failed: '2' })).toEqual({
      total: 12, done: 7, failed: 2,
    })
    expect(normalizeTaskProgress('{"total":10,"done":4,"failed":1}')).toEqual({
      total: 10, done: 4, failed: 1,
    })
  })

  it('turns missing, malformed and non-object payloads into zero progress', () => {
    for (const value of [null, undefined, '', 'not-json', [], 12, true]) {
      expect(normalizeTaskProgress(value)).toEqual({ total: 0, done: 0, failed: 0 })
    }
  })

  it('rejects non-finite and negative counters', () => {
    expect(normalizeTaskProgress({
      total: Number.POSITIVE_INFINITY,
      done: Number.NaN,
      failed: -3,
    })).toEqual({ total: 0, done: 0, failed: 0 })
  })

  it('floors fractional counters and ignores unrelated fields', () => {
    expect(normalizeTaskProgress({ total: 4.9, done: 2.8, failed: 0.2, extra: 1 })).toEqual({
      total: 4, done: 2, failed: 0,
    })
  })
})


describe('mergeRecentTaskLogs', () => {
  it('keeps newer SSE events ahead of a stale API snapshot and deduplicates IDs', () => {
    const api = [{ log_id: 1, message: 'old' }, { log_id: 2, message: 'snapshot' }]
    const live = [{ log_id: 3, message: 'new SSE' }, { log_id: 2, message: 'duplicate newer copy' }]
    expect(mergeRecentTaskLogs(api, live)).toEqual([
      { log_id: 3, message: 'new SSE' },
      { log_id: 2, message: 'duplicate newer copy' },
      { log_id: 1, message: 'old' },
    ])
  })

  it('limits the newest-first result and supports zero capacity', () => {
    const api = [{ log_id: 1 }, { log_id: 2 }, { log_id: 3 }]
    expect(mergeRecentTaskLogs(api, [], 2).map(log => log.log_id)).toEqual([3, 2])
    expect(mergeRecentTaskLogs(api, [], 0)).toEqual([])
  })

  it('retains entries without IDs instead of accidentally collapsing them', () => {
    const api = [{ message: 'older' }, { message: 'newer' }]
    expect(mergeRecentTaskLogs(api, []).map(log => log.message)).toEqual(['newer', 'older'])
  })
})
