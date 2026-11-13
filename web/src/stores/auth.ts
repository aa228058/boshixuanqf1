import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { RequestOptions } from '@/utils/request'
import { authApi } from '@/api/auth'
import type { Tenant } from '@/types'

// F-20 安全修复: access token 不再持久化到 localStorage（CSP 含 unsafe-inline，
// XSS 即可窃取长寿命令牌）。token 只活在内存；localStorage 仅保留一个不含密钥的
// "登录标志位"，供硬刷新后凭 HttpOnly refresh Cookie 静默恢复会话。
// 旧版本写入的 token 键在登录/登出/恢复时统一清理。
const SESSION_KEY = 'tg_sender_session'
const LEGACY_TOKEN_KEYS = ['tg_sender_token', 'tg_sender_refresh_token']

// 安全读写 localStorage（兼容隐私模式/禁用 localStorage 的浏览器）
function safeGetItem(key: string): string | null {
  try { return localStorage.getItem(key) } catch { return null }
}
function safeSetItem(key: string, val: string): void {
  try { localStorage.setItem(key, val) } catch {}
}
function safeRemoveItem(key: string): void {
  try { localStorage.removeItem(key) } catch {}
}

function clearLegacyTokenKeys() {
  LEGACY_TOKEN_KEYS.forEach(safeRemoveItem)
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref('') // 纯内存，绝不落 localStorage
  const tenant = ref<Tenant | null>(null)
  const loading = ref(false)
  // 登录/登出/续签可能交错返回；旧会话响应不得复活已退出的会话。
  let sessionGeneration = 0

  const isLoggedIn = computed(() => !!token.value)
  // 是否存在可恢复会话：标志位在即可用 refresh Cookie 续签
  const hasSession = computed(() => !!safeGetItem(SESSION_KEY))

  async function login(username: string, password: string) {
    const generation = ++sessionGeneration
    loading.value = true
    try {
      const res = await authApi.login(username, password)
      if (generation !== sessionGeneration) return res
      token.value = res.token
      safeSetItem(SESSION_KEY, '1')
      clearLegacyTokenKeys()
      // 注意: 不在这里调 me() —— 登录后 Layout.onMounted 会统一调 refreshTenant(),
      // 避免同接口 60 秒内重复请求, 省一次串行 RTT
      return res
    } finally {
      if (generation === sessionGeneration) loading.value = false
    }
  }

  async function logout() {
    // 立即使所有在途登录/续签/资料请求失效，防止迟到响应复活旧会话。
    sessionGeneration++
    token.value = ''
    tenant.value = null
    safeRemoveItem(SESSION_KEY)
    clearLegacyTokenKeys()
    try {
      await authApi.logout()
    } catch (e) {
      console.error('logout api failed:', e)
    }
  }

  // 刷新走 HttpOnly Cookie(credentials), refresh_token 不出现在前端任何存储
  async function refresh(): Promise<string> {
    const generation = sessionGeneration
    const res = await authApi.refresh()
    if (generation !== sessionGeneration) {
      // 续签期间若发生登出或重新登录，迟到的旧续签结果不能交给调用方重放请求。
      throw Object.assign(new Error('会话已切换，请重新登录'), { name: 'AbortError' })
    }
    token.value = res.token
    safeSetItem(SESSION_KEY, '1')
    return res.token
  }

  // 硬刷新后恢复会话：登录标志位在且内存无 token 时，用 refresh Cookie 静默换发。
  // 失败（Cookie 过期/被撤销）则清除标志位，守卫据此踢回登录页。
  async function restoreSession(): Promise<boolean> {
    if (token.value) return true
    if (!hasSession.value) return false
    try {
      await refresh()
      return !!token.value
    } catch {
      // request() 已在明确的 401 / 账号级 403 时清除会话标志。
      // 若仍有标志，说明本次恢复可能只是网络/服务端临时故障，保留会话并允许后续请求重试。
      return hasSession.value
    }
  }

  async function refreshTenant(options?: RequestOptions) {
    if (!token.value) return
    const generation = sessionGeneration
    const currentToken = token.value
    try {
      const meRes = await authApi.me(options)
      if (generation !== sessionGeneration || currentToken !== token.value) return
      // /auth/me 返回 tenant_name，而管理端租户模型使用 name；在 Store 入口统一字段。
      tenant.value = { ...meRes, name: meRes.tenant_name || meRes.name || '' }
    } catch (e) {
      if (!(e instanceof Error && e.name === 'AbortError')) console.error('刷新租户信息失败:', e)
    }
  }

  return { token, tenant, loading, isLoggedIn, hasSession, login, logout, refresh, refreshTenant, restoreSession }
})
