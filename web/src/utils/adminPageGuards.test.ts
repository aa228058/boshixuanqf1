import { describe, expect, it } from 'vitest'
import { clampPercentage, isValidAuditDateRange, validateAdminLogin } from './adminPageGuards'

describe('validateAdminLogin', () => {
  it('requires a username', () => expect(validateAdminLogin(' ', 'secret')).toEqual({ username: '请输入用户名' }))
  it('requires a password', () => expect(validateAdminLogin('admin', '  ')).toEqual({ password: '请输入密码' }))
  it('accepts non-empty credentials', () => expect(validateAdminLogin(' admin ', ' secret ')).toEqual({}))
})

describe('isValidAuditDateRange', () => {
  it('allows an unbounded range', () => expect(isValidAuditDateRange('', '')).toBe(true))
  it('allows an ordered inclusive range', () => expect(isValidAuditDateRange('2026-02-28', '2026-03-01')).toBe(true))
  it('rejects a reversed range', () => expect(isValidAuditDateRange('2026-03-02', '2026-03-01')).toBe(false))
  it('rejects impossible calendar dates', () => expect(isValidAuditDateRange('2026-02-29', '')).toBe(false))
})

describe('clampPercentage', () => {
  it('clamps values below zero', () => expect(clampPercentage(-12)).toBe(0))
  it('clamps values above one hundred', () => expect(clampPercentage(140)).toBe(100))
  it('normalizes invalid and numeric-string values', () => {
    expect(clampPercentage(Number.NaN)).toBe(0)
    expect(clampPercentage('42.5')).toBe(42.5)
  })
})
