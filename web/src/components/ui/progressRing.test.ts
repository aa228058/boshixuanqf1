import { describe, expect, it } from 'vitest'
import {
  normalizeRingPercent,
  normalizeRingSize,
  normalizeRingStroke,
  normalizeRingSuccess,
} from './progressRing'

describe('ProgressRing input normalization', () => {
  it('falls back to a usable size for non-finite values', () => {
    expect(normalizeRingSize(Number.NaN)).toBe(44)
    expect(normalizeRingSize(Number.POSITIVE_INFINITY)).toBe(44)
  })

  it('keeps size positive', () => {
    expect(normalizeRingSize(-10)).toBe(1)
  })

  it('caps finite stroke to the ring diameter', () => {
    expect(normalizeRingStroke(80, 40)).toBe(40)
    expect(normalizeRingStroke(-2, 40)).toBe(0)
  })

  it('caps fallback stroke to a tiny ring size', () => {
    expect(normalizeRingStroke(Number.NaN, 1)).toBe(1)
    expect(normalizeRingStroke(Number.POSITIVE_INFINITY, 2)).toBe(2)
  })

  it('normalizes percent and success within the visible progress', () => {
    expect(normalizeRingPercent(120)).toBe(100)
    expect(normalizeRingPercent(-5)).toBe(0)
    expect(normalizeRingPercent(Number.NaN)).toBe(0)
    expect(normalizeRingSuccess(90, 30)).toBe(30)
    expect(normalizeRingSuccess(-1, 30)).toBe(0)
  })
})
