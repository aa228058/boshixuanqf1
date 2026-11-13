import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// Mock request.ts 的外部依赖（auth/admin store、router），隔离被测逻辑
const { authMock } = vi.hoisted(() => ({
  authMock: {
    token: 'test-token',
    refresh: vi.fn(),
    logout: vi.fn(),
  },
}))
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authMock,
}))
vi.mock('@/stores/admin', () => ({
  useAdminStore: () => ({ token: '', logout: vi.fn() }),
}))
vi.mock('@/router', () => ({ default: { push: vi.fn() } }))

import { request } from '@/utils/request'

function mockFetchSequence(responses: Array<{ status: number; body: unknown }>) {
  let call = 0
  const spy = vi.fn().mockImplementation(() => {
    const r = responses[Math.min(call, responses.length - 1)]
    call++
    return Promise.resolve(new Response(JSON.stringify(r.body), { status: r.status }))
  })
  vi.stubGlobal('fetch', spy)
  return spy
}

describe('doFetchWithRetry 行为（经 request 导出验证）', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    authMock.token = 'test-token'
    authMock.refresh.mockReset().mockImplementation(async () => {
      authMock.token = 'new-token'
      return 'new-token'
    })
    authMock.logout.mockReset()
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('GET 5xx 幂等重试：前两次 500 第三次 200 应成功', async () => {
    const spy = mockFetchSequence([
      { status: 500, body: {} },
      { status: 502, body: {} },
      { status: 200, body: { code: 0, data: 'ok' } },
    ])
    const p = request('/x')
    await vi.runAllTimersAsync()
    const result = await p
    expect(result).toBe('ok')
    expect(spy).toHaveBeenCalledTimes(3)
  })

  it('POST 5xx 不重试（F1 修复）：网关 502 但后端可能已入库，重试会重复建任务', async () => {
    const spy = mockFetchSequence([{ status: 500, body: {} }])
    const p = request('/tasks', { method: 'POST', body: '{}' }).catch((e) => e)
    await vi.runAllTimersAsync()
    const err = await p
    expect(err).toBeInstanceOf(Error)
    expect(spy).toHaveBeenCalledTimes(1) // 关键：只调一次
  })

  it('超时定时器被清理（F2 修复）：fetch 正常返回后无悬挂 timer', async () => {
    mockFetchSequence([{ status: 200, body: { code: 0, data: 'ok' } }])
    const p = request('/x')
    await vi.runAllTimersAsync()
    await p
    expect(vi.getTimerCount()).toBe(0)
  })

  it('网络错误（fetch reject）GET 重试后成功', async () => {
    let call = 0
    const spy = vi.fn().mockImplementation(() => {
      call++
      return call === 1 ? Promise.reject(new Error('network down'))
        : Promise.resolve(new Response(JSON.stringify({ code: 0, data: 'ok' }), { status: 200 }))
    })
    vi.stubGlobal('fetch', spy)
    const p = request('/x')
    await vi.runAllTimersAsync()
    expect(await p).toBe('ok')
    expect(spy).toHaveBeenCalledTimes(2)
  })

  it('POST 网络错误不重试：避免结果未知时重复提交写操作', async () => {
    const spy = vi.fn().mockRejectedValue(new Error('network down'))
    vi.stubGlobal('fetch', spy)
    const p = request('/tasks', { method: 'POST', body: '{}' }).catch((e) => e)
    await vi.runAllTimersAsync()
    expect(await p).toBeInstanceOf(Error)
    expect(spy).toHaveBeenCalledTimes(1)
  })

  it('刷新令牌遇到临时 5xx 时不清除会话', async () => {
    mockFetchSequence([{ status: 401, body: { message: 'invalid token' } }])
    authMock.refresh.mockRejectedValueOnce(new Error('服务器暂时不可用'))
    const p = request('/protected').catch((e) => e)
    await vi.runAllTimersAsync()
    expect(await p).toBeInstanceOf(Error)
    expect(authMock.logout).not.toHaveBeenCalled()
  })

  it('刷新令牌遇到网络错误时不清除会话', async () => {
    mockFetchSequence([{ status: 401, body: { message: 'invalid token' } }])
    authMock.refresh.mockRejectedValueOnce(new Error('网络连接失败'))
    const p = request('/protected').catch((e) => e)
    await vi.runAllTimersAsync()
    expect(await p).toBeInstanceOf(Error)
    expect(authMock.logout).not.toHaveBeenCalled()
  })

  it('refresh 明确返回 401 时仍然清除会话', async () => {
    mockFetchSequence([
      { status: 401, body: { message: 'access token expired' } },
      { status: 401, body: { message: 'refresh token expired' } },
    ])
    authMock.refresh.mockImplementationOnce(() => request('/auth/refresh'))
    const p = request('/protected').catch((e) => e)
    await vi.runAllTimersAsync()

    expect(await p).toBeInstanceOf(Error)
    expect(authMock.logout).toHaveBeenCalledTimes(1)
  })

  it('refresh 明确返回账号停用/过期 403 时仍然清除会话', async () => {
    mockFetchSequence([
      { status: 401, body: { message: 'access token expired' } },
      { status: 403, body: { message: 'tenant expired' } },
    ])
    authMock.refresh.mockImplementationOnce(() => request('/auth/refresh'))
    const p = request('/protected').catch((e) => e)
    await vi.runAllTimersAsync()

    expect(await p).toBeInstanceOf(Error)
    expect(authMock.logout).toHaveBeenCalledTimes(1)
  })

  it('请求超时会 abort 底层请求', async () => {
    let signal: AbortSignal | undefined
    const spy = vi.fn().mockImplementation((_url: string, init: RequestInit) => {
      signal = init.signal as AbortSignal
      return new Promise<Response>(() => {})
    })
    vi.stubGlobal('fetch', spy)
    const p = request('/slow', { timeout: 10 }).catch((e) => e)
    await vi.advanceTimersByTimeAsync(10)
    // GET 超时会按幂等重试策略继续执行；推进退避定时器，避免 fake timer 下等待永远挂起。
    await vi.runAllTimersAsync()
    expect(await p).toBeInstanceOf(Error)
    expect(signal?.aborted).toBe(true)
    expect(spy).toHaveBeenCalledTimes(3)
  })
  it('调用方 AbortSignal 取消请求后不进行 GET 重试', async () => {
    let signal: AbortSignal | undefined
    const spy = vi.fn().mockImplementation((_url: string, init: RequestInit) => {
      signal = init.signal as AbortSignal
      return new Promise<Response>((_resolve, reject) => {
        signal?.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')), { once: true })
      })
    })
    vi.stubGlobal('fetch', spy)
    const controller = new AbortController()
    const p = request('/cancel', { signal: controller.signal }).catch((e) => e)

    controller.abort()
    const error = await p

    expect(error).toMatchObject({ name: 'AbortError', message: '请求已取消' })
    expect(signal?.aborted).toBe(true)
    expect(spy).toHaveBeenCalledTimes(1)
  })


  it('会话切换后不使用新账号 token 重放旧请求', async () => {
    let resolveRefresh!: (token: string) => void
    let markRefreshStarted!: () => void
    const refreshStarted = new Promise<void>(resolve => { markRefreshStarted = resolve })
    authMock.refresh.mockImplementationOnce(() => {
      markRefreshStarted()
      return new Promise(resolve => { resolveRefresh = resolve })
    })
    const spy = mockFetchSequence([
      { status: 401, body: { message: 'access token expired' } },
      { status: 200, body: { code: 0, data: 'must-not-be-requested' } },
    ])

    const pending = request('/tenant/private').catch(error => error)
    await refreshStarted

    // 模拟旧请求等待续签时，用户退出旧账号并登录了另一个账号。
    authMock.token = 'different-account-token'
    resolveRefresh('old-session-refreshed-token')
    const error = await pending

    expect(error).toMatchObject({ name: 'AbortError' })
    expect(spy).toHaveBeenCalledTimes(1)
    expect(spy.mock.calls[0][1].headers.Authorization).toBe('Bearer test-token')
  })

  it('调用前已取消的请求不会发出网络请求', async () => {
    const spy = vi.fn()
    vi.stubGlobal('fetch', spy)
    const controller = new AbortController()
    controller.abort()

    const error = await request('/already-cancelled', { signal: controller.signal }).catch((e) => e)

    expect(error).toMatchObject({ name: 'AbortError', message: '请求已取消' })
    expect(spy).not.toHaveBeenCalled()
  })
  it('GET 进入退避等待时收到 AbortSignal 会立即结束且不再重试', async () => {
    const spy = vi.fn().mockRejectedValue(new Error('network down'))
    vi.stubGlobal('fetch', spy)
    const controller = new AbortController()
    const p = request('/cancel-during-backoff', { signal: controller.signal }).catch((e) => e)

    // 让第一次 fetch 的 rejection 进入退避等待，再取消。
    await Promise.resolve()
    await Promise.resolve()
    await Promise.resolve()
    controller.abort()
    const error = await p

    expect(error).toMatchObject({ name: 'AbortError', message: '请求已取消' })
    expect(spy).toHaveBeenCalledTimes(1)
    expect(vi.getTimerCount()).toBe(0)
  })


})