import { describe, expect, it } from 'vitest'
import {
  formatGroupLibraryTime,
  normalizeGroupMemberCount,
  uniqueGroupIds,
  validateGroupImportFile,
  validateGroupJoinOptions,
} from '../groupLibraryGuards'

describe('group library guards', () => {
  it('accepts only non-empty TXT/CSV files up to 20MB', () => {
    expect(validateGroupImportFile({ name: 'groups.csv', size: 1, type: 'text/csv' })).toBeNull()
    expect(validateGroupImportFile({ name: 'groups.exe', size: 1 })).toBe('仅支持 TXT 或 CSV 文件')
    expect(validateGroupImportFile({ name: 'groups.txt', size: 0 })).toBe('文件不能为空')
    expect(validateGroupImportFile({ name: 'groups.txt', size: 20 * 1024 * 1024 + 1 })).toBe('文件不能超过 20MB')
  })

  it('formats invalid timestamps as a placeholder', () => {
    expect(formatGroupLibraryTime('invalid')).toBe('—')
    expect(formatGroupLibraryTime('')).toBe('—')
  })

  it('normalizes invalid member counts to zero and floors fractions', () => {
    expect(normalizeGroupMemberCount(Number.NaN)).toBe(0)
    expect(normalizeGroupMemberCount(-1)).toBe(0)
    expect(normalizeGroupMemberCount(12.8)).toBe(12)
    expect(normalizeGroupMemberCount('25')).toBe(25)
  })

  it('deduplicates and trims selected group IDs', () => {
    expect(uniqueGroupIds([' a ', 'a', '', 'b'])).toEqual(['a', 'b'])
  })

  it('validates join account state, batch limits and paired intervals', () => {
    const valid = { accountIds: ['acc'], targetIds: ['g1'], batchEnabled: true, perAccountCount: 10, intervalMin: 1, intervalMax: 3, onlineAccountIds: ['acc'] }
    expect(validateGroupJoinOptions(valid)).toBeNull()
    expect(validateGroupJoinOptions({ ...valid, onlineAccountIds: [] })).toBe('所选账号已离线或不可用，请重新选择')
    expect(validateGroupJoinOptions({ ...valid, perAccountCount: 1.5 })).toBe('每个账号分配的群数必须是 1–100 的整数')
    expect(validateGroupJoinOptions({ ...valid, intervalMax: null })).toBe('请同时填写最小和最大加群间隔')
    expect(validateGroupJoinOptions({ ...valid, intervalMin: 5, intervalMax: 2 })).toBe('最小加群间隔不能大于最大间隔')
  })
})
