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
