import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock, postMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
}))

vi.mock('@/utils/request', () => ({
  api: { get: getMock, post: postMock },
}))

import { useAccountAvatar } from './useAccountAvatar'

async function flushPromises() {
  for (let index = 0; index < 12; index++) await Promise.resolve()
}

describe('useAccountAvatar refresh lifecycle', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    getMock.mockReset()
    postMock.mockReset().mockResolvedValue(undefined)
    vi.stubGlobal('URL', {
      createObjectURL: vi.fn(() => `blob:test-${Math.random()}`),
      revokeObjectURL: vi.fn(),
    })
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('8 秒刷新窗口内取得的新头像不会被旧清理 timer 删除', async () => {
    const { avatarOf } = useAccountAvatar()
    const freshBlob = new Blob(['fresh-avatar'])
    getMock
      .mockRejectedValueOnce(Object.assign(new Error('missing'), { status: 404 }))
      .mockResolvedValueOnce(freshBlob)

    expect(avatarOf('avatar-refresh-cache-test')).toBeNull()
    await flushPromises()
    expect(postMock).toHaveBeenCalledTimes(1)

    expect(avatarOf('avatar-refresh-cache-test')).toBeNull()
    await flushPromises()
    await vi.advanceTimersByTimeAsync(8_000)

    const cachedUrl = avatarOf('avatar-refresh-cache-test')
    expect(cachedUrl).toMatch(/^blob:test-/)
    expect(URL.revokeObjectURL).not.toHaveBeenCalledWith(cachedUrl)
  })

  it('invalidate 会取消旧刷新清理任务并使迟到缓存失效', async () => {
    const { avatarOf, invalidate } = useAccountAvatar()
    getMock
      .mockRejectedValueOnce(Object.assign(new Error('missing'), { status: 404 }))
      .mockResolvedValueOnce(new Blob(['replacement']))

    expect(avatarOf('avatar-refresh-invalidate-test')).toBeNull()
    await flushPromises()
    expect(postMock).toHaveBeenCalledTimes(1)

    invalidate('avatar-refresh-invalidate-test')
    await vi.advanceTimersByTimeAsync(8_000)
    expect(avatarOf('avatar-refresh-invalidate-test')).toBeNull()
    expect(getMock).toHaveBeenCalledTimes(2)
  })
})
