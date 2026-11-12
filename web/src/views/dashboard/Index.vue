<template>
  <main class="min-w-0 w-full space-y-5 sm:space-y-6 lg:space-y-7">
    <!-- Workspace header -->
    <section class="relative overflow-hidden rounded-2xl border border-border bg-bg-primary p-5 sm:p-7 lg:p-8">
      <div class="pointer-events-none absolute -right-16 -top-24 h-64 w-64 rounded-full opacity-20"
           style="background: radial-gradient(circle, var(--t-brand), transparent 68%);" />
      <div class="pointer-events-none absolute bottom-0 right-24 h-px w-2/5 opacity-30"
           style="background: linear-gradient(90deg, transparent, var(--t-brand), transparent);" />
      <div class="relative flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between">
        <div class="min-w-0">
          <div class="flex items-center gap-2 text-xs font-semibold tracking-wide text-text-muted">
            <span class="h-1.5 w-1.5 rounded-full" style="background: var(--t-brand);" />
            工作台 <span class="text-text-muted/50">/</span> 运营总览
          </div>
          <h1 class="mt-3 text-2xl font-bold tracking-tight text-text-primary sm:text-3xl">{{ greeting }}，欢迎回来</h1>
          <p class="mt-2 max-w-xl text-sm leading-6 text-text-secondary">
            在这里掌握账号状态、群发数据与近期动态。所有数据会随当前选中账号同步更新。
          </p>
          <p class="mt-3 text-xs text-text-muted">{{ todayLabel }}<span v-if="currentAccount"> · 当前账号：{{ currentAccount.first_name || currentAccount.remark || currentAccount.phone || '已选账号' }}</span></p>
        </div>
        <div class="flex shrink-0 flex-wrap gap-2">
          <router-link to="/accounts"
            class="inline-flex min-h-10 items-center justify-center gap-2 rounded-xl border border-border px-4 text-sm font-semibold text-text-primary transition-colors hover:bg-bg-secondary">
            <span aria-hidden="true">＋</span> 添加账号
          </router-link>
          <router-link :to="joinPath"
            class="inline-flex min-h-10 items-center justify-center gap-2 rounded-xl border border-border px-4 text-sm font-semibold text-text-primary transition-colors hover:bg-bg-secondary">
            <span aria-hidden="true">＋</span> 创建加群任务
          </router-link>
          <router-link :to="broadcastPath"
            class="ui-native-button ui-native-button--primary min-h-11 rounded-xl px-4 shadow-sm">

            <span aria-hidden="true">↗</span> 创建群发任务
          </router-link>
        </div>
      </div>
    </section>

    <!-- KPI overview -->
    <section>
      <div class="mb-3 flex items-center justify-between gap-3">
        <div>
          <h2 class="text-sm font-bold text-text-primary">核心数据</h2>
          <p class="mt-0.5 text-xs text-text-muted">当前账号范围内的运营概况</p>
        </div>
        <span class="text-[11px] text-text-muted">实时统计</span>
      </div>
      <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <article v-for="(stat, i) in stats" :key="stat.label"
          class="group min-w-0 rounded-2xl border border-border bg-bg-primary p-4 transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md sm:p-5">
          <div class="flex items-center justify-between gap-2">
            <p class="truncate text-xs font-medium text-text-muted">{{ stat.label }}</p>
            <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl text-xs font-bold"
              :class="['bg-cyan-dim text-cyan','bg-emerald/10 text-emerald','bg-amber/10 text-amber','bg-rose/10 text-rose'][i]">
              {{ ['●','✓','▦','!'][i] }}
            </span>
          </div>
          <p class="mt-3 text-2xl font-bold tracking-tight text-text-primary sm:text-3xl">{{ stat.value }}</p>
          <p class="mt-1 text-[11px] text-text-muted">{{ ['当前可用账号','今日已完成任务','已收录群组','最近 24 小时'][i] }}</p>
        </article>
      </div>
    </section>

    <!-- Activity and analytics -->
    <section class="grid min-w-0 grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1.45fr)_minmax(280px,0.8fr)]">
      <div class="min-w-0 space-y-4">
        <article class="rounded-2xl border border-border bg-bg-primary p-4 sm:p-5">
          <div class="mb-4 flex items-start justify-between gap-3">
            <div>
              <h2 class="text-sm font-bold text-text-primary">发送日历</h2>
              <p class="mt-1 text-xs text-text-muted">每日成功发送量，追踪近期运营节奏</p>
            </div>
            <span class="rounded-lg bg-bg-secondary px-2.5 py-1 text-[11px] font-medium text-text-muted">近 6 个月</span>
          </div>
          <SendCalendar />
        </article>

        <div class="grid min-w-0 grid-cols-1 gap-4 md:grid-cols-2">
          <article class="min-w-0 rounded-2xl border border-border bg-bg-primary p-4 sm:p-5">
            <div class="mb-3">
              <h2 class="text-sm font-bold text-text-primary">近 7 天任务趋势</h2>
              <p class="mt-1 text-xs text-text-muted">每日完成任务数量</p>
            </div>
            <div v-if="loading" class="flex h-52 items-center justify-center">
              <span class="h-5 w-5 animate-spin rounded-full border-2" style="border-color: var(--t-brand); border-top-color: transparent;" />
            </div>
            <div v-else-if="!hasTaskData" class="flex h-52 flex-col items-center justify-center text-center">
              <span class="mb-3 flex h-11 w-11 items-center justify-center rounded-2xl bg-bg-secondary text-lg text-text-muted">↗</span>
              <p class="text-sm font-medium text-text-secondary">暂无任务趋势</p>
              <p class="mt-1 text-xs text-text-muted">创建任务后，这里会显示完成情况</p>
            </div>
            <v-chart v-else class="h-52 w-full" :option="taskChartOption" autoresize />
          </article>

          <article class="min-w-0 rounded-2xl border border-border bg-bg-primary p-4 sm:p-5">
            <div class="mb-3">
              <h2 class="text-sm font-bold text-text-primary">账号状态分布</h2>
              <p class="mt-1 text-xs text-text-muted">账号连接状态概览</p>
            </div>
            <div v-if="loading" class="flex h-52 items-center justify-center">
              <span class="h-5 w-5 animate-spin rounded-full border-2" style="border-color: var(--t-brand); border-top-color: transparent;" />
            </div>
            <div v-else-if="!hasAccountData" class="flex h-52 flex-col items-center justify-center text-center">
              <span class="mb-3 flex h-11 w-11 items-center justify-center rounded-2xl bg-bg-secondary text-lg text-text-muted">◎</span>
              <p class="text-sm font-medium text-text-secondary">还没有账号数据</p>
              <router-link to="/accounts" class="mt-2 text-xs font-semibold hover:underline" style="color: var(--t-brand);">前往账号管理 →</router-link>
            </div>
            <v-chart v-else class="h-52 w-full" :option="accountChartOption" autoresize />
          </article>
        </div>
      </div>

      <!-- Recent events -->
      <div class="min-w-0 space-y-4">
        <article class="rounded-2xl border border-border bg-bg-primary p-4 sm:p-5">
          <div class="mb-4 flex items-center justify-between gap-3">
            <div>
              <h2 class="text-sm font-bold text-text-primary">风险提醒</h2>
              <p class="mt-1 text-xs text-text-muted">最近 24 小时的账号风险事件</p>
            </div>
            <router-link to="/accounts" class="shrink-0 text-xs font-semibold hover:underline" style="color: var(--t-brand);">账号管理 →</router-link>
          </div>
          <div class="space-y-2.5">
            <div v-for="event in riskEvents" :key="event.risk_id"
              class="flex min-w-0 items-start gap-3 rounded-xl bg-bg-secondary/70 p-3">
              <span class="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-amber/10 text-amber">
                <AlertTriangle class="h-4 w-4" />
              </span>
              <div class="min-w-0 flex-1">
                <p class="truncate text-sm font-semibold text-text-primary">{{ riskTypeLabel(event.risk_type) }}</p>
                <p class="mt-1 truncate text-xs text-text-secondary">
                  <span v-if="event.account_id">{{ riskAccName(event.account_id) }}</span>
                  <template v-if="event.account_id && event.detail"> · </template>
                  <span v-if="event.detail">{{ event.detail }}</span>
                </p>
                <p class="mt-1 text-[11px] text-text-muted">{{ formatRelativeTime(event.created_at) }}</p>
              </div>
              <span class="shrink-0 rounded-md px-2 py-1 text-[10px] font-semibold" :class="severityClass(event.severity)">
                {{ severityLabel(event.severity) }}
              </span>
            </div>
            <div v-if="!riskEvents.length" class="flex flex-col items-center py-6 text-center">
              <Shield class="mb-2 h-7 w-7 text-emerald" />
              <p class="text-sm font-medium text-text-primary">暂时没有风险提醒</p>
              <p class="mt-1 text-xs text-text-muted">有新的风险事件时会显示在这里</p>
            </div>
          </div>
        </article>

        
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { riskTypeLabel, severityLabel } from '@/utils/labels'
import { ref, onMounted, computed, onUnmounted, onBeforeUnmount, watch } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, PieChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { useChartTheme } from '@/composables/useChartTheme'
import { AlertTriangle, Shield } from 'lucide-vue-next'
import { dashboardApi } from '@/api/dashboard'
import { toast } from '@/utils/toast'
import { useAccountStore } from '@/stores/account'
import { settingsApi } from '@/api/settings'
import StatCard from '@/components/charts/StatCard.vue'
import SendCalendar from '@/components/SendCalendar.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import { formatRelativeTime, formatDate } from '@/utils/format'
import type { RiskEvent } from '@/types'

const greeting = (() => {
  const h = new Date().getHours()
  if (h < 6) return '夜深了'
  if (h < 12) return '早上好'
  if (h < 14) return '中午好'
  if (h < 18) return '下午好'
  return '晚上好'
})()
const todayLabel = new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'long', day: 'numeric', weekday: 'long' }).format(new Date())
const accountStore = useAccountStore()
const currentAccount = computed(() => accountStore.currentAccount)
const broadcastPath = computed(() => {
  const accountId = currentAccount.value?.account_id
  return accountId ? `/workspace/${accountId}/broadcast` : '/accounts'
})
const joinPath = computed(() => {
  const accountId = currentAccount.value?.account_id
  return accountId ? `/workspace/${accountId}/join` : '/accounts'
})
let dashboardRequestId = 0
watch(currentAccount, () => { loadDashboardData() })

use([CanvasRenderer, LineChart, PieChart, GridComponent, TooltipComponent, LegendComponent])
const loading = ref(true)
const stats = ref([
  { label: '在线账号', value: '0', icon: 'Users', color: 'cyan', trend: '+0' },
  { label: '今日任务', value: '0', icon: 'ListTodo', color: 'cyan', trend: '+0' },
  { label: '总群组', value: '0', icon: 'MessageSquare', color: 'amber', trend: '+0' },
  { label: '风险事件', value: '0', icon: 'AlertTriangle', color: 'rose', trend: '0' },
])
const riskEvents = ref<RiskEvent[]>([])
function riskAccName(id: string) {
  const a = (accountStore.accounts || []).find((x: any) => String(x.account_id) === String(id))
  if (!a) return '账号 ' + String(id).slice(0, 6)
  return a.first_name || a.remark || a.phone || '账号 ' + String(id).slice(0, 6)
}
const taskData = ref<number[]>([])
const accountData = ref<{ value: number; name: string }[]>([])
const expiresAt = ref('')
const daysLeft = ref(0)
const hasTaskData = computed(() => (taskData.value?.length || 0) > 0 && taskData.value.some((v) => v > 0))
const hasAccountData = computed(() => (accountData.value?.length || 0) > 0 && (accountData.value || []).some((v) => v.value > 0))
const chartTheme = useChartTheme()
onMounted(() => chartTheme.start())
onBeforeUnmount(() => chartTheme.stop())

const taskChartOption = computed(() => ({
  backgroundColor: 'transparent',
  tooltip: {
    trigger: 'axis',
    backgroundColor: 'rgba(15, 23, 42, 0.95)',
    borderColor: 'rgba(255, 255, 255, 0.1)',
    borderWidth: 1,
    textStyle: { color: chartTheme.textPrimary.value },
    extraCssText: 'box-shadow: 0 4px 20px rgba(0,0,0,0.4); border-radius: 0.75rem;',
  },
  grid: { left: '2%', right: '3%', bottom: '2%', top: '8%', containLabel: true },
  xAxis: {
    type: 'category',
    data: ['周一', '周二', '周三', '周四', '周五', '周六', '周日'],
    axisLine: { show: false },
    axisTick: { show: false },
    axisLabel: { color: chartTheme.textMuted.value, fontSize: 11 },
  },
  yAxis: {
    type: 'value',
    minInterval: 1,
    axisLine: { show: false },
    axisTick: { show: false },
    splitLine: { lineStyle: { color: chartTheme.border.value + '33' } },
    axisLabel: { color: chartTheme.textMuted.value, fontSize: 11 },
  },
  series: [{
    name: '完成任务',
    type: 'line',
    smooth: true,
    symbol: 'circle',
    symbolSize: 6,
    showSymbol: false,
    data: taskData.value,
    lineStyle: { color: chartTheme.palette.value[0], width: 2.5 },
    itemStyle: { color: chartTheme.palette.value[0], borderWidth: 2, borderColor: 'rgba(15,23,42,0.8)' },
    areaStyle: {
      color: {
        type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
        colorStops: [
          { offset: 0, color: `${chartTheme.palette.value[0]}40` },
          { offset: 1, color: `${chartTheme.palette.value[0]}00` },
        ],
      },
    },
  }],
}))

const accountChartOption = computed(() => ({
  backgroundColor: 'transparent',
  tooltip: {
    backgroundColor: 'rgba(15, 23, 42, 0.95)',
    borderColor: 'rgba(255, 255, 255, 0.1)',
    borderWidth: 1,
    textStyle: { color: chartTheme.textPrimary.value },
    extraCssText: 'box-shadow: 0 4px 20px rgba(0,0,0,0.4); border-radius: 0.75rem;',
  },
  legend: {
    bottom: 0,
    itemWidth: 8,
    itemHeight: 8,
    itemGap: 16,
    textStyle: { color: chartTheme.textMuted.value, fontSize: 11 },
  },
  series: [{
    type: 'pie',
    radius: ['42%', '72%'],
    center: ['50%', '44%'],
    avoidLabelOverlap: false,
    padAngle: 3,
    itemStyle: { borderRadius: 6, borderColor: 'rgba(15,23,42,0.8)', borderWidth: 2 },
    data: accountData.value,
    label: { show: false },
    emphasis: {
      label: { show: true, fontSize: 13, fontWeight: 'bold', color: chartTheme.textPrimary.value },
      itemStyle: { shadowBlur: 10, shadowColor: 'rgba(0,0,0,0.1)' },
    },
  }],
}))

function severityClass(s: string) {
  const m: Record<string, string> = {
    low: 'bg-text-muted/15 text-text-secondary',
    medium: 'bg-[color:var(--t-warn-dim)] text-[var(--t-warn)]',
    high: 'bg-[color:var(--t-err-dim)] text-[var(--t-err)]',
    critical: 'bg-[color:var(--t-err-dim)] text-[var(--t-err)]',
  }
  return m[s] || ''
}

async function loadDashboardData() {
  const requestId = ++dashboardRequestId
  const accountId = currentAccount.value?.account_id
  try {
    loading.value = true
    const s = await dashboardApi.getStats(accountId)
    if (requestId !== dashboardRequestId) return
    if (s) {
      stats.value[0].value = String(s.online_accounts ?? 0)
      stats.value[1].value = String(s.completed_tasks_today ?? 0)
      stats.value[2].value = String(s.total_groups ?? 0)
      stats.value[3].value = String(s.risk_events_24h ?? 0)
      taskData.value = s.task_trend || []
      accountData.value = s.account_distribution || []
    }
    const re = await dashboardApi.getRiskEvents({ limit: 5 })
    if (requestId !== dashboardRequestId) return
    riskEvents.value = (re as any)?.list || (Array.isArray(re) ? re : [])
    try {
      const profile = await settingsApi.getProfile()
      if (requestId !== dashboardRequestId) return
      if (profile.expires_at) {
        expiresAt.value = formatDate(profile.expires_at)
        const diff = new Date(profile.expires_at).getTime() - Date.now()
        daysLeft.value = Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)))
      }
    } catch (e) { /* ignore */ }
  } catch (e: unknown) {
    if (requestId !== dashboardRequestId) return
    toast({ type: 'error', title: '仪表盘数据加载失败', message: (e instanceof Error ? e.message : '请检查网络或联系管理员') })
  } finally {
    if (requestId === dashboardRequestId) loading.value = false
  }
}

onMounted(() => { loadDashboardData() })
// 页面离开时使未完成的账号/仪表盘请求失效，避免卸载后继续写入旧视图状态。
onUnmounted(() => { dashboardRequestId++ })
</script>
