import { describe, expect, it } from 'vitest'
import { parseCountValue } from './countUp'

describe('parseCountValue', () => {
  it.each([
    ['¥1,234.56', 1234.56],
    ['$-2,500', -2500],
    ['0.75 USD', 0.75],
    ['1.2e3 users', 1200],
    ['not available', 0],
    ['', 0],
    [Number.NaN, 0],
    [Number.POSITIVE_INFINITY, 0],
  ])('parses %s', (value, expected) => {
    expect(parseCountValue(value)).toBe(expected)
  })
})
