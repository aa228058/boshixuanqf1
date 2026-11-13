import { onMounted, onUnmounted } from 'vue'

// 页面可见时自动轮询刷新数据; 切到后台自动暂停, 回来自动续上
// 用法: useAutoRefresh(fetchTasks, 5000)
export function useAutoRefresh(fn: () => void | Promise<void>, intervalMs = 5000) {
  let timer: ReturnType<typeof setInterval> | null = null
  let active = false
  let pending = false
  const run = () => {
    if (!active || pending || document.visibilityState !== 'visible') return
    pending = true
    Promise.resolve()
      .then(() => { if (active) return fn() })
      .catch((error) => { console.warn('[useAutoRefresh] 刷新失败:', error) })
      .finally(() => { pending = false })
  }
  const tick = () => run()
  const start = () => {
    if (!timer) timer = setInterval(tick, Number.isFinite(intervalMs) ? Math.max(250, intervalMs) : 5000)
  }
  const stop = () => {
    if (timer) { clearInterval(timer); timer = null }
  }
  const onVis = () => {
    if (document.visibilityState === 'visible') run() // 回到前台立即刷一次
  }
  onMounted(() => {
    active = true
    document.addEventListener('visibilitychange', onVis)
    run()
    start()
  })
  onUnmounted(() => {
    active = false
    document.removeEventListener('visibilitychange', onVis)
    stop()
  })
}
