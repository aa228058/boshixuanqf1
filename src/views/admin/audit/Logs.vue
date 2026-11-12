<template>
  <main class="space-y-4">
    <Breadcrumb :items="[{ path: '/admin/dashboard', label: '仪表盘' }, { path: '', label: '审计日志' }]" />
    <div class="flex items-center gap-2 overflow-x-auto pb-1">
      <input
        v-model="startDate"
        type="date"
        class="ui-input !w-auto !min-w-[140px] !min-h-[40px] !px-2.5 !py-1.5 !text-sm shrink-0"
        aria-label="开始日期" />
      <span class="text-text-muted shrink-0">至</span>
      <input
        v-model="endDate"
        type="date"
        class="ui-input !w-auto !min-w-[140px] !min-h-[40px] !px-2.5 !py-1.5 !text-sm shrink-0"
        aria-label="结束日期" />
      <t-button theme="primary" size="sm" class="ui-t-button ui-t-button--primary ui-t-button--sm shrink-0" @click="page = 1; fetchLogs()" :disabled="loading">查询</t-button>
    </div>

    <div v-if="loading" role="status" aria-live="polite" class="glass rounded-[var(--t-r-md)] p-8 text-center text-sm text-text-secondary">正在读取审计日志…</div>
    <div v-else-if="loadError" role="alert" class="glass rounded-[var(--t-r-md)] p-8 text-center space-y-3">
      <p class="text-sm text-[var(--t-err)]">{{ loadError }}</p>
      <t-button size="small" class="ui-t-button ui-t-button--secondary ui-t-button--sm" @click="fetchLogs">重新加载</t-button>
    </div>

    <!-- 桌面端表格 -->
    <div v-show="!loading && !loadError" class="hidden sm:block glass rounded-[var(--t-r-md)] overflow-hidden">
      <t-table
        row-key="log_id"
        :columns="columns.map(c => ({ colKey: c.key, title: c.title }))"
        :data="logs"
        hover
        :pagination="{ current: page, pageSize: 20, total: total, onChange: (p: any) => { page = p.current; fetchLogs() } }"
      >
        <template #created_at="{ row }">{{ new Date(row.created_at).toLocaleString('zh-CN') }}</template>
        <template #tenant_id="{ row }">{{ row.tenant_id ? row.tenant_id.slice(0, 8) + '...' : '系统' }}</template>
        <template #user_id="{ row }">{{ row.user_id ? row.user_id.slice(0, 8) + '...' : '-' }}</template>
        <template #action="{ row }">
          <span class="text-xs px-1.5 py-0.5 rounded" :class="actionClass(row.action as string)">
            {{ auditActionLabel(row.action as string) }}
          </span>
        </template>
      </t-table>
    </div>
    <!-- 移动端卡片 -->
    <div v-show="!loading && !loadError" class="sm:hidden space-y-3">
      <div
        v-for="log in logs"
        :key="log.log_id"
        class="ui-card p-4 space-y-2 lift"
      >
        <div class="flex flex-wrap items-center justify-between gap-2">
          <span class="text-xs px-1.5 py-0.5 rounded" :class="actionClass(log.action)">
            {{ auditActionLabel(log.action) }}
          </span>
          <span class="text-xs text-text-secondary">{{ new Date(log.created_at).toLocaleString('zh-CN') }}</span>
        </div>
        <div class="text-xs text-text-secondary">
          <span class="text-text-muted">租户:</span> {{ log.tenant_id ? log.tenant_id.slice(0, 8) + '...' : '系统' }}
          <span class="text-text-muted ml-2">用户:</span> {{ log.user_id ? log.user_id.slice(0, 8) + '...' : '-' }}
        </div>
        <div class="text-xs text-text-secondary">
          <span class="text-text-muted">目标:</span> {{ log.target || '-' }}
        </div>
        <div class="text-xs text-text-secondary">IP: {{ log.ip || '-' }}</div>
      </div>
      <EmptyState v-if="!logs.length" icon="ClipboardList" title="暂无审计日志" description="选择日期范围后点击查询" />
      <div v-if="total > 20" class="flex items-center justify-between gap-3 pt-1">
        <span class="text-xs text-text-muted">共 {{ total }} 条 · 第 {{ page }} / {{ Math.ceil(total / 20) }} 页</span>
        <div class="flex items-center gap-2">
          <t-button size="sm" variant="outline" class="ui-t-button ui-t-button--secondary ui-t-button--sm" :disabled="page <= 1" @click="page = page - 1; fetchLogs()">上一页</t-button>
          <t-button size="sm" variant="outline" class="ui-t-button ui-t-button--secondary ui-t-button--sm" :disabled="page >= Math.ceil(total / 20)" @click="page = page + 1; fetchLogs()">下一页</t-button>
        </div>
      </div>
    </div>
  </main>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { adminApiMod } from '@/api/admin'
import Breadcrumb from '@/components/ui/Breadcrumb.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import { toast } from '@/utils/toast'
import type { AuditLog } from '@/types'

const logs = ref<AuditLog[]>([])
const page = ref(1)
const total = ref(0)
const loading = ref(false)
const loadError = ref('')
const startDate = ref('')
const endDate = ref('')

const columns = [
  { key: 'created_at', title: '时间' },
  { key: 'tenant_id', title: '租户ID' },
  { key: 'user_id', title: '用户' },
  { key: 'action', title: '操作' },
  { key: 'target', title: '目标' },
  { key: 'ip', title: 'IP 地址' },
]

// P0-09 审计修复: 后端落库的 action 是英文码(account_delete/code_generated 等),
// 原实现判断中文字符串——颜色全部落到默认青,且用户直接看英文码。
const auditActionMap: Record<string, string> = {
  account_delete: '删除账号', account_import: '导入账号', account_create: '添加账号',
  account_update: '更新账号', account_batch_update: '批量改资料',
  tenant_created: '创建租户', tenant_updated: '更新租户', tenant_deleted: '删除租户',
  tenant_purged: '清除租户数据',
  plan_created: '创建套餐', plan_updated: '修改套餐', plan_deleted: '删除套餐',
  code_generated: '生成卡密', code_deleted: '删除卡密',
  announcement_created: '发布公告', announcement_updated: '更新公告', announcement_deleted: '删除公告',
  admin_login: '管理员登录', admin_logout: '管理员登出',
}
function auditActionLabel(action: string): string {
  return auditActionMap[action] || action
}
function actionClass(action: string) {
  // 按英文码前缀分类(值可能带动态后缀, 用 test 而非 includes)
  if (/delete|purge|revoke|ban/.test(action)) return 'bg-[var(--t-err)]/10 text-[var(--t-err)]'
  if (/create|enable|activate|import|login/.test(action)) return 'bg-[var(--t-ok)]/10 text-[var(--t-ok)]'
  if (/update|modify|edit/.test(action)) return 'bg-[var(--t-warn)]/10 text-[var(--t-warn)]'
  return 'bg-[color:var(--t-brand-dim)] text-[var(--t-brand)]'
}

async function fetchLogs() {
  loading.value = true
  loadError.value = ''
  try {
    const params: any = { page: page.value, page_size: 20 }
    // 时区修正：此前直接拼 'T00:00:00Z'（UTC 零点 = 北京时间 08:00），
    // 会漏掉当天 0-8 点的日志。改为本地零点/末点转 UTC ISO 串，与后端
    // parseAuditTime 的 RFC3339 分支对齐，筛选边界与管理员预期一致。
    if (startDate.value) params.start_time = new Date(startDate.value + 'T00:00:00').toISOString()
    if (endDate.value) params.end_time = new Date(endDate.value + 'T23:59:59').toISOString()
    const res = await adminApiMod.getAuditLogs(params)
    logs.value = (res.list || []) as any
    total.value = res.total || 0
  } catch (e: unknown) {
    loadError.value = e instanceof Error ? e.message : '查询审计日志失败，请重试'
    toast({ type: 'error', title: '查询失败', message: loadError.value })
  } finally {
    loading.value = false
  }
}

onMounted(fetchLogs)
</script>
