import { createRenderer, defineComponent, h } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const { fetchMock, setSseStatusMock, bumpSseRetryMock } = vi.hoisted(() => ({
  fetchMock: vi.fn(),
  setSseStatusMock: vi.fn(),
  bumpSseRetryMock: vi.fn(),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ token: 'test-token' }),
}))

vi.mock('@/stores/sse', () => ({
  setSseStatus: setSseStatusMock,
  bumpSseRetry: bumpSseRetryMock,
}))

import { reconnectNow, useTaskEvents } from './useTaskEvents'

type EventHandler = (event: Event | MessageEvent) => void

class MockEventSource {
  static instances: MockEventSource[] = []
  onopen: ((event: Event) => void) | null = null
  onerror: ((event: Event) => void) | null = null
  closed = false
  listeners = new Map<string, EventHandler[]>()

  constructor(public url: string) {
    MockEventSource.instances.push(this)
  }

  addEventListener(type: string, listener: EventHandler) {
    const entries = this.listeners.get(type) ?? []
    entries.push(listener)
    this.listeners.set(type, entries)
  }

  close() {
    this.closed = true
  }

  emit(type: string, event: Event | MessageEvent = new Event(type)) {
    this.listeners.get(type)?.forEach((listener) => listener(event))
  }
}

const renderer = createRenderer<any, any>({
  createElement: (type) => ({ type, children: [] as any[] }),
  createText: (text) => ({ text }),
  createComment: (text) => ({ comment: text }),
  setText: (node, text) => { node.text = text },
  setElementText: (node, text) => { node.text = text },
  parentNode: () => null,
  nextSibling: () => null,
  insert: (node, parent) => { parent.children.push(node) },
  remove: (node) => {
    const parent = node.parent
    if (parent) parent.children = parent.children.filter((child: any) => child !== node)
  },
  patchProp: () => {},
})

function mountSubscriptions(count = 1) {
  const component = defineComponent({
    setup() {
      for (let index = 0; index < count; index++) {
        useTaskEvents(() => {})
      }
      return () => h('div')
    },
  })
  const app = renderer.createApp(component)
  app.mount({ children: [] })
  return () => app.unmount()
}

async function flushPromises() {
  // fetch() and response.json() each add promise turns before EventSource is created.
  for (let index = 0; index < 10; index++) await Promise.resolve()
}

describe('useTaskEvents SSE lifecycle', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.stubGlobal('EventSource', MockEventSource)
    vi.stubGlobal('fetch', fetchMock)
    fetchMock.mockReset()
    setSseStatusMock.mockReset()
    bumpSseRetryMock.mockReset()
    MockEventSource.instances = []
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ ticket: 'one-time-ticket' }),
    })
  })

  afterEach(async () => {
    await vi.runOnlyPendingTimersAsync()
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('并发订阅时只换票并创建一个 EventSource', async () => {
    const unmount = mountSubscriptions(4)
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/events/ticket', expect.objectContaining({ method: 'POST' }))
    expect(MockEventSource.instances).toHaveLength(1)
    expect(MockEventSource.instances[0].url).toContain('/api/v1/events?ticket=')

    unmount()
    await vi.advanceTimersByTimeAsync(30000)
  })

  it('看门狗关闭假活连接后会清空引用并建立新连接', async () => {
    const unmount = mountSubscriptions()
    await flushPromises()

    const first = MockEventSource.instances[0]
    first.onopen?.(new Event('open'))
    await vi.advanceTimersByTimeAsync(45000)
    expect(first.closed).toBe(true)

    await vi.advanceTimersByTimeAsync(10000)
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(MockEventSource.instances).toHaveLength(2)
    expect(MockEventSource.instances[1].closed).toBe(false)

    unmount()
    await vi.advanceTimersByTimeAsync(30000)
  })


  it('手动重连会主动取消旧换票请求并立即使用新代次', async () => {
    let firstSignal: AbortSignal | undefined
    fetchMock.mockImplementationOnce((_url: string, options: RequestInit) => new Promise((_resolve, reject) => {
      firstSignal = options.signal as AbortSignal
      firstSignal.addEventListener('abort', () => {
        const error = new Error('aborted')
        error.name = 'AbortError'
        reject(error)
      }, { once: true })
    }))
    const unmount = mountSubscriptions()
    await flushPromises()

    reconnectNow()
    await flushPromises()
    expect(firstSignal?.aborted).toBe(true)
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(MockEventSource.instances).toHaveLength(1)
    expect(MockEventSource.instances[0].url).toContain('one-time-ticket')

    unmount()
    await vi.advanceTimersByTimeAsync(30000)
  })

  it('旧 EventSource 的迟到 error 不会关闭新连接', async () => {
    const unmount = mountSubscriptions()
    await flushPromises()
    const first = MockEventSource.instances[0]
    first.onerror?.(new Event('error'))
    await vi.advanceTimersByTimeAsync(10000)
    await flushPromises()
    const second = MockEventSource.instances[1]
    expect(second).toBeDefined()

    first.onerror?.(new Event('late-error'))
    expect(second.closed).toBe(false)
    expect(MockEventSource.instances).toHaveLength(2)

    unmount()
    await vi.advanceTimersByTimeAsync(30000)
  })

  it('最后一个订阅者离开并完成宽限关闭后会取消仍在进行的 ticket 请求', async () => {
    let firstSignal: AbortSignal | undefined
    fetchMock.mockImplementationOnce((_url: string, options: RequestInit) => new Promise((_resolve, reject) => {
      firstSignal = options.signal as AbortSignal
      firstSignal.addEventListener('abort', () => {
        const error = new Error('aborted')
        error.name = 'AbortError'
        reject(error)
      }, { once: true })
    }))

    const unmount = mountSubscriptions()
    unmount()
    await vi.advanceTimersByTimeAsync(30000)
    await flushPromises()

    expect(firstSignal?.aborted).toBe(true)
    expect(MockEventSource.instances).toHaveLength(0)
  })
})
