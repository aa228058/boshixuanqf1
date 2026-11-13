<template>
  <main class="space-y-4">
    <Breadcrumb :items="[{ path: '/admin/dashboard', label: '仪表盘' }, { path: '', label: '卡密管理' }]" />
    <div class="flex items-center gap-2 overflow-x-auto pb-1">
      <t-button theme="primary" size="sm" class="shrink-0 ui-t-button ui-t-button--primary ui-t-button--sm" @click="showGenerate = true"> 生成卡密
      </t-button>
      <t-select v-model="statusFilter" :options="[
        { label: '全部状态', value: '' },
        { label: '未使用', value: 'unused' },
        { label: '已使用', value: 'used' }
      ]" class="w-24 shrink-0" />
      <t-button theme="primary" size="sm" variant="outline" class="shrink-0 ui-t-button ui-t-button--secondary ui-t-button--sm" :loading="exportingCodes" :disabled="exportingCodes" @click="exportCodes">
        {{ exportingCodes ? '导出中…' : '导出' }}
      </t-button>
      <t-button v-if="selectedIds.length" size="sm" variant="outline" class="shrink-0 ui-t-button ui-t-button--secondary ui-t-button--sm" @click="copySelected">
        复制选中({{ selectedIds.length }})
      </t-button>
    </div>

    <!-- 桌面端表格 -->
    <div class="hidden sm:block glass rounded-[var(--t-r-md)] overflow-hidden">
      <t-table
        row-key="code_id"
        :columns="tdColumns"
        :data="codes"
        hover
        :pagination="{ current: page, pageSize: 20, total: total, onChange: onPageChange }"
      >
        <template #select="{ row }">
          <input type="checkbox" :value="row.code_id" v-model="selectedIds" class="accent-[var(--t-brand)]" @click.stop />
        </template>
        <template #used_by="{ row }">
          <span v-if="row.used_by_tenant_name" class="text-sm">{{ row.used_by_tenant_name }}</span>
          <span v-else-if="row.used_by" class="text-sm text-text-secondary">{{ row.used_by.slice(0, 8) }}...</span>
          <span v-else class="text-text-secondary">-</span>
        </template>
        <template #code="{ row }">
          <span class="font-mono text-sm">{{ row.code }}</span>
        </template>
        <template #plan_id="{ row }">
          <span class="text-sm">{{ planLabel(row.plan_id as string) }}</span>
        </template>
        <template #created_at="{ row }">
          <span class="text-sm">{{ formatValidDate(row.created_at || '') }}</span>
        </template>
        <template #status="{ row }">
          <span class="text-xs px-2 py-0.5 rounded-full" :class="codeStatusClass(row.status as string)">
            {{ row.status === 'unused' ? '未使用' : row.status === 'used' ? '已使用' : row.status === 'expired' ? '已过期' : '已作废' }}
          </span>
        </template>
        <template #actions="{ row }">
          <t-button theme="primary" variant="text" size="sm" class="ui-t-button ui-t-button--text ui-t-button--sm" @click="copyCode(row.code as string)">复制</t-button>
          <t-button theme="primary" variant="text" size="sm" class="ui-t-button ui-t-button--text ui-t-button--sm" @click="viewCode(row as any)">查看</t-button>
          <t-button v-if="row.status === 'unused'" theme="default" variant="outline" size="sm" class="ui-t-button ui-t-button--secondary ui-t-button--sm" @click="revokeCode(row as any)" :disabled="revokingCodeIds.includes(row.code_id as string)">作废</t-button>
          <t-button theme="danger" size="sm" class="ui-t-button ui-t-button--danger ui-t-button--sm" @click="delCode(row as any)" :disabled="deletingCodeIds.includes(row.code_id as string)">删除</t-button>
        </template>
      </t-table>
    </div>
    <!-- 移动端卡片 -->
    <div class="sm:hidden space-y-3">
      <div
        v-for="c in codes"
        :key="c.code_id"
        class="ui-card p-4 space-y-2 lift"
      >
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex items-center gap-2 min-w-0">
            <input type="checkbox" :value="c.code_id" v-model="selectedIds" class="accent-[var(--t-brand)] shrink-0" />
            <span class="font-mono text-sm truncate">{{ c.code }}</span>
          </div>
          <span class="text-xs px-2 py-0.5 rounded-full shrink-0" :class="codeStatusClass(c.status)">
            {{ c.status === 'unused' ? '未使用' : c.status === 'used' ? '已使用' : c.status === 'expired' ? '已过期' : '已作废' }}
          </span>
        </div>
        <div class="flex items-center gap-3 text-xs text-text-secondary">
          <span>{{ planLabel(c.plan_id) }}</span>
          <span v-if="c.used_by_tenant_name">使用租户: {{ c.used_by_tenant_name }}</span>
          <span v-else-if="c.used_by">使用者: {{ c.used_by.slice(0, 8) }}...</span>
        </div>
        <div class="flex items-center justify-between text-xs text-text-secondary">
          <span>{{ formatValidDate(c.created_at || '') }}</span>
          <div>
            <t-button theme="primary" variant="text" size="sm" class="ui-t-button ui-t-button--text ui-t-button--sm" @click="copyCode(c.code)">复制</t-button>
            <t-button theme="primary" variant="text" size="sm" class="ui-t-button ui-t-button--text ui-t-button--sm" @click="viewCode(c)">查看</t-button>
            <t-button v-if="c.status === 'unused'" theme="default" variant="outline" size="sm" class="ui-t-button ui-t-button--secondary ui-t-button--sm" @click="revokeCode(c)" :disabled="revokingCodeIds.includes(c.code_id)">作废</t-button>
            <t-button theme="danger" size="sm" class="ui-t-button ui-t-button--danger ui-t-button--sm" @click="delCode(c)" :disabled="deletingCodeIds.includes(c.code_id)">删除</t-button>
          </div>
        </div>
      </div>
      <EmptyState v-if="!codes.length" icon="Key" title="暂无卡密" description="点击上方按钮生成卡密" />
      <div v-if="total > 20" class="flex items-center justify-between gap-3 pt-1">
        <span class="text-xs text-text-muted">共 {{ total }} 条 · 第 {{ page }} / {{ Math.ceil(total / 20) }} 页</span>
        <div class="flex items-center gap-2">
          <t-button size="sm" variant="outline" class="ui-t-button ui-t-button--secondary ui-t-button--sm" :disabled="page <= 1" @click="page = page - 1; fetchCodes()">上一页</t-button>
          <t-button size="sm" variant="outline" class="ui-t-button ui-t-button--secondary ui-t-button--sm" :disabled="page >= Math.ceil(total / 20)" @click="page = page + 1; fetchCodes()">下一页</t-button>
        </div>
      </div>
    </div>

    <Modal v-model="showGenerate" title="生成卡密">
      <div class="space-y-4">
        <div>
          <label for="code-plan" class="block text-sm text-text-secondary mb-1.5">套餐</label>
          <t-select v-model="generateForm.plan_id" :options="planOptions" />
        </div>
        <div>
          <label for="code-timing" class="block text-sm text-text-secondary mb-1.5">生效方式</label>
          <t-select v-model="generateForm.timing_type" :options="[
            { label: '立即生效（生成后开始计时）', value: 'immediate' },
            { label: '使用时生效（激活后开始计时）', value: 'on_use' }
          ]" />
        </div>
        <div>
          <label for="code-count" class="block text-sm text-text-secondary mb-1.5">数量</label>
          <input v-model.number="generateForm.count" type="number" min="1" max="100" id="code-count" class="ui-input !min-h-[36px] !px-2.5 !py-1.5 !text-sm" aria-label="生成数量" />
        </div>
        <div>
          <label for="code-days" class="block text-sm text-text-secondary mb-1.5">有效期（天，可选）</label>
          <input v-model.number="generateForm.expires_days" type="number" min="1" id="code-days" class="ui-input !min-h-[36px] !px-2.5 !py-1.5 !text-sm" aria-label="有效期天数" />
        </div>
      </div>
      <template #footer>
        <t-button type="button" theme="default" variant="outline" size="sm" class="ui-t-button ui-t-button--secondary ui-t-button--sm" @click="showGenerate = false">取消</t-button>
        <t-button class="ui-t-button ui-t-button--primary ui-t-button--sm" theme="primary" size="sm" @click="doGenerate" :loading="generatingCodes" :disabled="generatingCodes">生成</t-button>
      </template>
    </Modal>

    <Modal v-model="showView" title="卡密详情">
      <div v-if="viewingCode" class="space-y-3 text-sm">
        <div class="flex justify-between gap-4"><span class="text-text-secondary shrink-0">卡密</span><span class="font-mono break-all text-right">{{ viewingCode.code }}</span></div>
        <div class="flex justify-between"><span class="text-text-secondary">套餐</span><span>{{ planLabel(viewingCode.plan_id) }}</span></div>
        <div class="flex justify-between"><span class="text-text-secondary">状态</span><span>{{ viewingCode.status === 'unused' ? '未使用' : viewingCode.status === 'used' ? '已使用' : viewingCode.status === 'expired' ? '已过期' : '已作废' }}</span></div>
        <div class="flex justify-between"><span class="text-text-secondary">使用租户</span><span>{{ viewingCode.used_by_tenant_name || '-' }}</span></div>
        <div class="flex justify-between"><span class="text-text-secondary">使用时间</span><span>{{ viewingCode.used_at ? formatValidDateTime(viewingCode.used_at) : '—' }}</span></div>
        <div class="flex justify-between"><span class="text-text-secondary">过期时间</span><span>{{ viewingCode.expires_at ? formatValidDateTime(viewingCode.expires_at) : '—' }}</span></div>
        <div class="flex justify-between"><span class="text-text-secondary">生效方式</span><span>{{ viewingCode.timing_type === 'immediate' ? '立即生效' : '使用时生效' }}</span></div>
        <div v-if="viewingCode.notes" class="flex justify-between gap-4"><span class="text-text-secondary shrink-0">备注</span><span class="text-right">{{ viewingCode.notes }}</span></div>
      </div>
      <template #footer>
        <t-button type="button" theme="default" variant="outline" size="sm" class="ui-t-button ui-t-button--secondary ui-t-button--sm" @click="showView = false">关闭</t-button>
      </template>
    </Modal>
  </main>

  <!-- 批量操作吸底栏：选中卡密时弹出 -->
  <Teleport to="body">
    <Transition name="batch-bar">
      <div v-if="selectedIds.length" class="fixed bottom-[calc(92px_+_env(safe-area-inset-bottom,0px))] md:bottom-5 left-1/2 -translate-x-1/2 z-50 flex items-center gap-2 pl-4 pr-2 py-1.5 max-w-[calc(100vw-24px)] overflow-x-auto scrollbar-hide rounded-full glass border border-border elev-pop">
        <span class="text-xs text-text-muted whitespace-nowrap">已选 {{ selectedIds.length }} 张</span>
        <t-button theme="default" variant="outline" size="sm" class="ui-t-button ui-t-button--secondary ui-t-button--sm whitespace-nowrap" @click="batchRevoke" :disabled="batchRevoking || batchDeleting">批量作废</t-button>
        <t-button theme="danger" size="sm" class="ui-t-button ui-t-button--danger ui-t-button--sm whitespace-nowrap" @click="batchDelete" :disabled="batchDeleting || batchRevoking">批量删除</t-button>
        <button type="button" @click="selectedIds = []" class="ui-icon-btn" aria-label="清空已选卡密" title="清空已选卡密">×</button>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { Plus, Key } from 'lucide-vue-next'
import { adminApiMod } from '@/api/admin'
import { toast } from '@/utils/toast'
import { downloadBlob } from '@/utils/export'
import Modal from '@/components/ui/Modal.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import Breadcrumb from '@/components/ui/Breadcrumb.vue'
import type { ActivationCode, Plan } from '@/types'
// R115: 恢复 codeStatusClass(复核脚本误判, 正文实际有使用)
import { codeStatusClass, planLabel } from '@/utils/format'
import { formatValidDate, formatValidDateTime, validateCodeGenerationInput } from '@/utils/adminPageGuards'
import { ask } from '@/utils/confirm'

const codes = ref<ActivationCode[]>([])

// 模板 17/48 行的"复制"按钮此前调用不存在的 copySelected/copyCode（点击无反应）。
async function copyText(text: string, successMessage: string) {
  if (!text) return
  if (!navigator.clipboard?.writeText) {
    toast({ type: 'warning', title: '当前浏览器不支持复制', message: '请手动选择并复制内容' })
    return
  }
  try {
    await navigator.clipboard.writeText(text)
    toast({ type: 'success', title: '已复制', message: successMessage })
  } catch {
    toast({ type: 'error', title: '复制失败', message: '请检查浏览器剪贴板权限后重试' })
  }
}
function copyCode(code: string) {
  void copyText(code, code)
}
function copySelected() {
  const text = codes.value.filter(c => selectedIds.value.includes(c.code_id)).map(c => c.code).join('\n')
  if (!text) return
  void copyText(text, `${selectedIds.value.length} 条卡密`)
}
const plans = ref<Plan[]>([])
const statusFilter = ref('')
const page = ref(1)
const total = ref(0)
const showGenerate = ref(false)
const generatingCodes = ref(false)
const revokingCodeIds = ref<string[]>([])
const deletingCodeIds = ref<string[]>([])
const batchRevoking = ref(false)
const batchDeleting = ref(false)
const generateForm = ref({ plan_id: 'basic', count: 10, timing_type: 'immediate', expires_days: undefined as number | undefined })

const planOptions = computed(() => plans.value.map(p => ({ label: p.name, value: p.plan_id })))

const columns = [
  { key: 'select', title: '', class: 'w-10' },
  { key: 'code', title: '卡密' },
  { key: 'plan_id', title: '套餐', format: (v: any) => planLabel(v) },
  { key: 'status', title: '状态' },
  { key: 'used_by', title: '使用租户' },
  { key: 'created_at', title: '创建时间', format: (v: any) => new Date(v).toLocaleDateString('zh-CN') },
  { key: 'actions', title: '操作', class: 'w-36' },
]

// DataTable → TDesign 表格列映射(key 字段对齐插槽名)
const tdColumns = computed(() => columns.map(c => ({ colKey: c.key, title: c.title, width: c.class?.includes('w-10') ? 48 : (c.class?.includes('w-36') ? 160 : undefined) })))

// P0-22 审计修复: 原 statusFilter 无任何 watch——下拉切换不触发查询, 列表纹丝不动。
// 切换即重置到第 1 页并重新拉取(状态筛选走后端, 页码不重置会看不到结果)。
watch(statusFilter, () => { page.value = 1; fetchCodes() })

function onPageChange(p: { current: number }) {
  if (p.current === page.value) return
  page.value = p.current
  void fetchCodes()
}

let codesRequestId = 0
let codesPageActive = true
let codesController: AbortController | null = null
let codesPlansController: AbortController | null = null
async function fetchCodes() {
  codesController?.abort()
  const controller = new AbortController()
  codesController = controller
  const requestId = ++codesRequestId
  const params = { page: page.value, page_size: 20, ...(statusFilter.value ? { status: statusFilter.value } : {}) }
  try {
    // 页码与筛选变化可能产生重叠请求；仅允许最新请求写入列表。
    const res = await adminApiMod.listCodes(params, { signal: controller.signal })
    if (requestId !== codesRequestId) return
    codes.value = res.list || []
    total.value = res.total || 0
  } catch (e: unknown) {
    if (controller.signal.aborted || requestId !== codesRequestId) return
    toast({ type: 'error', title: '获取卡密失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  } finally {
    if (codesController === controller) codesController = null
  }
}

async function doGenerate() {
  if (generatingCodes.value) return
  const validationError = validateCodeGenerationInput(generateForm.value, planOptions.value.map(option => option.value))
  if (validationError) {
    toast({ type: 'warning', title: '生成参数无效', message: validationError })
    return
  }
  const payload = { ...generateForm.value }
  generatingCodes.value = true
  try {
    await adminApiMod.generateCodes(payload)
    showGenerate.value = false
    generateForm.value = { plan_id: 'basic', count: 10, timing_type: 'immediate', expires_days: undefined }
    await fetchCodes()
    toast({ type: 'success', title: '卡密已生成' })
  } catch (e: unknown) {
    toast({ type: 'error', title: '生成失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  } finally {
    generatingCodes.value = false
  }
}

async function revokeCode(c: ActivationCode) {
  if (revokingCodeIds.value.includes(c.code_id)) return
  revokingCodeIds.value = [...revokingCodeIds.value, c.code_id]
  try {
    await adminApiMod.revokeCode(c.code_id)
    await fetchCodes()
    toast({ type: 'success', title: '卡密已作废' })
  } catch (e: unknown) {
    toast({ type: 'error', title: '作废失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  } finally {
    revokingCodeIds.value = revokingCodeIds.value.filter(id => id !== c.code_id)
  }
}

// 导出全部未使用卡密：走后端 /admin/codes/export（此前只导出当前页过滤结果）
const exportingCodes = ref(false)
async function exportCodes() {
  if (exportingCodes.value) return
  exportingCodes.value = true
  try {
    const text = await adminApiMod.exportCodes()
    if (!text) {
      toast({ type: 'info', title: '没有可导出的卡密' })
      return
    }
    const blob = new Blob([text], { type: 'text/plain;charset=utf-8' })
    downloadBlob(`codes_export_${new Date().toISOString().slice(0, 10)}.txt`, blob)
  } catch (e: unknown) {
    toast({ type: 'error', title: '导出失败', message: e instanceof Error ? e.message : '请稍后重试' })
  } finally {
    exportingCodes.value = false
  }
}

const selectedIds = ref<string[]>([])

async function batchRevoke() {
  if (batchRevoking.value || batchDeleting.value || !selectedIds.value.length) return
  const ids = [...new Set(selectedIds.value)]
  if (!await ask(`批量作废 ${ids.length} 张卡密？`)) return
  batchRevoking.value = true
  try {
    await adminApiMod.batchRevokeCodes({ ids })
    selectedIds.value = selectedIds.value.filter(id => !ids.includes(id))
    await fetchCodes()
    toast({ type: 'success', title: '已批量作废' })
  } catch (e: unknown) {
    toast({ type: 'error', title: '批量作废失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  } finally { batchRevoking.value = false }
}

async function batchDelete() {
  if (batchDeleting.value || batchRevoking.value || !selectedIds.value.length) return
  const ids = [...new Set(selectedIds.value)]
  if (!await ask(`批量删除 ${ids.length} 张卡密？不可恢复！`)) return
  batchDeleting.value = true
  try {
    await adminApiMod.batchDeleteCodes({ ids })
    selectedIds.value = selectedIds.value.filter(id => !ids.includes(id))
    await fetchCodes()
    toast({ type: 'success', title: '已批量删除' })
  } catch (e: unknown) {
    toast({ type: 'error', title: '批量删除失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  } finally { batchDeleting.value = false }
}

async function delCode(c: ActivationCode) {
  if (deletingCodeIds.value.includes(c.code_id)) return
  if (!await ask(`确认删除卡密 ${c.code}？不可恢复！`)) return
  deletingCodeIds.value = [...deletingCodeIds.value, c.code_id]
  try {
    await adminApiMod.batchDeleteCodes({ ids: [c.code_id] })
    selectedIds.value = selectedIds.value.filter(id => id !== c.code_id)
    await fetchCodes()
    toast({ type: 'success', title: '卡密已删除' })
  } catch (e: unknown) {
    toast({ type: 'error', title: '删除失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  } finally {
    deletingCodeIds.value = deletingCodeIds.value.filter(id => id !== c.code_id)
  }
}

onUnmounted(() => { codesPageActive = false; codesRequestId++; codesController?.abort(); codesPlansController?.abort() })

const viewingCode = ref<ActivationCode | null>(null)
const showView = ref(false)
function viewCode(c: ActivationCode) {
  viewingCode.value = c
  showView.value = true
}

onMounted(async () => {
  codesPageActive = true
  await fetchCodes()
  if (!codesPageActive) return
  try {
    codesPlansController = new AbortController()
    const res = await adminApiMod.listPlans({ signal: codesPlansController.signal })
    if (!codesPageActive) return
    plans.value = res.list || []
  } catch (e: unknown) {
    if (codesPageActive) toast({ type: 'error', title: '获取套餐失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  }
})
</script>

<style scoped>
.batch-bar-enter-active,
.batch-bar-leave-active {
  transition: all 0.25s ease;
}
.batch-bar-enter-from,
.batch-bar-leave-to {
  opacity: 0;
  transform: translate(-50%, 12px);
}
</style>
