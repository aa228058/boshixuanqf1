<template>
  <Modal v-model="visible" title="选择群组">
    <div class="flex flex-col max-h-[min(520px,calc(85vh-180px))]">
      <!-- 固定头部：标签 + 操作 + 搜索 -->
      <div class="shrink-0 space-y-3 pb-3">
        <div class="flex items-center justify-between gap-2">
          <div class="flex gap-2">
            <button type="button"
              v-for="tab in tabs"
              :key="tab.value"
              @click="activeTab = tab.value"
              class="px-3 py-1.5 rounded-lg text-xs font-medium transition-colors shrink-0 whitespace-nowrap"
              :class="activeTab === tab.value ? 'bg-cyan-glow/50 text-cyan border border-cyan/30' : 'text-text-secondary hover:text-text-primary'"
            >
              {{ tab.label }}
            </button>
          </div>
          <div class="flex items-center gap-2">
            <button type="button"
              v-if="props.mode === 'multiple' && activeTab !== 'dm' && filtered.length > 0"
              @click="selectAll"
              class="px-2.5 py-1.5 rounded-lg text-xs border border-border hover:border-cyan/30 hover:text-cyan transition-colors"
            >
              {{ allSelected ? '取消全选' : '全选' }}
            </button>
            <button type="button"
              v-else-if="props.mode === 'multiple' && activeTab === 'dm' && dmFiltered.length > 0"
              @click="selectAllDm"
              class="px-2.5 py-1.5 rounded-lg text-xs border border-border hover:border-cyan/30 hover:text-cyan transition-colors"
            >
              {{ allDmSelected ? '取消全选' : '全选' }}
            </button>
            <button type="button"
              @click="onRefresh"
              :disabled="loading"
              class="px-2.5 py-1.5 rounded-lg text-xs border border-border hover:border-cyan/30 hover:text-cyan transition-colors flex items-center gap-1 disabled:opacity-50"
              title="从Telegram同步并刷新"
             aria-label="刷新">
              <RefreshCw :class="['w-3.5 h-3.5', loading && 'animate-spin']" />
              {{ loading ? '同步中...' : '同步刷新' }}
            </button>
          </div>
        </div>
        <t-input v-model="search" placeholder="搜索群组名称或用户名..." clearable class="w-full">
          <template #prefix-icon>
            <Search class="w-4 h-4" />
          </template>
        </t-input>
      </div>

      <!-- 滚动列表 -->
      <div class="flex-1 overflow-y-auto space-y-2 min-h-0">
        <!-- 加载中 -->
        <div v-if="loading" class="py-12 text-center">
          <div class="animate-spin w-6 h-6 border-2 border-cyan border-t-transparent rounded-full mx-auto mb-3" />
          <p class="text-sm text-text-muted max-w-[65ch]">
            {{ syncStatus === 'syncing' ? '正在同步 Telegram 群组...' : '加载中...' }}
          </p>
          <p v-if="syncStatus === 'syncing'" class="text-xs text-text-secondary mt-1">
            首次同步约需 15-30 秒，请稍候
          </p>
        </div>
        <!-- 群组 tab: 普通群 + 超级群(频道已排除) -->
        <template v-else-if="activeTab !== 'dm'">
          <div
            v-for="g in displayGroups"
            :key="g.group_id"
            @click="toggle(g)"
            class="flex items-center gap-3 p-3 rounded-lg border border-border cursor-pointer transition-[transform,opacity,border-color,background-color]"
            :class="isSelected(g) ? 'border-cyan/50 bg-cyan/5' : 'hover:bg-bg-hover'"
          >
            <t-checkbox
              :aria-label="'选择群组 ' + g.title"
            :model-value="isSelected(g)"
              @click.stop
              @update:model-value="toggle(g)"
            />
            <div class="w-10 h-10 rounded-lg bg-bg-card flex items-center justify-center shrink-0">
              <Users v-if="g.type === 'group'" class="w-5 h-5 text-cyan" />
              <Megaphone v-else-if="g.type === 'channel'" class="w-5 h-5 text-cyan" />
              <UsersRound v-else class="w-5 h-5 text-amber" />
            </div>
            <div class="flex-1 min-w-0">
              <div class="font-medium text-sm truncate max-w-[65ch]">{{ g.title }}</div>
              <div class="flex items-center gap-2 mt-0.5">
                <span class="text-xs text-text-muted">{{ g.member_count }} 成员</span>
                <t-tag v-if="g.is_joined" theme="success" variant="light">已加入</t-tag>
              </div>
            </div>
          </div>
          <button v-if="displayGroups.length < filtered.length" type="button" @click="visibleGroupCount += 200" class="w-full py-2 text-xs text-cyan hover:text-cyan/80">加载更多群组（{{ displayGroups.length }} / {{ filtered.length }}）</button>
        </template>
        <!-- 私聊 tab: 采集过的成员 -->
        <template v-else>
          <div
            v-for="u in dmDisplay"
            :key="u.group_id"
            @click="toggle(u)"
            class="flex items-center gap-3 p-3 rounded-lg border border-border cursor-pointer transition-[transform,opacity,border-color,background-color]"
            :class="isSelected(u) ? 'border-cyan/50 bg-cyan/5' : 'hover:bg-bg-hover'"
           tabindex="0" role="button">
            <t-checkbox
              :model-value="isSelected(u)"
              @click.stop
              @update:model-value="toggle(u)"
            />
            <div class="w-10 h-10 rounded-lg bg-bg-card flex items-center justify-center shrink-0">
              <UsersRound class="w-5 h-5 text-amber" />
            </div>
            <div class="flex-1 min-w-0">
              <div class="font-medium text-sm truncate max-w-[65ch]">{{ u.title }}</div>
              <div class="flex items-center gap-2 mt-0.5">
                <span class="text-xs text-text-muted">{{ u.username ? '@' + u.username : 'TG 用户 ' + u.tg_chat_id }}</span>
                <t-tag theme="warning" variant="light">私聊</t-tag>
              </div>
            </div>
          </div>
          <button v-if="dmDisplay.length < dmFiltered.length" type="button" @click="dmVisibleCount += 200" class="w-full py-2 text-xs text-cyan hover:text-cyan/80">加载更多私聊对象（{{ dmDisplay.length }} / {{ dmFiltered.length }}）</button>
        </template>
        <div v-if="!loading && (activeTab === 'dm' ? !dmFiltered.length : !filtered.length)" class="py-8 text-center">
          <div class="w-16 h-16 rounded-[var(--t-r-lg)] bg-bg-card flex items-center justify-center mx-auto mb-3">
            <Search class="w-8 h-8 text-text-muted" />
          </div>
          <template v-if="activeTab === 'dm'">
            <p class="text-sm text-text-secondary mb-1 max-w-[65ch]">暂无私聊对象</p>
            <p class="text-xs text-text-secondary">先在「成员采集」里采集任意群的成员，成员会自动成为私聊目标</p>
          </template>
          <template v-else>
            <p class="text-sm text-text-secondary mb-1 max-w-[65ch]">
              {{ syncStatus === 'timeout' ? '同步超时，请重试' : '没有找到群组' }}
            </p>
            <p v-if="syncStatus === 'timeout'" class="text-xs text-text-secondary">
              该账号可能未加入任何群组，或 Telegram 连接异常
            </p>
            <p v-else class="text-xs text-text-secondary">该账号尚未加入任何群组</p>
          </template>
        </div>
      </div>
    </div>
    <template #footer>
      <span class="text-xs text-text-secondary">已选 {{ selected.length }} 个</span>
      <div class="flex gap-3">
        <t-button variant="outline" theme="default" class="ui-t-button ui-t-button--secondary" @click="visible = false">取消</t-button>
        <t-button theme="primary" class="ui-t-button ui-t-button--primary" @click="confirm">确认</t-button>
      </div>
    </template>
  </Modal>
</template>

<script setup lang="ts">

import { ref, computed, watch, nextTick } from 'vue'
import { Users, Megaphone, UsersRound, Search, RefreshCw } from 'lucide-vue-next'
import { groupApi } from '@/api/group'
// R115: needsSync 死导入已删(正文零使用)
import { syncLock, markSyncing } from '@/utils/syncLock'
import { useGroupStore } from '@/stores/group'
import Modal from '@/components/ui/Modal.vue'
import { toggleSelectionByKey } from './selection'
import { uniqueByNormalizedId } from './selectorRuntime'
import type { Group } from '@/types'

const props = defineProps<{
  modelValue: boolean
  groups: Group[]
  mode: 'single' | 'multiple'
  accountId?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'select', v: Group[]): void
  (e: 'refresh'): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v)
})

const tabs = [
  { label: '群组', value: 'group' },
]

const activeTab = ref('group')
const search = ref('')
const selected = ref<Group[]>([])
const loading = ref(false)
const syncStatus = ref<'idle' | 'syncing' | 'done' | 'timeout' | 'error'>('idle')
const localGroups = ref<Group[]>([])
const groupStore = useGroupStore()
// 账号切换/关闭弹窗后，旧请求不能再写入当前选择器状态。
let selectorRequestId = 0
let dmRequestId = 0

// props.groups 是父组件的权威快照；空数组也必须同步，不能把旧账号的列表留在界面。
watch(() => props.groups, (groups) => {
  localGroups.value = [...(groups || [])]
}, { deep: true, immediate: true })

async function loadAccountGroups(requestId = ++selectorRequestId, accountId = props.accountId || '') {
  if (!accountId) {
    localGroups.value = []
    return
  }
  const suppliedGroups = props.groups || []
  if (suppliedGroups.length) {
    localGroups.value = [...suppliedGroups]
    return
  }
  loading.value = true
  try {
    const groups = await groupStore.ensure(accountId)
    if (requestId === selectorRequestId && visible.value && props.accountId === accountId) {
      localGroups.value = [...groups]
    }
  } catch (e) {
    if (requestId === selectorRequestId && visible.value && props.accountId === accountId) {
      localGroups.value = []
      console.error('[GroupSelector] ensure error:', e)
    }
  } finally {
    if (requestId === selectorRequestId && visible.value && props.accountId === accountId) {
      loading.value = false
    }
  }
}

watch(() => props.accountId, (accountId, previousAccountId) => {
  if (accountId === previousAccountId) return
  selectorRequestId++
  dmRequestId++
  loading.value = false
  localGroups.value = []
  dmUsers.value = []
  dmLoaded.value = false
  syncStatus.value = 'idle'
  if (visible.value) loadAccountGroups(selectorRequestId, accountId || '')
})

const filtered = computed(() => {
  // 群组 tab = 普通群 + 超级群, 明确排除频道(channel)
  let list = localGroups.value.filter(g => g.type === 'group' || g.type === 'supergroup')
  if (!search.value) return list
  const s = search.value.toLowerCase()
  return list.filter(g =>
    g.title.toLowerCase().includes(s) ||
    (g.username && g.username.toLowerCase().includes(s))
  )
})
const visibleGroupCount = ref(200)
const displayGroups = computed(() => filtered.value.slice(0, visibleGroupCount.value))
watch(search, () => { visibleGroupCount.value = 200 })

// ===== 私聊 tab: contacts 表缓存(同步群组时已写入), 按 tg_user_id 去重, 伪装 Group 结构供父组件取 tg_chat_id =====
const dmUsers = ref<Group[]>([])
const dmLoaded = ref(false)

async function loadDmUsers() {
  if (dmLoaded.value || !props.accountId) return
  const accountId = props.accountId
  const requestId = ++dmRequestId
  try {
    const raw: any = await groupApi.listMembersGlobal(accountId)
    if (requestId !== dmRequestId || !visible.value || props.accountId !== accountId) return
    const arr: any[] = Array.isArray(raw) ? raw : (raw?.list || raw?.data || [])
    dmUsers.value = uniqueByNormalizedId(arr, (m: any) => m.tg_user_id).map((m: any) => ({
      group_id: 'dm_' + m.tg_user_id,
      title: [m.first_name, m.last_name].filter(Boolean).join(' ') || m.username || String(m.tg_user_id),
      tg_chat_id: String(m.tg_user_id),
      username: m.username || '',
      member_count: 0,
      is_joined: true,
      type: 'dm',
      access_hash: m.access_hash || 0,
    })) as unknown as Group[]
    dmLoaded.value = true
  } catch (e) {
    if (requestId === dmRequestId && visible.value && props.accountId === accountId) {
      console.error('[GroupSelector] loadDmUsers:', e)
      dmUsers.value = []
      // Keep the request retryable when the user revisits the DM tab. A failed
      // fetch is not a successfully loaded empty contact list.
      dmLoaded.value = false
    }
  }
}

const dmFiltered = computed(() => {
  if (!search.value) return dmUsers.value
  const s = search.value.toLowerCase()
  return dmUsers.value.filter(u => u.title.toLowerCase().includes(s) || (u.username && u.username.toLowerCase().includes(s)))
})

const allDmSelected = computed(() =>
  dmFiltered.value.length > 0 && dmFiltered.value.every(user => selectedGroupIds.value.has(user.group_id))
)
const dmVisibleCount = ref(200)
const dmDisplay = computed(() => dmFiltered.value.slice(0, dmVisibleCount.value))
watch(search, () => { dmVisibleCount.value = 200 })

watch(activeTab, t => { if (t === 'dm') loadDmUsers() })

const selectedGroupIds = computed(() => new Set(selected.value.map(item => item.group_id)))

const allSelected = computed(() =>
  filtered.value.length > 0 && filtered.value.every(group => selectedGroupIds.value.has(group.group_id))
)


function selectAll() {
  selected.value = toggleSelectionByKey(selected.value, filtered.value, item => item.group_id)
}

function selectAllDm() {
  selected.value = toggleSelectionByKey(selected.value, dmFiltered.value, item => item.group_id)
}

function isSelected(g: Group) {
  return selectedGroupIds.value.has(g.group_id)
}

function toggle(g: Group) {
  if (props.mode === 'single') {
    selected.value = [g]
  } else {
    if (isSelected(g)) {
      selected.value = selected.value.filter(s => s.group_id !== g.group_id)
    } else {
      selected.value.push(g)
    }
  }
}

function confirm() {
  emit('select', [...selected.value])
  visible.value = false
}

async function onRefresh() {
  if (loading.value) return
  const accountId = props.accountId || ''
  const requestId = ++selectorRequestId
  const isCurrent = () =>
    requestId === selectorRequestId && visible.value && props.accountId === accountId

  // 如果其他组件正在同步同一账号，只等待并读取该账号结果；等待期间切换/关闭即放弃 UI 写入。
  const syncToken = accountId ? syncLock.acquire(accountId) : null
  if (accountId && syncToken === null) {
    loading.value = true
    syncStatus.value = 'syncing'
    try {
      const syncFinished = await waitForGlobalSync(accountId, isCurrent)
      if (!isCurrent()) return
      if (!syncFinished || syncLock.isSyncing(accountId)) {
        syncStatus.value = 'timeout'
        return
      }
      try {
        const raw: any = await groupApi.list(accountId)
        if (!isCurrent()) return
        const list = raw.list || raw.data || (Array.isArray(raw) ? raw : [])
        localGroups.value = list
        groupStore.setGroups(accountId, list)
        emit('refresh')
      } catch (e) {
        if (isCurrent()) console.error('[GroupSelector] list after global sync:', e)
      }
    } finally {
      if (isCurrent()) loading.value = false
    }
    return
  }

  if (!accountId) {
    if (isCurrent()) emit('refresh')
    return
  }

  loading.value = true
  syncStatus.value = 'syncing'
  try {
    await groupApi.sync(accountId)
    if (!isCurrent()) return
    // POST /groups/sync now returns only after the Telegram sync is complete.
    // An empty result is valid; never use list.length as a completion signal.
    const raw: any = await groupApi.list(accountId)
    if (!isCurrent()) return
    const list = raw.list || raw.data || (Array.isArray(raw) ? raw : [])
    localGroups.value = list
    groupStore.setGroups(accountId, list)
    syncStatus.value = 'done'
    markSyncing(accountId)
    emit('refresh')
    await nextTick()
  } catch (e: unknown) {
    if (isCurrent()) {
      console.error('[GroupSelector] sync failed:', e)
      syncStatus.value = 'error'
    }
  } finally {
    if (isCurrent()) loading.value = false
    // 这次调用持有全局锁；即使 UI 已切换，也必须释放原账号的锁。
    if (syncToken !== null) syncLock.release(accountId, syncToken)
  }
}

watch(() => props.modelValue, (v) => {
  selectorRequestId++
  dmRequestId++
  if (v) {
    selected.value = []
    search.value = ''
    activeTab.value = 'group'
    syncStatus.value = 'idle'
    loadAccountGroups(selectorRequestId, props.accountId || '')
  } else {
    loading.value = false
  }
})

// Wait for another component to finish syncing, then return
async function waitForGlobalSync(accountId: string, shouldContinue: () => boolean = () => true): Promise<boolean> {
  const checkInterval = 500
  const maxWait = 120000
  const start = Date.now()
  while (Date.now() - start < maxWait) {
    if (!shouldContinue()) return false
    if (!syncLock.isSyncing(accountId)) {
      return true
    }
    await new Promise(r => setTimeout(r, checkInterval))
  }
  return !syncLock.isSyncing(accountId)
}
</script>
