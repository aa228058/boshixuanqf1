import { describe, expect, it } from 'vitest'
import { canStartPullRefresh } from './pullDownRefresh'

describe('canStartPullRefresh', () => {
  it('allows a gesture only at the top edge', () => {
    expect(canStartPullRefresh(0, false)).toBe(true)
    expect(canStartPullRefresh(-1, false)).toBe(true)
    expect(canStartPullRefresh(1, false)).toBe(false)
    expect(canStartPullRefresh(240, false)).toBe(false)
  })

  it('does not start while a refresh is active', () => {
    expect(canStartPullRefresh(0, true)).toBe(false)
  })

  it('rejects invalid scroll positions', () => {
    expect(canStartPullRefresh(Number.NaN, false)).toBe(false)
    expect(canStartPullRefresh(Number.POSITIVE_INFINITY, false)).toBe(false)
  })
})
