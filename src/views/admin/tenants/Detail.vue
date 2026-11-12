<template>
  <!-- P0-12: 无效租户ID的错误态——原 setup throw 白屏, 这里给返回出口。
       v-if 链顺序: invalidId 先行, tenant 内容 v-else-if 跟随 -->
  <main v-if="invalidId" class="max-w-4xl mx-auto py-16 text-center space-y-4">
    <p class="text-text-muted">租户 ID 无效或链接已损坏</p>
    <t-button @click="router.replace('/admin/tenants')" class="ui-t-button--secondary ui-t-button">返回租户列表</t-button>
  </main>
  <main v-else-if="loadingTenant" class="max-w-4xl mx-auto py-16 text-center text-text-secondary" role="status" aria-live="polite">
    正在读取租户详情…
  </main>
  <main v-else-if="loadError" class="max-w-4xl mx-auto py-16 text-center space-y-4">
    <p class="text-text-secondary">{{ loadError }}</p>
    <t-button theme="primary" @click="fetchTenant" class="ui-t-button ui-t-button--primary">重新加载</t-button>
    <t-button @click="router.replace('/admin/tenants')" class="ui-t-button ui-t-button--secondary">返回租户列表</t-button>
  </main>
  <main class="max-w-4xl mx-auto space-y-6" v-else-if="tenant">
    <Breadcrumb :items="[{ path: '/admin/tenants', label: '租户管理' }, { path: '', label: '租户详情' }]" />
    <div class="flex items-center gap-3">
      <button type="button" @click="router.replace('/admin/tenants')" class="ui-icon-btn" aria-label="返回" title="返回">
        <ArrowLeft class="w-5 h-5" />
      </button>
      <h2 class="text-lg font-semibold text-gradient">租户详情</h2>
    </div>

    <div class="glass rounded-[var(--t-r-md)] p-5">
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <p class="text-xs text-text-secondary">租户名称</p>
          <p class="text-lg font-medium mt-1">{{ tenant.tenant_name }}</p>
        </div>
        <div>
          <p class="text-xs text-text-secondary">租户ID</p>
          <p class="font-mono text-sm mt-1 max-w-[65ch]">{{ tenant.tenant_id }}</p>
        </div>
        <div>
          <p class="text-xs text-text-secondary">当前套餐</p>
          <p class="text-lg font-medium text-[var(--t-brand)] mt-1">{{ planLabel(tenant.plan_id) }}</p>
        </div>
        <div>
          <p class="text-xs text-text-secondary">状态</p>
          <span class="inline-block mt-1 text-xs px-2 py-0.5 rounded-full" :class="statusClass(tenant.status)">
            {{ tenant.status === 'active' ? '正常' : tenant.status === 'suspended' ? '已停用' : '未激活' }}
          </span>
        </div>
        <div>
          <p class="text-xs text-text-secondary">账号上限</p>
          <p class="text-lg font-mono mt-1">{{ tenant.max_accounts }}</p>
        </div>
        <div>
          <p class="text-xs text-text-secondary">并发上限</p>
          <p class="text-lg font-mono mt-1">{{ tenant.max_concurrent_tasks }}</p>
        </div>
        <div>
          <p class="text-xs text-text-secondary">到期时间</p>
          <p class="text-sm mt-1 max-w-[65ch]">{{ tenant.expires_at ? formatDate(tenant.expires_at) : '永久' }}</p>
        </div>
        <div>
          <p class="text-xs text-text-secondary">注册时间</p>
          <p class="text-sm mt-1 max-w-[65ch]">{{ formatDate(tenant.created_at) }}</p>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2 mt-6 pt-6 border-t border-border">
        <t-button theme="primary" size="sm" :disabled="plansLoading" class="ui-t-button ui-t-button--primary ui-t-button--sm" @click="showEdit = true">修改套餐</t-button>
        <t-button theme="primary" size="sm" variant="outline" class="ui-t-button ui-t-button--secondary ui-t-button--sm shrink-0 whitespace-nowrap" @click="resetKey">重置 API Key</t-button>
        <t-button theme="primary" size="sm" class="ui-t-button ui-t-button--primary ui-t-button--sm" @click="toggleStatus">
          {{ tenant.status === 'active' ? '停用租户' : '启用租户' }}
        </t-button>
      </div>
    </div>

    <Modal v-model="showEdit" title="修改套餐">
      <div class="space-y-4">
        <p v-if="plansLoading" class="text-sm text-text-secondary" role="status" aria-live="polite">正在读取套餐列表…</p>
        <div v-else-if="plansError" class="space-y-2" role="alert">
          <p class="text-sm text-[var(--t-err)]">{{ plansError }}</p>
          <t-button size="small" variant="outline" :disabled="plansLoading" class="ui-t-button ui-t-button--secondary ui-t-button--sm" @click="fetchPlans">重新加载套餐</t-button>
        </div>
        <div v-else>
        <div>
          <label for="tenant-detail-plan" class="block text-sm text-text-secondary mb-1.5">套餐</label>
          <t-select id="tenant-detail-plan" v-model="editForm.plan_id" :options="planOptions" />
        </div>
        <div>
          <label for="tenant-detail-expires" class="block text-sm text-text-secondary mb-1.5">到期时间</label>
          <input id="tenant-detail-expires" v-model="editForm.expires_at" type="datetime-local" class="ui-input !text-sm" />
        </div>
        </div>
      </div>
      <template #footer>
        <t-button type="button" @click="showEdit = false" class="ui-t-button--secondary ui-t-button">取消</t-button>
        <t-button theme="primary" size="sm" :disabled="plansLoading || !!plansError || !plans.length" class="ui-t-button ui-t-button--primary ui-t-button--sm" @click="saveEdit">保存</t-button>
      </template>
    </Modal>
  </main>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft } from 'lucide-vue-next'
import { adminApiMod } from '@/api/admin'
import { formatDate, formatPrice } from '@/utils/format'
import { toast } from '@/utils/toast'
import { ask } from '@/utils/confirm'
import Modal from '@/components/ui/Modal.vue'
import Breadcrumb from '@/components/ui/Breadcrumb.vue'
import type { Tenant, Plan } from '@/types'

function isValidUUID(id: string): boolean {
  const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
  return uuidRegex.test(id)
}

const route = useRoute()
const router = useRouter()
const tenantId = route.params.id as string
// P0-12 审计修复: 原 throw 在 setup 顶层——整个组件渲染失败整页白屏, 用户无路可退。
// 改为导航回租户列表(错误出口), 并渲染错误态而非崩溃。
if (!isValidUUID(tenantId)) {
  router.replace('/admin/tenants')
}
const invalidId = ref(!isValidUUID(tenantId))

const tenant = ref<Tenant | null>(null)
const loadingTenant = ref(true)
const loadError = ref('')
const plans = ref<Plan[]>([])
const plansLoading = ref(true)
const plansError = ref('')
const showEdit = ref(false)
const editForm = ref({ plan_id: '', expires_at: '' })

const planOptions = computed(() => plans.value.map(p => ({ label: p.name + ' - ' + formatPrice(p.price_monthly) + '/月', value: p.plan_id })))

function toDateTimeLocal(v: string): string {
  if (!v) return ''
  const d = new Date(v)
  if (isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

async function fetchTenant() {
  loadingTenant.value = true
  loadError.value = ''
  try {
    tenant.value = await adminApiMod.getTenant(tenantId)
    editForm.value = { plan_id: tenant.value?.plan_id || '', expires_at: toDateTimeLocal(tenant.value?.expires_at || '') }
  } catch (e: unknown) {
    loadError.value = e instanceof Error ? e.message : '获取租户失败，请稍后重试'
    toast({ type: 'error', title: '获取租户失败', message: loadError.value })
  } finally {
    loadingTenant.value = false
  }
}

async function saveEdit() {
  try {
    // P2-49: datetime-local 给的是无时区本地时间, 原样提交后端按 UTC 解析——
    // 东八区设 10:00 被存成 18:00。转 ISO(本地时区→UTC)后提交。
    const payload = {
      ...editForm.value,
      expires_at: editForm.value.expires_at ? new Date(editForm.value.expires_at).toISOString() : '',
    }
    await adminApiMod.updateTenant(tenantId, payload)
    showEdit.value = false
    await fetchTenant()
    toast({ type: 'success', title: '租户信息已更新' })
  } catch (e: unknown) {
    toast({ type: 'error', title: '更新失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  }
}

async function resetKey() {
  try {
    // 重置即作废旧 Key（明文只显示一次），误触代价高——加确认
    const ok = await ask('重置 API Key 后旧 Key 将立即失效，确定重置？')
    if (!ok) return
    await adminApiMod.resetApiKey(tenantId)
    await fetchTenant()
    toast({ type: 'success', title: 'API Key 已重置' })
  } catch (e: unknown) {
    toast({ type: 'error', title: '重置失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  }
}

async function toggleStatus() {
  try {
    const newStatus = tenant.value?.status === 'active' ? 'suspended' : 'active'
    await adminApiMod.updateTenant(tenantId, { status: newStatus })
    await fetchTenant()
    toast({ type: 'success', title: newStatus === 'active' ? '租户已启用' : '租户已停用' })
  } catch (e: unknown) {
    toast({ type: 'error', title: '操作失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  }
}

function statusClass(s: string) {
  const map: Record<string, string> = { active: 'bg-[var(--t-ok)]/10 text-[var(--t-ok)]', inactive: 'bg-text-muted/15 text-text-muted', suspended: 'bg-[var(--t-err)]/10 text-[var(--t-err)]' }
  return map[s] || ''
}

function planLabel(p: string) {
  const map: Record<string, string> = { free: '免费版', basic: '基础版', pro: '专业版', enterprise: '企业版' }
  return map[p] || p
}

async function fetchPlans() {
  plansLoading.value = true
  plansError.value = ''
  try {
    const res = await adminApiMod.listPlans()
    plans.value = res.list || []
    if (!plans.value.length) plansError.value = '当前没有可用套餐，请检查套餐配置'
  } catch (e: unknown) {
    plansError.value = e instanceof Error ? e.message : '获取套餐失败，请稍后重试'
    toast({ type: 'error', title: '获取套餐失败', message: plansError.value })
  } finally {
    plansLoading.value = false
  }
}

onMounted(async () => {
  if (invalidId.value) return
  await Promise.all([fetchTenant(), fetchPlans()])
})
</script>
