<template>
  <div class="min-w-0 w-full p-3 sm:p-5 md:p-6 max-w-3xl mx-auto">
    <div class="flex items-center justify-between gap-3 mb-4">
      <div class="min-w-0">
        <h1 class="text-title font-extrabold tracking-tight text-text-primary truncate">进群邀请</h1>
        <!-- R109: 移除"当前账号"标签(执行账号在创建弹窗里下拉选择) -->
        <p class="text-sm text-text-muted mt-1">执行账号在创建任务时选择</p>
      </div>
      <t-button @click="showCreate = true" class="ui-t-button ui-t-button--primary flex items-center gap-2 shrink-0"> 新建加群任务
      </t-button>
    </div>

    <!-- 实时进度面板: 创建后在此显示, 无需跳转 -->
    <LiveTaskFeed v-if="currentAccount" :account-id="currentAccount.account_id" task-type="join_groups" class="mb-4" />

    <!-- 本账号的加群任务列表 -->
    <div v-if="tasksLoading" class="glass rounded-[var(--t-r-md)] p-4 space-y-3 mb-4">
      <div v-for="i in 3" :key="i" class="space-y-2.5">
        <div class="flex items-center justify-between">
          <Skeleton height="1.25rem" width="40%" rounded="rounded-md" />
          <Skeleton height="0.875rem" width="25%" rounded="rounded-md" />
        </div>
        <Skeleton height="0.5rem" rounded="rounded-full" />
      </div>
    </div>
    <template v-else>
      <EmptyState v-if="tasksError" icon="Users" :title="tasksError" description="点击重试">
        <t-button @click="fetchTasks" class="ui-t-button ui-t-button--secondary ui-t-button--sm mt-3">重新加载</t-button>
      </EmptyState>
      <EmptyState v-else-if="!tasks.length" icon="Users" title="暂无加群任务" description="点击右上角「新建加群任务」批量加入群组" />
      <div v-else class="space-y-3 mb-4">
        <pull-down-refresh v-if="tasks.length" v-model="pullRefreshing" @refresh="onPullRefresh">
        <div v-for="(t, ti) in tasks" :key="t.task_id" :class="['glass rounded-[var(--t-r-md)] p-4 card-lift card-enter hover:border-[color:var(--t-brand-dim)] mb-3', t.status === 'running' ? 'card-running' : '']" :style="{ '--i': ti }">
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 min-w-0">
              <span class="px-2 py-0.5 rounded text-xs shrink-0 inline-flex items-center gap-1.5" :class="statusClass(t.status)">
                <span v-if="t.status === 'running'" class="w-1.5 h-1.5 rounded-full bg-current dot-pulse" />
                {{ statusLabel(t.status) }}
              </span>
              <span class="text-xs text-text-muted truncate">账号 {{ accountStore.accounts.find((a: any) => a.account_id === t.account_id)?.remark || accountStore.accounts.find((a: any) => a.account_id === t.account_id)?.phone || t.account_id?.slice(0, 8) }} · 目标 {{ (t.target_ids || []).length || progOf(t).total || '-' }} 个群</span>
            </div>
            <span class="text-xs text-text-muted shrink-0">{{ formatTime(t.created_at) }}</span>
          </div>
          <div class="mt-3">
            <div class="flex items-center justify-between text-xs mb-1.5">
              <span class="text-text-secondary">{{ progOf(t).done }} / {{ progOf(t).total }}</span>
              <span class="text-[var(--t-brand)] font-medium">{{ percentOf(t) }}%</span>
            </div>
            <div class="h-2 rounded-full bg-bg-hover overflow-hidden">
              <div class="h-full rounded-full transition-all duration-500 bg-[var(--t-brand)]" :style="{ width: percentOf(t) + '%' }" />
            </div>
          </div>
          <div class="flex items-center justify-end gap-2 mt-3">
            <t-button v-if="['failed','cancelled','completed'].includes(t.status) && progOf(t).failed > 0" @click.stop="retryFailed(t.task_id)" :disabled="taskActionsPending.has(t.task_id)" class="ui-t-button ui-t-button--sm ui-t-button--secondary">重试失败 ({{ progOf(t).failed }})</t-button>
            <t-button v-if="['running','queued','pending'].includes(t.status)" @click.stop="stopTask(t.task_id)" :disabled="taskActionsPending.has(t.task_id)" class="ui-t-button ui-t-button--sm ui-t-button--danger">停止</t-button>
            <t-button @click.stop="$router.push(`/workspace/${route.params.accountId}/tasks/${t.task_id}`)" class="ui-t-button ui-t-button--sm ui-t-button--secondary">查看详情</t-button>
          </div>
        </div>
        </pull-down-refresh>
      </div>
    </template>

    <!-- 新建模态框 -->
    <Modal v-model="showCreate" title="新建加群任务">
    <div class="space-y-4">
      <!-- R108: 执行账号选择(普通号模式) —— 不再依赖顶栏切换器 -->
      <div v-if="joinMode === 'invite'">
        <label class="block text-sm font-medium mb-2">执行账号</label>
        <select v-model="phoneAccId" class="ui-input !text-sm" aria-label="执行账号">
          <option v-for="a in onlineAccounts" :key="a.account_id" :value="a.account_id">{{ a.first_name || a.phone }}</option>
        </select>
      </div>
      <div v-else class="space-y-2">
        <div class="flex items-center justify-between"><label class="block text-sm font-medium">执行账号（可多选）</label><span class="text-xs text-text-muted">已选 {{ joinAccounts.length }} 个在线账号</span></div>
        <div class="max-h-36 overflow-y-auto rounded-lg border border-border p-2 space-y-1">
          <label v-for="a in onlineAccounts" :key="a.account_id" class="flex items-center gap-2 p-2 rounded hover:bg-bg-hover cursor-pointer">
            <input type="checkbox" :value="a.account_id" v-model="joinAccounts" class="accent-[var(--t-brand)]" />
            <span class="text-sm">{{ a.remark || a.first_name || a.phone }}</span><span class="text-xs text-text-muted ml-auto">{{ a.phone }}</span>
          </label>
          <p v-if="!onlineAccounts.length" class="text-xs text-text-muted text-center py-3">没有在线账号，请先启动账号。</p>
        </div>
        <div v-if="joinAccounts.length > 1" class="space-y-1">
          <label class="block text-xs text-text-secondary">多账号分配方式</label>
          <select v-model="joinDistribution" class="ui-input !min-h-[36px] !px-3 !py-1.5 !text-sm" aria-label="多账号分配方式"><option value="all">每个账号都执行全部链接</option><option value="round_robin">均分链接（每个群只分配给一个账号）</option></select>
        </div>
      </div>
      <!-- R127 令牌化: seg 容器深浅主题自适应(原写死深底在浅色下违和) -->
      <div class="flex rounded-[var(--t-r-lg)] p-1 gap-1 mb-4" style="background: color-mix(in srgb, var(--t-text-1) 8%, transparent);">
        <button type="button" class=" ui-segmented__item flex-1"
          :style="joinMode === 'link'
            ? 'background: var(--t-surface-2); color: var(--t-text-1); box-shadow: 0 2px 8px rgba(0,0,0,0.2);'
            : 'color: var(--t-text-3);'"
          @click="joinMode = 'link'">链接加群</button>
        <button type="button" class=" ui-segmented__item flex-1"
          :style="joinMode === 'invite'
            ? 'background: var(--t-surface-2); color: var(--t-text-1); box-shadow: 0 2px 8px rgba(0,0,0,0.2);'
            : 'color: var(--t-text-3);'"
          @click="joinMode = 'invite'">邀请成员入群</button>
      </div>

      <!-- 邀请成员模式: 目标群(我的群) + 来源群(有采集成员的群) -->
      <div v-if="joinMode === 'invite'" class="space-y-3">
        <div>
          <label class="block text-xs font-semibold mb-1">我的群(邀请进这个群)</label>
          <select v-model="inviteTargetId" class="ui-input !text-sm">
            <option value="">选择目标群</option>
            <option v-for="g in joinedGroups" :key="g.group_id" :value="g.group_id">{{ g.title || g.username || ('群' + g.tg_chat_id) }}</option>
          </select>
        </div>
        <div>
          <label class="block text-xs font-semibold mb-1">来源群(邀请这个群的成员)</label>
          <select v-model="inviteSourceId" class="ui-input !text-sm">
            <option value="">选择来源群</option>
            <option v-for="g in inviteGroups" :key="g.group_id" :value="g.group_id">{{ g.title || g.username || ('群' + g.tg_chat_id) }} ({{ g.member_count || 0 }}人)</option>
          </select>
        </div>
        <p class="text-xs text-text-muted">需要先在「成员采集」里采集来源群成员；邀请间隔 1.5~3.5 秒, 被限流会自动暂停该账号。</p>
      </div>
      <!-- 协议号协同模式 -->
      <!-- R111: 协议协同死分支已删(view 恒 'phone')。群链接输入两模式本共用 -->
      <div>
        <label class="block text-sm font-medium mb-2">群链接 / 邀请链接</label>
        <textarea v-model="linksInput" rows="6" placeholder="每行一个链接，如:&#10;https://t.me/groupname&#10;https://t.me/+AbCdEfGhIjK"
                  class="ui-input !min-h-[144px] resize-none font-mono text-sm" aria-label="群链接或邀请链接" />
        <div class="flex items-center justify-between mt-2">
          <label class="px-3 py-1.5 rounded-lg border border-border text-xs cursor-pointer hover:border-[color:var(--t-brand)] transition-colors">
            导入文件<input type="file" accept=".txt,.csv,text/plain,text/csv" class="hidden" :disabled="linkFileReading || submitting" @change="onImportFile">
          </label>
          <span v-if="linkFileReading" class="text-xs text-text-muted" role="status">正在读取文件…</span>
          <span class="text-xs text-text-muted">已识别 {{ validLinks.length }} 个有效链接</span>
          <!-- R128 无效链接可展开查看: 原只报数量, 用户不知哪行错了 -->
          <button type="button" v-if="invalidLinks.length > 0"
                  class="text-xs text-[var(--t-err)] underline decoration-dotted underline-offset-2"
                  @click="showInvalidLinks = !showInvalidLinks">{{ invalidLinks.length }} 个格式无效{{ showInvalidLinks ? '▲' : '▼' }}</button>
        </div>
        <div v-if="showInvalidLinks && invalidLinks.length" class="mt-2 rounded-lg p-2.5 space-y-1 max-h-28 overflow-y-auto"
             style="background: color-mix(in srgb, var(--t-err) 8%, transparent); border: 1px solid color-mix(in srgb, var(--t-err) 25%, transparent);">
          <div v-for="(bad, i) in invalidLinks" :key="i" class="text-[11px] font-mono truncate" style="color: var(--t-err);">{{ bad }}</div>
        </div>
        <!-- 加群间隔(租户自定义): 最小/最大分钟, 每次随机取区间内 -->
        <div class="grid grid-cols-2 gap-2">
          <label class="block text-xs text-text-secondary">最小间隔(分钟)
            <input v-model.number="joinIntervalMin" type="number" min="0" step="0.5" placeholder="默认2"
              class="ui-input mt-0.5 !min-h-[36px] !px-2.5 !py-1.5 !text-sm" />
          </label>
          <label class="block text-xs text-text-secondary">最大间隔(分钟)
            <input v-model.number="joinIntervalMax" type="number" min="0" step="0.5" placeholder="默认3"
              class="ui-input mt-0.5 !min-h-[36px] !px-2.5 !py-1.5 !text-sm" />
          </label>
        </div>
      </div>

    </div>
    <template #footer>
      <div class="flex items-center justify-end gap-3">
        <t-button type="button" class="ui-t-button ui-t-button--secondary" @click="showCreate = false">取消</t-button>
        <t-button type="button" class="ui-t-button ui-t-button--primary flex items-center gap-2" @click="submit" :disabled="validLinks.length === 0 || submitting">
          <Loader2 v-if="submitting" class="w-4 h-4 animate-spin" />
          {{ submitting ? '创建中...' : `创建加群任务 (${validLinks.length})` }}
        </t-button>
      </div>
    </template>
    </Modal>
  </div>

</template>

<script setup lang="ts">
import { parseTargets } from '@/utils/importTargets'
import { findInvalidJoinLinks, normalizeJoinLinks } from '@/utils/joinLinks'
import { toast } from '@/utils/toast'
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAccountStore } from '@/stores/account'
import { useGroupStore } from '@/stores/group'
import type { Group } from '@/types'
// R111 已删 ProtocolPicker/campaignApi/CAMPAIGN_STATUS 导入(死代码)
import { useAuthStore } from '@/stores/auth'
import { taskApi } from '@/api/task'
import { Loader2, Plus, Users } from 'lucide-vue-next'
import LiveTaskFeed from '@/components/LiveTaskFeed.vue'
import Modal from '@/components/ui/Modal.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import { useTaskEvents } from '@/composables/useTaskEvents'
import { validateJoinLinkFile } from '@/utils/workspaceGuards'

const route = useRoute()
const router = useRouter()
const accountStore = useAccountStore()
const groupStore = useGroupStore()
const authStore = useAuthStore()
const currentAccount = computed(() => accountStore.currentAccount)
const accName = (a: any) => a?.first_name || a?.remark || a?.phone || (a?.account_id || '').slice(0, 8)

// ===== 模态框化: 本页任务列表 + 新建弹窗 =====
const showCreate = ref(false)
// R108: 手机号模式的执行账号(默认当前账号, 可下拉改选, 不必切换顶栏)
const phoneAccId = ref('')
const joinAccounts = ref<string[]>([])
const joinDistribution = ref<'all' | 'round_robin'>('round_robin')
watch(currentAccount, (acc: any) => {
  if (acc?.account_id && !phoneAccId.value) phoneAccId.value = acc.account_id
  if (acc?.account_id && joinAccounts.value.length === 0) joinAccounts.value = [acc.account_id]
}, { immediate: true })
const onlineAccounts = computed(() => (accountStore.accounts || []).filter((a: any) => a.status === 'online'))
const tasks = ref<any[]>([])
const tasksLoading = ref(true)
const tasksError = ref('')
const taskActionsPending = ref(new Set<string>())
const linkFileReading = ref(false)
let linkFileReader: FileReader | null = null
let linkFileReadRequestId = 0

let taskListRequestId = 0
let joinPageActive = true
let taskListController: AbortController | null = null

async function fetchTasks() {
  const accountId = currentAccount.value?.account_id || ''
  const requestId = ++taskListRequestId
  taskListController?.abort()
  taskListController = null
  if (!accountId) {
    tasks.value = []
    tasksError.value = ''
    tasksLoading.value = false
    return
  }
  tasksLoading.value = true
  const controller = new AbortController()
  taskListController = controller
  try {
    const res: any = await taskApi.list({ type: 'join_groups', page: 1, page_size: 50, hide_loops: 1 } as any, { signal: controller.signal })
    if (!joinPageActive || requestId !== taskListRequestId || accountId !== (currentAccount.value?.account_id || '')) return
    tasks.value = res.list || []
    tasksError.value = ''
  } catch (e: any) {
    if (controller.signal.aborted || !joinPageActive || requestId !== taskListRequestId || accountId !== (currentAccount.value?.account_id || '')) return
    tasksError.value = e?.message || '加载失败, 点击重试'
  } finally {
    if (joinPageActive && requestId === taskListRequestId) tasksLoading.value = false
    if (taskListController === controller) taskListController = null
  }
}
onUnmounted(() => {
  joinPageActive = false
  taskListRequestId++
  taskListController?.abort()
  taskListController = null
  linkFileReadRequestId++
  linkFileReader?.abort()
  linkFileReader = null
})
useAutoRefresh(fetchTasks, 15000)

// SSE 实时: 进度事件即时更新对应卡片, 状态事件刷新列表——亚秒级, 不再等轮询
useTaskEvents((evt: any) => {
  if (evt.type === 'task_status') { if (!evt.task_type || evt.task_type === 'join_groups') fetchTasks(); return }
  if (evt.progress && evt.task_id) {
    const t = tasks.value.find((x: any) => x.task_id === evt.task_id)
    if (t) t.progress = evt.progress
  }
})
// 邀请成员入群模式
const joinMode = ref<'link' | 'invite'>('link')
const inviteTargetId = ref('')
const inviteSourceId = ref('')
// 加群间隔(分钟, 0=用系统默认)
const joinIntervalMin = ref(0)
const joinIntervalMax = ref(0)
const inviteGroups = ref<Group[]>([])

// P0-32 审计修复: 原过滤条件 g.is_joined 在 TelegramGroup 类型里根本不存在——
// 过滤恒 false, 邀请目标群下拉永远空白。数据源 groupStore.ensure() 拉的就是
// 该账号已加入的群(dialogs 语义), 过滤本就冗余, 直接透传。
const joinedGroups = computed(() => inviteGroups.value)

// R108: 群列表跟随"执行账号"下拉(原绑 currentAccount, 选号与选群错位必失败)
let inviteGroupRequest = 0
async function loadInviteGroups() {
  const accountId = phoneAccId.value
  if (!accountId) {
    inviteGroups.value = []
    return
  }
  const requestId = ++inviteGroupRequest
  try {
    const groups = await groupStore.ensure(accountId)
    // 账号切换期间旧请求可能后返回；只允许当前选择账号的结果更新下拉。
    if (requestId === inviteGroupRequest && phoneAccId.value === accountId) {
      inviteGroups.value = groups
    }
  } catch {
    if (requestId === inviteGroupRequest && phoneAccId.value === accountId) {
      inviteGroups.value = []
      toast({ type: 'error', title: '群组加载失败', message: '请检查账号连接后重试' })
    }
  }
}
watch(phoneAccId, () => {
  // 使此前尚未完成的账号请求失效，避免旧账号群列表覆盖新选择。
  inviteGroupRequest++
  inviteGroups.value = []
  inviteTargetId.value = ''
  inviteSourceId.value = ''
  loadInviteGroups()
})

watch(currentAccount, () => {
  // 群组下拉以 phoneAccId（弹窗内显式选择）为唯一数据源；
  // 顶栏账号变化不应清空或重新加载用户已选的执行账号群组。
  fetchTasks()
})

const statusLabelMap: Record<string, string> = {
  pending: '等待调度', queued: '排队中', running: '运行中',
  completed: '已完成', failed: '失败', cancelled: '已取消',
}
function statusLabel(s: string) { return statusLabelMap[s] || s }
function statusClass(s: string): string {
  return ({
    pending: 'bg-[color:var(--t-warn-dim)] text-[var(--t-warn)]', queued: 'bg-[color:var(--t-warn-dim)] text-[var(--t-warn)]',
    running: 'bg-[color:var(--t-brand-dim)] text-[var(--t-brand)]', completed: 'bg-[color:var(--t-ok-dim)] text-[var(--t-ok)]',
    failed: 'bg-[color:var(--t-err-dim)] text-[var(--t-err)]', cancelled: 'bg-text-muted/10 text-text-muted',
  } as any)[s] || 'bg-bg-hover text-text-muted'
}
function progOf(t: any) {
  let p = t.progress
  if (typeof p === 'string') { try { p = JSON.parse(p) } catch { p = {} } }
  return { total: p?.total || 0, done: p?.done || 0, failed: p?.failed || 0 }
}
function percentOf(t: any) {
  const pr = progOf(t)
  return pr.total ? Math.min(100, Math.round((pr.done / pr.total) * 100)) : 0
}
function formatTime(iso: string) { return iso ? new Date(iso).toLocaleString('zh-CN') : '' }
const pullRefreshing = ref(false)
async function onPullRefresh() {
  if (pullRefreshing.value) return
  pullRefreshing.value = true
  try {
    await fetchTasks()
  } finally {
    pullRefreshing.value = false
  }
}

async function retryFailed(id: string) {
  if (taskActionsPending.value.has(id)) return
  taskActionsPending.value = new Set(taskActionsPending.value).add(id)
  try { await taskApi.retry(id, 'failed'); await fetchTasks() }
  catch (e: any) { toast({ type: 'error', title: '重试失败', message: e?.message }) }
  finally {
    const next = new Set(taskActionsPending.value)
    next.delete(id)
    taskActionsPending.value = next
  }
}
async function stopTask(id: string) {
  if (taskActionsPending.value.has(id)) return
  taskActionsPending.value = new Set(taskActionsPending.value).add(id)
  try { await taskApi.cancel(id); await fetchTasks() }
  catch (e: any) { toast({ type: 'error', title: '停止失败', message: e?.message }) }
  finally {
    const next = new Set(taskActionsPending.value)
    next.delete(id)
    taskActionsPending.value = next
  }
}

const linksInput = ref('')
// ===== 协议号协同模式 =====
// R111 已删 isProtocol/protocolIds(协议协同死代码, view 恒 'phone')
// R111 已删 protoGroups
// 当前账号的已同步TG群列表(群勾选数据源; Group类型: title/username/member_count/tg_chat_id)
// R111 已删 protoGroupList/toggleProtoGroup(协议协同死代码)
// R111 已删 campaigns/loadCampaigns(协议协同死代码)
const submitting = ref(false)

function onImportFile(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file || !joinPageActive) return
  const requestId = ++linkFileReadRequestId
  linkFileReader?.abort()
  linkFileReader = null
  linkFileReading.value = false
  const error = validateJoinLinkFile(file)
  if (error) {
    toast({ type: 'warning', title: '文件不可用', message: error })
    return
  }
  const reader = new FileReader()
  linkFileReader = reader
  linkFileReading.value = true
  reader.onload = () => {
    if (!joinPageActive || requestId !== linkFileReadRequestId || reader !== linkFileReader) return
    const content = typeof reader.result === 'string' ? reader.result : ''
    if (!content.trim()) {
      toast({ type: 'warning', title: '文件内容为空', message: '请检查文件后重新选择' })
      return
    }
    const items = parseTargets(content, { keepTmeLinks: true })
    linksInput.value = [...new Set([...linksInput.value.split('\n').map(x => x.trim()).filter(Boolean), ...items])].join('\n')
  }
  reader.onerror = () => {
    if (joinPageActive && requestId === linkFileReadRequestId) toast({ type: 'error', title: '读取文件失败', message: '无法读取所选文件，请重新选择' })
  }
  reader.onloadend = () => {
    if (requestId === linkFileReadRequestId) {
      linkFileReading.value = false
      if (linkFileReader === reader) linkFileReader = null
    }
  }
  reader.readAsText(file)
}

// 2026-09-22 宽松识别+归一化：缺协议头的 t.me/telegram.me、@开头 都接受，
// 统一归一为 https://t.me/... 再提交。此前严格 startsWith 会把 "t.me/xxx" 等常见写法
// 静默滤掉——用户明明填了链接，validLinks 却是 0，建出零目标任务（服务端日志实证）。
const validLinks = computed(() => normalizeJoinLinks(linksInput.value))
const invalidLinks = computed(() => findInvalidJoinLinks(linksInput.value))
// R128 无效链接展开查看(模板引用, 编译必需)
const showInvalidLinks = ref(false)

async function submit() {
  if (submitting.value || linkFileReading.value) return
  // R111: 协议协同死分支已删
  // ===== 邀请成员入群模式 =====
  if (joinMode.value === 'invite') {
    if (!phoneAccId.value || !inviteTargetId.value || !inviteSourceId.value) {
      toast({ type: 'warning', title: '请选择执行账号、目标群和来源群' })
      return
    }
    const accountId = phoneAccId.value
    const targetGroupId = inviteTargetId.value
    const sourceGroupId = inviteSourceId.value
    submitting.value = true
    try {
      await taskApi.create({
        type: 'invite_members',
        account_id: accountId,
        payload: {
          target_group_id: targetGroupId,
          source_group_id: sourceGroupId,
        },
      })
      toast({ type: 'success', title: '邀请入群任务已创建', message: '成员邀请已排队执行' })
      inviteTargetId.value = ''
      inviteSourceId.value = ''
      joinMode.value = 'link'
      showCreate.value = false
      fetchTasks()
    } catch (e: any) {
      toast({ type: 'error', title: '创建失败', message: e?.message || '请稍后重试' })
    } finally {
      submitting.value = false
    }
    return
  }
  if (joinAccounts.value.length === 0) {
    toast({ type: 'warning', title: '请至少选择一个执行账号' })
    return
  }
  if (validLinks.value.length === 0) {
    toast({ type: 'warning', title: '请先添加有效的群链接' })
    return
  }
  // Freeze the complete batch before the first await. Edits made while requests are
  // in flight must apply to the next batch, not produce inconsistent per-account tasks.
  const accountIds = [...new Set(joinAccounts.value)]
  const links = [...new Set(validLinks.value)]
  const distribution = joinDistribution.value
  const intervalMin = joinIntervalMin.value || undefined
  const intervalMax = joinIntervalMax.value || undefined
  const assignments = distribution === 'round_robin' && accountIds.length > 1
    ? accountIds.map((accountId, accountIndex) => ({ accountId, links: links.filter((_, linkIndex) => linkIndex % accountIds.length === accountIndex) })).filter(x => x.links.length > 0)
    : accountIds.map(accountId => ({ accountId, links }))
  submitting.value = true
  try {
    let created = 0
    const errors: string[] = []
    for (const assignment of assignments) {
      try {
        await taskApi.create({ type: 'join_groups', account_id: assignment.accountId, links: assignment.links,
          interval_min_minutes: intervalMin, interval_max_minutes: intervalMax })
        created++
      } catch (err: any) { errors.push(err?.message || '任务创建失败') }
    }
    if (created === 0) throw new Error(errors[0] || '所有账号任务创建失败')
    linksInput.value = ''
    showCreate.value = false
    toast({ type: errors.length ? 'warning' : 'success', title: '已创建 ' + created + '/' + assignments.length + ' 个加群任务', message: (distribution === 'round_robin' ? '链接已按账号均分' : '每个账号执行全部链接') + (errors.length ? '；失败：' + errors.join('；') : '') })
    fetchTasks()
  } catch (e: any) {
    toast({ type: 'error', title: '创建失败', message: e?.message || '请稍后重试' })
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  fetchTasks()
  loadInviteGroups()
  // 入口分流(2026-10-20): 侧边栏"邀请入群"项带 ?mode=invite 进入自动切模式
  const q = new URLSearchParams(window.location.search)
  if (q.get('mode') === 'invite' && authStore.tenant?.plan_features?.invite) {
    joinMode.value = 'invite'
  }
  // R118: 守卫改并集 —— 加群/邀请合并入口后, 任一权限可用即可留在页内
  // (joinMode 默认 'link'; 无加群权限但有邀请权限时自动落到 invite Tab)
  const pf = authStore.tenant?.plan_features
  if (!pf?.join_groups && !pf?.invite) {
    toast({ type: 'error', title: '当前套餐不支持进群邀请，请联系客服升级套餐' })
    router.push('/dashboard')
    return
  }
  if (!pf?.join_groups && pf?.invite && !q.get('mode')) {
    joinMode.value = 'invite'
  }
})
</script>
