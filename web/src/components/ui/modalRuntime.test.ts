import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { createModalRuntime } from './modalRuntime'

function createKeyboardEnvironment() {
  const listeners = new Map<string, Set<EventListenerOrEventListenerObject>>()
  const viewport = {
    height: 500,
    offsetTop: 0,
    addEventListener: vi.fn((type: string, listener: EventListenerOrEventListenerObject) => {
      const entries = listeners.get(type) ?? new Set()
      entries.add(listener)
      listeners.set(type, entries)
    }),
    removeEventListener: vi.fn((type: string, listener: EventListenerOrEventListenerObject) => {
      listeners.get(type)?.delete(listener)
    }),
  } as unknown as VisualViewport
  const setProperty = vi.fn()
  const environment = {
    window: { innerHeight: 800, visualViewport: viewport } as unknown as Window,
    document: { documentElement: { style: { setProperty } } } as unknown as Document,
  }
  return { environment, viewport, listeners, setProperty }
}

describe('modalRuntime', () => {
  afterEach(() => vi.restoreAllMocks())

  it('keeps z-index lookup pure and compacts the active stack when a modal closes', () => {
    const runtime = createModalRuntime(() => ({}))
    const firstId = Symbol('first')
    const secondId = Symbol('second')
    const firstOrder = ref(0)
    const secondOrder = ref(0)

    runtime.registerModal(firstId, firstOrder)
    runtime.registerModal(secondId, secondOrder)

    expect(firstOrder.value).toBe(1)
    expect(secondOrder.value).toBe(2)
    expect(runtime.getZIndex(secondOrder.value)).toBe(2540)
    expect(runtime.getZIndex(secondOrder.value)).toBe(2540)

    runtime.unregisterModal(firstId, firstOrder)
    expect(firstOrder.value).toBe(0)
    expect(secondOrder.value).toBe(1)
    expect(runtime.getZIndex(secondOrder.value, 3000)).toBe(3000)

    runtime.unregisterModal(secondId, secondOrder)
    expect(secondOrder.value).toBe(0)
  })

  it('shares viewport listeners until the last modal closes', () => {
    const { environment, viewport, listeners, setProperty } = createKeyboardEnvironment()
    const runtime = createModalRuntime(() => environment)
    const firstId = Symbol('first')
    const secondId = Symbol('second')

    runtime.activateKeyboardGuard(firstId)
    runtime.activateKeyboardGuard(secondId)

    expect(viewport.addEventListener).toHaveBeenCalledTimes(2)
    expect(listeners.get('resize')?.size).toBe(1)
    expect(listeners.get('scroll')?.size).toBe(1)
    expect(setProperty).toHaveBeenLastCalledWith('--modal-kb', '300px')

    ;(viewport as unknown as { height: number }).height = 620
    for (const listener of listeners.get('resize') || []) {
      if (typeof listener === 'function') listener(new Event('resize'))
      else listener.handleEvent(new Event('resize'))
    }
    expect(setProperty).toHaveBeenLastCalledWith('--modal-kb', '180px')

    runtime.deactivateKeyboardGuard(firstId)
    expect(viewport.removeEventListener).not.toHaveBeenCalled()
    expect(setProperty).toHaveBeenLastCalledWith('--modal-kb', '180px')

    runtime.deactivateKeyboardGuard(secondId)
    expect(viewport.removeEventListener).toHaveBeenCalledTimes(2)
    expect(listeners.get('resize')?.size).toBe(0)
    expect(listeners.get('scroll')?.size).toBe(0)
    expect(setProperty).toHaveBeenLastCalledWith('--modal-kb', '0px')
  })

  it('ignores small viewport changes and duplicate modal registration', () => {
    const { environment, viewport, setProperty } = createKeyboardEnvironment()
    const runtime = createModalRuntime(() => environment)
    const id = Symbol('modal')
    const order = ref(0)

    runtime.registerModal(id, order)
    runtime.registerModal(id, order)
    expect(order.value).toBe(1)

    ;(viewport as unknown as { height: number }).height = 750
    runtime.activateKeyboardGuard(id)
    expect(setProperty).toHaveBeenLastCalledWith('--modal-kb', '0px')
    runtime.deactivateKeyboardGuard(id)
  })
  it('preserves stack order when a middle modal closes and later reopens', () => {
    const runtime = createModalRuntime(() => ({}))
    const firstId = Symbol('first')
    const middleId = Symbol('middle')
    const lastId = Symbol('last')
    const firstOrder = ref(0)
    const middleOrder = ref(0)
    const lastOrder = ref(0)

    runtime.registerModal(firstId, firstOrder)
    runtime.registerModal(middleId, middleOrder)
    runtime.registerModal(lastId, lastOrder)
    runtime.unregisterModal(middleId, middleOrder)

    expect([firstOrder.value, middleOrder.value, lastOrder.value]).toEqual([1, 0, 2])

    runtime.registerModal(middleId, middleOrder)
    expect([firstOrder.value, lastOrder.value, middleOrder.value]).toEqual([1, 2, 3])

    // A repeated close/unmount cleanup must not disturb the remaining stack.
    runtime.unregisterModal(middleId, middleOrder)
    runtime.unregisterModal(middleId, middleOrder)
    expect([firstOrder.value, lastOrder.value, middleOrder.value]).toEqual([1, 2, 0])
  })

  it('keeps keyboard listeners registered once for duplicate activation of one modal', () => {
    const { environment, viewport, listeners, setProperty } = createKeyboardEnvironment()
    const runtime = createModalRuntime(() => environment)
    const id = Symbol('modal')

    runtime.activateKeyboardGuard(id)
    runtime.activateKeyboardGuard(id)
    expect(viewport.addEventListener).toHaveBeenCalledTimes(2)
    expect(listeners.get('resize')?.size).toBe(1)

    runtime.deactivateKeyboardGuard(id)
    runtime.deactivateKeyboardGuard(id)
    expect(viewport.removeEventListener).toHaveBeenCalledTimes(2)
    expect(listeners.get('resize')?.size).toBe(0)
    expect(setProperty).toHaveBeenLastCalledWith('--modal-kb', '0px')
  })

  it('does not attach keyboard listeners when the visual viewport is unavailable', () => {
    const setProperty = vi.fn()
    const runtime = createModalRuntime(() => ({
      window: { innerHeight: 800, visualViewport: undefined } as unknown as Window,
      document: { documentElement: { style: { setProperty } } } as unknown as Document,
    }))
    const id = Symbol('modal')

    runtime.activateKeyboardGuard(id)
    runtime.deactivateKeyboardGuard(id)

    expect(setProperty).not.toHaveBeenCalled()
  })

})
