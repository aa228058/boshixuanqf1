import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const { listMock, syncMock } = vi.hoisted(() => ({
  listMock: vi.fn(),
  syncMock: vi.fn(),
}))
vi.mock('@/api/group', () => ({
  groupApi: { list: listMock, sync: syncMock },
}))

import { useGroupStore } from './group'

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>(res => { resolve = res })
  return { promise, resolve }
}

describe('group store refresh lifecycle', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    listMock.mockReset()
    syncMock.mockReset()
    sessionStorage.clear()
  })

  it('失效后到达的旧群组响应不会返回给调用方或写入缓存', async () => {
    const pending = deferred<any>()
    listMock.mockReturnValueOnce(pending.promise)
    const store = useGroupStore()

    const refresh = store.refresh('account-1')
    store.invalidate('account-1')
    pending.resolve({ list: [{ id: 'stale', title: 'stale' }] })

    await expect(refresh).resolves.toEqual([])
    expect(store.get('account-1')).toEqual([])
  })

  it('损坏的 syncedAt 被视为过期并触发重新同步', async () => {
    sessionStorage.setItem('tg_groups_cache', JSON.stringify({
      'account-2': {
        accountId: 'account-2',
        groups: [{ id: 'cached' }],
        syncedAt: 'not-a-date',
      },
    }))
    listMock.mockResolvedValueOnce({ list: [{ id: 'fresh' }] })
    const store = useGroupStore()

    await expect(store.ensure('account-2')).resolves.toEqual([{ id: 'fresh' }])
    expect(listMock).toHaveBeenCalledTimes(1)
    expect(store.get('account-2')).toEqual([{ id: 'fresh' }])
  })
})
