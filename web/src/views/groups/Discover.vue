<template>
  <div class="space-y-4">
    <!-- 页头 -->
    <div class="flex items-center justify-between flex-wrap gap-3">
      <div>
        <h2 class="text-lg font-bold text-text-primary">发现群组</h2>
        <p class="text-xs text-text-secondary mt-1">
          所有租户共同维护的 Telegram 群组库；可贡献链接完善资料，也可选择群组创建加群任务。
        </p>
      </div>
      <div class="flex items-center gap-2">
        <span class="text-xs text-text-secondary">共 {{ total }} 条 · 已跨页选中 {{ selected.size }} 条</span>
        <t-button size="small" variant="outline" :disabled="!aggregatedList.length" @click="toggleCurrentPage">{{ currentPageAllSelected ? '取消本页全选' : '本页全选' }}</t-button>
        <t-button v-if="selected.size > 0" size="small" variant="text" @click="selected.clear()" title="清空所有勾选(含其他页)">
          清空
        </t-button>
        <t-button size="small" variant="outline" @click="importModalVisible = true" title="批量导入 t.me 链接/@用户名, 自动解析群名">
          导入
        </t-button>
        <t-button theme="primary" :disabled="selected.size === 0 || joining" @click="joinModalVisible = true">
          <CheckSquare class="w-4 h-4 mr-1" />
          加入所选 ({{ selected.size }})
        </t-button>
      </div>
    </div>

    <!-- R128 统一示例2 seg 语言(与进群邀请/链接采集同款) -->
    <div class="flex rounded-[var(--t-r-lg)] p-1 gap-1 w-fit" style="background: color-mix(in srgb, var(--t-text-1) 8%, transparent);">
      <button class="px-4 py-2.5 rounded-[14px] text-sm font-bold transition-all"
        :style="tab === 'library' ? 'background: var(--t-surface-2); color: var(--t-text-1); box-shadow: 0 2px 8px rgba(0,0,0,0.18);' : 'color: var(--t-text-3);'"
        @click="tab = 'library'">雷达群库</button>
      <button class="px-4 py-2.5 rounded-[14px] text-sm font-bold transition-all"
        :style="tab === 'diy' ? 'background: var(--t-surface-2); color: var(--t-text-1); box-shadow: 0 2px 8px rgba(0,0,0,0.18);' : 'color: var(--t-text-3);'"
        @click="tab = 'diy'">定向采集</button>
    </div>

    <!-- 系统群库 -->
    <template v-if="tab === 'library'">
      <!-- 筛选条 -->
      <div class="glass rounded-[var(--t-r-md)] p-3 flex flex-wrap items-center gap-2 lift">
        <div class="relative">
          <Search class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
          <input v-model="filters.keyword" class="pl-9 pr-3 py-2 rounded-lg glow-input text-sm w-full sm:w-56 bg-bg-card" placeholder="群名称 / 用户名 / 链接" @keyup.enter="search" />
        </div>
        <select v-model="filters.gtype" class="px-3 py-2 rounded-lg text-sm bg-bg-card" @change="search">
          <option value="">全部类型</option>
          <option value="public">公开群</option>
          <option value="private">私密群</option>
          <option value="channel">频道</option>
        </select>
        <select v-model="filters.status" class="px-3 py-2 rounded-lg text-sm bg-bg-card" @change="search">
          <option value="">全部状态</option>
          <option value="active">可用链接 / 待验证</option>
                    <option value="invalid">已失效</option>
        </select>
        <select v-model="filters.source" class="px-3 py-2 rounded-lg text-sm bg-bg-card" @change="search">
          <option value="">全部来源</option>
          <option value="scraped">链接采集</option>
                    <option value="bio">简介里发现</option>
        </select>
        <!-- 本地群库过滤: 回车/筛选走 keyword ILIKE 后端过滤(群库内搜索); TG 全网搜独立入口 -->
        <t-button size="small" theme="primary" @click="search">群库内搜索</t-button>
        <t-button size="small" variant="outline" :disabled="!filters.keyword.trim() || kwSearching" @click="doSearch" title="借在线账号在 Telegram 全网搜索, 结果入群库">
          <Loader2 v-if="kwSearching" class="w-3.5 h-3.5 animate-spin mr-1" />
          TG全网搜
        </t-button>
        <span class="flex-1" />
        <t-button size="small" variant="text" @click="load">刷新</t-button>
      </div>

      <!-- 群列表: 窄屏可横向滑动, 不再挤压变形 -->
      <div class="glass rounded-[var(--t-r-md)] overflow-x-auto lift">
        <!-- R87-M10: 表格横向滚动(移动端不再挤压截断), 卡片化改造留待专项批 -->
        <!-- R98 移动端卡片化: 小屏用卡片列表替代表格(不再挤压截断), 桌面仍用表格 -->
        <div class="md:hidden space-y-2">
          <div v-for="row in aggregatedList" :key="row.sg_id"
               class="glass rounded-[var(--t-r-md)] p-3 flex items-center gap-3">
            <input type="checkbox" :checked="selected.has(row.sg_id)" @change="toggle(row.sg_id)"
                   class="w-4 h-4 accent-[var(--t-brand)] shrink-0" />
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium text-text-primary truncate">{{ row.title }}</span>
                <span class="text-[10px] px-1.5 py-0.5 rounded whitespace-nowrap shrink-0"
                      :class="row.gtype === 'private' ? 'bg-amber/10 text-amber' : row.gtype === 'channel' ? 'bg-cyan-glow/40 text-cyan' : 'bg-emerald/10 text-emerald'">
                  {{ row.gtype === 'private' ? '私密群' : row.gtype === 'channel' ? '频道' : '公开群' }}
                </span>
              </div>
              <div class="text-[10.5px] text-text-muted truncate mt-0.5">{{ row.tme_link || row.username }}</div>
              <div class="text-[10.5px] text-text-muted mt-0.5">{{ Number(row.member_count).toLocaleString() }} 成员</div>
              <div v-if="row.contributor_name" class="text-[10.5px] text-text-muted mt-0.5">贡献者：{{ row.contributor_name }}</div>
            </div>
            <span v-if="row.source" class="text-[10px] text-text-muted/70 shrink-0 whitespace-nowrap">{{ sourceLabel(row.source) }}</span>
          </div>
          <div v-if="!aggregatedList.length && !loading" class="text-center text-xs text-text-muted py-8">暂无数据</div>
        </div>
        <t-table class="hidden md:block" :data="aggregatedList" :columns="tableCols" row-key="sg_id" :loading="loading" size="small" hover table-layout="auto" :scroll="{ type: 'scroll', width: 720 }">
          <template #checked="{ row }">
            <t-checkbox :checked="selected.has(row.sg_id)" @change="toggle(row.sg_id)" />
          </template>
          <template #title="{ row }">
            <div class="min-w-0">
              <div class="text-sm text-text-primary truncate">
                {{ row.title || row.tme_link || '—' }}
                <!-- 同群聚合徽标: 多入口时可点击展开全部链接 -->
                <span v-if="(row as any).entry_count > 1"
                      class="ml-1.5 px-1.5 py-0.5 rounded bg-cyan-glow/30 text-cyan text-[10px] whitespace-nowrap align-middle cursor-pointer hover:bg-cyan-glow/50"
                      :title="isExpanded(row.sg_id) ? '收起' : '展开全部入口链接'"
                      @click.stop="toggleExpand(row.sg_id)">
                  同群 {{ (row as any).entry_count }} 入口 {{ isExpanded(row.sg_id) ? '▲' : '▼' }}
                </span>
              </div>
              <div class="text-xs text-text-muted truncate">{{ row.tme_link || ('@' + (row.username || '')) }}</div>
              <!-- 展开: 该群全部入口链接(同群多链接场景) -->
              <div v-if="isExpanded(row.sg_id) && (row as any).entries?.length" class="mt-1 space-y-0.5 border-l-2 border-cyan-glow/40 pl-2">
                <div v-for="e in (row as any).entries" :key="e.link_id || e.tme_link || e.sg_id" class="flex items-center gap-2 text-xs min-w-0">
                  <a v-if="e.tme_link" :href="e.tme_link" target="_blank" rel="noopener"
                     class="text-cyan hover:underline truncate">{{ e.tme_link }}</a>
                  <span v-else class="text-text-muted truncate">{{ '@' + (e.username || '') }}</span>
                  <span class="px-1 rounded bg-cyan-glow/20 text-cyan text-[10px] whitespace-nowrap">{{ sourceLabel(e.source) }}</span>
                  <span :class="e.status === 'invalid' ? 'text-text-muted line-through' : 'text-cyan'" class="whitespace-nowrap">{{ statusLabel(e.status) }}</span>
                  <button class="text-text-muted hover:text-text-primary whitespace-nowrap" title="复制链接" @click.stop="copyLink(e.tme_link || ('https://t.me/' + (e.username || '')))">⧉</button>
                </div>
              </div>
            </div>
          </template>
          <template #gtype="{ row }">
            <span class="px-1.5 py-0.5 rounded text-[11px] whitespace-nowrap"
              :class="row.gtype === 'private' ? 'bg-amber/10 text-amber' : row.gtype === 'channel' ? 'bg-cyan-glow/40 text-cyan' : 'bg-emerald/10 text-emerald'">
              {{ gtypeLabel(row.gtype) }}
            </span>
          </template>
          <!-- R87-M8: "已加入"状态列已删除 —— 切账号后状态不跟随该视角, 对租户是误导信息;
               未加入的群在标题行内已有"可加入"入口标识 -->
          <template #source="{ row }">
            <span class="text-xs text-text-secondary whitespace-nowrap">{{ sourceLabel(row.source) }}</span>
          </template>
          <template #contributor_name="{ row }">
            <span class="text-xs text-text-secondary">{{ row.contributor_name || "—" }}</span>
          </template>
          <template #last_seen_at="{ row }">
            <span class="text-xs text-text-muted whitespace-nowrap">{{ fmtTime(row.last_seen_at) }}</span>
          </template>
          <template #empty>
            <div class="empty-pretty">
              <div class="ep-icon">
                <svg class="w-7 h-7" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <circle cx="11" cy="11" r="7" /><path d="m21 21-4.3-4.3" stroke-linecap="round" />
                </svg>
              </div>
              <p class="text-sm">群库还是空的</p>
              <p class="text-xs">导入公开链接或邀请链接，帮助所有租户共同完善群组资料</p>
            </div>
          </template>
        </t-table>

        <!-- 分页 -->
        <div class="flex items-center justify-between px-4 py-3 border-t border-border">
          <span class="text-xs text-text-secondary">第 {{ page }} 页 / 共 {{ totalPages }} 页</span>
          <div class="flex gap-2">
            <t-button size="small" variant="outline" :disabled="page <= 1" @click="prevPage">上一页</t-button>
            <t-button size="small" variant="outline" :disabled="page >= totalPages" @click="nextPage">下一页</t-button>
          </div>
        </div>
      </div>
    </template>

    <!-- 自助采集: 引导到工作区链接采集 -->
    <div v-else class="glass rounded-[var(--t-r-md)] p-8 text-center space-y-3 lift">
      <Radar class="w-10 h-10 mx-auto text-cyan" />
      <div class="text-text-primary font-medium">自助采集</div>
      <p class="text-sm text-text-secondary max-w-md mx-auto">
        用你自己的账号做定向采集：监听指定群/频道里的 t.me 链接、挖成员简介，采集结果自动汇入系统群库。
      </p>
      <t-button theme="primary" class="whitespace-nowrap" @click="goDiy">去工作区采集</t-button>
    </div>

    <!-- 批量导入弹层: 粘贴链接文本, 自动解析群名/成员数入库 -->
    <Modal v-model="importModalVisible" title="导入群链接" class="max-w-xl">
      <div class="space-y-3 pb-2">
        <label class="block text-xs font-medium text-text-secondary">选择 TXT / CSV 文件（最大 20MB）</label>
        <input type="file" accept=".txt,.csv,text/plain,text/csv" class="w-full text-xs"
          @change="onImportFile" />
        <p v-if="importFileName" class="text-xs text-text-secondary">已选择：{{ importFileName }}</p>
        <div class="text-xs text-text-muted">也可以直接粘贴链接，每行一条或使用逗号分隔。</div>
        <textarea v-model="importText" rows="8" class="w-full px-3 py-2 rounded-lg glow-input text-sm resize-y"
          placeholder="t.me/群用户名&#10;@群用户名&#10;https://t.me/+邀请hash"></textarea>
        <p class="text-[11px] leading-relaxed" style="color: var(--t-text-3);">
          提交后由后台分批处理；只会完善公共群组库，不会让任何账号加入群组。
          同一批任务可处理数万条链接，文件大小上限 20MB。
        </p>
        <div v-if="importJob" class="text-xs rounded-lg p-3 space-y-2" style="background: color-mix(in srgb, var(--t-text-1) 6%, transparent);">
          <div class="flex justify-between gap-2">
            <span class="font-medium">{{ importJob.status === 'completed' ? '导入完成' : importJob.status === 'failed' ? '导入失败' : '后台处理中' }}</span>
            <span>{{ importJob.processed_count }} / {{ importJob.total_count }}</span>
          </div>
          <div class="h-2 rounded-full overflow-hidden" style="background: var(--t-line-light);">
            <div class="h-full rounded-full" style="background: var(--t-brand);"
              :style="{ width: (importJob.total_count ? Math.min(100, importJob.processed_count / importJob.total_count * 100) : 0) + '%' }"></div>
          </div>
          <div class="text-text-muted">入库 {{ importJob.imported_count }} · 失败 {{ importJob.failed_count }} · 跳过 {{ importJob.skipped_count }}</div>
          <div v-if="importJob.last_error" class="text-rose break-words">{{ importJob.last_error }}</div>
        </div>
      </div>
      <template #footer>
        <t-button size="small" variant="text" @click="importModalVisible = false">关闭</t-button>
        <t-button size="small" theme="primary" :loading="importing" @click="doImport">开始导入</t-button>
      </template>
    </Modal>

    <!-- 选账号弹层(子弹层, 保持传统模态) -->
    <t-dialog v-model:visible="joinModalVisible" header="选择执行账号" :footer="false" width="480px">
      <div class="space-y-3">
        <p class="text-xs text-text-secondary">将对每个勾选的在线账号创建一个加群任务（每任务上限 100 个群）。</p>
        <div class="max-h-64 overflow-y-auto space-y-2">
          <label v-for="acc in joinableAccounts" :key="acc.account_id"
            class="flex items-center gap-3 p-2.5 rounded-lg bg-bg-card hover:bg-bg-hover cursor-pointer">
            <t-checkbox :checked="joinAccounts.includes(acc.account_id)" @change="toggleJoinAccount(acc.account_id)" />
            <span class="text-sm text-text-primary">{{ acc.remark || acc.first_name || acc.phone }}</span>
            <span class="text-xs text-text-muted">{{ acc.phone }}</span>
          </label>
          <p v-if="joinableAccounts.length === 0" class="text-xs text-text-muted text-center py-4">没有在线账号, 先去账号管理启动一个。</p>
        </div>
        <!-- 分批与节奏控制: 每账号分配数量 + 自定义加群间隔 -->
        <div class="grid grid-cols-2 gap-2 rounded-lg p-2.5" style="background: color-mix(in srgb, var(--t-text-1) 5%, transparent);">
          <label class="col-span-2 flex items-center gap-2 text-xs text-text-secondary cursor-pointer select-none">
            <input type="checkbox" v-model="joinBatchEnabled" class="accent-emerald" />
            分批加入（每个账号只加分配到的群，默认每个账号全量加一遍）
          </label>
          <div v-if="joinBatchEnabled" class="col-span-2 flex items-center gap-2 text-xs text-text-secondary">
            每个账号分配
            <input type="number" v-model.number="joinPerAccount" min="1" max="100" class="w-20 px-2 py-1.5 rounded-lg glow-input text-center" />
            个群（共 {{ selected.size }} 个群 / {{ joinAccounts.length || 0 }} 个账号）
          </div>
          <label class="col-span-2 text-xs text-text-secondary">
            加群间隔（分钟，区间内随机；撞 TG 限流会按官方要求自动冷却等待）
          </label>
          <input type="number" v-model.number="joinIntervalMin" min="0" step="1" placeholder="最小间隔" class="px-2 py-1.5 rounded-lg glow-input text-sm" />
          <input type="number" v-model.number="joinIntervalMax" min="0" step="1" placeholder="最大间隔" class="px-2 py-1.5 rounded-lg glow-input text-sm" />
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <t-button variant="outline" @click="joinModalVisible = false">取消</t-button>
          <t-button theme="primary" :disabled="joinAccounts.length === 0 || joining" @click="doJoin">
            <Loader2 v-if="joining" class="w-4 h-4 animate-spin" />
            开始加入
          </t-button>
        </div>
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Search, CheckSquare, Radar, Loader2 } from 'lucide-vue-next'
import { getGroupLibrary, joinLibraryGroups, searchLibraryGroups, importLibraryGroups, getGroupLibraryImportJob } from '@/api/groups'
import { useAccountStore } from '@/stores/account'
import { toast } from '@/utils/toast'
import Modal from '@/components/ui/Modal.vue'
import type { SystemGroup } from '@/types/models'

const router = useRouter()
const accountStore = useAccountStore()

const tab = ref<'library' | 'diy'>('library')
const list = ref<SystemGroup[]>([])

// ===== 同群聚合(2026-10-21): 同一群多个邀请链接合并为一行 =====
// 现象: "汤臣社区-核销"等群被采集到 6+ 条不同邀请链接, 列表全是重复行。
// 规则: 按群标题合并, 成员数取最大(入口数不同采集值不同), 已加入任一入口视为已加入,
//       勾选/加入用首个入口的 sg_id, 徽标显示入口数。
const aggregatedList = computed<SystemGroup[]>(() => list.value.map(g => {
  const entries = g.entries?.length ? g.entries : [{
    sg_id: g.sg_id,
    tme_link: g.tme_link,
    username: g.username,
    status: g.status,
    source: g.source,
    member_count: g.member_count,
  }]
  return { ...g, entries, entry_count: g.entry_count || entries.length }
}))

// 同群多入口的展开状态(用户反馈: 徽标只能看数量,看不到其余链接)
const expandedSg = ref(new Set<string>())
function toggleExpand(id: string) {
  const s = new Set(expandedSg.value)
  if (s.has(id)) s.delete(id); else s.add(id)
  expandedSg.value = s
}
const isExpanded = (id: number) => expandedSg.value.has(id)
function copyLink(link: string) {
  if (!link) return
  navigator.clipboard?.writeText(link).catch(() => {})
}
const total = ref(0)
const page = ref(1)
const pageSize = 20
// 翻页必须走函数: 模板里 page++ 对 ref 无效(编译成 $setup.page++ 作用在对象上), page 永远=1 所以每页都是第一页内容
function nextPage() { page.value++; load() }
function prevPage() { if (page.value > 1) { page.value--; load() } }
const loading = ref(false)
const selected = reactive(new Set<string>())
const filters = reactive({ keyword: '', gtype: '', status: '', source: '' })

const joinModalVisible = ref(false)
const joinAccounts = ref<string[]>([])
const joining = ref(false)
// 分批/间隔表单状态
const joinBatchEnabled = ref(false)
const joinPerAccount = ref(10)
const joinIntervalMin = ref<number | null>(null)
const joinIntervalMax = ref<number | null>(null)
// 批量导入弹层状态
const importModalVisible = ref(false)
const importText = ref('')
const importing = ref(false)
const importFileName = ref('')
const importJob = ref<any>(null)

async function onImportFile(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (file.size > 20 * 1024 * 1024) {
    toast({ type: 'error', message: '文件不能超过 20MB' })
    ;(event.target as HTMLInputElement).value = ''
    return
  }
  try {
    importText.value = await file.text()
    importFileName.value = file.name
  } catch {
    toast({ type: 'error', message: '文件读取失败，请检查文件编码' })
  }
}

async function pollImportJob(jobId: string) {
  try {
    const job = await getGroupLibraryImportJob(jobId)
    importJob.value = job
    if (job.status === 'completed' || job.status === 'failed') {
      load()
      toast({ type: job.status === 'completed' ? 'success' : 'warning',
        message: `导入结束：入库 ${job.imported_count}，失败 ${job.failed_count}，跳过 ${job.skipped_count}` })
      return
    }
  } catch {
    // 网络抖动时继续轮询，任务本身由服务端持久化。
  }
  window.setTimeout(() => pollImportJob(jobId), 2000)
}

async function doImport() {
  if (importing.value || !importText.value.trim()) return
  importing.value = true
  importJob.value = null
  try {
    const job = await importLibraryGroups(importText.value, importFileName.value)
    importJob.value = job
    toast({ type: 'success', message: `已提交后台任务，共 ${job.total_count} 条` })
    pollImportJob(job.job_id)
  } catch (err: any) {
    toast({ type: 'error', message: err?.message || '提交导入任务失败' })
  } finally {
    importing.value = false
  }
}
// 关键词搜群(用筛选条的关键字输入框, 借本租户在线账号做 TG 全局搜索)
const kwSearching = ref(false)

async function doSearch() {
  if (!filters.keyword.trim() || kwSearching.value) return
  kwSearching.value = true
  try {
    const res = await searchLibraryGroups(filters.keyword.trim())
    if (res.count > 0) {
      toast({ type: 'success', message: `搜到 ${res.count} 个群, 已放入群组大厅` })
      load()
    } else {
      toast({ type: 'info', message: '没搜到相关群组, 换个关键词试试' })
    }
  } catch (err: any) {
    toast({ type: 'error', message: err?.message || '搜索失败' })
  } finally {
    kwSearching.value = false
  }
}

const tableCols = [
  { colKey: 'checked', title: '', width: 44 },
  { colKey: 'title', title: '群组', ellipsis: true },
  { colKey: 'gtype', title: '类型', width: 80 },
  { colKey: 'member_count', title: '成员', width: 70 },
  // R87-M8: 删除"状态"列(已加入标识对租户是误导, 切账号后不跟随)
  { colKey: 'source', title: '来源', width: 110 },
  { colKey: 'contributor_name', title: '最早贡献者', width: 120 },
  { colKey: 'last_seen_at', title: '最近收录', width: 132 },
]

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const joinableAccounts = computed(() =>
  (accountStore.phoneAccounts || []).filter((a: any) => a.status === 'online')
)

function gtypeLabel(t: string) {
  return t === 'private' ? '私密群' : t === 'channel' ? '频道' : t === 'public' ? '公开群' : '未知'
}
function statusLabel(s: string) {
  return s === 'valid' ? '链接有效' : s === 'invalid' ? '链接无效' : '待验证'
}
function sourceLabel(s: string) {
  return s === 'bio' ? '简介发现' : s === 'import' ? '手动/文件导入' : s === 'search' ? 'Telegram 搜索' : '链接采集'
}
function fmtTime(t: string) {
  if (!t) return '—'
  const d = new Date(t)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getMonth() + 1}/${d.getDate()} ${p(d.getHours())}:${p(d.getMinutes())}`
}

async function load() {
  loading.value = true
  try {
    const res = await getGroupLibrary({ ...filters, page: page.value, page_size: pageSize })
    list.value = res.list || []
    total.value = res.total || 0
    // 跨页勾选保留: 翻页/加载不再清空已选(分批加群需要跨页凑量);
    // 已失效条目保留勾选无意义, 顺手清掉
    for (const id of [...selected]) {
      const g = list.value.find((x) => x.sg_id === id)
      if (g && g.status === 'invalid') selected.delete(id)
    }
  } catch (err: any) {
    toast({ type: 'error', message: err?.message || '加载群库失败' })
  } finally {
    loading.value = false
  }
}
function search() {
  page.value = 1
  load()
}

const currentPageIds = computed(() => aggregatedList.value.map(g => g.sg_id))
const currentPageAllSelected = computed(() => currentPageIds.value.length > 0 && currentPageIds.value.every(id => selected.has(id)))
function toggleCurrentPage() {
  if (currentPageAllSelected.value) currentPageIds.value.forEach(id => selected.delete(id))
  else currentPageIds.value.forEach(id => selected.add(id))
}
function toggle(id: string) {
  selected.has(id) ? selected.delete(id) : selected.add(id)
}
function toggleJoinAccount(id: string) {
  const i = joinAccounts.value.indexOf(id)
  i >= 0 ? joinAccounts.value.splice(i, 1) : joinAccounts.value.push(id)
}

async function doJoin() {
  if (joining.value) return
  joining.value = true
  try {
    const res = await joinLibraryGroups([...joinAccounts.value], [...selected], {
      perAccountCount: joinBatchEnabled.value ? (joinPerAccount.value || 0) : 0,
      intervalMinMinutes: joinIntervalMin.value || 0,
      intervalMaxMinutes: joinIntervalMax.value || 0,
    })
    toast({ type: 'success', message: `已创建 ${res.created} 个加群任务（${res.links} 个群）${res.failed_err ? '；部分失败: ' + res.failed_err : ''}` })
    joinModalVisible.value = false
    selected.clear()
  } catch (err: any) {
    toast({ type: 'error', message: err?.message || '创建加群任务失败' })
  } finally {
    joining.value = false
  }
}

function goDiy() {
  const acc = accountStore.currentAccount
  if (acc?.account_id) router.push(`/workspace/${acc.account_id}/scrape-links`)
  else router.push('/accounts')
}

onMounted(() => {
  load()
  if (!accountStore.accounts?.length) accountStore.fetchAccounts(true)
})
</script>
