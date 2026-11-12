<template>
  <main class="space-y-3 md:space-y-4 min-w-0">
    <div class="flex items-center justify-between mb-6 flex-wrap gap-3">
      <div>
        <h1 class="text-title font-extrabold tracking-tight text-text-primary truncate">任务列表</h1>
        <!-- R109: 任务列表为全租户视角(各账号任务汇总), 移除误导性的"当前账号"标签 -->
        <p class="text-sm text-text-muted mt-1 max-w-[65ch]">全部账号的任务汇总, 点卡片查看详情</p>
      </div>
      <!-- 类型 Tab: 点一次直接过滤 -->
      <div class="ui-segmented flex gap-1.5 overflow-x-auto pb-1 -mx-1 px-1" role="group" aria-label="按任务类型筛选">
        <button type="button" v-for="t in typeTabs" :key="t.value" @click="typeFilter = t.value"
                class="ui-segmented__item whitespace-nowrap"
                :class="{ 'is-active': typeFilter === t.value }">
          {{ t.label }}
        </button>
      </div>
      <div class="ui-segmented flex gap-2 mb-3" role="group" aria-label="按创建时间筛选">
  <button type="button" v-for="r in [['today','今日'],['7d','近7日'],['all','全部']]" :key="r[0]"
    @click="range = r[0]; page = 1; fetchTasks()"
    class="ui-segmented__item"
    :class="{ 'is-active': range === r[0] }">
    {{ r[1] }}
  </button>
</div>
<div class="flex items-center gap-2 flex-wrap">
        <div class="flex items-center gap-1.5">
          <input v-model="numberSearch" class="ui-input !w-36 !min-h-[36px] !px-3 !py-1.5 !text-xs" placeholder="按任务编号搜索" aria-label="按任务编号搜索" @keyup.enter="searchByNumber" />
          <button type="button" class="ui-icon-btn" @click="searchByNumber" aria-label="搜索任务编号" title="搜索编号"><Search class="w-4 h-4" /></button>
          <button v-if="numberSearch" type="button" class="ui-icon-btn" @click="clearNumberSearch" aria-label="清除编号搜索" title="清除搜索"><X class="w-4 h-4" /></button>
        </div>
        <!-- R128 原生 select: t-select 的 tdesign 下拉面板在移动端定位滑到页面底部(用户反馈),
             原生 select 视觉已令牌化且系统自带可靠下拉 -->
        <select v-model="statusFilter"
                class="ui-input !w-32 !min-h-[36px] !px-3 !py-1.5 !text-xs font-medium cursor-pointer"
                style="background: var(--t-surface-2); border: 1px solid var(--t-line); color: var(--t-text-1);">
          <option value="">全部状态</option>
          <option value="running">运行中</option>
          <option value="queued">排队中</option>
          <option value="completed">已完成</option>
          <option value="failed">失败</option>
          <option value="cancelled">已取消</option>
        </select>
        <button type="button" class="ui-icon-btn ui-icon-btn--danger" @click="clearFinished" title="清空已完成任务" aria-label="清空已完成任务">
          <Trash2 class="w-4 h-4" />
        </button>
        <button type="button" @click="fetchTasks" class="ui-icon-btn" aria-label="刷新任务">
          <RefreshCw class="w-4 h-4" />
        </button>
      </div>
    </div>

    <!-- 群组链接导入任务：任务属于任务中心，不属于正式群组库 -->
    <section v-if="importJobs.length" class="space-y-3">
      <div class="flex items-center justify-between">
        <h2 class="text-sm font-semibold text-text-primary">群组导入与解析</h2>
        <span class="text-xs text-text-muted">{{ importJobs.length }} 条任务</span>
      </div>
      <article v-for="job in importJobs" :key="job.job_id" class="ui-card p-4 space-y-2">
        <div class="flex items-center justify-between gap-2">
          <div class="min-w-0">
            <div class="text-sm font-semibold text-text-primary truncate">{{ job.file_name || '文本导入' }}</div>
            <div class="text-xs text-text-muted mt-1"><span class="font-semibold tabular-nums text-[var(--t-brand)]">{{ formatTaskNumber(job.task_number) }}</span> · 群组链接导入 · {{ formatDate(job.created_at) }}</div>
          </div>
          <span class="px-2 py-1 rounded text-xs whitespace-nowrap" :class="job.status === 'completed' ? 'text-[var(--t-ok)] bg-[color:var(--t-ok-dim)]' : job.status === 'failed' ? 'text-[var(--t-err)] bg-[color:var(--t-err-dim)]' : job.status === 'cancelled' ? 'text-text-muted bg-bg-hover' : 'text-[var(--t-warn)] bg-[color:var(--t-warn-dim)]'">
            {{ importStatusLabel(job.status) }}
          </span>
        </div>
        <div class="flex items-center justify-between text-xs text-text-secondary">
          <span>已处理 {{ job.processed_count || 0 }} / {{ job.total_count || 0 }}</span>
          <span>入库 {{ job.imported_count || 0 }} · 失败 {{ job.failed_count || 0 }} · 跳过 {{ job.skipped_count || 0 }}</span>
        </div>
        <div class="h-1.5 rounded-full overflow-hidden bg-bg-secondary">
          <div class="h-full rounded-full bg-[var(--t-brand)] transition-all" :style="{ width: (job.total_count ? Math.min(100, (job.processed_count || 0) / job.total_count * 100) : 0) + '%' }"></div>
        </div>
        <div class="flex flex-wrap items-center justify-end gap-2">
          <t-button v-if="job.status === 'review'" class="ui-t-button ui-t-button--primary ui-t-button--sm" size="small" theme="primary" @click="confirmGroupImport(job.job_id)">确认筛选结果并开始解析</t-button>
          <t-button v-if="['review', 'pending', 'processing'].includes(job.status)" class="ui-t-button ui-t-button--danger ui-t-button--sm" size="small" theme="danger" @click="cancelGroupImport(job.job_id)">取消导入</t-button>
          <t-button v-if="['completed', 'failed', 'cancelled'].includes(job.status) && job.failed_count > 0" class="ui-t-button ui-t-button--warning ui-t-button--sm" size="small" theme="warning" @click="retryGroupImport(job.job_id)">重试失败链接 ({{ job.failed_count }})</t-button>
          <t-button v-if="['completed', 'failed', 'cancelled'].includes(job.status)" class="ui-t-button ui-t-button--secondary ui-t-button--sm" size="small" theme="default" @click="clearGroupImportJob(job)">清除记录</t-button>
        </div>
        <p v-if="job.last_error" class="text-xs text-[var(--t-err)] break-words">{{ job.last_error }}</p>
      </article>
    </section>

    <!-- 任务列表 -->
    <div class="space-y-3">
      <div v-if="loading" class="space-y-3 py-2">
        <div v-for="i in 5" :key="i" class="bg-bg-card rounded-[var(--t-r-md)] p-4 space-y-3">
          <div class="flex items-center gap-3">
            <div class="rounded-full bg-bg-hover/70 animate-pulse" style="width:2.5rem;height:2.5rem"></div>
            <div class="flex-1 space-y-2">
              <div class="rounded bg-bg-hover/70 animate-pulse" style="width:40%;height:0.875rem"></div>
              <div class="rounded bg-bg-hover/70 animate-pulse" style="width:25%;height:0.75rem"></div>
            </div>
            <div class="rounded bg-bg-hover/70 animate-pulse" style="width:4rem;height:1.25rem"></div>
          </div>
          <div class="rounded bg-bg-hover/70 animate-pulse" style="width:100%;height:0.5rem"></div>
        </div>
      </div>
      <div v-else-if="tasks.length === 0 && importJobs.length === 0" class="py-6">
        <EmptyState title="暂无任务" description="创建群发/加群/采集任务后会显示在这里" />
      </div>
      <!-- 电脑端: TDesign 表格骨架 -->
      <div v-if="tasks.length" class="hidden md:block mt-2 w-full max-w-full overflow-x-auto">
        <t-table row-key="task_id" :data="tasks" :columns="tableCols" hover style="min-width: 860px"
                 @row-click="(ctx: any) => $router.push(`/workspace/${route.params.accountId}/tasks/${ctx.row.task_id}`)">
          <template #task_number="{ row }"><span class="font-semibold tabular-nums text-[var(--t-brand)]">{{ formatTaskNumber(row.task_number) }}</span></template>
          <template #status="{ row }">
            <span class="px-2 py-0.5 rounded text-xs font-medium" :class="statusClass(row.status)">{{ statusLabel(row.status) }}</span>
          </template>
          <template #type="{ row }">{{ taskTypeLabel(row.type) }}</template>
          <!-- 消息概览: 群发类任务预览文案前 24 字 -->
          <template #overview="{ row }">
            <span class="text-xs text-text-secondary max-w-[180px] truncate inline-block align-middle" :title="msgPreview(row)">{{ msgPreview(row) || '—' }}</span>
          </template>
          <!-- 目标数: 任务总目标量 -->
          <template #targets="{ row }">
            <span class="text-xs text-text-secondary">{{ progressOf(row.progress).total }}</span>
          </template>
          <!-- 累计下发: 成功/失败/等待 + 双色进度条 + 部分失败标签 -->
          <template #progress="{ row }">
            <div class="space-y-1">
              <div class="flex items-center gap-2">
                <div class="w-[120px] h-1.5 bg-bg-secondary rounded-full overflow-hidden shrink-0 flex">
                  <div class="h-full bg-[var(--t-brand)] transition-all duration-500" :style="{ width: donePct(row) + '%' }" />
                  <div class="h-full bg-[var(--t-err)] transition-all duration-500" :style="{ width: failedPct(row) + '%' }" />
                </div>
                <span v-if="progressOf(row.progress).failed > 0" class="px-1.5 py-0.5 rounded text-[10px] bg-[color:var(--t-warn-dim)] text-[var(--t-warn)] whitespace-nowrap">部分失败</span>
              </div>
              <div class="text-xs text-text-secondary whitespace-nowrap">
                成功 <span class="text-[var(--t-ok)] font-medium">{{ progressOf(row.progress).done }}</span>
                / 失败 <span class="text-[var(--t-err)] font-medium">{{ progressOf(row.progress).failed }}</span>
                / 等待 <span class="text-[var(--t-warn)]">{{ Math.max(0, progressOf(row.progress).total - progressOf(row.progress).done - progressOf(row.progress).failed) }}</span>
              </div>
              <!-- R118: 等待倒计时(仅运行中) -->
              <div v-if="row.status === 'running' && waitInfo[row.task_id]" class="text-[11px] text-[var(--t-warn)] whitespace-nowrap">
                <span class="tabular-nums font-medium">{{ waitInfo[row.task_id].remaining }}s</span>
                <span class="text-text-secondary"> · 账号 {{ accNameById(waitInfo[row.task_id].account_id) }} 等待中</span>
              </div>
            </div>
          </template>
          <template #created_at="{ row }">
            <span class="text-xs">{{ formatDate(row.created_at) }}</span>
          </template>
          <template #op="{ row }">
            <div class="flex items-center gap-1.5">
              <t-button v-if="row.status === 'running'" class="ui-t-button ui-t-button--danger ui-t-button--sm" size="small" theme="danger" @click.stop="cancelTask(row.task_id)">停止</t-button>
              <t-button v-if="row.status === 'failed'" class="ui-t-button ui-t-button--primary ui-t-button--sm" size="small" theme="primary" @click.stop="retryTask(row.task_id)">重试</t-button>
              <t-button v-if="['failed','cancelled','completed'].includes(row.status) && progressOf(row.progress).failed > 0"
                         class="ui-t-button ui-t-button--warning ui-t-button--sm" size="small" theme="warning" @click.stop="retryTask(row.task_id, 'failed')">重试失败 ({{ progressOf(row.progress).failed }})</t-button>
              <!-- 每账号统计: 点击弹层查看该任务各账号成功/失败/跳过 -->
              <t-tooltip content="每账号统计" position="top">
                <t-button class="ui-t-button ui-t-button--secondary ui-t-button--sm" size="small" variant="outline" @click.stop="showAccStats(row)" title="每账号统计">
                  <BarChart3 class="w-3.5 h-3.5" />
                </t-button>
              </t-tooltip>
            </div>
          </template>
        </t-table>
      </div>

      <!-- 手机端: 卡片列表(支持下拉刷新) -->
      <pull-down-refresh v-if="tasks.length" v-model="pullRefreshing" class="md:hidden" @refresh="onPullRefresh">
      <div v-for="(task, ti) in tasks" :key="task.task_id" :style="{ '--i': ti }"
           :class="['p-3 sm:p-4 rounded-[var(--t-r-md)] border bg-bg-card card-lift transition-colors', task.status === 'running' ? 'card-running' : 'border-border']"
           role="group" :aria-label="'任务 ' + taskTypeLabel(task.type) + ' ' + task.task_id">
        <RouterLink :to="'/workspace/' + route.params.accountId + '/tasks/' + task.task_id"
                    class="block rounded-md focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--t-brand)]">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-bold whitespace-nowrap" :class="statusClass(task.status)">
              <i class="w-1.5 h-1.5 rounded-full bg-current opacity-80"></i>{{ statusLabel(task.status) }}
            </span>
            <span class="text-sm font-medium text-text-primary">{{ taskTypeLabel(task.type) }}</span>
            <span class="text-[10px] font-semibold tabular-nums text-[var(--t-brand)]">{{ formatTaskNumber(task.task_number) }}</span>
            <!-- R112: 执行账号列 —— 全租户汇总后标识每条任务属于哪个号 -->
            <span v-if="accountNameOf(task)" class="text-[10.5px] px-1.5 py-0.5 rounded bg-[color:var(--t-brand-dim)] text-[var(--t-brand)] whitespace-nowrap"
                  :title="'执行账号: ' + accountNameOf(task)">
              {{ accountNameOf(task) }}
            </span>
          </div>
          <span class="text-xs text-text-secondary">{{ formatDate(task.created_at) }}</span>
          <span v-if="task.status === 'queued'" class="text-xs text-[var(--t-warn)]">已等待 {{ elapsed(task.created_at) }}</span>
          <span v-else-if="task.status === 'running'" class="text-xs text-[var(--t-brand)]">已运行 {{ elapsed(task.created_at) }}</span>
        </div>
        <div class="mt-2">
          <div class="flex items-center justify-between text-xs text-text-secondary mb-1">
            <span>成功 {{ progressOf(task.progress).done }} / 失败 {{ progressOf(task.progress).failed }} / 等待 {{ Math.max(0, progressOf(task.progress).total - progressOf(task.progress).done - progressOf(task.progress).failed) }}</span>
            <span v-if="progressOf(task.progress).failed > 0" class="px-1.5 py-0.5 rounded text-[10px] bg-[color:var(--t-warn-dim)] text-[var(--t-warn)]">部分失败</span>
          </div>
          <div class="w-full h-1.5 bg-bg-secondary rounded-full overflow-hidden flex">
            <div class="h-full bg-[var(--t-brand)] transition-all duration-500" :class="task.status === 'running' ? 'bar-animated' : ''" :style="{ width: donePct(task.progress) + '%' }" />
            <div class="h-full bg-[var(--t-err)] transition-all duration-500" :style="{ width: failedPct(task.progress) + '%' }" />
          </div>
          <!-- R118: 等待倒计时 —— 等待过程每秒可见 -->
          <div v-if="task.status === 'running' && waitInfo[task.task_id]" class="mt-1.5 flex items-center gap-1.5 text-[11px] text-[var(--t-warn)]">
            <span class="tabular-nums font-medium">{{ waitInfo[task.task_id].remaining }}s</span>
            <span class="text-text-secondary">· 账号 {{ accNameById(waitInfo[task.task_id].account_id) }} 等待中</span>
            <span v-if="waitInfo[task.task_id].total" class="text-text-muted">({{ waitInfo[task.task_id].next_index }}/{{ waitInfo[task.task_id].total }})</span>
          </div>
        </div>
        </RouterLink>
        <div class="mt-2 flex items-center gap-2">
          <t-button v-if="task.status === 'running'" class="ui-t-button ui-t-button--danger ui-t-button--sm" size="small" theme="danger" @click.stop="cancelTask(task.task_id)">取消</t-button>
          <t-button v-if="task.status === 'failed'" class="ui-t-button ui-t-button--primary ui-t-button--sm" size="small" theme="primary" @click.stop="retryTask(task.task_id)">重试</t-button>
          <t-button v-if="['failed','cancelled','completed'].includes(task.status) && progressOf(task.progress).failed > 0"
                     size="small" theme="warning" class="ui-t-button ui-t-button--warning ui-t-button--sm" @click.stop="retryTask(task.task_id, 'failed')">重试失败 ({{ progressOf(task.progress).failed }})</t-button>
          <t-button class="ui-t-button ui-t-button--secondary ui-t-button--sm" size="small" variant="outline" @click.stop="showAccStats(task)">统计</t-button>
        </div>
      </div>
      </pull-down-refresh>
    </div>

    <!-- 分页 -->
    <div v-if="total > pageSize" class="flex flex-wrap items-center justify-between gap-3 mt-6">
      <span class="text-xs text-text-secondary">共 {{ total }} 条</span>
      <!-- P0-07: TDesign total 是总条数不是页数, 原传 totalPages 导致分页严重错误 -->
      <t-pagination class="ui-pagination" :page="page" :total="total" :page-size="pageSize" @change="(p: any) => page = p.current" />
    </div>

    <!-- 每账号统计弹层: 任务内各账号成功/失败/跳过 -->
    <Modal v-model="accStatsVisible" :title="`每账号统计 — ${accStatsTaskName}`" width="480px">
      <div v-if="accStatsLoading" class="py-8 flex justify-center">
        <div class="skeleton-shimmer w-full h-8 rounded mb-2" v-for="i in 4" :key="i" />
      </div>
      <div v-else-if="accStatsData.length" class="w-full overflow-x-auto">
      <table class="w-full min-w-[420px] text-sm">
        <thead>
          <tr class="text-left text-xs text-text-muted border-b border-border">
            <th class="py-2 pr-2 font-medium">执行账号</th>
            <th class="py-2 pr-2 text-right font-medium text-[var(--t-ok)]">成功</th>
            <th class="py-2 pr-2 text-right font-medium text-[var(--t-err)]">失败</th>
            <th class="py-2 text-right font-medium">跳过</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in accStatsData" :key="row.phone" class="border-b border-border/50 row-hover">
            <td class="py-2.5 pr-2 text-text-primary">{{ row.phone }}</td>
            <td class="py-2.5 pr-2 text-right tabular-nums text-[var(--t-ok)]">{{ row.success }}</td>
            <td class="py-2.5 pr-2 text-right tabular-nums text-[var(--t-err)]">{{ row.failed }}</td>
            <td class="py-2.5 text-right tabular-nums text-text-secondary">{{ row.skipped }}</td>
          </tr>
        </tbody>
        <tfoot>
          <tr class="text-text-primary font-medium">
            <td class="py-2.5 pr-2">合计</td>
            <td class="py-2.5 pr-2 text-right tabular-nums text-[var(--t-ok)]">
              {{ accStatsData.reduce((s, r) => s + r.success, 0) }}</td>
            <td class="py-2.5 pr-2 text-right tabular-nums text-[var(--t-err)]">
              {{ accStatsData.reduce((s, r) => s + r.failed, 0) }}</td>
            <td class="py-2.5 text-right tabular-nums text-text-secondary">
              {{ accStatsData.reduce((s, r) => s + r.skipped, 0) }}</td>
          </tr>
        </tfoot>
      </table>
      </div>
      <p v-else class="py-8 text-center text-sm text-text-muted">该任务暂无执行记录</p>
    </Modal>
  </main>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import { useRoute, useRouter } from 'vue-router'
import { useAccountStore } from '@/stores/account'
import { taskApi } from '@/api/task'
import { listGroupLibraryImportJobs, confirmImportLibraryGroups, cancelGroupLibraryImportJob, deleteGroupLibraryImportJob, retryFailedGroupLibraryImportJob } from '@/api/groups'
import { api } from '@/utils/request'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import { useTaskEvents } from '@/composables/useTaskEvents'
import { toast } from '@/utils/toast'
import { ask } from '@/utils/confirm'
import { ListTodo, RefreshCw, Trash2, Search, X } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const accountStore = useAccountStore()
const currentAccount = computed(() => accountStore.currentAccount)
// R112: 任务→执行账号名(account_ids 首号; 名称取 first_name/remark/phone)
const accountNameOf = (task: any) => {
  const id = task?.account_ids?.[0]
  if (!id) return ''
  const a = (accountStore.accounts || []).find((x: any) => x.account_id === id)
  return a ? (a.first_name || a.remark || a.phone || '') : ''
}
// R118: accName 死函数已删(模板无引用)

// 模板 12 行的类型过滤 Tab 此前引用未定义（Tab 栏整体不渲染，过滤功能失效）。
const typeTabs = [
  { value: '', label: '全部' },
  { value: 'broadcast', label: '群发' },
  { value: 'scrape_members', label: '采集' },
  { value: 'join_groups', label: '加群' },
  { value: 'monitor_scrape', label: '监控' },
  { value: 'scheduled_broadcast', label: '定时' },
]

const tasks = ref<any[]>([])
const numberSearch = ref('')
const activeNumberSearch = ref('')
const importJobs = ref<any[]>([])
const loading = ref(false)
const loadError = ref('')
// E-02: 分页/筛选状态与 URL 双向同步——刷新/分享/前进后退均不丢状态
const query = route.query as Record<string, string | undefined>
const page = ref(Math.max(1, parseInt(query.page || '1', 10) || 1))
const pageSize = 20
const total = ref(0)
const range = ref(query.range || 'today') // 今日(默认) | 7d | all
const statusFilter = ref(query.status || '')
const typeFilter = ref(query.type || '')

const totalPages = computed(() => Math.ceil(total.value / pageSize))

// 清空已结束任务: 复用现有 batch/delete 接口, 零后端改动
// 跨页拉全量(每状态最多 500 条), 避免只清当前页漏其他页
// R109: 任务列表全租户化 —— 移除 account_id 过滤(原绑全局切换器, 已删除;
// 页头文案已是"全部账号的任务汇总")
async function clearFinished() {
  const ids: string[] = []
  for (const status of ['completed', 'failed', 'cancelled']) {
    try {
      const res: any = await taskApi.list({
        status,
        page: 1,
        page_size: 500,
        hide_loops: 1,
        range: 'all',
      } as any)
      const list = Array.isArray(res) ? res : (res?.data ?? res?.list ?? [])
      for (const t of list) if (t?.task_id) ids.push(t.task_id)
    } catch { /* 单状态拉取失败不阻塞整体 */ }
  }
  if (!ids.length) {
    toast({ type: 'info', title: '没有已结束的任务' })
    return
  }
  const ok = await ask(`确定清空 ${ids.length} 条已结束任务？此操作不可恢复`,
    { title: '清空任务', danger: true, confirmText: '清空' })
  if (!ok) return
  try {
    await taskApi.batchDelete(ids)
    toast({ type: 'success', title: `已清空 ${ids.length} 条任务` })
    page.value = 1
    fetchTasks()
  } catch (e: any) {
    toast({ type: 'error', title: '清空失败', message: e?.message || '请稍后重试' })
  }
}

let importJobsRequestId = 0
let importJobsController: AbortController | null = null
let taskListController: AbortController | null = null
async function fetchImportJobs() {
  importJobsController?.abort()
  const controller = new AbortController()
  importJobsController = controller
  const requestId = ++importJobsRequestId
  const taskNumber = activeNumberSearch.value
  try {
    const res: any = await listGroupLibraryImportJobs(taskNumber || undefined, { signal: controller.signal })
    // 5 秒轮询、自动刷新和编号搜索可能并发；只接受当前编号筛选对应的最新响应。
    if (requestId !== importJobsRequestId || taskNumber !== activeNumberSearch.value) return
    importJobs.value = res.list || []
  } catch {
    // keep the last known task list during transient network errors
  } finally {
    if (importJobsController === controller) importJobsController = null
  }
}

function importStatusLabel(status: string) {
  const labels: Record<string, string> = {
    review: '待确认', pending: '排队中', processing: '解析中', cancelling: '取消中', cancelled: '已取消',
    completed: '已完成', failed: '失败',
  }
  return labels[status] || status || '未知'
}

async function confirmGroupImport(jobId: string) {
  try {
    await confirmImportLibraryGroups(jobId)
    toast({ type: 'success', title: '已开始解析', message: '群组导入任务已进入任务中心队列' })
    await fetchImportJobs()
  } catch (e: any) {
    toast({ type: 'error', title: '确认失败', message: e?.message || '无法确认导入任务' })
  }
}

async function cancelGroupImport(jobId: string) {
  const ok = await ask('确定取消这条群组导入任务？已完成的链接不会回滚。', {
    title: '取消导入任务', danger: true, confirmText: '取消任务',
  })
  if (!ok) return
  try {
    await cancelGroupLibraryImportJob(jobId)
    toast({ type: 'success', title: '已提交取消请求' })
    await fetchImportJobs()
  } catch (e: any) {
    toast({ type: 'error', title: '取消失败', message: e?.message || '无法取消导入任务' })
  }
}

async function retryGroupImport(jobId: string) {
  try {
    await retryFailedGroupLibraryImportJob(jobId)
    toast({ type: 'success', title: '失败链接已重新排队' })
    await fetchImportJobs()
  } catch (e: any) {
    toast({ type: 'error', title: '重试失败', message: e?.message || '无法重试失败链接' })
  }
}

async function clearGroupImportJob(job: any) {
  const ok = await ask('确定清除这条已结束的导入任务记录？只删除任务记录及失败明细，不会删除已经入库的群组。', {
    title: '清除导入记录', danger: true, confirmText: '清除记录',
  })
  if (!ok) return
  try {
    await deleteGroupLibraryImportJob(job.job_id)
    // Delete API 已确认成功后立即从当前列表移除，避免等待下一次列表请求造成“清不掉”的错觉。
    importJobs.value = importJobs.value.filter(item => item.job_id !== job.job_id)
    await fetchImportJobs()
    toast({ type: 'success', title: '导入任务记录已清除' })
  } catch (e: any) {
    toast({ type: 'error', title: '清除失败', message: e?.message || '任务可能仍在运行，请稍后重试' })
    await fetchImportJobs()
  }
}

let taskListRequestId = 0

function formatTaskNumber(value: number | string | undefined): string {
  const number = Number(value)
  return Number.isSafeInteger(number) && number > 0 ? '#' + String(number).padStart(6, '0') : '#------'
}

async function searchByNumber() {
  const value = numberSearch.value.trim().replace(/^#/, '')
  if (value && !/^\d+$/.test(value)) {
    toast({ type: 'warning', title: '请输入有效的任务编号' })
    return
  }
  numberSearch.value = value
  activeNumberSearch.value = value
  page.value = 1
  await Promise.all([fetchTasks(), fetchImportJobs()])
}

async function clearNumberSearch() {
  numberSearch.value = ''
  activeNumberSearch.value = ''
  page.value = 1
  await Promise.all([fetchTasks(), fetchImportJobs()])
}

async function fetchTasks() {
  taskListController?.abort()
  const controller = new AbortController()
  taskListController = controller
  const requestId = ++taskListRequestId
  // 请求参数在发起时快照化，避免等待期间筛选/分页变化造成响应与当前视图错配。
  const params: any = { page: page.value, page_size: pageSize, hide_loops: 1 }
   if (activeNumberSearch.value) params.task_number = activeNumberSearch.value
  if (range.value !== 'all') params.range = range.value
  if (statusFilter.value) params.status = statusFilter.value
  if (typeFilter.value) params.type = typeFilter.value
  loading.value = true
  loadError.value = ''
  try {
    const res = await taskApi.list(params, { signal: controller.signal })
    if (requestId !== taskListRequestId) return
    const rawList = Array.isArray(res?.list) ? res.list : []
    // 按 task_id 去重，防止接口异常或重试造成重复行。
    tasks.value = [...new Map(rawList.filter((t: any) => t?.task_id).map((t: any) => [t.task_id, t])).values()]
    total.value = Number(res?.total) || 0
  } catch (e) {
    if (controller.signal.aborted || requestId !== taskListRequestId) return
    loadError.value = e instanceof Error ? e.message : '获取任务失败，请稍后重试'
    console.error('获取任务失败:', e)
  } finally {
    if (taskListController === controller) taskListController = null
    if (requestId === taskListRequestId) loading.value = false
  }
}

// 筛选变化时重置到第一页再拉取(避免停留在后面的页码导致筛选"看起来没反应")
watch([statusFilter, typeFilter], () => {
  if (page.value !== 1) page.value = 1
  else fetchTasks()
})

// E-02: 状态回写 URL（默认值不写，保持 URL 干净）
watch([page, range, statusFilter, typeFilter], () => {
  router.replace({
    query: {
      ...route.query,
      page: page.value > 1 ? String(page.value) : undefined,
      range: range.value !== 'today' ? range.value : undefined,
      status: statusFilter.value || undefined,
      type: typeFilter.value || undefined,
    },
  })
})

const taskActionIds = ref(new Set<string>())
async function cancelTask(id: string) {
  if (!id || taskActionIds.value.has(id)) return
  taskActionIds.value = new Set(taskActionIds.value).add(id)
  try {
    await taskApi.cancel(id)
    void fetchTasks()
    toast({ type: 'success', title: '任务已取消' })
  } catch (e: unknown) {
    toast({ type: 'error', title: '取消失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  } finally {
    const next = new Set(taskActionIds.value)
    next.delete(id)
    taskActionIds.value = next
  }
}

// 每账号统计弹层: 点任务行"统计"按钮, 懒加载该任务各账号成功/失败/跳过
const accStatsVisible = ref(false)
const accStatsLoading = ref(false)
const accStatsData = ref<{ phone: string; success: number; failed: number; skipped: number }[]>([])
const accStatsTaskName = ref('')

async function showAccStats(row: any) {
  accStatsVisible.value = true
  accStatsLoading.value = true
  accStatsTaskName.value = row.name || row.task_id?.slice(0, 8) || ''
  accStatsData.value = []
  try {
    const raw = await api.get<Record<string, { success: number; failed: number; skipped: number }>>(
      `/tasks/${row.task_id}/stats-by-account`)
    // account_id → 手机号映射(从任务 account_ids + 账号 store)
    const idSet = new Set<string>()
    const collect = (v: any) => { if (Array.isArray(v)) v.forEach((x: any) => idSet.add(String(x))) }
    collect(row.account_ids); collect(row.payload?.account_ids); collect(row.payload?.accounts)
    const phoneMap: Record<string, string> = {}
    for (const acc of accountStore.accounts) {
      if (idSet.has(String(acc.account_id))) phoneMap[String(acc.account_id)] = acc.phone
    }
    accStatsData.value = Object.entries(raw).map(([id, v]) => ({
      phone: phoneMap[id] || `账号 ${id.slice(0, 6)}…`,
      success: v.success, failed: v.failed, skipped: v.skipped,
    })).sort((a, b) => b.success - a.success)
  } catch { /* 请求层统一报错 */ }
  finally { accStatsLoading.value = false }
}

async function retryTask(id: string, scope?: 'failed') {
  if (!id || taskActionIds.value.has(id)) return
  taskActionIds.value = new Set(taskActionIds.value).add(id)
  try {
    await taskApi.retry(id, scope)
    void fetchTasks()
    toast({ type: 'success', title: scope === 'failed' ? '已创建续发任务(仅失败目标)' : '任务已重试' })
  } catch (e: unknown) {
    toast({ type: 'error', title: '重试失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  } finally {
    const next = new Set(taskActionIds.value)
    next.delete(id)
    taskActionIds.value = next
  }
}

function statusClass(s: string) {
  const map: Record<string, string> = {
    running: 'bg-[color:var(--t-brand-dim)] text-[var(--t-brand)]', queued: 'bg-[color:var(--t-warn-dim)] text-[var(--t-warn)]',
    completed: 'bg-[color:var(--t-ok-dim)] text-[var(--t-ok)]', failed: 'bg-[color:var(--t-err-dim)] text-[var(--t-err)]',
    cancelled: 'bg-text-muted/15 text-text-muted', paused: 'bg-text-muted/15 text-text-muted',
    pending: 'bg-[color:var(--t-warn-dim)] text-[var(--t-warn)]'
  }
  return map[s] || 'bg-text-muted/15 text-text-muted'
}

function statusLabel(s: string) {
  const map: Record<string, string> = {
    running: '运行中', queued: '排队中', completed: '已完成',
    failed: '失败', cancelled: '已取消', paused: '已暂停', pending: '待执行'
  }
  return map[s] || s
}

function taskTypeLabel(t: string) {
  const map: Record<string, string> = {
    broadcast: '群发消息', join_groups: '加群任务', join: '加群任务',
    scrape_members: '成员采集', scrape: '成员采集', scrape_links: '链接采集',
    scheduled_broadcast: '定时广播', monitor_scrape: '监控采集', invite_members: '邀请入群'
  }
  return map[t] || t
}

function progressBarClass(s: string) {
  return s === 'failed' ? 'bg-[var(--t-err)]' : s === 'completed' ? 'bg-[var(--t-ok)]' : 'bg-[var(--t-brand)]'
}


// 电脑端表格列定义
const pullRefreshing = ref(false)
async function onPullRefresh() {
  await fetchTasks()
  pullRefreshing.value = false
}

const tableCols = [
  { colKey: 'task_number', title: '编号', width: 110 },
  { colKey: 'status', title: '状态', width: 100 },
  { colKey: 'type', title: '类型', width: 120 },
  { colKey: 'overview', title: '消息内容', width: 200 },
  { colKey: 'targets', title: '目标数', width: 80 },
  { colKey: 'progress', title: '投放明细', width: 260 },
  { colKey: 'created_at', title: '创建时间', width: 170 },
  { colKey: 'op', title: '操作', width: 240 },
]

// 消息概览: 群发类任务取 payload.message 前 24 字
function msgPreview(t: any): string {
  const m = t?.payload?.message || t?.payload?.text || ''
  return typeof m === 'string' ? (m.length > 24 ? m.slice(0, 24) + '…' : m) : ''
}

function donePct(p: any) {
  const { total, done } = progressOf(p)
  return total ? Math.round((done / total) * 100) : 0
}

function failedPct(p: any) {
  const { total, failed } = progressOf(p)
  return total ? Math.round((failed / total) * 100) : 0
}

function progressOf(p: any) {
  if (typeof p === 'string') { try { p = JSON.parse(p) } catch { p = {} } }
  return { total: p?.total || 0, done: p?.done || 0, failed: p?.failed || 0 }
}

// R128 紧凑时间: 原 toLocaleString 全格式("2026/10/24 06:33:03")在任务卡占两行;
// 同年省年份, 一律秒级省略 → "10/24 06:33"
function formatDate(d: string) {
  const dt = new Date(d)
  const now = new Date()
  const sameYear = dt.getFullYear() === now.getFullYear()
  const mm = String(dt.getMonth() + 1).padStart(2, '0')
  const dd = String(dt.getDate()).padStart(2, '0')
  const hh = String(dt.getHours()).padStart(2, '0')
  const mi = String(dt.getMinutes()).padStart(2, '0')
  return `${sameYear ? '' : dt.getFullYear() + '/'}${mm}/${dd} ${hh}:${mi}`
}

// 等待/运行时长: 人性化显示 分/秒
function elapsed(iso: string): string {
  const s = Math.max(0, Math.floor((Date.now() - new Date(iso).getTime()) / 1000))
  if (s < 60) return `${s} 秒`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m} 分钟`
  return `${Math.floor(m / 60)} 小时 ${m % 60} 分`
}

watch(() => route.params.accountId, () => {
  // 切换执行账号路由时立即清除旧列表并使旧请求失效，避免切换期间短暂展示上一账号数据。
  taskListRequestId++
  taskListController?.abort()
  tasks.value = []
  total.value = 0
  loading.value = false
  loadError.value = ''
  if (page.value !== 1) page.value = 1
  else void fetchTasks()
}, { immediate: true })

// 15 秒轮询兜底 + SSE 亚秒级: 进度事件即时更新对应行, 状态事件即时刷新
// R118: 等待倒计时 —— task_wait 帧(每秒)驱动"当前动作"显示
const waitInfo = ref<Record<string, { account_id: string; remaining: number; next_index: number; total: number; _t: number }>>({})
function pruneWaitInfo() {
  const now = Date.now()
  for (const [id, info] of Object.entries(waitInfo.value)) {
    if (now - info._t > 5000) { const copy = { ...waitInfo.value }; delete copy[id]; waitInfo.value = copy }
  }
}
const waitInfoTimer = window.setInterval(pruneWaitInfo, 3000)
onUnmounted(() => window.clearInterval(waitInfoTimer))
const accNameById = (id: string) => {
  const a = (accountStore.accounts || []).find((x: any) => String(x.account_id) === String(id))
  return a ? (a.first_name || a.remark || a.phone || id.slice(0, 6)) : id.slice(0, 6)
}

useAutoRefresh(() => { void fetchTasks(); void fetchImportJobs() }, 15000)
void fetchImportJobs()
const importJobsTimer = window.setInterval(fetchImportJobs, 5000)
onUnmounted(() => {
  window.clearInterval(importJobsTimer)
  importJobsRequestId++
  importJobsController?.abort()
  taskListController?.abort()
})
useTaskEvents((evt: any) => {
  if (evt.type === 'task_wait') {
    waitInfo.value = { ...waitInfo.value, [String(evt.task_id)]: { account_id: String(evt.account_id || ''), remaining: Number(evt.remaining || 0), next_index: Number(evt.next_index || 0), total: Number(evt.total || 0), _t: Date.now() } }
    return
  }
  if (evt.type === 'task_status') {
    if (evt.status !== 'running' && waitInfo.value[evt.task_id]) { const copy = { ...waitInfo.value }; delete copy[evt.task_id]; waitInfo.value = copy }
    fetchTasks(); return
  }
  if (evt.progress && evt.task_id) {
    const t = tasks.value.find((x: any) => x.task_id === evt.task_id)
    if (t) t.progress = evt.progress
  }
})
watch(page, fetchTasks)
</script>
