import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const { authApiMock } = vi.hoisted(() => ({
  authApiMock: {
    login: vi.fn(),
    logout: vi.fn(),
    refresh: vi.fn(),
    me: vi.fn(),
    register: vi.fn(),
    renew: vi.fn(),
  },
}))

vi.mock('@/api/auth', () => ({ authApi: authApiMock }))

import { useAuthStore } from '@/stores/auth'

function installLocalStorage() {
  const values = new Map<string, string>()
  vi.stubGlobal('localStorage', {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => values.set(key, value),
    removeItem: (key: string) => values.delete(key),
    clear: () => values.clear(),
  })
  return values
}

describe('auth store session restoration', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    authApiMock.refresh.mockReset()
    authApiMock.login.mockReset()
    authApiMock.logout.mockReset()
    authApiMock.me.mockReset()
  })

  it('临时网络/服务端错误时保留可恢复会话标志', async () => {
    const storage = installLocalStorage()
    storage.set('tg_sender_session', '1')
    authApiMock.refresh.mockRejectedValueOnce(new Error('temporary network failure'))

    const store = useAuthStore()
    await expect(store.restoreSession()).resolves.toBe(true)

    expect(storage.get('tg_sender_session')).toBe('1')
    expect(store.isLoggedIn).toBe(false)
    expect(authApiMock.refresh).toHaveBeenCalledTimes(1)
  })

  it('没有会话标志时不尝试刷新', async () => {
    installLocalStorage()
    const store = useAuthStore()

    await expect(store.restoreSession()).resolves.toBe(false)
    expect(authApiMock.refresh).not.toHaveBeenCalled()
  })

  it('刷新成功后恢复内存 token', async () => {
    const storage = installLocalStorage()
    storage.set('tg_sender_session', '1')
    authApiMock.refresh.mockResolvedValueOnce({ token: 'restored-token' })

    const store = useAuthStore()
    await expect(store.restoreSession()).resolves.toBe(true)

    expect(store.token).toBe('restored-token')
    expect(store.isLoggedIn).toBe(true)
    expect(storage.get('tg_sender_session')).toBe('1')
  })

  it('登出并重新登录后，迟到的续签结果不能返回旧 token', async () => {
    installLocalStorage()
    let resolveRefresh!: (value: { token: string }) => void
    authApiMock.refresh.mockReturnValueOnce(new Promise(resolve => { resolveRefresh = resolve }))
    authApiMock.logout.mockResolvedValueOnce(undefined)
    authApiMock.login.mockResolvedValueOnce({ token: 'new-session-token' })
    const store = useAuthStore()

    const pendingRefresh = store.refresh()
    await store.logout()
    await store.login('new-user', 'password')
    resolveRefresh({ token: 'stale-refresh-token' })

    await expect(pendingRefresh).rejects.toMatchObject({ name: 'AbortError' })
    expect(store.token).toBe('new-session-token')
  })

  it('登出后迟到的登录响应不能重新建立会话', async () => {
    installLocalStorage()
    let resolveLogin!: (value: { token: string }) => void
    authApiMock.login.mockReturnValueOnce(new Promise(resolve => { resolveLogin = resolve }))
    authApiMock.logout.mockResolvedValueOnce(undefined)
    const store = useAuthStore()

    const pendingLogin = store.login('user', 'password')
    await store.logout()
    resolveLogin({ token: 'stale-token' })
    await pendingLogin

    expect(store.token).toBe('')
    expect(store.isLoggedIn).toBe(false)
    expect(store.hasSession).toBe(false)
  })

})