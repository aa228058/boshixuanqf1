import { onMounted, onUnmounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { setSseStatus, bumpSseRetry } from '@/stores/sse'

// ===== 任务实时事件(SSE) · 单连接常驻版 =====
let es: EventSource | null = null
let connecting: Promise<void> | null = null
let ticketController: AbortController | null = null
const listeners = new Set<(evt: any) => void>()
let refs = 0
let reconnects = 0
let retryTimer: ReturnType<typeof setTimeout> | null = null
let disconnectTimer: ReturnType<typeof setTimeout> | null = null
let watchdogTimer: ReturnType<typeof setTimeout> | null = null
let connectionGeneration = 0
const DISCONNECT_GRACE_MS = 30000
const MAX_RETRIES = 8
const WATCHDOG_TIMEOUT_MS = 45000
const API_BASE_URL = (import.meta as any).env?.VITE_API_BASE_URL || '/api/v1'

async function fetchTicket(baseUrl: string, signal: AbortSignal): Promise<string | null> {
  try {
    const token = useAuthStore().token
    if (!token) return null
    const res = await fetch(`${baseUrl}/events/ticket`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
      signal,
    })
    if (!res.ok) return null
    const data = await res.json()
    return data.ticket || null
  } catch {
    return null
  }
}

function clearWatchdog() {
  if (watchdogTimer) {
    clearTimeout(watchdogTimer)
    watchdogTimer = null
  }
}

function resetWatchdog(source: EventSource) {
  clearWatchdog()
  watchdogTimer = setTimeout(() => {
    if (es !== source) return
    console.warn('[task-events] SSE 45s 无消息，判定假活，强制重连')
    es = null
    source.close()
    clearWatchdog()
    setSseStatus('reconnecting')
    bumpSseRetry()
    scheduleReconnect()
  }, WATCHDOG_TIMEOUT_MS)
}

function scheduleReconnect() {
  if (refs <= 0) return
  if (reconnects >= MAX_RETRIES) {
    console.warn('[task-events] SSE 重连已达上限，转 60s 低频无限重试')
    scheduleRetry(60000)
    return
  }
  reconnects++
  scheduleRetry(Math.min(10000 * reconnects, 30000))
}

function scheduleRetry(ms: number) {
  if (retryTimer) clearTimeout(retryTimer)
  retryTimer = setTimeout(() => {
    retryTimer = null
    void ensureConnected()
  }, ms)
}

function ensureConnected(): Promise<void> {
  if (es || refs <= 0) return Promise.resolve()
  if (connecting) return connecting

  const generation = connectionGeneration
  const controller = new AbortController()
  ticketController = controller
  const pending = (async () => {
    try {
      const ticket = await fetchTicket(API_BASE_URL, controller.signal)
      // 页面已全部卸载或连接代次已变化时，丢弃迟到的换票结果。
      if (generation !== connectionGeneration || refs <= 0) return
      if (!ticket) {
        scheduleReconnect()
        return
      }

      const source = new EventSource(`${API_BASE_URL}/events?ticket=${encodeURIComponent(ticket)}`)
      es = source

      source.onopen = () => {
        if (es !== source) return
        reconnects = 0
        setSseStatus('connected')
        resetWatchdog(source)
      }

      source.addEventListener('heartbeat', () => {
        if (es === source) resetWatchdog(source)
      })

      source.addEventListener('events', (m: MessageEvent) => {
        if (es !== source) return
        try {
          resetWatchdog(source)
          const raw = JSON.parse(m.data)
          // 全局事件帧为 broker Event{topic,payload}；业务字段位于 payload。
          const evt = raw && raw.payload ? raw.payload : raw
          ;[...listeners].forEach((fn) => {
            try { fn(evt) } catch { /* 单个回调异常不影响其他订阅者 */ }
          })
        } catch {
          // 忽略非 JSON 帧
        }
      })

      source.onerror = () => {
        // 旧连接的迟到 error 不得关闭后来建立的新连接。
        if (es !== source) return
        console.warn('[task-events] SSE 异常，退避重建')
        clearWatchdog()
        es = null
        source.close()
        setSseStatus('reconnecting')
        bumpSseRetry()
        scheduleReconnect()
      }
    } catch {
      if (generation === connectionGeneration) scheduleReconnect()
    }
  })()

  connecting = pending
  void pending.finally(() => {
    if (connecting === pending) connecting = null
    if (ticketController === controller) ticketController = null
    // teardown 后若有新页面重新订阅，补建新代次的连接。
    if (generation !== connectionGeneration && refs > 0 && !es) {
      void ensureConnected()
    }
  })
  return pending
}

function teardown() {
  connectionGeneration++
  ticketController?.abort()
  ticketController = null
  setSseStatus('disconnected')
  clearWatchdog()
  if (es) {
    es.close()
    es = null
  }
  if (retryTimer) {
    clearTimeout(retryTimer)
    retryTimer = null
  }
  if (disconnectTimer) {
    clearTimeout(disconnectTimer)
    disconnectTimer = null
  }
  refs = 0
  reconnects = 0
}

export function useTaskEvents(onEvent: (evt: any) => void) {
  onMounted(() => {
    listeners.add(onEvent)
    refs++
    if (disconnectTimer) {
      clearTimeout(disconnectTimer)
      disconnectTimer = null
    }
    void ensureConnected()
  })

  onUnmounted(() => {
    listeners.delete(onEvent)
    refs--
    if (refs <= 0) {
      refs = 0
      // 页面切换保留 30 秒连接；最后一个订阅者离开后再统一关闭。
      if (disconnectTimer) clearTimeout(disconnectTimer)
      disconnectTimer = setTimeout(() => {
        disconnectTimer = null
        if (refs <= 0) teardown()
      }, DISCONNECT_GRACE_MS)
    }
  })
}

// 手动重连：关闭旧 EventSource、清除退避等待并立即换取新 ticket。
export function reconnectNow() {
  if (retryTimer) {
    clearTimeout(retryTimer)
    retryTimer = null
  }
  // Invalidate any in-flight ticket request so it cannot win this explicit reconnect.
  // Its finally handler will start the new generation after the old request settles.
  connectionGeneration++
  ticketController?.abort()
  ticketController = null
  clearWatchdog()
  if (es) {
    es.close()
    es = null
  }
  reconnects = 0
  void ensureConnected()
}
