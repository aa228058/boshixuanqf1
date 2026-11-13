import { describe, expect, it } from 'vitest'
import { formatCountValue, parseCountValue } from './countUp'

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

describe('formatCountValue', () => {
  it.each([
    ['¥1,234.56', 9876.54, '¥9,876.54'],
    ['$-2,500', -1250, '$-1,250'],
    ['0.75 USD', 1.25, '1.25 USD'],
    ['1.2e3 users', 2500, '2500 users'],
    [42, 12, '12'],
  ])('formats %s without losing its affixes', (value, amount, expected) => {
    expect(formatCountValue(value, amount)).toBe(expected)
  })
})
