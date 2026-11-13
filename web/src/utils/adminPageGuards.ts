export interface AdminLoginErrors {
  username?: string
  password?: string
}

export function validateAdminLogin(username: string, password: string): AdminLoginErrors {
  const errors: AdminLoginErrors = {}
  if (!username.trim()) errors.username = '请输入用户名'
  if (!password.trim()) errors.password = '请输入密码'
  return errors
}

function isCalendarDate(value: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false
  const [year, month, day] = value.split('-').map(Number)
  const date = new Date(year, month - 1, day)
  return date.getFullYear() === year && date.getMonth() === month - 1 && date.getDate() === day
}

export function isValidAuditDateRange(startDate: string, endDate: string): boolean {
  if (startDate && !isCalendarDate(startDate)) return false
  if (endDate && !isCalendarDate(endDate)) return false
  return !startDate || !endDate || startDate <= endDate
}

export function clampPercentage(value: unknown): number {
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) return 0
  return Math.min(100, Math.max(0, numeric))
}

export function finiteMetric(value: unknown): number {
  const numeric = Number(value)
  return Number.isFinite(numeric) ? numeric : 0
}

export function isAdminMenuPathActive(currentPath: string, menuPath: string): boolean {
  return currentPath === menuPath || currentPath.startsWith(`${menuPath}/`)
}

export function validateTenantCreateInput(input: { name: string; username: string; password: string; plan_id: string }): string | null {
  if (!input.name.trim()) return '请输入租户名称'
  if (!input.username.trim()) return '请输入登录用户名'
  if (!input.password.trim()) return '请输入登录密码'
  if (!input.plan_id.trim()) return '请选择套餐'
  return null
}

export function isSupportedQrImage(type: string, size: number, maxSize = 500 * 1024): boolean {
  return (type === 'image/png' || type === 'image/jpeg') && Number.isFinite(size) && size > 0 && size <= maxSize
}

export function isValidDateTimeLocal(value: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(value)) return false
  const [datePart, timePart] = value.split('T')
  if (!isValidCalendarDate(datePart)) return false
  const [hour, minute] = timePart.split(':').map(Number)
  return hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59
}

function isValidCalendarDate(value: string): boolean {
  const [year, month, day] = value.split('-').map(Number)
  const date = new Date(year, month - 1, day)
  return date.getFullYear() === year && date.getMonth() === month - 1 && date.getDate() === day
}

export function formatValidDate(value: string, locale = 'zh-CN'): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleDateString(locale)
}


export function formatValidDateTime(value: string, locale = 'zh-CN'): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString(locale)
}

export interface CodeGenerationInput {
  plan_id: string
  count: number
  timing_type: string
  expires_days?: number
}

export function validateCodeGenerationInput(input: CodeGenerationInput, availablePlanIds: string[]): string | null {
  if (!input.plan_id || !availablePlanIds.includes(input.plan_id)) return '请选择有效套餐'
  if (!Number.isInteger(input.count) || input.count < 1 || input.count > 100) return '生成数量必须是 1–100 的整数'
  if (input.timing_type !== 'immediate' && input.timing_type !== 'on_use') return '请选择有效的生效方式'
  if (input.expires_days !== undefined && (!Number.isInteger(input.expires_days) || input.expires_days < 1)) return '有效期必须是正整数天数'
  return null
}

export function validateAnnouncementInput(input: {
  title: string
  content: string
  priority: string
  target_type: string
  target_plans?: string[]
}): string | null {
  if (!input.title.trim()) return '请输入公告标题'
  if (!input.content.trim()) return '请输入公告内容'
  if (input.priority !== 'normal' && input.priority !== 'urgent') return '请选择有效的公告优先级'
  if (input.target_type !== 'all' && input.target_type !== 'specific_plan') return '请选择有效的公告目标'
  if (input.target_type === 'specific_plan' && !(input.target_plans || []).some(id => id.trim())) return '指定套餐公告至少选择一个套餐'
  return null
}

export function normalizePlanFeatures(features: unknown): string[] {
  if (Array.isArray(features)) return features.filter((value): value is string => typeof value === 'string' && value.length > 0)
  if (!features || typeof features !== 'object') return []
  return Object.entries(features as Record<string, unknown>)
    .filter(([key, enabled]) => key.length > 0 && enabled === true)
    .map(([key]) => key)
}
