export interface GroupImportFileLike {
  name: string
  size: number
  type?: string
}

export function validateGroupImportFile(file: GroupImportFileLike, maxBytes = 20 * 1024 * 1024): string | null {
  const extension = file.name.split('.').pop()?.toLowerCase()
  const supported = extension === 'txt' || extension === 'csv' ||
    file.type === 'text/plain' || file.type === 'text/csv' || file.type === 'application/vnd.ms-excel'
  if (!supported) return '仅支持 TXT 或 CSV 文件'
  if (!Number.isFinite(file.size) || file.size <= 0) return '文件不能为空'
  if (file.size > maxBytes) return '文件不能超过 20MB'
  return null
}

export function formatGroupLibraryTime(value: string | null | undefined): string {
  if (!value) return '—'
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return '—'
  const pad = (number: number) => String(number).padStart(2, '0')
  return `${date.getMonth() + 1}/${date.getDate()} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

export function normalizeGroupMemberCount(value: unknown): number {
  const count = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(count) && count > 0 ? Math.floor(count) : 0
}

export function validateGroupJoinOptions(input: {
  accountIds: string[]
  targetIds: string[]
  batchEnabled: boolean
  perAccountCount: number
  intervalMin: number | null
  intervalMax: number | null
  onlineAccountIds: string[]
}): string | null {
  if (!input.targetIds.length) return '请至少选择一个群组'
  if (!input.accountIds.length) return '请至少选择一个执行账号'
  if (input.accountIds.some(id => !input.onlineAccountIds.includes(id))) return '所选账号已离线或不可用，请重新选择'
  if (input.batchEnabled && (!Number.isInteger(input.perAccountCount) || input.perAccountCount < 1 || input.perAccountCount > 100)) {
    return '每个账号分配的群数必须是 1–100 的整数'
  }
  const min = input.intervalMin
  const max = input.intervalMax
  if ((min !== null && (!Number.isInteger(min) || min < 0)) ||
      (max !== null && (!Number.isInteger(max) || max < 0))) return '加群间隔必须是非负整数分钟'
  if ((min === null) !== (max === null)) return '请同时填写最小和最大加群间隔'
  if (min !== null && max !== null && min > max) return '最小加群间隔不能大于最大间隔'
  return null
}

export function uniqueGroupIds(ids: string[]): string[] {
  return [...new Set(ids.filter(id => typeof id === 'string' && id.trim()).map(id => id.trim()))]
}
