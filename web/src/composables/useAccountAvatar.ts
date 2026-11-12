import { reactive } from 'vue'
import { api } from '@/utils/request'

// R166: 头像缓存响应式化 —— 原实现两个问题:
//   1) 异步 blob 加载完成后写入普通对象, 无响应式触发, 组件永不重绘 → 字母头像永远不变成真头像
//   2) 把 blob: ObjectURL 存 sessionStorage —— 刷新页面即失效, 只会得到死链头像
// 现改为 reactive 缓存(加载完成自动重绘) + 失败自动重试(不缓存失败态) + 1h 内存 TTL。

const TTL = 60 * 60 * 1000
const REFRESH_RETRY_DELAY_MS = 8_000
const cache = reactive<Record<string, { url: string; ts: number }>>({})
const pending = new Set<string>()
const requestIds = new Map<string, number>()
const refreshing = new Set<string>() // R172: 正在自动重拉头像的账号, 防重复触发
const refreshTimers = new Map<string, ReturnType<typeof setTimeout>>()

export function useAccountAvatar() {
  // List.vue passes the full account object; other callers may pass the ID directly.
  const avatarOf = (account: string | { account_id?: string } | null | undefined): string | null => {
    const accountId = typeof account === 'string' ? account : account?.account_id || ''
    if (!accountId) return null
    const hit = cache[accountId]
    if (hit && Date.now() - hit.ts < TTL) return hit.url
    if (pending.has(accountId)) return null
    const requestId = (requestIds.get(accountId) || 0) + 1
    requestIds.set(accountId, requestId)
    pending.add(accountId)
    api
      .get(`/accounts/${accountId}/avatar`, { responseType: 'blob' })
      .then((res: any) => {
        const url = URL.createObjectURL(res)
        if (requestIds.get(accountId) !== requestId) {
          URL.revokeObjectURL(url)
          return
        }
        const previous = cache[accountId]
        if (previous) URL.revokeObjectURL(previous.url)
        cache[accountId] = { url, ts: Date.now() }
      })
      .catch((err: any) => {
        if (requestIds.get(accountId) !== requestId) return
        // R166: 失败不缓存, 下次渲染自动重试(原实现缓存了失败态, 24h 内不再尝试)
        // R172: 404(文件丢失, 如容器重建) → 自动通知后端重新拉取头像, 全程无感,
        // 30s 冷却后可再次触发(头像拉取是异步任务, 立即重试多半还是 404)
        if (err?.status === 404 && !refreshing.has(accountId)) {
          refreshing.add(accountId)
          const refreshGeneration = requestIds.get(accountId) || 0
          const refreshStartedAt = Date.now()
          api
            .post(`/accounts/${accountId}/avatar/refresh`, {})
            .catch(() => {})
            .finally(() => {
              // invalidate/新请求发生后，旧刷新流程不得再安排清理新缓存的 timer。
              if (requestIds.get(accountId) !== refreshGeneration) {
                refreshing.delete(accountId)
                return
              }
              const previousTimer = refreshTimers.get(accountId)
              if (previousTimer) clearTimeout(previousTimer)
              const timer = setTimeout(() => {
                refreshTimers.delete(accountId)
                refreshing.delete(accountId)
                // 只清理刷新请求开始前已有的旧缓存；8 秒内成功取得的新头像必须保留。
                const stale = cache[accountId]
                if (
                  requestIds.get(accountId) === refreshGeneration &&
                  stale &&
                  stale.ts <= refreshStartedAt
                ) {
                  try { URL.revokeObjectURL(stale.url) } catch { /* 已失效 */ }
                  delete cache[accountId]
                }
              }, REFRESH_RETRY_DELAY_MS)
              refreshTimers.set(accountId, timer)
            })
        }
      })
      .finally(() => {
        if (requestIds.get(accountId) === requestId) pending.delete(accountId)
      })
    return null
  }

  const invalidate = (accountId: string) => {
    requestIds.set(accountId, (requestIds.get(accountId) || 0) + 1)
    const refreshTimer = refreshTimers.get(accountId)
    if (refreshTimer) {
      clearTimeout(refreshTimer)
      refreshTimers.delete(accountId)
    }
    refreshing.delete(accountId)
    const old = cache[accountId]
    if (old) {
      try { URL.revokeObjectURL(old.url) } catch { /* 已失效 */ }
      delete cache[accountId]
    }
    pending.delete(accountId)
  }

  return { avatarOf, invalidate }
}
