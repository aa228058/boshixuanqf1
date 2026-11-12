export type TaskSseConnectionState = 'connected' | 'disconnected'

type TaskSseSource = { close(): void }

type TaskSseRuntimeOptions<TSource extends TaskSseSource> = {
  isActive: () => boolean
  getSource: () => TSource | null
  setSource: (source: TSource | null) => void
  setConnectionState: (state: TaskSseConnectionState) => void
  connect: () => void
  onWatchdogTimeout?: () => void
  reconnectDelayMs?: number
  watchdogDelayMs?: number
}

/**
 * Owns reconnect/watchdog timers and ignores callbacks from superseded streams.
 * Event parsing and task-specific state updates remain in the view.
 */
export function createTaskSseRuntime<TSource extends TaskSseSource>(options: TaskSseRuntimeOptions<TSource>) {
  const reconnectDelayMs = options.reconnectDelayMs ?? 3000
  const watchdogDelayMs = options.watchdogDelayMs ?? 45000
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let watchdogTimer: ReturnType<typeof setTimeout> | null = null

  function clearReconnect() {
    if (reconnectTimer) clearTimeout(reconnectTimer)
    reconnectTimer = null
  }

  function clearWatchdog() {
    if (watchdogTimer) clearTimeout(watchdogTimer)
    watchdogTimer = null
  }

  function scheduleReconnect() {
    if (!options.isActive()) return
    clearReconnect()
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      if (options.isActive()) options.connect()
    }, reconnectDelayMs)
  }

  function resetWatchdog(source: TSource) {
    clearWatchdog()
    watchdogTimer = setTimeout(() => {
      watchdogTimer = null
      if (!options.isActive() || options.getSource() !== source) return
      options.onWatchdogTimeout?.()
      disconnect(source)
    }, watchdogDelayMs)
  }

  function disconnect(source: TSource) {
    if (!options.isActive() || options.getSource() !== source) return false
    try { source.close() } catch { /* ignore close errors */ }
    options.setSource(null)
    options.setConnectionState('disconnected')
    clearWatchdog()
    scheduleReconnect()
    return true
  }

  function handleOpen(source: TSource) {
    if (!options.isActive() || options.getSource() !== source) return false
    options.setConnectionState('connected')
    clearReconnect()
    resetWatchdog(source)
    return true
  }

  function handleMessage(source: TSource) {
    if (!options.isActive() || options.getSource() !== source) return false
    resetWatchdog(source)
    return true
  }

  function handleError(source: TSource) {
    return disconnect(source)
  }

  function dispose() {
    clearReconnect()
    clearWatchdog()
  }

  return {
    scheduleReconnect,
    handleOpen,
    handleMessage,
    handleError,
    dispose,
  }
}
