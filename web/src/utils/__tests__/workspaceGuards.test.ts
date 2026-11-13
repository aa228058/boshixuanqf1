import { describe, expect, it } from 'vitest'
import {
  normalizeTaskProgress, relativeTaskTime, safeProgressPercent, safeTaskDate,
  uniqueTaskIds, validateJoinLinkFile,
} from '../workspaceGuards'

describe('workspace guards', () => {
  it('validates join link import type, emptiness and size', () => {
    expect(validateJoinLinkFile({ name: 'links.txt', size: 1, type: 'text/plain' })).toBeNull()
    expect(validateJoinLinkFile({ name: 'links.exe', size: 1 })).toBe('仅支持 TXT 或 CSV 文件')
    expect(validateJoinLinkFile({ name: 'links.csv', size: 0 })).toBe('文件不能为空')
    expect(validateJoinLinkFile({ name: 'links.txt', size: 5 * 1024 * 1024 + 1 })).toBe('文件不能超过 5MB')
  })

  it('normalizes malformed and out-of-range progress', () => {
    expect(normalizeTaskProgress('{bad')).toEqual({ total: 0, done: 0, failed: 0 })
    expect(normalizeTaskProgress({ total: 10, done: 12, failed: -1 })).toEqual({ total: 10, done: 10, failed: 0 })
    expect(normalizeTaskProgress({ total: Infinity, done: NaN })).toEqual({ total: 0, done: 0, failed: 0 })
    expect(safeProgressPercent({ total: 0, done: 3 })).toBe(0)
    expect(safeProgressPercent({ total: 4, done: 9 })).toBe(100)
  })

  it('protects task date and relative time formatting', () => {
    expect(safeTaskDate('not-a-date')).toBe('—')
    expect(relativeTaskTime('not-a-date')).toBe('—')
    expect(relativeTaskTime('2026-01-01T00:00:00Z', Date.parse('2026-01-01T00:00:10Z'))).toBe('10秒前')
    expect(relativeTaskTime('2026-01-01T00:01:00Z', Date.parse('2026-01-01T00:00:00Z'))).toBe('0秒前')
  })

  it('trims and deduplicates task IDs', () => {
    expect(uniqueTaskIds([' a ', 'a', '', null, 'b'])).toEqual(['a', 'b'])
  })
})
