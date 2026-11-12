<template>
  <main class="w-full min-w-0 max-w-4xl mx-auto space-y-3 sm:space-y-4 md:space-y-6">
    <Breadcrumb :items="[{ path: `/workspace/${route.params.accountId}/tasks`, label: '任务列表' }, { path: '', label: '任务详情' }]" />
    <!-- 页面级错误态: 加载失败可重试 -->
    <EmptyState v-if="pageError && !task" icon="ListTodo" :title="pageError" description="任务详情暂时无法显示">
      <t-button @click="fetchTask" class="ui-t-button ui-t-button--secondary ui-t-button--sm mt-3">重新加载</t-button>
    </EmptyState>
    <div v-else-if="pageLoading && !task" class="glass rounded-[var(--t-r-lg)] p-6 sm:p-12 text-center text-sm text-text-muted lift">加载中...</div>
    <div v-if="task" class="flex flex-wrap items-center gap-3">
      <button type="button" @click="$router.push(`/workspace/${route.params.accountId}/tasks`)" class="ui-icon-btn" aria-label="返回">
        <ArrowLeft class="w-5 h-5" />
      </button>
      <div>
        <h2 class="text-xl tracking-tight font-semibold text-text-primary truncate"><span class="text-[var(--t-brand)] tabular-nums mr-2">{{ task?.task_number ? '#' + String(task.task_number).padStart(6, '0') : '' }}</span>{{ taskTypeLabel(task?.type || '') }}</h2>
        <p class="text-xs text-text-secondary">创建于 {{ task ? formatDateTime(task.created_at) : '' }}</p>
      </div>
      <span
        class="ml-auto text-xs px-2 py-0.5 rounded-full border"
        :class="task ? taskStatusClass(task.status) : ''"
      >
        {{ task ? taskStatusLabel(task.status) : '' }}
      </span>
    </div>

    <!-- R106: 桌面端两栏 —— 左列进度卡(固定宽), 右列实时日志(自适应); 移动端保持单列堆叠 -->
    <div class="grid grid-cols-1 lg:grid-cols-[minmax(0,340px)_minmax(0,1fr)] gap-4 items-start">
    <div class="space-y-4 min-w-0">
    <!-- 进度 -->
    <div class="glass rounded-[var(--t-r-md)] p-4 sm:p-6 lift">
      <div class="flex items-center justify-between mb-4">
        <div class="flex items-center gap-3 sm:gap-5 min-w-0">
          <!-- R126 示例2 ring 进度环: conic-gradient 环 + 内圈百分比 -->
          <div class="relative w-[76px] h-[76px] sm:w-[92px] sm:h-[92px] rounded-full grid place-items-center shrink-0"
               :style="{ background: `conic-gradient(var(--t-brand) ${progressPercent * 3.6}deg, var(--t-surface-2) 0deg)` }">
            <div class="absolute inset-[9px] rounded-full grid place-items-center" style="background: var(--t-bg);">
              <span class="text-lg sm:text-title font-black tabular-nums">{{ progressPercent }}<span class="text-[11px] font-bold">%</span></span>
            </div>
          </div>
          <div>
            <p class="text-sm text-text-secondary max-w-[60ch]">
              <span class="tabular-nums">{{ formatNumber((task?.progress.done || 0) + (task?.progress.failed || 0)) }}</span>/<span class="tabular-nums">{{ formatNumber(task?.progress.total || 0) }}</span> 已处理
              <span class="text-[var(--t-ok)] ml-2"><span class="tabular-nums">{{ formatNumber(task?.progress.done || 0) }}</span> 成功</span>
              <span v-if="task?.progress.failed" class="text-[var(--t-err)] ml-2"><span class="tabular-nums">{{ formatNumber(task.progress.failed) }}</span> 失败</span>
            </p>
          </div>
        </div>
        <div class="flex flex-wrap gap-2">
          <t-button v-if="['running','queued','scheduled'].includes(task?.status || '')" @click="pause" :disabled="!!actionPending" :loading="actionPending === 'pause'" class="ui-t-button ui-t-button--secondary ui-t-button--sm">暂停</t-button>
          <t-button v-else-if="task?.status === 'paused'" @click="resume" :disabled="!!actionPending" :loading="actionPending === 'resume'" class="ui-t-button ui-t-button--secondary ui-t-button--sm">恢复</t-button>
          <t-button v-if="['running','queued','scheduled','paused','pending'].includes(task?.status || '')" @click="cancel" :disabled="!!actionPending" :loading="actionPending === 'cancel'" class="ui-t-button ui-t-button--danger ui-t-button--sm" aria-label="取消">取消</t-button>
          <t-button v-if="['completed','success','failed','cancelled'].includes(task?.status || '') && (task?.progress?.failed || 0) > 0" @click="retryFailed" :disabled="!!actionPending" :loading="actionPending === 'retry'" class="ui-t-button ui-t-button--primary ui-t-button--sm">重试失败目标 ({{ task?.progress?.failed }})</t-button>
        </div>
      </div>
      <div class="w-full h-3 bg-bg-card rounded-full overflow-hidden">
        <div
          class="h-full rounded-full transition-[transform,opacity,border-color,background-color] duration-300 relative"
          :class="progressColor"
          :style="{ width: `${progressPercent}%` }"
        >
          <div v-if="task?.status === 'running'" class="absolute inset-0 overflow-hidden">
            <div class="shimmer-bar" />
          </div>
        </div>
      </div>
      <!-- R118: 等待倒计时 —— task_wait 帧驱动, 等待过程每秒可见 -->
      <div v-if="waitState && task?.status === 'running'" class="mt-3 flex items-center gap-2 text-xs text-text-secondary">
        <span class="relative flex h-2 w-2 shrink-0">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-[var(--t-warn)] opacity-60" />
          <span class="relative inline-flex rounded-full h-2 w-2 bg-[var(--t-warn)]" />
        </span>
        <span class="tabular-nums text-[var(--t-warn)] font-medium">{{ waitState.remaining }}s</span>
        <span>·</span>
        <span class="truncate">账号 {{ accountMap[waitState.account_id] || '...' }}</span>
        <span>等待后进入下一目标</span>
        <span class="truncate text-text-muted">→ {{ targetDisplayName(waitState.target_id) }}<template v-if="waitState.total">（{{ waitState.next_index }}/{{ waitState.total }}）</template></span>
      </div>
    </div>

    <!-- 成员采集结果 -->
    <div v-if="task?.type === 'scrape_members'" class="glass rounded-[var(--t-r-md)] p-3.5 sm:p-5 lift">
      <div class="flex items-center justify-between gap-2 mb-3">
        <div class="flex items-center gap-2">
          <Users class="w-4 h-4 text-text-muted" />
          <h3 class="text-sm font-medium">采集到的成员</h3>
          <span class="text-xs text-text-secondary">({{ members.length }})</span>
        </div>
        <t-button @click="fetchMembers" :disabled="membersLoading" class="ui-t-button ui-t-button--secondary ui-t-button--sm disabled:opacity-40">
          {{ membersLoading ? '加载中...' : '刷新' }}
        </t-button>
      </div>
      <div v-if="membersLoading" class="py-6 text-center text-sm text-text-muted">加载中...</div>
      <div v-else-if="membersError" class="py-6 text-center text-sm text-[var(--t-err)]">{{ membersError }}</div>
      <div v-else-if="!members.length" class="py-6 text-center text-sm text-text-muted">暂无成员数据</div>
      <div v-else class="space-y-1 max-h-80 overflow-y-auto">
        <div v-for="m in members" :key="m.member_id || m.tg_user_id" class="flex items-center gap-3 px-2 py-1.5 rounded hover:bg-bg-hover/50 text-sm">
          <span class="text-[var(--t-brand)] font-mono text-xs w-32 shrink-0">{{ m.tg_user_id }}</span>
          <span class="truncate">{{ m.username || m.first_name || '未设置用户名' }}</span>
        </div>
      </div>
    </div>

    </div><!-- /左列 -->
    <div class="min-w-0">
    <!-- 实时日志 -->
    <div class="glass rounded-[var(--t-r-md)] p-5 lift">
      <div class="flex items-center justify-between mb-3">
        <div class="flex items-center gap-2">
          <Clock class="w-4 h-4 text-text-muted" />
          <h3 class="text-sm font-medium">执行日志</h3>
          <span class="text-xs text-text-secondary">({{ visibleRoundLogs.length }})</span>
        </div>
        <div class="flex items-center gap-2">
          <span v-if="showLiveConnection && connState === 'connected'" class="flex items-center gap-1 text-xs text-[var(--t-ok)]">
            <span class="w-1.5 h-1.5 rounded-full bg-[var(--t-ok)] animate-pulse" />
            实时连接中
          </span>
          <span v-else-if="showLiveConnection && connState === 'connecting'" class="flex items-center gap-1 text-xs text-text-secondary">
            <span class="w-1.5 h-1.5 rounded-full bg-[var(--t-brand)] animate-pulse" />
            连接中…
          </span>
          <span v-else-if="showLiveConnection" class="text-xs text-text-secondary">已断开,3秒后重连</span>
          <t-button
            @click="clearLogs"
            :disabled="!['completed', 'failed', 'cancelled'].includes(task?.status || '')"
            class="ui-t-button ui-t-button--secondary ui-t-button--sm"
            :title="['completed', 'failed', 'cancelled'].includes(task?.status || '') ? '永久删除该任务的全部日志' : '任务结束后才能清除日志'"
          >清除日志</t-button>
        </div>
      </div>
      <div v-if="roundOptions.length" class="mb-3 flex flex-wrap items-center gap-2">
        <label for="task-round-filter" class="text-xs text-text-secondary">执行轮次</label>
        <select id="task-round-filter" v-model="selectedRound" class="min-w-0 max-w-full rounded-lg border border-border bg-bg-card px-3 py-2 text-xs text-text-primary">
          <option value="all">全部轮次</option>
          <option v-for="round in roundOptions" :key="round.value" :value="round.value">{{ round.label }} · 成功 {{ round.success }} / 失败 {{ round.failed }}</option>
        </select>
        <span v-if="selectedRound !== 'all'" class="text-[11px] text-text-muted">仅显示该轮的原始执行记录</span>
      </div>
      <div class="ui-segmented mb-2 flex-wrap" role="group" aria-label="筛选任务日志">
        <button
          v-for="pill in logPills"
          :key="pill.value"
          type="button"
          class="ui-segmented__item"
          :aria-pressed="logFilter === pill.value"
          :class="{ 'is-active': logFilter === pill.value }"
          @click="logFilter = pill.value"
        >
          {{ pill.label }}
        </button>
      </div>
      <div v-if="hasMoreLogs" class="mb-2 flex justify-center">
        <t-button size="small" variant="text" class="ui-t-button ui-t-button--text ui-t-button--sm" :loading="loadingMoreLogs" @click="loadMoreLogs">加载更早日志</t-button>
      </div>
      <div ref="logContainer" class="max-h-[min(60dvh,420px)] lg:max-h-[420px] overflow-y-auto space-y-1.5">
        <div
          v-for="(log, i) in filteredDisplayLogs"
          :key="log.log_id || `${log.executed_at}-${i}`"
          class="flex items-start gap-2.5 py-1.5 px-2 rounded-lg text-xs"
          :class="logCardClass(log)"
        >
          <!-- 状态图标 -->
          <div class="shrink-0 mt-0.5">
            <CheckCircle2 v-if="log.status === 'success'" class="w-3.5 h-3.5 text-[var(--t-ok)]" />
            <XCircle v-else-if="log.status === 'failed'" class="w-3.5 h-3.5 text-[var(--t-err)]" />
            <AlertCircle v-else class="w-3.5 h-3.5 text-text-muted" />
          </div>
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 flex-wrap">
              <span class="text-text-muted/70 font-mono text-[10.5px] shrink-0">{{ formatTime(log.executed_at) }}</span>
              <!-- R87-M8: 操作标签弱化(去底色, 与内容拉开主次) -->
              <span class="text-[10.5px] font-medium text-text-muted/70 shrink-0 w-14">{{ getActionLabel(log.action) }}</span>
              <span class="flex items-center gap-1 text-text-primary truncate">
                <Send class="w-3 h-3 text-[var(--t-brand)] shrink-0" />
                <span class="truncate">{{ targetDisplayName(log.target_id, log.target_name) }}</span>
              </span>
            </div>
            <!-- message 渲染(2026-10-22): 成功行的"耗时 X.Xs"和限速行的"N 秒后进入下一目标"
                 写在 message 字段但此前模板没渲染, 用户看不到——统一展示 -->
            <div v-if="log.message || log._waitMsg" class="mt-1 text-text-muted text-xs inline-block">
              {{ log.message }}
              <span v-if="log._waitMsg" class="ml-1 text-text-muted/60">⏳ {{ log._waitMsg }}</span>
            </div>
            <div v-if="log.tg_message_id" class="mt-1 text-text-muted text-[11px] font-mono">
              Telegram 消息 ID：{{ log.tg_message_id }}
            </div>
            <div v-if="log.error_code" class="mt-1 text-xs rounded px-2 py-1 inline-block"
                 :class="log.error_code === 'checkpoint_persist_failed' ? 'text-[var(--t-warning)] bg-[var(--t-warning)]/10' : 'text-[var(--t-err)] bg-[var(--t-err)]/5'">
              {{ log.error_code === 'checkpoint_persist_failed' ? '发送结果待核实：请先到 Telegram 检查目标消息，禁止自动续发' : statusLabel(log.error_code) }}
            </div>
          </div>
        </div>
        <div v-if="!filteredDisplayLogs.length" class="text-text-muted text-center py-10 text-sm">
          <Clock class="w-8 h-8 mx-auto mb-2 opacity-30" />
          {{ logs.length ? '当前筛选条件下没有日志' : '暂无执行日志，任务运行后会在这里显示~' }}
        </div>
      </div>
    </div>
    </div><!-- /右列 -->
    </div><!-- /R106 两栏 -->
  </main>
</template>

<script setup lang="ts">
import { statusLabel } from '@/utils/labels'
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, CheckCircle2, XCircle, AlertCircle, Send, Clock, Users } from 'lucide-vue-next'
import { taskApi } from '@/api/task'
import { accountApi } from '@/api/account'
import { groupApi } from '@/api/group'
import { ask } from '@/utils/confirm'
const pageError = ref('')
const pageLoading = ref(true)
import { formatDateTime, formatNumber, taskStatusClass, taskStatusLabel, taskTypeLabel } from '@/utils/format'
import { toast } from '@/utils/toast'
import { useAuthStore } from '@/stores/auth'
import Breadcrumb from '@/components/ui/Breadcrumb.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import type { Task, TaskLog } from '@/types'
import { getTaskLogPageState, mergeTaskLogs, taskLogKey } from './taskLogRuntime'
import { createTaskSseRuntime } from './taskSseRuntime'

const route = useRoute()
const router = useRouter()
const taskId = ((Array.isArray(route.params.id) ? route.params.id[0] : route.params.id) as string) || ''

function isValidUUID(id: string): boolean {
  const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
  return uuidRegex.test(id)
}

const actionLabels: Record<string, string> = {
  send_message: '发送消息',
  join_group: '加入群组',
  scrape_members: '采集成员',
  scrape_links: '采集链接',
  forward_message: '转发消息',
  reply_message: '回复消息',
  broadcast: '群发执行',
  scrape: '采集执行',
  scrape_private_link: '私密链接解析',
  scrape_public_group: '公开群解析',
  monitor_start: '监听启动',
  monitor_complete: '监听完成',
  // 群发执行细分(修复: 这些码缺失导致任务详情显示英文)
  // 2026-10-21: send_message 映射表上方已有('发送消息'), 此前重复声明导致构建失败;
  // 日志"全是等待发送"的真因是 target_name 列缺失+COPY 列数不匹配(见 r76 修复), 非映射表
  send_wait: '限速等待',
  send_success: '发送成功',
  send_failed: '发送失败',
  send_skip_duplicate: '跳过重复',
  send_skip_restricted: '跳过受限',
  wait: '等待',
  send: '发送',
}

function getActionLabel(action: string): string {
  return actionLabels[action] || action
}

const task = ref<Task | null>(null)
const actionPending = ref<'pause' | 'resume' | 'cancel' | 'retry' | null>(null)
let taskFetchRequestId = 0
let pageActive = true
let membersRequestId = 0
let completionRefreshTimer: ReturnType<typeof setTimeout> | null = null
const members = ref<any[]>([])
const membersLoading = ref(false)
const membersError = ref('')

async function fetchMembers() {
  const t = task.value as any
  const groupId = t?.payload?.group_id || t?.target_ids?.[0]
  if (!pageActive) return
  if (!groupId) { membersError.value = '任务缺少群组信息'; return }
  const requestId = ++membersRequestId
  membersLoading.value = true
  membersError.value = ''
  try {
    const res: any = await groupApi.getMembers(groupId)
    if (!pageActive || requestId !== membersRequestId) return
    members.value = res?.members || res?.data?.members || res?.list || res?.items || (Array.isArray(res) ? res : [])
    const entries: [string, string][] = []
    members.value.forEach((member: any) => {
      const name = [member.display_name, member.name, [member.first_name, member.last_name].filter(Boolean).join(' '), member.username ? '@' + String(member.username).replace(/^@/, '') : '']
        .map(value => String(value || '').trim()).find(Boolean)
      if (!name) return
      for (const id of [member.tg_user_id, member.user_id, member.telegram_id, member.member_id, member.id]) {
        if (id !== undefined && id !== null && String(id).trim()) entries.push([normId(id), name])
      }
      if (member.username) entries.push([normId(member.username), name])
    })
    memberMap.value = Object.fromEntries(entries)
  } catch (e: unknown) {
    if (pageActive && requestId === membersRequestId) {
      membersError.value = e instanceof Error ? e.message : '加载成员失败'
    }
  } finally {
    if (pageActive && requestId === membersRequestId) membersLoading.value = false
  }
}

const logs = ref<TaskLog[]>([])
type RoundLogOption = { value: string; label: string; from: string; to?: string; success: number; failed: number }
const selectedRound = ref('all')
const roundOptions = computed<RoundLogOption[]>(() => {
  let payload: any = task.value?.payload || {}
  if (typeof payload === 'string') {
    try { payload = JSON.parse(payload) } catch { payload = {} }
  }
  const history = Array.isArray(payload?.round_history) ? payload.round_history : []
  const options: RoundLogOption[] = history
    .filter((item: any) => item?.round && item?.started_at)
    .map((item: any) => ({
      value: String(item.round),
      label: `第 ${item.round} 轮`,
      from: String(item.started_at),
      to: item.completed_at ? String(item.completed_at) : undefined,
      success: Number(item.success || 0),
      failed: Number(item.failed || 0),
    }))
  const activeFrom = payload?.round_started_at
  const terminal = ['completed', 'failed', 'cancelled'].includes(task.value?.status || '')
  if (activeFrom && !terminal) {
    const activeRound = Number(payload?.round_count || 0) + 1
    options.push({
      value: String(activeRound),
      label: `第 ${activeRound} 轮（当前）`,
      from: String(activeFrom),
      success: Number(task.value?.progress?.done || 0),
      failed: Number(task.value?.progress?.failed || 0),
    })
  }
  return options.sort((a, b) => Number(b.value) - Number(a.value))
})
const selectedRoundWindow = computed(() => {
  if (selectedRound.value === 'all') return undefined
  const option = roundOptions.value.find(item => item.value === selectedRound.value)
  return option ? { from: option.from, to: option.to } : undefined
})
const showLiveConnection = computed(() => {
  if (selectedRound.value === 'all') return true
  return roundOptions.value.find(item => item.value === selectedRound.value)?.label.includes('当前') || false
})
const visibleRoundLogs = computed(() => {
  const range = selectedRoundWindow.value
  if (!range) return logs.value
  const from = Date.parse(range.from)
  const to = range.to ? Date.parse(range.to) : Number.POSITIVE_INFINITY
  return logs.value.filter(log => {
    const at = Date.parse(String(log.executed_at || ''))
    return Number.isFinite(at) && at >= from && at < to
  })
})
const logKeys = new Set<string>()
const logsPage = ref(1)
const logPageCursors = new Map<number, string>()
const hasMoreLogs = ref(false)
const loadingMoreLogs = ref(false)
const LOG_PAGE_SIZE = 50


// R87-M10: 100% 后状态收敛兜底 —— SSE 事件丢失时, 进度满后 3s 自动补拉任务状态,
// 避免"发完了还显示执行中"必须手动刷新
watch(() => [(task.value as any)?.progress?.done, (task.value as any)?.progress?.failed, (task.value as any)?.status], () => {
  if (completionRefreshTimer) {
    clearTimeout(completionRefreshTimer)
    completionRefreshTimer = null
  }
  const t = task.value as any
  if (!pageActive || !t || t.status !== 'running') return
  const processed = (t.progress?.done || 0) + (t.progress?.failed || 0)
  if (t.progress?.total && processed >= t.progress.total) {
    completionRefreshTimer = setTimeout(() => {
      completionRefreshTimer = null
      const cur = task.value as any
      if (pageActive && cur && cur.status === 'running') void fetchTask(true)
    }, 3000)
  }
})

// R87-M8: 日志合并展示 —— "限速等待"行不再独立占行,
// 其"等待N秒后进入下一目标"信息并入上一条日志显示(时间-消息-群组-结果 的干净时间线)
const displayLogs = computed(() => {
  const out: (TaskLog & { _waitMsg?: string })[] = []
  for (const log of visibleRoundLogs.value) {
    const isWait = log.action === '限速等待' || (log.message || '').includes('秒后进入下一目标')
    if (isWait) {
      if (out.length && log.message) out[out.length - 1] = { ...out[out.length - 1], _waitMsg: log.message }
      continue
    }
    out.push({ ...log })
  }
  return out
})

// 日志级别筛选: 全部/成功/失败/其他
const logFilter = ref<'all' | 'success' | 'failed' | 'other'>('all')
const logPills = [
  { value: 'all', label: '全部' },
  { value: 'success', label: '成功' },
  { value: 'failed', label: '失败' },
  { value: 'other', label: '其他' },
] as const
const filteredLogs = computed(() => {
  if (logFilter.value === 'all') return visibleRoundLogs.value
  return visibleRoundLogs.value.filter((log: any) => {
    // 与 logCardClass 保持同一套状态归一化(任务完成时的 wait 视为成功)
    const s = (task.value?.status === 'completed' && (log.status === 'wait' || log.status === 'pending')) ? 'success' : log.status
    if (logFilter.value === 'success') return s === 'success'
    if (logFilter.value === 'failed') return s === 'failed'
    return s !== 'success' && s !== 'failed'
  })
})

// R118: 等待倒计时态 —— task_wait 帧(每秒)更新, 不刷日志列表
// {account_id, target_id, remaining, next_index, total}
const waitState = ref<{ account_id: string; target_id: string; remaining: number; next_index: number; total: number } | null>(null)
watch(() => task.value?.status, status => {
  if (status !== 'running') waitState.value = null
})

const filteredDisplayLogs = computed(() => {
  const visibleKeys = new Set(filteredLogs.value.map(taskLogKey))
  return displayLogs.value.filter(log => visibleKeys.has(taskLogKey(log)))
})

async function loadMoreLogs() {
  if (loadingMoreLogs.value || !hasMoreLogs.value || !pageActive) return
  const cursor = logPageCursors.get(logsPage.value)
  if (!cursor) { hasMoreLogs.value = false; return }
  loadingMoreLogs.value = true
  try {
    await fetchLogs(logsPage.value + 1, cursor)
  } finally {
    if (pageActive) loadingMoreLogs.value = false
  }
}

async function clearLogs() {
  if (!pageActive || !task.value || !['completed', 'failed', 'cancelled'].includes(task.value.status)) return
  const ok = await ask('确定永久删除这条任务的全部日志？任务本身和已导入的群组不会删除，日志删除后无法恢复。', {
    title: '清除任务日志',
    danger: true,
    confirmText: '永久清除',
  })
  if (!ok || !pageActive) return
  try {
    await taskApi.deleteTaskLogs(taskId)
    // 使正在进行的分页请求失效，防止旧响应在清空后重新填入日志。
    for (const page of logPageRequestIds.keys()) logPageRequestIds.set(page, ++nextLogPageRequestId)
    logs.value = []
    logKeys.clear()
    logsPage.value = 1
    logPageCursors.clear()
    hasMoreLogs.value = false
    toast({ type: 'success', title: '任务日志已永久清除' })
  } catch (e: any) {
    toast({ type: 'error', title: '清除日志失败', message: e?.message || '请稍后重试' })
  }
}
// 2026-09-23 修复: 初始态是"连接中"而非"已断开"——换ticket+建连要1-2秒,
// 原实现初始 false 直接显示"已断开", 用户每次打开详情都看到一次假断线
const connState = ref<'connecting' | 'connected' | 'disconnected'>('connecting')
const isConnected = computed(() => connState.value === 'connected')
const logContainer = ref<HTMLDivElement>()
const accountMap = ref<Record<string, string>>({})
const groupMap = ref<Record<string, string>>({})
const memberMap = ref<Record<string, string>>({})

// 规范化群ID: 去掉 -100 前缀, 兼容 groups.tg_chat_id 与日志 target_id 的两种存法
const normId = (id: any): string => {
  const raw = String(id ?? '').trim()
  const linkUser = raw.match(/(?:t\.me\/|@)([A-Za-z0-9_]+)/i)?.[1]
  return (linkUser || raw.replace(/^-100/, '')).toLowerCase()
}

// 优先显示日志快照名称，其次匹配本租户群组库；无法匹配时不把长数字 ID/邀请链接直接暴露在日志行。
function targetDisplayName(targetId: any, explicitName?: any): string {
  const explicit = String(explicitName ?? '').trim()
  if (explicit && !/^[-+]?\d{5,}$/.test(explicit) && !/^https?:\/\//i.test(explicit)) return explicit
  const raw = String(targetId ?? '').trim()
  const member = memberMap.value[normId(raw)]
  if (member) return member
  const mapped = groupMap.value[normId(raw)]
  if (mapped) return mapped
  const username = raw.match(/(?:t\.me\/|@)([A-Za-z0-9_]+)/i)?.[1]
  if (username) return '@' + username
  return raw ? '未匹配名称' : '未知目标'
}
let eventSource: EventSource | null = null
let refreshTimer: ReturnType<typeof setInterval>
let sseConnecting = false
let onVisCleanup: (() => void) | null = null

const sseRuntime = createTaskSseRuntime({
  isActive: () => pageActive,
  getSource: () => eventSource,
  setSource: source => { eventSource = source },
  setConnectionState: state => { connState.value = state },
  connect: () => { void connectSSE() },
  onWatchdogTimeout: () => console.warn('[task-detail] SSE 45s 无消息, 判定假活, 强制重连'),
})

const // 2026-10-21 修正: 进度按"已处理"(成功+失败)计算——失败也是处理结果, 不应让进度条永远卡住
progressPercent = computed(() => {
  if (!task.value?.progress.total) return 0
  const percent = ((task.value.progress.done + (task.value.progress.failed || 0)) / task.value.progress.total) * 100
  return Math.max(0, Math.min(100, Math.round(percent)))
})

const progressColor = computed(() => {
  if (!task.value) return 'bg-[var(--t-brand)]'
  if (task.value.status === 'failed') return 'bg-[var(--t-err)]'
  if (task.value.status === 'completed') return 'bg-[var(--t-ok)]'
  return 'bg-[var(--t-brand)]'
})

function logCardClass(log: TaskLog) {
  // 如果任务已完成，wait/pending 的日志视为成功
  const effectiveStatus = (task.value?.status === 'completed' && (log.status === 'wait' || log.status === 'pending'))
    ? 'success' : log.status
  if (effectiveStatus === 'success') return 'bg-[var(--t-ok)]/5 hover:bg-[color:var(--t-ok-dim)] border border-[color:var(--t-ok)]/10'
  if (effectiveStatus === 'failed') return 'bg-[var(--t-err)]/5 hover:bg-[color:var(--t-err-dim)] border border-[color:var(--t-err)]/10'
  return 'bg-ink-01 hover:bg-bg-card border border-transparent'
}

function formatTime(date: string | undefined | null) {
  if (!date) return '--:--:--'
  const d = new Date(date)
  if (isNaN(d.getTime())) return '--:--:--'
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

async function fetchTask(silent = false) {
  const requestId = ++taskFetchRequestId
  pageLoading.value = true
  pageError.value = ''
  try {
    const res = await taskApi.getById(taskId)
    if (!pageActive || requestId !== taskFetchRequestId) return
    // 兼容: 直接对象 / {data: {...}}
    if (res && typeof res === 'object') {
      task.value = (res as any).data || res
    } else {
      task.value = res as Task
    }
  } catch (e: unknown) {
    if (!pageActive || requestId !== taskFetchRequestId) return
    pageError.value = e instanceof Error ? e.message : '请稍后重试'
    if (!silent) toast({ type: 'error', title: '获取任务失败', message: pageError.value })
  } finally {
    if (pageActive && requestId === taskFetchRequestId) pageLoading.value = false
  }
}

const logPageRequestIds = new Map<number, number>()
let nextLogPageRequestId = 0

watch(selectedRound, () => {
  if (!pageActive) return
  for (const page of logPageRequestIds.keys()) logPageRequestIds.set(page, ++nextLogPageRequestId)
  logs.value = []
  logKeys.clear()
  logsPage.value = 1
  logPageCursors.clear()
  hasMoreLogs.value = false
  void fetchLogs()
})

async function fetchLogs(page = 1, beforeLogId?: string) {
  const requestId = ++nextLogPageRequestId
  logPageRequestIds.set(page, requestId)
  const container = logContainer.value
  const previousHeight = container?.scrollHeight || 0
  const previousTop = container?.scrollTop || 0
  const wasAtBottom = !container || container.scrollHeight - container.scrollTop - container.clientHeight < 48
  try {
    const res = await taskApi.getLogs(taskId, page, LOG_PAGE_SIZE, beforeLogId, selectedRoundWindow.value)
    if (!pageActive || logPageRequestIds.get(page) !== requestId) return
    let data: TaskLog[] = []
    if (Array.isArray(res)) {
      data = res
    } else if (res && typeof res === 'object') {
      const payload = res as any
      data = Array.isArray(payload.list) ? payload.list
        : Array.isArray(payload.data) ? payload.data
        : Array.isArray(payload.data?.list) ? payload.data.list : []
    }

    // Preserve live SSE rows received while this HTTP page was in flight.
    logs.value = mergeTaskLogs(logs.value, data)
    logKeys.clear()
    for (const log of logs.value) logKeys.add(taskLogKey(log))

    const pageState = getTaskLogPageState(data, LOG_PAGE_SIZE)
    if (pageState.cursor) logPageCursors.set(page, pageState.cursor)
    else logPageCursors.delete(page)
    if (page > logsPage.value) logsPage.value = page
    if (page === logsPage.value) hasMoreLogs.value = pageState.hasMore

    await nextTick()
    if (pageActive && logContainer.value) {
      if (page > 1) {
        // Older rows are prepended; compensate for their height so the current
        // reading position stays anchored to the same visible log.
        logContainer.value.scrollTop = previousTop + (logContainer.value.scrollHeight - previousHeight)
      } else if (wasAtBottom) {
        logContainer.value.scrollTop = logContainer.value.scrollHeight
      }
    }
  } catch (e: unknown) {
    if (pageActive && logPageRequestIds.get(page) === requestId) {
      toast({ type: 'error', title: '获取日志失败', message: e instanceof Error ? e.message : '请稍后重试' })
    }
  }
}

async function connectSSE() {
  if (sseConnecting || (eventSource && eventSource.readyState !== EventSource.CLOSED)) return
  sseConnecting = true
  try {
    const authStore = useAuthStore()
    const token = authStore.token
    if (!token) {
      connState.value = 'disconnected'
      sseRuntime.scheduleReconnect()
      return
    }

    // Short-lived ticket replaces the JWT in the EventSource URL.
    const apiBase = import.meta.env.VITE_API_BASE_URL || '/api/v1'
    const ticketController = new AbortController()
    const ticketTimeout = setTimeout(() => ticketController.abort(), 15000)
    let ticketResponse: Response
    let ticket = ''
    try {
      ticketResponse = await fetch(`${apiBase}/events/ticket`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
        signal: ticketController.signal,
      })
      if (!pageActive) return
      if (!ticketResponse.ok) {
        connState.value = 'disconnected'
        sseRuntime.scheduleReconnect()
        return
      }
      ticket = (await ticketResponse.json()).ticket || ''
    } finally {
      clearTimeout(ticketTimeout)
    }
    if (!pageActive) return
    if (!ticket) {
      connState.value = 'disconnected'
      sseRuntime.scheduleReconnect()
      return
    }

    const source = new EventSource(`${apiBase}/tasks/${taskId}/progress/stream?ticket=${encodeURIComponent(ticket)}`)
    eventSource = source
    source.onopen = () => {
      if (!sseRuntime.handleOpen(source)) return
      // This stream has no replay cursor; reconcile persisted rows after every reconnect.
      void fetchLogs()
    }
    source.onmessage = (e) => {
      if (!sseRuntime.handleMessage(source)) return
      try {
        const data = JSON.parse(e.data)
        if (!data || !task.value) return

        if (data.type === 'resync_required') {
          // 服务端慢客户端队列溢出时会发送此标记；从持久化接口补齐日志和状态。
          void fetchTask(true)
          void fetchLogs()
          return
        }
        if (data.type === 'task_wait') {
          const remaining = Number(data.remaining || 0)
          waitState.value = remaining > 0 ? {
            account_id: String(data.account_id || ''),
            target_id: String(data.target_id || ''),
            remaining,
            next_index: Number(data.next_index || 0),
            total: Number(data.total || 0),
          } : null
          return
        }
        if (data.type === 'task_log' && data.log) {
          const key = taskLogKey(data.log)
          if (!logKeys.has(key)) {
            logKeys.add(key)
            const container = logContainer.value
            const shouldFollow = !container || container.scrollHeight - container.scrollTop - container.clientHeight < 48
            logs.value.push(data.log)
            if (shouldFollow) {
              void nextTick(() => {
                if (pageActive && logContainer.value) logContainer.value.scrollTop = logContainer.value.scrollHeight
              })
            }
          }
          return
        }
        if (data.type === 'task_status') {
          void fetchTask(true)
          return
        }
        const tp = task.value.progress || {}
        task.value.progress = {
          total: data.total ?? data.progress?.total ?? tp.total ?? 0,
          done: data.done ?? data.progress?.done ?? tp.done ?? 0,
          failed: data.failed ?? data.progress?.failed ?? tp.failed ?? 0,
          current_target: data.current_target ?? data.progress?.current_target ?? tp.current_target ?? '',
          account_progress: data.account_progress ?? data.progress?.account_progress ?? tp.account_progress ?? []
        }
      } catch (err) { /* ignore malformed SSE data */ }
    }
    source.onerror = () => {
      sseRuntime.handleError(source)
    }
  } catch {
    if (pageActive) {
      connState.value = 'disconnected'
      sseRuntime.scheduleReconnect()
    }
  } finally {
    sseConnecting = false
  }
}

async function performTaskAction(
  kind: 'pause' | 'resume' | 'cancel',
  request: () => Promise<unknown>,
  successTitle: string,
  errorTitle: string,
) {
  if (!pageActive || actionPending.value) return
  actionPending.value = kind
  try {
    await request()
    if (!pageActive) return
    await fetchTask()
    if (pageActive) toast({ type: 'success', title: successTitle })
  } catch (e: unknown) {
    if (pageActive) toast({ type: 'error', title: errorTitle, message: e instanceof Error ? e.message : '请稍后重试' })
  } finally {
    if (pageActive) actionPending.value = null
  }
}
async function pause() {
  return performTaskAction('pause', () => taskApi.pause(taskId), '任务已暂停', '暂停失败')
}
async function resume() {
  return performTaskAction('resume', () => taskApi.resume(taskId), '任务已恢复', '恢复失败')
}
async function cancel() {
  return performTaskAction('cancel', () => taskApi.cancel(taskId), '任务已取消', '取消失败')
}

// 重试失败目标: 后端 scope=failed 克隆新任务(仅含失败目标, 续传防重保证不重复)
async function retryFailed() {
  if (!pageActive || !task.value || actionPending.value) return
  const failed = task.value.progress?.failed || 0
  const ok = await ask(`仅对 ${failed} 个失败目标重新发送？将创建新任务`, { title: '重试失败目标' })
  if (!ok || !pageActive || actionPending.value) return
  actionPending.value = 'retry'
  try {
    await taskApi.retry(taskId, 'failed')
    if (!pageActive) return
    toast({ type: 'success', title: '已创建重试任务', message: '可在任务列表查看进度' })
    router.push(`/workspace/${route.params.accountId}/tasks`)
  } catch (e: unknown) {
    if (pageActive) toast({ type: 'error', title: '重试失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  } finally {
    if (pageActive) actionPending.value = null
  }
}


onMounted(async () => {
  if (!isValidUUID(taskId)) {
    router.push(`/workspace/${route.params.accountId}/tasks`)
    return
  }

  await fetchTask()
  if (!pageActive) return
  await fetchLogs()
  if (!pageActive) return

  if ((task.value as any).type === 'scrape_members') void fetchMembers()

  // Start the task-scoped stream before loading optional display-name dictionaries.
  connectSSE()

  const onVis = () => {
    if (!pageActive || document.visibilityState !== 'visible') return
    void fetchLogs()
    if (connState.value !== 'connected') void connectSSE()
  }
  document.addEventListener('visibilitychange', onVis)
  onVisCleanup = () => document.removeEventListener('visibilitychange', onVis)

  refreshTimer = setInterval(() => {
    if (!pageActive) return
    // SSE 是实时通道，但队列满/短暂断线时可能丢掉终态事件；对未结束任务做轻量状态对账。
    if (task.value && !['completed', 'failed', 'cancelled'].includes(task.value.status)) {
      void fetchTask(true)
    }
    if (!isConnected.value) {
      void fetchLogs()
      if (connState.value === 'disconnected') void connectSSE()
    }
  }, 5000)

  // These dictionaries are supplementary. Their latency must not delay SSE or
  // disconnect recovery, and each result is discarded after page unmount.
  void (async () => {
    try {
      const accs = await accountApi.list()
      if (!pageActive) return
      const accData = (accs as any).data || accs || []
      accountMap.value = Object.fromEntries(accData.map((a: any) => [a.account_id, a.first_name || a.remark || a.username || a.phone || '未知账号']))
    } catch { /* optional labels */ }
  })()

  void (async () => {
    try {
      // Group library is tenant-wide because a task may span multiple accounts.
      const grps = await groupApi.list()
      if (!pageActive) return
      const raw: any = grps
      const grpData = Array.isArray(raw) ? raw
        : Array.isArray(raw?.data) ? raw.data
        : Array.isArray(raw?.data?.list) ? raw.data.list
        : Array.isArray(raw?.list) ? raw.list : []
      const entries: [string, string][] = []
      grpData.forEach((g: any) => {
        const name = g.title || g.name || g.group_name || (g.username ? '@' + String(g.username).replace(/^@/, '') : '') || '未命名群组'
        if (g.group_id) entries.push([normId(g.group_id), name])
        if (g.tg_chat_id) entries.push([normId(g.tg_chat_id), name])
        if (g.username) entries.push([normId(g.username), name])
      })
      groupMap.value = Object.fromEntries(entries)
    } catch { /* optional labels */ }
  })()
})

onUnmounted(() => {
  pageActive = false
  taskFetchRequestId++
  membersRequestId++
  if (completionRefreshTimer) { clearTimeout(completionRefreshTimer); completionRefreshTimer = null }
  eventSource?.close()
  eventSource = null
  clearInterval(refreshTimer)
  sseRuntime.dispose()
  if (onVisCleanup) onVisCleanup()
})
</script>

<style scoped>
.shimmer-bar {
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, var(--t-shimmer-highlight), transparent);
  animation: shimmer-slide 1.5s infinite;
}
@keyframes shimmer-slide {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(100%); }
}
</style>
