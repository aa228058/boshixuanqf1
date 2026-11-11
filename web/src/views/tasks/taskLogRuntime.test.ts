import { describe, expect, it } from 'vitest'
import type { TaskLog } from '@/types'
import { getTaskLogPageState, mergeTaskLogs, taskLogKey } from './taskLogRuntime'

const log = (overrides: Partial<TaskLog> = {}): TaskLog => ({
  action: 'send',
  status: 'success',
  executed_at: '2026-11-10T10:00:00Z',
  ...overrides,
})

describe('task log runtime', () => {
  it('uses persisted IDs as stable identity across HTTP and SSE', () => {
    expect(taskLogKey(log({ log_id: '42', message: 'HTTP copy' })))
      .toBe(taskLogKey(log({ log_id: '42', message: 'SSE copy' })))
  })

  it('uses collision-safe structural identity when an ID is unavailable', () => {
    expect(taskLogKey(log({ message: 'a|b' }))).not.toBe(taskLogKey(log({ message: 'a', target_id: 'b' })))
  })

  it('merges live and persisted rows without duplicates and sorts deterministically', () => {
    const live = log({ log_id: '2', message: 'live' })
    const older = log({ log_id: '1', executed_at: '2026-11-10T09:00:00Z' })
    const duplicate = log({ log_id: '2', message: 'persisted copy' })
    const merged = mergeTaskLogs([live], [older, duplicate])
    expect(merged.map(row => row.log_id)).toEqual(['1', '2'])
    expect(merged[1].message).toBe('persisted copy')
  })

  it('uses log ID as a deterministic tie-breaker for equal timestamps', () => {
    const rows = mergeTaskLogs([], [log({ log_id: 'b' }), log({ log_id: 'a' })])
    expect(rows.map(row => row.log_id)).toEqual(['a', 'b'])
  })

  it('only exposes a next-page cursor when a full page has a cursor ID', () => {
    expect(getTaskLogPageState([log({ log_id: 'last' })], 1)).toEqual({ cursor: 'last', hasMore: true })
    expect(getTaskLogPageState([log()], 1)).toEqual({ cursor: null, hasMore: false })
    expect(getTaskLogPageState([], 1)).toEqual({ cursor: null, hasMore: false })
  })
})
