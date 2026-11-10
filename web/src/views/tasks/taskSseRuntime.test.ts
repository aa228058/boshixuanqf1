import { afterEach, describe, expect, it, vi } from 'vitest'
import { createTaskSseRuntime } from './taskSseRuntime'

type FakeSource = { close: () => void }

function createRuntime() {
  let active = true
  let currentSource: FakeSource | null = null
  const setSource = vi.fn((source: FakeSource | null) => { currentSource = source })
  const setConnectionState = vi.fn()
  const connect = vi.fn()
  const onWatchdogTimeout = vi.fn()
  const runtime = createTaskSseRuntime({
    isActive: () => active,
    getSource: () => currentSource,
    setSource,
    setConnectionState,
    connect,
    onWatchdogTimeout,
    reconnectDelayMs: 3000,
    watchdogDelayMs: 45000,
  })
  return {
    runtime,
    setActive: (value: boolean) => { active = value },
    setCurrentSource: (source: FakeSource | null) => { currentSource = source },
    setSource,
    setConnectionState,
    connect,
    onWatchdogTimeout,
    source: (): FakeSource => ({ close: vi.fn() }),
  }
}

describe('task SSE runtime', () => {
  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('coalesces reconnect requests and reconnects only while the page is active', () => {
    vi.useFakeTimers()
    const ctx = createRuntime()

    ctx.runtime.scheduleReconnect()
    ctx.runtime.scheduleReconnect()
    vi.advanceTimersByTime(2999)
    expect(ctx.connect).not.toHaveBeenCalled()

    vi.advanceTimersByTime(1)
    expect(ctx.connect).toHaveBeenCalledTimes(1)

    ctx.setActive(false)
    ctx.runtime.scheduleReconnect()
    vi.advanceTimersByTime(3000)
    expect(ctx.connect).toHaveBeenCalledTimes(1)
  })

  it('clears a pending reconnect when the current stream opens', () => {
    vi.useFakeTimers()
    const ctx = createRuntime()
    const source = ctx.source()
    ctx.setCurrentSource(source)

    ctx.runtime.scheduleReconnect()
    expect(ctx.runtime.handleOpen(source)).toBe(true)
    vi.advanceTimersByTime(3000)

    expect(ctx.connect).not.toHaveBeenCalled()
    expect(ctx.setConnectionState).toHaveBeenCalledWith('connected')
  })

  it('ignores stale stream errors and disconnects only the current source', () => {
    vi.useFakeTimers()
    const ctx = createRuntime()
    const stale = ctx.source()
    const current = ctx.source()
    ctx.setCurrentSource(current)

    expect(ctx.runtime.handleError(stale)).toBe(false)
    expect(stale.close).not.toHaveBeenCalled()
    expect(ctx.setSource).not.toHaveBeenCalled()

    expect(ctx.runtime.handleError(current)).toBe(true)
    expect(current.close).toHaveBeenCalledTimes(1)
    expect(ctx.setSource).toHaveBeenCalledWith(null)
    expect(ctx.setConnectionState).toHaveBeenCalledWith('disconnected')
    vi.advanceTimersByTime(3000)
    expect(ctx.connect).toHaveBeenCalledTimes(1)
  })

  it('restarts the watchdog on messages and ignores a superseded source timeout', () => {
    vi.useFakeTimers()
    const ctx = createRuntime()
    const oldSource = ctx.source()
    const current = ctx.source()
    ctx.setCurrentSource(current)

    expect(ctx.runtime.handleOpen(current)).toBe(true)
    vi.advanceTimersByTime(30000)
    expect(ctx.runtime.handleMessage(current)).toBe(true)
    vi.advanceTimersByTime(30000)
    expect(current.close).not.toHaveBeenCalled()

    ctx.setCurrentSource(oldSource)
    vi.advanceTimersByTime(15000)
    expect(current.close).not.toHaveBeenCalled()
    expect(ctx.onWatchdogTimeout).not.toHaveBeenCalled()
  })

  it('closes and reconnects a current stream after watchdog timeout', () => {
    vi.useFakeTimers()
    const ctx = createRuntime()
    const source = ctx.source()
    ctx.setCurrentSource(source)

    ctx.runtime.handleOpen(source)
    vi.advanceTimersByTime(45000)

    expect(ctx.onWatchdogTimeout).toHaveBeenCalledTimes(1)
    expect(source.close).toHaveBeenCalledTimes(1)
    expect(ctx.setSource).toHaveBeenCalledWith(null)
    expect(ctx.setConnectionState).toHaveBeenCalledWith('disconnected')
    vi.advanceTimersByTime(3000)
    expect(ctx.connect).toHaveBeenCalledTimes(1)
  })

  it('disposes pending timers without reconnecting after unmount', () => {
    vi.useFakeTimers()
    const ctx = createRuntime()

    ctx.runtime.scheduleReconnect()
    ctx.runtime.dispose()
    ctx.setActive(false)
    vi.advanceTimersByTime(60000)

    expect(ctx.connect).not.toHaveBeenCalled()
  })
})
