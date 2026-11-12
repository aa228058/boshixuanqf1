import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const { listMock } = vi.hoisted(() => ({ listMock: vi.fn() }))
vi.mock('@/api/account', () => ({
  accountApi: {
    list: listMock,
    create: vi.fn(),
    delete: vi.fn(),
  },
}))

import { useAccountStore } from './account'

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

describe('account store request lifecycle', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    listMock.mockReset()
    localStorage.clear()
  })

  it('只允许最后发起的账号列表请求覆盖状态', async () => {
    const first = deferred<any>()
    const second = deferred<any>()
    listMock.mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise)
    const store = useAccountStore()

    const oldRequest = store.fetchAccounts(true)
    const newRequest = store.fetchAccounts(true)
    second.resolve({ list: [{ account_id: 'new', status: 'online' }] })
    await newRequest
    first.resolve({ list: [{ account_id: 'old', status: 'offline' }] })
    await oldRequest

    expect(store.accounts.map(account => account.account_id)).toEqual(['new'])
    expect(store.currentAccount?.account_id).toBe('new')
  })

  it('本地存储不可用时仍可加载账号列表', async () => {
    listMock.mockResolvedValue({ list: [{ account_id: 'safe', status: 'online' }] })
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('storage disabled') })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('storage disabled') })
    vi.spyOn(Storage.prototype, 'removeItem').mockImplementation(() => { throw new Error('storage disabled') })
    const store = useAccountStore()

    await expect(store.fetchAccounts(true)).resolves.toBeUndefined()
    expect(store.accounts.map(account => account.account_id)).toEqual(['safe'])
  })
})
