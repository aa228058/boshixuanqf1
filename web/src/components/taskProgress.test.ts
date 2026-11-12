import { describe, expect, it } from 'vitest'
import { normalizeTaskProgress } from './taskProgress'

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
