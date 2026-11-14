<template>
  <main class="space-y-6">
    <Breadcrumb :items="[{ path: '/admin/dashboard', label: '仪表盘' }, { path: '', label: '系统监控' }]" />

    <!-- 首次拉取失败必须说清楚，否则整页 0.0% 会被误读成"系统完全空闲" -->
    <div v-if="metricsError" role="alert" class="glass rounded-[var(--t-r-md)] p-4 border border-[color:var(--t-err)]/30 text-sm text-[var(--t-err)] lift flex items-center justify-between gap-3">
      <span>系统指标加载失败：{{ metricsError }}</span>
      <t-button size="small" variant="outline" class="ui-t-button ui-t-button--secondary" @click="fetchMetrics()">重试</t-button>
    </div>
    <!-- 骨架：避免首屏把 0 当成真实值 -->
    <div v-if="metricsLoading" class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div v-for="n in 3" :key="n" class="glass rounded-[var(--t-r-md)] p-5 lift space-y-4">
        <div class="h-3 w-1/3 rounded bg-bg-hover animate-pulse-soft" />
        <div class="h-8 w-1/2 rounded bg-bg-hover animate-pulse-soft" />
        <div class="h-2 w-full rounded bg-bg-hover animate-pulse-soft" />
      </div>
    </div>
    <!-- 资源监控 -->
    <div v-if="!metricsLoading" class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div class="glass rounded-[var(--t-r-md)] p-5 lift">
        <div class="flex items-center justify-between mb-4">
          <span class="text-sm text-text-secondary">CPU 使用率</span>
          <Cpu class="w-5 h-5 text-[var(--t-brand)]" />
        </div>
        <p class="text-3xl tracking-tight font-bold font-mono">{{ clampPercentage(metrics.cpu).toFixed(1) }}%</p>
        <div class="w-full h-2 bg-bg-card rounded-full mt-3 overflow-hidden">
          <div class="h-full bg-gradient-to-r rounded-full glow-progress transition-[width,background-color] duration-500" :class="usageColor(clampPercentage(metrics.cpu))" :style="{ width: `${clampPercentage(metrics.cpu)}%` }" />
        </div>
      </div>
      <div class="glass rounded-[var(--t-r-md)] p-5 lift">
        <div class="flex items-center justify-between mb-4">
          <span class="text-sm text-text-secondary">内存使用</span>
          <MemoryStick class="w-5 h-5 text-[var(--t-brand)]" />
        </div>
        <p class="text-3xl tracking-tight font-bold font-mono">{{ clampPercentage(metrics.memory).toFixed(1) }}%</p>
        <p class="text-xs text-text-secondary mt-1">{{ finiteMetric(metrics.memoryUsed).toFixed(1) }} / {{ finiteMetric(metrics.memoryTotal).toFixed(1) }} GB</p>
        <div class="w-full h-2 bg-bg-card rounded-full mt-2 overflow-hidden">
          <div class="h-full rounded-full transition-[width,background-color] duration-500" :class="usageBarColor(clampPercentage(metrics.memory))" :style="{ width: `${clampPercentage(metrics.memory)}%` }" />
        </div>
      </div>
      <div class="glass rounded-[var(--t-r-md)] p-5 lift">
        <div class="flex items-center justify-between mb-4">
          <span class="text-sm text-text-secondary">磁盘使用</span>
          <HardDrive class="w-5 h-5 text-[var(--t-warn)]" />
        </div>
        <p class="text-3xl tracking-tight font-bold font-mono">{{ clampPercentage(metrics.disk).toFixed(1) }}%</p>
        <!-- P1-36: diskTotal 为 0(未加载或后端未给)时显示占位, 不造假容量 -->
        <p class="text-xs text-text-secondary mt-1">{{ finiteMetric(metrics.diskTotal) > 0 ? finiteMetric(metrics.diskUsed).toFixed(1) + ' / ' + finiteMetric(metrics.diskTotal).toFixed(1) + ' GB' : '—' }}</p>
        <t-progress :percentage="Math.round(clampPercentage(metrics.disk) * 10) / 10"  class="mt-2" />
      </div>
    </div>

    <!-- 在线连接 + 队列 -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <div class="glass rounded-[var(--t-r-md)] p-5 lift">
        <h3 class="text-sm font-medium text-text-secondary mb-4">在线 TG 连接</h3>
        <div class="flex items-center gap-4">
          <div class="w-20 h-20 rounded-full border-4 border-[color:var(--t-brand)]/20 flex items-center justify-center">
            <span class="text-2xl tracking-tight font-bold font-mono text-[var(--t-brand)]">{{ finiteMetric(metrics.onlineConnections) }}</span>
          </div>
          <div class="space-y-1">
            <p class="text-sm max-w-[65ch]">在线账号</p>
            <p class="text-xs text-text-secondary">总账号: {{ finiteMetric(metrics.totalAccounts) }}</p>
            <p class="text-xs text-text-secondary">在线率: {{ clampPercentage(metrics.onlineRate).toFixed(1) }}%</p>
          </div>
        </div>
      </div>
      <div class="glass rounded-[var(--t-r-md)] p-5 lift">
        <h3 class="text-sm font-medium text-text-secondary mb-4">任务队列</h3>
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-sm">排队中</span>
            <span class="font-mono text-[var(--t-warn)]">{{ finiteMetric(metrics.queuedTasks) }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm">执行中</span>
            <span class="font-mono text-[var(--t-brand)]">{{ finiteMetric(metrics.runningTasks) }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm">今日完成</span>
            <span class="font-mono text-[var(--t-ok)]">{{ finiteMetric(metrics.completedToday) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 实时日志 -->
    <div class="glass rounded-[var(--t-r-md)] p-5 lift">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-sm font-medium text-text-secondary">系统日志</h3>
        <span class="flex items-center gap-1 text-xs text-[var(--t-ok)]">
          <span class="w-1.5 h-1.5 rounded-full bg-[var(--t-ok)] animate-pulse" />
          实时
        </span>
      </div>
      <div class="max-h-[300px] overflow-y-auto space-y-1 font-mono text-xs bg-bg-card rounded-lg p-4">
        <div v-for="(log, i) in systemLogs" :key="`${log.time}-${i}`" class="flex items-start gap-2 py-0.5">
          <span class="text-text-muted shrink-0">{{ log.time }}</span>
          <span class="shrink-0" :class="logLevelClass(log.level)">[{{ log.level }}]</span>
          <span class="text-text-secondary">{{ log.message }}</span>
        </div>
      </div>
    </div>
  </main>
</template>

<script setup lang="ts">
import Breadcrumb from '@/components/ui/Breadcrumb.vue'
import { ref, onMounted, onUnmounted } from 'vue'
import { Cpu, MemoryStick, HardDrive } from 'lucide-vue-next'
import { adminApiMod } from '@/api/admin'
import { clampPercentage, finiteMetric } from '@/utils/adminPageGuards'

const metrics = ref({
  cpu: 0, memory: 0, memoryUsed: 0, memoryTotal: 4,
  // P1-36: 原 diskTotal 硬编码 100——首屏闪"0.0 / 100.0 GB"假容量, 后端字段
  // 缺失/请求失败时永远显示假容量。初值改 0, 模板按 0 显示占位符, 真值到达后覆盖。
  disk: 0, diskUsed: 0, diskTotal: 0,
  onlineConnections: 0, totalAccounts: 0, onlineRate: 0,
  queuedTasks: 0, runningTasks: 0, completedToday: 0,
})

const systemLogs = ref<Array<{ time: string; level: string; message: string }>>([])
let refreshTimer: ReturnType<typeof setInterval>
let alertCooldownTimer: ReturnType<typeof setTimeout> | null = null

// 平滑采样：内存历史值（5次采样）
const memoryHistory = ref<number[]>([])
const alertCooldown = ref(false)
const ALERT_COOLDOWN_MS = 60000 // 警报冷却1分钟

function logLevelClass(level: string) {
  const map: Record<string, string> = { INFO: 'text-[var(--t-brand)]', WARN: 'text-[var(--t-warn)]', ERROR: 'text-[var(--t-err)]', DEBUG: 'text-text-muted' }
  return map[level] || 'text-text-secondary'
}

function pushLog(level: string, message: string) {
  systemLogs.value.unshift({
    time: new Date().toLocaleTimeString('zh-CN'),
    level,
    message,
  })
  if (systemLogs.value.length > 50) systemLogs.value.pop()
}

function checkAlert(key: string, value: number, threshold: number) {
  if (alertCooldown.value) return
  if (key === 'memory') {
    memoryHistory.value.push(value)
    if (memoryHistory.value.length > 5) memoryHistory.value.shift()
    // 5次采样中至少3次超过阈值才报警
    const exceedCount = memoryHistory.value.filter(v => v > threshold).length
    if (exceedCount >= 3) {
      pushLog('WARN', `内存使用率连续异常: ${value.toFixed(1)}%`)
      triggerCooldown()
    }
  } else if (value > threshold) {
    pushLog('WARN', `${key.toUpperCase()} 使用率过高: ${value.toFixed(1)}%`)
    triggerCooldown()
  }
}

function triggerCooldown() {
  alertCooldown.value = true
  if (alertCooldownTimer) clearTimeout(alertCooldownTimer)
  alertCooldownTimer = setTimeout(() => { alertCooldown.value = false; alertCooldownTimer = null }, ALERT_COOLDOWN_MS)
}

// 动态颜色：根据使用率返回对应颜色类
function usageColor(percent: number): string {
  if (percent >= 90) return 'from-[var(--t-err)] to-[var(--t-err)]'
  if (percent >= 75) return 'from-[var(--t-warn)] to-[var(--t-warn)]'
  return 'from-[var(--t-brand)] to-[var(--t-brand-hover)]'
}

function usageBarColor(percent: number): string {
  if (percent >= 90) return 'bg-[var(--t-err)]'
  if (percent >= 75) return 'bg-[var(--t-warn)]'
  return 'bg-[var(--t-brand)]'
}

// 首屏还没有数据时，页面上是一排 0.0%——看起来像"系统空闲"，
// 实际是"还没拉到"。用 loading/error 两个状态把这件事说清楚。
const metricsLoading = ref(true)
const metricsError = ref('')

let metricsRequestPending = false
let metricsMounted = false
let metricsController: AbortController | null = null
async function fetchMetrics() {
  // 轮询间隔短于慢请求耗时会产生重叠请求；保持单飞，避免旧快照覆盖新指标。
  if (!metricsMounted || metricsRequestPending) return
  metricsRequestPending = true
  const controller = new AbortController()
  metricsController = controller
  try {
    const data = await adminApiMod.getSystemMetrics({ signal: controller.signal })
    if (!metricsMounted) return
    metricsLoading.value = false
    metricsError.value = ''
    metrics.value = { ...metrics.value, ...data }
    metrics.value.cpu = clampPercentage(data.cpu)
    metrics.value.memory = clampPercentage(data.memory)
    metrics.value.memoryUsed = finiteMetric(data.memoryUsed)
    metrics.value.memoryTotal = finiteMetric(data.memoryTotal)
    metrics.value.disk = clampPercentage(data.disk)
    metrics.value.diskUsed = finiteMetric(data.diskUsed)
    metrics.value.diskTotal = finiteMetric(data.diskTotal)
    metrics.value.onlineConnections = finiteMetric(data.onlineConnections)
    metrics.value.totalAccounts = finiteMetric((data as any).total_accounts)
    metrics.value.onlineRate = clampPercentage((data as any).online_rate)
    metrics.value.queuedTasks = finiteMetric(data.queuedTasks)
    metrics.value.runningTasks = finiteMetric(data.runningTasks)
    metrics.value.completedToday = finiteMetric(data.completedToday)
    // 平滑采样警报检查
    checkAlert('memory', metrics.value.memory, 85)
    checkAlert('cpu', metrics.value.cpu, 90)
    checkAlert('disk', metrics.value.disk, 90)
    // P1-39: 原 systemLogs 只在超阈值警报时写入——"系统日志"面板永远空。
    // 每次成功拉取记一条 DEBUG(50 条环形缓冲, 不堆积)。
    pushLog('DEBUG', `指标已更新 CPU ${metrics.value.cpu}% 内存 ${metrics.value.memory}%`)
  } catch (e: unknown) {
    if (metricsMounted) {
      metricsLoading.value = false
      metricsError.value = (e as Error)?.message || '指标拉取失败'
      pushLog('ERROR', `指标拉取失败: ${metricsError.value}`)
    }
  } finally {
    if (metricsController === controller) metricsController = null
    metricsRequestPending = false
  }
}

// 审计UI-P1-04: visibility感知轮询(页面隐藏暂停, 可见立即刷新)
// 修复：原代码定义了 startAutoRefresh 却忘记调用，监控页只加载一次就永远静止
const tickHandler = () => { if (!document.hidden) fetchMetrics() }

function startAutoRefresh() {
  refreshTimer = setInterval(tickHandler, 5000)
  document.addEventListener('visibilitychange', tickHandler)
}

onMounted(() => {
  metricsMounted = true
  fetchMetrics()
  startAutoRefresh()
})

onUnmounted(() => {
  clearInterval(refreshTimer)
  // 防止组件离开后未完成的请求继续向已卸载页面写入状态。
  metricsMounted = false
  metricsController?.abort()
  metricsController = null
  if (alertCooldownTimer) clearTimeout(alertCooldownTimer)
  document.removeEventListener('visibilitychange', tickHandler)
})
</script>
