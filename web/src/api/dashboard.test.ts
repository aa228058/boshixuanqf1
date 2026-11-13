import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }))
vi.mock('@/utils/request', () => ({
  api: { get: getMock },
}))

import { dashboardApi } from '@/api/dashboard'

describe('dashboardApi query encoding', () => {
  beforeEach(() => getMock.mockReset())

  it('encodes account IDs before placing them in the query string', () => {
    dashboardApi.getStats('tenant id&scope=other')

    expect(getMock).toHaveBeenCalledWith('/dashboard/stats?account_id=tenant%20id%26scope%3Dother', undefined)
  })
})
