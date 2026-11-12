<template>
  <main class="max-w-3xl mx-auto space-y-4">
    <Breadcrumb :items="[{ path: `/workspace/${accountStore.currentAccount?.account_id || ''}`, label: '工作台' }, { path: '', label: '链接采集' }]" />
    <div class="flex items-center gap-3">
      <button type="button" @click="$router.back()" class="ui-icon-btn" aria-label="返回" title="返回">
        <ArrowLeft class="w-5 h-5" />
      </button>
      <h2 class="text-lg font-semibold text-gradient">群链接采集</h2>
    </div>

    <!-- R127 令牌化: seg 容器深浅主题自适应 -->
    <div class="ui-segmented w-full max-w-xs" role="tablist" aria-label="链接采集页面">
      <button
        v-for="t in [{ key: 'create', label: '链接监听' }, { key: 'results', label: '采集结果' }]"
        :key="t.key"
        @click="switchTab(t.key as 'create' | 'results')"
        class="ui-segmented__item"
        type="button"
        role="tab"
        :aria-selected="activeTab === t.key"
        :class="{ 'is-active': activeTab === t.key }"
      >
        {{ t.label }}
      </button>
    </div>

    <!-- R111: 协议协同创建Tab死块已删(view 恒 'phone'), 统一走监听开关流程 -->
    <!-- ===== 链接监听开关 ===== -->
    <div v-if="activeTab === 'create'" class="glass rounded-[var(--t-r-md)] p-5 space-y-4">
      <div class="flex items-center justify-between gap-4">
        <div class="min-w-0">
          <div class="font-medium">链接监听</div>
          <p class="text-xs text-text-muted mt-1 leading-relaxed">开启后, 当前账号自动采集<b>所有群组</b>消息中的邀请链接(公开+私密), 无需再选群</p>
        </div>
        <button
          type="button"
          @click="toggleListening"
           :aria-busy="listeningSaving"
          class="relative w-12 h-7 rounded-full transition-colors shrink-0"
          role="switch"
          :aria-checked="listening"
           :disabled="listeningSaving || !linksAccId"
          :class="listening ? 'bg-cyan' : 'bg-bg-hover'"
          aria-label="链接监听开关"
        ><span class="absolute top-1 w-5 h-5 rounded-full bg-white transition-all" :class="listening ? 'left-6' : 'left-1'" /></button>
      </div>
      <p v-if="listening" class="text-xs font-bold" style="color: var(--t-ok);">● 监听中, 结果在"采集结果"Tab 查看与导出</p>
      <p v-else class="text-xs text-text-muted">
        账号:
        <select v-model="linksAccId" class="ui-input !w-auto !min-h-[36px] !px-2 !py-1.5 !text-xs ml-1 align-middle" aria-label="链接监听账号">
          <option v-for="a in accountStore.accounts" :key="a.account_id" :value="a.account_id">{{ a.first_name || a.phone }}</option>
        </select>
      </p>
    </div>


    <div v-if="activeTab === 'create'" class="glass rounded-[var(--t-r-md)] p-5 space-y-3">
      <div class="flex items-center justify-between gap-4">
        <div class="min-w-0">
          <div class="font-medium">用户简介链接实时解析</div>
          <p class="text-xs text-text-muted mt-1 leading-relaxed">群内用户发言时读取发言者 Bio，提取简介中的 Telegram 链接并写入成员资料，供群库聚合器归并。与消息正文链接监听独立。</p>
        </div>
        <button type="button" @click="toggleBioListening" :disabled="bioSaving || !linksAccId"
          :aria-busy="bioSaving" role="switch" :aria-checked="bioListening"
          class="relative w-12 h-7 rounded-full transition-colors shrink-0"
          :class="bioListening ? 'bg-cyan' : 'bg-bg-hover'" aria-label="用户简介链接实时解析开关">
          <span class="absolute top-1 w-5 h-5 rounded-full bg-white transition-all" :class="bioListening ? 'left-6' : 'left-1'" />
        </button>
      </div>
      <p v-if="bioListening" class="text-xs font-bold" style="color: var(--t-ok);">● 简介实时解析已开启</p>
      <p v-else class="text-xs text-text-muted">关闭状态不会读取发言用户的个人简介。</p>
    </div>

    <!-- ===== 采集结果 ===== -->
    <div v-else class="space-y-3">
      <div v-if="tasksLoading" class="flex justify-center py-8">
        <t-loading size="small" loading class="inline-flex" />
      </div>

      <template v-else>
      <!-- R87-M8: 空态与下方总开关卡的空提示二选一(原双重空态), 展开面板时不再显示大空态 -->
      <EmptyState v-if="!tasks.length && !allLinksLoading && !allLinks.length && !allLinksOpen" icon="Link" title="暂无数据" description="暂无历史任务，也没有采集到链接" class="mb-3" />

      <!-- 总开关模式: 全部采集链接 -->
      <div class="glass rounded-[var(--t-r-md)] overflow-hidden mb-3">
        <div class="flex items-center justify-between gap-3 p-4 flex-wrap">
          <button type="button" @click="toggleAllLinks" class="flex min-w-0 flex-1 items-center justify-between gap-3 text-left hover:text-cyan transition-colors">
          <div class="flex items-center gap-3 min-w-0">
            <ChevronRight class="w-4 h-4 shrink-0 transition-transform" :class="{ 'rotate-90': allLinksOpen }" />
            <span class="text-sm font-medium">当前账号全部采集链接</span>
            <span v-if="allLinks.length" class="text-xs text-text-muted">{{ allLinks.length }} 条</span>
          </div>
          </button>
          <div class="flex items-center gap-2 shrink-0">
            <t-button type="button" class="ui-t-button--secondary ui-t-button" @click="exportAllLinks('txt')">导出 TXT</t-button>
            <t-button type="button" class="ui-t-button--secondary ui-t-button" @click="exportAllLinks('csv')">导出 CSV</t-button>
          </div>
        </div>
        <div v-if="allLinksOpen" class="border-t border-border">
          <div v-if="allLinksLoading" class="flex justify-center py-6 text-sm text-text-muted">加载中...</div>
          <div v-else-if="!allLinks.length" class="p-6 text-center text-text-muted text-sm">暂无链接, 先在"链接监听"Tab 开启监听, 收到含链接的群消息后会自动入库</div>
          <div v-else>
            <!-- 单行横向布局: 链接截断 + 类型徽标 + 复制, 手机上不再一链接占一整块 -->
            <div class="px-4 py-2 text-xs text-text-muted">
              共 {{ allLinks.length }} 条 · 私密 {{ allLinks.filter(l => l.link_type === 'private').length }} · 公开 {{ allLinks.filter(l => l.link_type !== 'private').length }}
            </div>
            <div class="divide-y divide-border/50">
              <pull-down-refresh v-model="pullRefreshing" @refresh="onPullRefresh">
              <div v-for="link in allLinks" :key="link.link_id" class="px-4 py-2.5 hover:bg-bg-hover/30">
                <!-- 上行: 时间 + 来源群 -->
                <div class="flex items-center gap-2 text-xs text-text-muted mb-1">
                  <span>{{ new Date(link.created_at).toLocaleString('zh-CN') }}</span>
                  <span v-if="link.source_chat_title" class="truncate max-w-[45%]">来自 {{ link.source_chat_title }}</span>
                  <span v-else-if="link.source_chat_id">来自 {{ link.source_chat_id }}</span>
                </div>
                <!-- 下行: 链接 -->
                <div class="flex items-center gap-2">
                  <!-- P2修复(前端审查): full_url 来自被抓取的TG消息(攻击者可控制内容),
                       Vue 不净化 :href 绑定, javascript: 协议可造成同源 XSS。
                       仅 http/https 渲染为链接, 其余降级为纯文本(与 safeUrl 同款思路)。 -->
                  <a v-if="isHttpUrl(link.full_url)" :href="link.full_url" target="_blank" rel="noopener" class="text-cyan text-sm truncate min-w-0 flex-1" :title="link.full_url">{{ link.full_url }}</a>
                  <span v-else class="text-cyan text-sm truncate min-w-0 flex-1" :title="link.full_url">{{ link.full_url }}</span>
                  <span class="shrink-0 text-xs px-1.5 py-0.5 rounded bg-bg-hover text-text-muted">{{ link.link_type === 'private' ? '私密' : '公开' }}</span>
                  <t-button type="button" @click="copyLink(link.full_url)" class="ui-t-button--secondary ui-t-button shrink-0">复制</t-button>
                </div>
              </div>
              </pull-down-refresh>
            </div>
          </div>
        </div>
      </div>

      <div v-if="tasks.length" class="space-y-2">
        <div
          v-for="task in tasks"
          :key="task.task_id"
          class="glass rounded-[var(--t-r-md)] overflow-hidden"
        >
          <!-- 任务行 -->
          <div class="flex items-center justify-between gap-3 p-4 flex-wrap">
            <button
              type="button"
              @click="toggleTask(task)"
              class="flex min-w-0 flex-1 items-center gap-3 text-left hover:text-cyan transition-colors"
              :aria-expanded="expandedTaskId === task.task_id"
            >
              <ChevronRight class="w-4 h-4 shrink-0 transition-transform" :class="{ 'rotate-90': expandedTaskId === task.task_id }" />
              <span class="text-sm font-medium truncate">任务 #{{ task.task_id.slice(0, 8) }}</span>
              <span
                class="text-xs px-2 py-0.5 rounded-full shrink-0"
                :class="{
                  'bg-green/10 text-green': task.status === 'completed',
                  'bg-yellow/10 text-yellow': task.status === 'running',
                  'bg-red/10 text-red': task.status === 'failed',
                  'bg-bg-hover text-text-muted': !['completed', 'running', 'failed'].includes(task.status),
                }"
              >{{ statusLabel(task.status) }}</span>
              <span class="text-xs text-text-muted hidden sm:inline">{{ new Date(task.created_at).toLocaleString() }}</span>
            </button>
            <div class="flex items-center gap-2 shrink-0">
              <t-button type="button" class="ui-t-button--secondary ui-t-button" @click="exportTaskLinks(task, 'txt')">导出 TXT</t-button>
              <t-button type="button" class="ui-t-button--secondary ui-t-button" @click="exportTaskLinks(task, 'csv')">导出 CSV</t-button>
            </div>
          </div>

          <!-- 展开: 链接列表 -->
          <div v-if="expandedTaskId === task.task_id" class="border-t border-border">
            <div v-if="linksLoading[task.task_id]" class="flex justify-center py-6">
              <t-loading size="small" loading class="inline-flex" />
            </div>
            <div v-else-if="!taskLinks[task.task_id]?.length" class="p-6 text-center text-text-muted text-sm">
              该任务窗口内暂无采集到链接(链接可能被去重过滤, 或服务未监听到消息)
            </div>
            <div v-else class="overflow-x-auto">
              <table class="w-full text-sm">
                <thead>
                  <tr class="text-left text-text-muted border-b border-border">
                    <th class="px-4 py-2 font-normal">链接</th>
                    <th class="px-4 py-2 font-normal">类型</th>
                    <th class="px-4 py-2 font-normal">解析结果</th>
                    <th class="px-4 py-2 font-normal">成员数</th>
                    <th class="px-4 py-2 font-normal">来源群</th>
                    <th class="px-4 py-2 font-normal">采集时间</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="link in taskLinks[task.task_id]" :key="link.link_id" class="border-b border-border/50 hover:bg-bg-hover/30">
                    <td class="px-4 py-2">
                      <a v-if="isHttpUrl(link.full_url)" :href="link.full_url" target="_blank" rel="noopener" class="text-cyan hover:underline break-all">{{ link.full_url }}</a>
                      <span v-else class="text-cyan break-all">{{ link.full_url }}</span>
                    </td>
                    <td class="px-4 py-2">{{ link.link_type === 'private' ? '私密' : '公开' }}</td>
                    <td class="px-4 py-2">{{ link.resolved_title || '-' }}</td>
                    <td class="px-4 py-2">{{ link.member_count ?? '-' }}</td>
                    <td class="px-4 py-2">{{ link.source_chat_title || link.source_chat_id }}</td>
                    <td class="px-4 py-2 text-text-muted whitespace-nowrap">{{ new Date(link.created_at).toLocaleString() }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
      </template>
    </div>
  </main>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import { useAuthStore } from '@/stores/auth'
import { useAccountStore } from '@/stores/account'
import { toast } from '@/utils/toast'
import { downloadBlob } from '@/utils/export'
// R111 已删 ProtocolPicker/campaignApi/CAMPAIGN_STATUS 导入(协议协同死代码)
import { api } from '@/utils/request'

import { useRouter } from 'vue-router'
import { taskApi } from '@/api/task'
import type { ScrapedLink } from '@/api/task'

import Breadcrumb from '@/components/ui/Breadcrumb.vue'
// R115: Group/Account 死类型导入已删(正文零使用)

// 仅放行 http/https 协议, 拦截 javascript:/data: 等危险协议(同 safeUrl 思路)
const isHttpUrl = (u: string): boolean => /^https?:\/\//i.test(u || '')

const router = useRouter()
const authStore = useAuthStore()
const accountStore = useAccountStore()

// ===== Tab: create=新建采集 results=采集结果 =====
const activeTab = ref<'create' | 'results'>('create')

function switchTab(tab: 'create' | 'results') {
  activeTab.value = tab
  if (tab === 'results') {
    fetchTasks()
    // 预载链接总量, 避免 EmptyState 因 allLinks 未加载而误报"暂无数据"
    if (!allLinks.value.length && !allLinksLoading.value) loadAllLinks()
  }
}

// ===== 采集结果 Tab =====
interface ScrapeTask {
  task_id: string
  status: string
  account_ids: string[]
  created_at: string
}
// 总开关模式: 直接展示当前账号采集到的全部链接(不依赖任务)
const allLinks = ref<ScrapedLink[]>([])
const allLinksOpen = ref(false)
const allLinksLoading = ref(false)

const tasks = ref<ScrapeTask[]>([])
const tasksLoading = ref(false)
const expandedTaskId = ref<string | null>(null)
const taskLinks = ref<Record<string, ScrapedLink[]>>({})
const linksLoading = ref<Record<string, boolean>>({})

// R108: 视图级账号切换 —— 监听配置按账号独立, 下拉选择管理哪个号(不依赖顶栏切换器)
const linksAccId = ref('')
watch(accountStore.currentAccount, (acc: any) => { if (acc?.account_id && !linksAccId.value) linksAccId.value = acc.account_id }, { immediate: true })
let taskListRequestId = 0
let allLinksRequestId = 0
let taskListController: AbortController | null = null
let allLinksController: AbortController | null = null
const taskLinkRequestIds = new Map<string, number>()
const taskLinkControllers = new Map<string, AbortController>()
let messageStatusController: AbortController | null = null
let bioStatusController: AbortController | null = null
watch(linksAccId, () => {
  // 切换账号时立即取消旧账号网络请求并清空旧数据。
  messageStatusController?.abort()
  bioStatusController?.abort()
  taskListController?.abort()
  allLinksController?.abort()
  for (const controller of taskLinkControllers.values()) controller.abort()
  taskLinkControllers.clear()
  taskListRequestId++
  allLinksRequestId++
  taskLinkRequestIds.clear()
  tasks.value = []
  allLinks.value = []
  taskLinks.value = {}
  linksLoading.value = {}
  expandedTaskId.value = null
  fetchTasks()
  loadAllLinks()
})

async function fetchTasks() {
  const accId = linksAccId.value
  const requestId = ++taskListRequestId
  taskListController?.abort()
  taskListController = null
  if (!accId) {
    tasks.value = []
    tasksLoading.value = false
    return
  }
  tasksLoading.value = true
  const controller = new AbortController()
  taskListController = controller
  try {
    const res = await taskApi.list({ type: 'scrape_links', account_id: accId, page: 1, page_size: 50 } as any, { signal: controller.signal })
    if (requestId !== taskListRequestId || accId !== linksAccId.value) return
    const raw = res as any
    tasks.value = raw.list || raw.items || []
  } catch (e: unknown) {
    if (!controller.signal.aborted && requestId === taskListRequestId && accId === linksAccId.value) {
      toast({ type: 'error', title: '加载任务失败', message: (e instanceof Error ? e.message : '请稍后重试') })
    }
  } finally {
    if (requestId === taskListRequestId) tasksLoading.value = false
    if (taskListController === controller) taskListController = null
  }
}

const pullRefreshing = ref(false)
async function onPullRefresh() {
  // P1-28 审计修复: 原 Promise.all 任一请求失败, 后续赋值不执行——下拉转圈永不
  // 停止。finally 保证收尾, allSettled 允许部分成功。
  try {
    await Promise.allSettled([loadAllLinks(), fetchTasks()])
  } finally {
    pullRefreshing.value = false
  }
}

async function loadAllLinks() {
  const accId = linksAccId.value
  const requestId = ++allLinksRequestId
  allLinksController?.abort()
  allLinksController = null
  if (!accId) {
    allLinks.value = []
    allLinksLoading.value = false
    return
  }
  allLinksLoading.value = true
  const controller = new AbortController()
  allLinksController = controller
  try {
    const res = await taskApi.listScrapedLinks({ account_id: accId, page: 1, page_size: 100 }, { signal: controller.signal })
    if (requestId !== allLinksRequestId || accId !== linksAccId.value) return
    const raw = res as any
    allLinks.value = raw.list || raw.items || []
  } catch (e: any) {
    if (!controller.signal.aborted && requestId === allLinksRequestId && accId === linksAccId.value) {
      toast({ type: 'error', title: '加载链接失败', message: e?.message || String(e) })
    }
  } finally {
    if (requestId === allLinksRequestId) allLinksLoading.value = false
    if (allLinksController === controller) allLinksController = null
  }
}

async function toggleAllLinks() {
  allLinksOpen.value = !allLinksOpen.value
  if (allLinksOpen.value && !allLinks.value.length) await loadAllLinks()
}

async function toggleTask(task: ScrapeTask) {
  if (expandedTaskId.value === task.task_id) {
    expandedTaskId.value = null
    taskLinkControllers.get(task.task_id)?.abort()
    taskLinkControllers.delete(task.task_id)
    taskLinkRequestIds.set(task.task_id, (taskLinkRequestIds.get(task.task_id) || 0) + 1)
    linksLoading.value = { ...linksLoading.value, [task.task_id]: false }
    return
  }
  expandedTaskId.value = task.task_id
  for (const [taskId, pending] of taskLinkControllers) {
    if (taskId === task.task_id) continue
    pending.abort()
    taskLinkControllers.delete(taskId)
    taskLinkRequestIds.set(taskId, (taskLinkRequestIds.get(taskId) || 0) + 1)
    linksLoading.value = { ...linksLoading.value, [taskId]: false }
  }
  if (taskLinks.value[task.task_id] || linksLoading.value[task.task_id]) return
  const accId = linksAccId.value
  const requestId = (taskLinkRequestIds.get(task.task_id) || 0) + 1
  taskLinkRequestIds.set(task.task_id, requestId)
  taskLinkControllers.get(task.task_id)?.abort()
  const controller = new AbortController()
  taskLinkControllers.set(task.task_id, controller)
  linksLoading.value = { ...linksLoading.value, [task.task_id]: true }
  try {
    // since=任务创建时间, 圈出该任务的采集窗口
    const res = await taskApi.listScrapedLinks({
      account_id: task.account_ids[0],
      since: task.created_at,
      page: 1,
      page_size: 100,
    }, { signal: controller.signal })
    if (accId !== linksAccId.value || taskLinkRequestIds.get(task.task_id) !== requestId) return
    const raw = res as any
    taskLinks.value = { ...taskLinks.value, [task.task_id]: raw.list || raw.items || [] }
  } catch (e: unknown) {
    if (!controller.signal.aborted && accId === linksAccId.value && taskLinkRequestIds.get(task.task_id) === requestId) {
      toast({ type: 'error', title: '加载链接失败', message: (e instanceof Error ? e.message : '请稍后重试') })
    }
  } finally {
    if (accId === linksAccId.value && taskLinkRequestIds.get(task.task_id) === requestId) {
      linksLoading.value = { ...linksLoading.value, [task.task_id]: false }
    }
    if (taskLinkControllers.get(task.task_id) === controller) taskLinkControllers.delete(task.task_id)
  }
}

// 复制链接到剪贴板(clipboard API 不可用时降级)
async function copyLink(url: string) {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(url)
      toast({ type: 'success', title: '已复制链接' })
      return
    }
  } catch {
    // Clipboard API may be unavailable or denied; try the legacy browser command.
  }

  const ta = document.createElement('textarea')
  ta.value = url
  ta.setAttribute('readonly', '')
  ta.style.position = 'fixed'
  ta.style.opacity = '0'
  document.body.appendChild(ta)
  ta.select()
  let copied = false
  try {
    copied = document.execCommand('copy')
  } catch {
    copied = false
  } finally {
    ta.remove()
  }
  if (copied) toast({ type: 'success', title: '已复制链接' })
  else toast({ type: 'error', title: '复制失败', message: '请手动选择并复制链接' })
}

// 导出纯文本(每行一个链接), 与"自动加群-导入链接"的格式一致
async function exportTaskLinks(task: ScrapeTask, format: 'txt' | 'csv' = 'txt') {
  const accountId = task.account_ids?.[0]
  if (!accountId) {
    toast({ type: 'error', title: '导出失败', message: '该任务没有关联的账号，无法导出' })
    return
  }
  try {
    const base = import.meta.env.VITE_API_BASE_URL || '/api/v1'
    const params = new URLSearchParams({ account_id: accountId, since: task.created_at, format })
    const resp = await fetch(`${base}/scrape-links/export?${params}`, {
      headers: { Authorization: `Bearer ${authStore.token}` },
    })
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
    const text = await resp.text()
    const mimeType = format === 'csv' ? 'text/csv;charset=utf-8' : 'text/plain;charset=utf-8'
    const blob = new Blob([text], { type: mimeType })
    downloadBlob(`scrape-links-${task.task_id.slice(0, 8)}.${format}`, blob)
  } catch (e: unknown) {
    toast({ type: 'error', title: '导出失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  }
}

// 导出全部链接(不带时间窗口)
async function exportAllLinks(format: 'txt' | 'csv') {
  const acc = accountStore.accounts.find((a: any) => a.account_id === linksAccId.value)
  if (!acc) { toast({ type: 'error', title: '请先选择账号' }); return }
  try {
    const base = import.meta.env.VITE_API_BASE_URL || '/api/v1'
    const params = new URLSearchParams({ account_id: acc.account_id, format })
    const resp = await fetch(`${base}/scrape-links/export?${params}`, {
      headers: { Authorization: `Bearer ${authStore.token}` },
    })
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
    const text = await resp.text()
    const mimeType = format === 'csv' ? 'text/csv;charset=utf-8' : 'text/plain;charset=utf-8'
    const blob = new Blob([text], { type: mimeType })
    downloadBlob(`scrape-links-all.${format}`, blob)
  } catch {
    toast({ type: 'error', title: '导出失败' })
  }
}

function statusLabel(s: string) {
  const m: Record<string, string> = {
    queued: '排队中', running: '运行中', paused: '已暂停',
    completed: '已完成', failed: '失败', cancelled: '已取消',
  }
  return m[s] || s
}


onUnmounted(() => {
  messageStatusController?.abort()
  bioStatusController?.abort()
  taskListController?.abort()
  allLinksController?.abort()
  for (const controller of taskLinkControllers.values()) controller.abort()
  taskLinkControllers.clear()
  taskListRequestId++
  allLinksRequestId++
  for (const [taskId, requestId] of taskLinkRequestIds) taskLinkRequestIds.set(taskId, requestId + 1)
})

onMounted(() => {
  if (!authStore.tenant?.plan_features?.scrape_links) {
    toast({ type: 'error', title: '当前套餐不支持链接采集，请联系客服升级套餐' })
    router.push('/dashboard')
  }
})

// ===== 链接监听总开关 =====
const listening = ref(false)
const listeningSaving = ref(false)
const bioListening = ref(false)
const bioSaving = ref(false)

async function loadBioListeningStatus(accountId: string) {
  bioStatusController?.abort()
  bioStatusController = null
  if (!accountId) { bioListening.value = false; return }
  const controller = new AbortController()
  bioStatusController = controller
  try {
    const res = await fetch(`${import.meta.env.VITE_API_BASE_URL || '/api/v1'}/scrape-links/status?account_id=${encodeURIComponent(accountId)}&mode=bio`, {
      headers: { Authorization: `Bearer ${authStore.token}` },
      signal: controller.signal,
    })
    if (res.ok && linksAccId.value === accountId) bioListening.value = (await res.json()).enabled
  } catch {} finally {
    if (bioStatusController === controller) bioStatusController = null
  }
}

async function toggleBioListening() {
  if (bioSaving.value) return
  const accountId = linksAccId.value
  if (!accountId) return
  const next = !bioListening.value
  bioSaving.value = true
  try {
    const res = await fetch(`${import.meta.env.VITE_API_BASE_URL || '/api/v1'}/scrape-links/toggle`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${authStore.token}` },
      body: JSON.stringify({ account_id: accountId, enabled: next, mode: 'bio' }),
    })
    if (!res.ok) throw new Error()
    if (linksAccId.value === accountId) {
      bioListening.value = next
      toast({ type: 'success', title: next ? '简介实时解析已开启' : '简介实时解析已关闭' })
    }
  } catch {
    if (linksAccId.value === accountId) toast({ type: 'error', title: '简介解析开关操作失败', message: '请稍后重试' })
  } finally { bioSaving.value = false }
}

async function loadMessageListeningStatus(accountId: string) {
  messageStatusController?.abort()
  messageStatusController = null
  if (!accountId) { listening.value = false; return }
  const controller = new AbortController()
  messageStatusController = controller
  try {
    const res = await fetch(`${import.meta.env.VITE_API_BASE_URL || '/api/v1'}/scrape-links/status?account_id=${encodeURIComponent(accountId)}`, {
      headers: { Authorization: `Bearer ${authStore.token}` },
      signal: controller.signal,
    })
    if (res.ok && linksAccId.value === accountId) listening.value = (await res.json()).enabled
  } catch {} finally {
    if (messageStatusController === controller) messageStatusController = null
  }
}

onMounted(() => {
  loadMessageListeningStatus(linksAccId.value)
  loadBioListeningStatus(linksAccId.value)
})
watch(linksAccId, accountId => {
  loadMessageListeningStatus(accountId)
  loadBioListeningStatus(accountId)
})

// ===== R111: 协议协同脚本已全部删除(loadCampaigns/submitProtocol/controlCampaign/exportLinks) =====

async function toggleListening() {
  if (listeningSaving.value) return
  const accountId = linksAccId.value
  const acc = accountStore.accounts.find((a: any) => a.account_id === accountId)
  if (!acc) { toast({ type: 'error', title: '请先选择账号' }); return }
  const next = !listening.value
  listeningSaving.value = true
  try {
    const res = await fetch(`${import.meta.env.VITE_API_BASE_URL || '/api/v1'}/scrape-links/toggle`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${authStore.token}` },
      body: JSON.stringify({ account_id: acc.account_id, enabled: next }),
    })
    if (!res.ok) throw new Error()
    if (linksAccId.value === accountId) {
      listening.value = next
      toast({ type: 'success', title: next ? '链接监听已开启' : '链接监听已关闭' })
      if (next) switchTab('results')
    }
  } catch {
    if (linksAccId.value === accountId) toast({ type: 'error', title: '操作失败, 请重试' })
  } finally {
    listeningSaving.value = false
  }
}
</script>
