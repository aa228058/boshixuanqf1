<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-lg font-bold text-text-primary">套餐管理</h2>
        <p class="text-xs text-text-secondary mt-1">
          勾选控制每档套餐能用的功能；右侧数值改规模与限额（0 = 不限）。保存后立即生效。
        </p>
      </div>
      <t-button size="small" theme="primary" :disabled="saving" class="whitespace-nowrap shrink-0 ui-t-button ui-t-button--primary ui-t-button--sm" @click="saveAll">
        <Loader2 v-if="saving" class="w-3.5 h-3.5 animate-spin mr-1 shrink-0" />
        保存全部
      </t-button>
    </div>

    <!-- 加载失败提示(不再静默空白) -->
    <div v-if="loadError" role="alert" class="glass rounded-[var(--t-r-md)] p-4 bg-[var(--t-warn)]/10 text-[var(--t-warn)] text-sm lift">
      {{ loadError }}
    </div>

    <!-- 加载骨架 -->
    <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-3">
      <div v-for="n in 4" :key="n" class="glass rounded-[var(--t-r-md)] p-4 space-y-3 lift">
        <div class="h-4 w-1/2 rounded bg-bg-hover animate-pulse-soft" />
        <div class="h-3 w-2/3 rounded bg-bg-hover animate-pulse-soft" />
        <div class="h-9 w-full rounded bg-bg-hover animate-pulse-soft" />
      </div>
    </div>
    <!-- 空态 -->
    <EmptyState
      v-else-if="!plans.length"
      icon="Inbox"
      title="还没有套餐配置"
      description="套餐数据来自 plans 表。新部署请先确认迁移已执行，或联系平台管理员创建套餐。"
    />

    <!-- 数值区: 每档一卡 -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-3">
      <div v-for="p in plans" :key="p.plan_id" class="glass rounded-[var(--t-r-md)] p-4 space-y-2.5 lift">
        <div class="flex items-center justify-between">
          <span class="font-semibold text-sm">{{ p.name }}</span>
          <span class="text-xs text-text-muted">{{ p.plan_id }}</span>
        </div>
        <label class="block text-xs text-text-secondary">月价（元）
          <input v-model.number="p.price_monthly" type="number" min="0" step="1"
            class="ui-input mt-0.5 !min-h-[36px] !px-2.5 !py-1.5 !text-sm" />
        </label>
        <div class="grid grid-cols-2 gap-2">
          <label class="block text-xs text-text-secondary">账号数
            <input v-model.number="p.max_accounts" type="number" min="1"
              class="ui-input mt-0.5 !min-h-[36px] !px-2.5 !py-1.5 !text-sm" />
          </label>
          <label class="block text-xs text-text-secondary">并发
            <input v-model.number="p.max_concurrent" type="number" min="1"
              class="ui-input mt-0.5 !min-h-[36px] !px-2.5 !py-1.5 !text-sm" />
          </label>
          <label class="block text-xs text-text-secondary">群发日限(0不限)
            <input v-model.number="p.max_daily_broadcasts" type="number" min="0"
              class="ui-input mt-0.5 !min-h-[36px] !px-2.5 !py-1.5 !text-sm" />
          </label>
          <label class="block text-xs text-text-secondary">搜群日限(0不限)
            <input v-model.number="p.daily_search_limit" type="number" min="0"
              class="ui-input mt-0.5 !min-h-[36px] !px-2.5 !py-1.5 !text-sm" />
          </label>
        </div>
      </div>
    </div>

    <!-- 功能矩阵 -->
    <div class="glass rounded-[var(--t-r-md)] overflow-x-auto lift">
      <table class="w-full min-w-[720px] text-sm">
        <thead>
          <tr class="border-b border-border text-left">
            <th class="px-4 py-3 text-text-secondary font-medium w-56">功能</th>
            <th v-for="p in plans" :key="p.plan_id" class="px-3 py-3 text-center text-text-secondary font-medium">
              {{ p.name }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="f in featureList" :key="f.key" class="border-b border-border/40 last:border-0 hover:bg-milk-01">
            <td class="px-4 py-2.5 text-text-primary">{{ f.label }}</td>
            <td v-for="p in plans" :key="p.plan_id" class="px-3 py-2.5 text-center">
              <input type="checkbox" class="accent-[var(--t-brand)] w-4 h-4 cursor-pointer"
                :checked="hasFeature(p, f.key)"
                @change="toggleFeature(p, f.key)" :aria-label="p.name + ' plan: ' + f.label" />
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <p class="text-xs text-text-muted">
      提示：「免费体验版」建议保持全功能（转化钩子）；基础版勾少一点形成阶梯；改动实时写入，租户下次请求即按新权限生效。
    </p>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Loader2 } from 'lucide-vue-next'
import { adminApiMod } from '@/api/admin'
import { toast } from '@/utils/toast'
import { normalizePlanFeatures } from '@/utils/adminPageGuards'
import EmptyState from '@/components/ui/EmptyState.vue'

interface PlanRow {
  plan_id: string
  name: string
  price_monthly: number
  max_accounts: number
  max_concurrent: number
  max_daily_broadcasts: number
  daily_search_limit: number
  features: string[]
}

const plans = ref<PlanRow[]>([])
const saving = ref(false)
// 首屏还没有数据时必须显示骨架：否则用户先看到空网格，几秒后才"长出"卡片。
const loading = ref(true)

// 套餐功能矩阵(2026-10-20 适配): 已随 UI 重构清理——
// 删: 关键词搜群(search 并入 discover)/邀请入群(并入加群)/账号检测/代理设置/AI图片验证/数据导出
// 增: 循环群发(loop_broadcast)/抽奖助手(raffle)/邀请入群独立 feature
const featureList = [
  { key: 'broadcast', label: '群发消息' },
  { key: 'loop_broadcast', label: '循环群发' },
  { key: 'join_groups', label: '加群任务' },
  { key: 'invite', label: '邀请入群' },
  { key: 'scrape_members', label: '成员采集' },
  { key: 'scrape_links', label: '链接采集' },
  { key: 'monitor', label: '消息监控' },
  { key: 'templates', label: '消息模板' },
  { key: 'scheduled_tasks', label: '定时任务' },
  { key: 'discover', label: '发现群组·群库(含搜群)' },
  { key: 'verifier', label: '入群验证(AI 图片验证已并入)' },
  { key: 'dict', label: '文案变量（高级）' },
  { key: 'api', label: 'API 接入' },
  { key: 'raffle', label: '抽奖助手' },
  { key: 'notify', label: '邮件通知' },
]

function hasFeature(p: PlanRow, key: string) {
  return (p.features || []).includes(key)
}
function toggleFeature(p: PlanRow, key: string) {
  const set = new Set(p.features || [])
  set.has(key) ? set.delete(key) : set.add(key)
  p.features = [...set]
}

const loadError = ref('')
let plansPageActive = false
let plansRequestId = 0
let plansController: AbortController | null = null

async function load() {
  plansController?.abort()
  const controller = new AbortController()
  plansController = controller
  const requestId = ++plansRequestId
  loadError.value = ''
  loading.value = true
  try {
    const res = await adminApiMod.listPlans({ signal: controller.signal })
    if (!plansPageActive || requestId !== plansRequestId) return
    plans.value = (res.list || []).map((p: any) => ({
      ...p,
      // 兼容双格式(2026-10-20): 后端存储已从数组改为 map {"raffle":true,...},
      // 加载时统一转成数组供勾选框使用(旧数组数据原样, 新 map 数据取 keys)
      features: normalizePlanFeatures(p.features),
    }))
    if (!plans.value.length) {
      loadError.value = '套餐列表为空 —— 请检查迁移 051 是否已执行(plans 表结构是否含 daily_search_limit 列)'
    }
  } catch (e: any) {
    loading.value = false
    if (controller.signal.aborted || !plansPageActive || requestId !== plansRequestId) return
    loadError.value = e?.message || '请求失败(401/500?)'
    toast({ type: 'error', message: '套餐加载失败: ' + loadError.value })
  } finally {
    if (plansPageActive) loading.value = false
    if (plansController === controller) plansController = null
  }
}

// P2-47 审计修复: 原 4 档套餐 for+await 串行保存(每档一次 RTT, 全量保存 4 倍耗时)。
// 改并行: 各档独立 then/catch, 失败仍逐档提示, 成功计数不变。
async function saveAll() {
  if (saving.value) return
  if (!plans.value.length) {
    toast({ type: 'warning', message: '没有可保存的套餐，请先重新加载套餐列表' })
    return
  }
  const invalidPlan = plans.value.find((p) =>
    !Number.isFinite(Number(p.price_monthly)) || Number(p.price_monthly) < 0 ||
    !Number.isInteger(Number(p.max_accounts)) || Number(p.max_accounts) < 1 ||
    !Number.isInteger(Number(p.max_concurrent)) || Number(p.max_concurrent) < 1 ||
    !Number.isInteger(Number(p.max_daily_broadcasts)) || Number(p.max_daily_broadcasts) < 0 ||
    !Number.isInteger(Number(p.daily_search_limit)) || Number(p.daily_search_limit) < 0
  )
  if (invalidPlan) {
    toast({ type: 'warning', title: '套餐数值无效', message: `${invalidPlan.name} 的价格、账号数、并发或日限额不符合要求` })
    return
  }
  saving.value = true
  const snapshot = plans.value.map((p) => ({
    plan_id: p.plan_id,
    name: p.name,
    price_monthly: Number(p.price_monthly),
    max_accounts: Number(p.max_accounts),
    max_concurrent: Number(p.max_concurrent),
    max_daily_broadcasts: Number(p.max_daily_broadcasts),
    daily_search_limit: Number(p.daily_search_limit),
    features: [...(p.features || [])],
  }))
  let ok = 0
  try {
    const jobs = snapshot.map((p) =>
      adminApiMod.updatePlan(p.plan_id, {
        name: p.name,
        price_monthly: p.price_monthly,
        max_accounts: p.max_accounts,
        max_concurrent: p.max_concurrent,
        max_daily_broadcasts: p.max_daily_broadcasts,
        daily_search_limit: p.daily_search_limit,
        // 保持数组格式：Plan.Features 后端结构体为 []string。
        features: p.features,
      }).then(() => { ok++ }).catch((e: any) => {
        if (plansPageActive) toast({ type: 'error', message: `${p.name} 保存失败: ${e?.message || '未知错误'}` })
      })
    )
    await Promise.all(jobs)
    if (!plansPageActive) return
    if (ok === snapshot.length) {
      toast({ type: 'success', message: '全部套餐已保存，即时生效' })
    } else {
      toast({ type: 'info', message: `已保存 ${ok}/${snapshot.length} 档` })
    }
  } finally {
    if (plansPageActive) saving.value = false
  }
}

onMounted(() => { plansPageActive = true; void load() })
onUnmounted(() => { plansPageActive = false; plansRequestId++; plansController?.abort(); plansController = null })
</script>
