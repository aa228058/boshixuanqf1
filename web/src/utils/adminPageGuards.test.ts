import { describe, expect, it } from 'vitest'
import { clampPercentage, finiteMetric, formatValidDate, formatValidDateTime, isAdminMenuPathActive, isSupportedQrImage, isValidAuditDateRange, isValidDateTimeLocal, normalizePlanFeatures, validateAdminLogin, validateAnnouncementInput, validateCodeGenerationInput, validateTenantCreateInput } from './adminPageGuards'

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

describe('finiteMetric', () => {
  it('replaces non-finite values with zero', () => {
    expect(finiteMetric(Number.NaN)).toBe(0)
    expect(finiteMetric(Number.POSITIVE_INFINITY)).toBe(0)
  })
  it('accepts finite numeric strings', () => expect(finiteMetric('12.5')).toBe(12.5))
})

describe('isAdminMenuPathActive', () => {
  it('matches exact paths and nested routes only', () => {
    expect(isAdminMenuPathActive('/admin/tenants', '/admin/tenants')).toBe(true)
    expect(isAdminMenuPathActive('/admin/tenants/123', '/admin/tenants')).toBe(true)
    expect(isAdminMenuPathActive('/admin/tenants-old', '/admin/tenants')).toBe(false)
  })
})

describe('validateTenantCreateInput', () => {
  const valid = { name: '租户', username: 'tenant', password: 'secret', plan_id: 'free' }
  it('requires each field and a selected plan', () => {
    expect(validateTenantCreateInput({ ...valid, name: ' ' })).toBe('请输入租户名称')
    expect(validateTenantCreateInput({ ...valid, username: ' ' })).toBe('请输入登录用户名')
    expect(validateTenantCreateInput({ ...valid, password: ' ' })).toBe('请输入登录密码')
    expect(validateTenantCreateInput({ ...valid, plan_id: '' })).toBe('请选择套餐')
    expect(validateTenantCreateInput(valid)).toBeNull()
  })
})

describe('isSupportedQrImage', () => {
  it('accepts non-empty JPEG and PNG within the size limit', () => {
    expect(isSupportedQrImage('image/jpeg', 1024)).toBe(true)
    expect(isSupportedQrImage('image/png', 500 * 1024)).toBe(true)
  })
  it('rejects unsupported, empty, oversized and invalid sizes', () => {
    expect(isSupportedQrImage('image/gif', 1024)).toBe(false)
    expect(isSupportedQrImage('image/png', 0)).toBe(false)
    expect(isSupportedQrImage('image/png', 500 * 1024 + 1)).toBe(false)
    expect(isSupportedQrImage('image/png', Number.NaN)).toBe(false)
  })
})

describe('isValidDateTimeLocal', () => {
  it('accepts a real local datetime and rejects invalid dates or times', () => {
    expect(isValidDateTimeLocal('2026-02-28T23:59')).toBe(true)
    expect(isValidDateTimeLocal('2026-02-29T10:00')).toBe(false)
    expect(isValidDateTimeLocal('2026-01-01T24:00')).toBe(false)
  })
})

describe('formatValidDate', () => {
  it('returns a placeholder for invalid timestamps', () => expect(formatValidDate('not-a-date')).toBe('—'))
})


describe('formatValidDateTime', () => {
  it('returns a placeholder for invalid timestamps', () => expect(formatValidDateTime('bad-date')).toBe('—'))
})

describe('validateCodeGenerationInput', () => {
  const valid = { plan_id: 'basic', count: 10, timing_type: 'immediate' }
  it('validates the selected plan, integer quantity, timing and optional expiry', () => {
    expect(validateCodeGenerationInput(valid, ['basic'])).toBeNull()
    expect(validateCodeGenerationInput({ ...valid, plan_id: 'missing' }, ['basic'])).toBe('请选择有效套餐')
    expect(validateCodeGenerationInput({ ...valid, count: 1.5 }, ['basic'])).toBe('生成数量必须是 1–100 的整数')
    expect(validateCodeGenerationInput({ ...valid, count: 101 }, ['basic'])).toBe('生成数量必须是 1–100 的整数')
    expect(validateCodeGenerationInput({ ...valid, timing_type: 'later' }, ['basic'])).toBe('请选择有效的生效方式')
    expect(validateCodeGenerationInput({ ...valid, expires_days: 0 }, ['basic'])).toBe('有效期必须是正整数天数')
    expect(validateCodeGenerationInput({ ...valid, expires_days: 1.2 }, ['basic'])).toBe('有效期必须是正整数天数')
  })
})

describe('validateAnnouncementInput', () => {
  const valid = { title: '标题', content: '内容', priority: 'normal', target_type: 'all', target_plans: [] as string[] }
  it('requires meaningful content and valid targeting', () => {
    expect(validateAnnouncementInput(valid)).toBeNull()
    expect(validateAnnouncementInput({ ...valid, title: ' ' })).toBe('请输入公告标题')
    expect(validateAnnouncementInput({ ...valid, content: ' ' })).toBe('请输入公告内容')
    expect(validateAnnouncementInput({ ...valid, target_type: 'specific_plan' })).toBe('指定套餐公告至少选择一个套餐')
    expect(validateAnnouncementInput({ ...valid, priority: 'unknown' })).toBe('请选择有效的公告优先级')
  })
})

describe('normalizePlanFeatures', () => {
  it('keeps only enabled object flags and valid array entries', () => {
    expect(normalizePlanFeatures({ broadcast: true, raffle: false, api: 1 })).toEqual(['broadcast'])
    expect(normalizePlanFeatures(['broadcast', '', 1, 'raffle'])).toEqual(['broadcast', 'raffle'])
  })
})
