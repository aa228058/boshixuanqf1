import { defineStore } from 'pinia'
import { ref, onScopeDispose } from 'vue'
import { groupApi } from '@/api/group'
import type { Group } from '@/types'
import { onMutation } from '@/utils/mutationBus'

// R151: 群组缓存 + 自动同步。任何页面打开秒显缓存; 缓存超过 30 分钟后台自动补同步(不阻塞);
// 手动"刷新" = 全量同步。syncedAt 随条目持久化到 sessionStorage, 刷新页面不丢。
const STORAGE_KEY = 'tg_groups_cache'
const STALE_MS = 30 * 60 * 1000

interface CacheEntry {
  accountId: string
  groups: Group[]
  syncedAt: string
}

function loadCache(): Record<string, CacheEntry> {
  try {
    return JSON.parse(sessionStorage.getItem(STORAGE_KEY) || '{}')
  } catch {
    return {}
  }
}

export const useGroupStore = defineStore('group', () => {
  const cache = ref<Record<string, CacheEntry>>(loadCache())
  const refreshRequests = new Map<string, number>()
  const cacheEpoch = new Map<string, number>()

  function persist() {
    try {
      sessionStorage.setItem(STORAGE_KEY, JSON.stringify(cache.value))
    } catch { /* 存储满了或隐私模式, 忽略 */ }
  }

  // 读取缓存(可能为空数组)
  function get(accountId: string): Group[] {
    return cache.value[accountId]?.groups || []
  }

  // R151: 状态栏用 —— 某账号上次同步时间(毫秒), 无记录返回 0
  function lastSyncAt(accountId: string): number {
    const t = cache.value[accountId]?.syncedAt
    const parsed = t ? new Date(t).getTime() : 0
    return Number.isFinite(parsed) ? parsed : 0
  }

  // 打开选择器时调用: 有缓存直接返回(过期后台补同步, 不阻塞界面); 无缓存才查库
  async function ensure(accountId: string): Promise<Group[]> {
    const hit = cache.value[accountId]
    if (hit && hit.groups.length) {
      // R151 自动同步: 缓存超 30 分钟后台补一次(秒开 + 保鲜)
      const syncedAt = new Date(hit.syncedAt).getTime()
      if (!Number.isFinite(syncedAt) || Date.now() - syncedAt > STALE_MS) {
        // Do not return the stale snapshot while refreshing in the background:
        // selectors copy the returned array and otherwise never see the new result.
        return refresh(accountId)
      }
      return hit.groups
    }
    return refresh(accountId)
  }

  // R151: 同步 = 调后端拉群接口 + 立刻重拉列表写回缓存。
  // 必须写回: POST /groups/* 会触发全局 mutation 总线把缓存整个擦掉,
  // 不同步写回就会出现"同步成功但列表空白"。
  async function sync(accountId: string): Promise<Group[]> {
    await groupApi.sync(accountId)
    return refresh(accountId)
  }

  // 同步成功后写回最新列表
  function setGroups(accountId: string, groups: Group[]) {
    cache.value[accountId] = { accountId, groups, syncedAt: new Date().toISOString() }
    persist()
  }

  // 点"同步"之后重新拉取库数据
  async function refresh(accountId: string): Promise<Group[]> {
    const requestId = (refreshRequests.get(accountId) || 0) + 1
    refreshRequests.set(accountId, requestId)
    const epoch = cacheEpoch.get(accountId) || 0
    const res: any = await groupApi.list(accountId)
    // 后端返回 ListResponse{code,data:{list,total}} 或直返数组, 兼容四种形态
    // 注意 res.data 可能是 {list,total} 对象本身，必须先取其 .list
    const raw = res.list || res.data?.list || (Array.isArray(res.data) ? res.data : []) || (Array.isArray(res) ? res : [])
    const list = Array.isArray(raw) ? raw.filter((group: any) => group && typeof group === 'object') : []
    // 仅允许最新请求且期间未发生失效/写操作的结果进入缓存。
    if (refreshRequests.get(accountId) === requestId && (cacheEpoch.get(accountId) || 0) === epoch) {
      setGroups(accountId, list)
      return list
    }
    // 过期响应不能被返回给调用方继续渲染；优先返回当前缓存快照。
    return cache.value[accountId]?.groups || []
  }

  function invalidate(accountId: string) {
    cacheEpoch.set(accountId, (cacheEpoch.get(accountId) || 0) + 1)
    refreshRequests.set(accountId, (refreshRequests.get(accountId) || 0) + 1)
    delete cache.value[accountId]
    persist()
  }

  // 全局变更总线：任何 /groups 相关写操作成功后自动清缓存重拉
  const stopGroupMutation = onMutation(/\/groups/, () => {
    for (const accountId of refreshRequests.keys()) {
      cacheEpoch.set(accountId, (cacheEpoch.get(accountId) || 0) + 1)
      refreshRequests.set(accountId, (refreshRequests.get(accountId) || 0) + 1)
    }
    cache.value = {}
    try { sessionStorage.removeItem(STORAGE_KEY) } catch {}
  })
  onScopeDispose(stopGroupMutation)

  return { get, lastSyncAt, ensure, setGroups, refresh, invalidate, sync }
})
