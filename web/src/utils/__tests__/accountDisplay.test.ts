import { describe, expect, it } from 'vitest'
import { cooldownText, statusDotClass } from '../accountDisplay'

describe('cooldownText', () => {
  const now = Date.parse('2026-01-01T12:00:00Z')

  it('无冷却时返回空串', () => {
    expect(cooldownText(undefined, now)).toBe('')
    expect(cooldownText(null, now)).toBe('')
    expect(cooldownText('not-a-date', now)).toBe('')
  })

  it('已过期不显示', () => {
    expect(cooldownText('2026-01-01T11:59:00Z', now)).toBe('')
  })

  it('小于一分钟按秒显示', () => {
    expect(cooldownText('2026-01-01T12:00:30Z', now)).toBe('30秒')
  })

  it('分钟与小时格式', () => {
    expect(cooldownText('2026-01-01T12:05:00Z', now)).toBe('5分钟')
    expect(cooldownText('2026-01-01T13:30:00Z', now)).toBe('1小时30分')
    expect(cooldownText('2026-01-01T14:00:00Z', now)).toBe('2小时')
  })
})

describe('statusDotClass', () => {
  it('按状态返回对应配色', () => {
    expect(statusDotClass('online')).toContain('t-ok')
    expect(statusDotClass('banned')).toContain('t-err')
    expect(statusDotClass('restricted')).toContain('t-warn')
  })

  it('未知/空状态兜底为灰点', () => {
    expect(statusDotClass(undefined)).toBe('bg-text-muted')
    expect(statusDotClass('weird')).toBe('bg-text-muted')
  })
})
