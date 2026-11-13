import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const { adminApiMock } = vi.hoisted(() => ({
  adminApiMock: { login: vi.fn(), logout: vi.fn() },
}))
vi.mock('@/api/admin', () => ({ adminApiMod: adminApiMock }))

import { useAdminStore } from './admin'

function installLocalStorage() {
  const values = new Map<string, string>()
  vi.stubGlobal('localStorage', {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => values.set(key, value),
    removeItem: (key: string) => values.delete(key),
  })
  return values
}

describe('admin auth session race guards', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    adminApiMock.login.mockReset()
    adminApiMock.logout.mockReset()
    vi.unstubAllGlobals()
  })

  it('登出后迟到的登录响应不能重新建立管理员会话', async () => {
    const storage = installLocalStorage()
    let resolveLogin!: (value: { token: string }) => void
    adminApiMock.login.mockReturnValueOnce(new Promise(resolve => { resolveLogin = resolve }))
    adminApiMock.logout.mockResolvedValueOnce(undefined)
    const store = useAdminStore()

    const pendingLogin = store.login('admin', 'password')
    await store.logout()
    resolveLogin({ token: 'stale-admin-token' })
    await pendingLogin

    expect(store.token).toBe('')
    expect(store.isLoggedIn).toBe(false)
    expect(storage.get('tg_sender_admin_session')).toBeUndefined()
  })

  it('较早的登出响应不能清除之后建立的新会话', async () => {
    installLocalStorage()
    let resolveLogout!: () => void
    adminApiMock.logout.mockReturnValueOnce(new Promise<void>(resolve => { resolveLogout = resolve }))
    adminApiMock.login.mockResolvedValueOnce({ token: 'new-admin-token' })
    const store = useAdminStore()

    const pendingLogout = store.logout()
    await store.login('admin', 'password')
    resolveLogout()
    await pendingLogout

    expect(store.token).toBe('new-admin-token')
    expect(store.isLoggedIn).toBe(true)
  })
})
