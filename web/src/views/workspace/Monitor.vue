<template>
  <div class="min-w-0 w-full p-3 sm:p-5 md:p-6 max-w-4xl mx-auto">
    <h1 class="text-title font-extrabold tracking-tight text-text-primary truncate">消息监控</h1>
    <p class="text-sm text-text-muted mt-1 mb-6">
      <!-- R109: 账号下拉(源群组选择/规则展示跟随), 不依赖顶栏切换器 -->
      <span>账号:
        <select v-model="monAccId" class="ui-input !w-auto !min-h-[36px] !px-2 !py-1.5 !text-xs ml-1 align-middle" aria-label="监控执行账号">
          <option v-for="a in accountStore.accounts" :key="a.account_id" :value="a.account_id">{{ accName(a) }}</option>
        </select>
      </span>
    </p>

    <div class="glass rounded-[var(--t-r-md)] p-4 sm:p-6 space-y-4">
      <!-- 监控状态(自动模式: 规则即开关) -->
      <div class="p-4 rounded-lg border border-border bg-bg-card">
        <div class="flex items-center gap-3">
          <div class="w-3 h-3 rounded-full shrink-0" :class="anyRuleActive ? 'bg-[var(--t-ok)] animate-pulse' : 'bg-text-muted'" />
          <span class="text-sm font-medium whitespace-nowrap">{{ anyRuleActive ? '监控运行中' : '监控已停止' }}</span>
        </div>
        <p class="text-xs text-text-muted mt-1.5 pl-6">启用任意规则即自动开始，全部停用自动停止</p>
      </div>

      <!-- 规则列表 -->
      <div>
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-sm font-medium">监控规则</h3>
          <!-- P0-28: 走 openAddRule 先清空 editingRule, 防编辑残留变覆盖 -->
          <t-button @click="openAddRule" class="ui-t-button--primary ui-t-button">添加规则</t-button>
        </div>
        <div class="space-y-2">
          <EmptyState v-if="rules.length === 0" icon="Eye" title="暂无监控规则" description="添加关键词规则，命中消息的群会被自动监控" class="py-8" />
          <div v-for="rule in rules" :key="rule.rule_id"
               class="p-4 rounded-[18px] flex items-center justify-between"
               style="background: var(--t-surface-1); border: 1px solid var(--t-line);">
            <div class="min-w-0">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="text-sm font-extrabold">{{ rule.name }}</span>
                <!-- R126 示例2 badge 语言: 圆点+文字胶囊 -->
                <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-bold"
                      :class="rule.is_active ? 'bg-[color:var(--t-ok-dim)] text-[var(--t-ok)]' : 'bg-text-muted/15 text-text-muted'">
                  <i class="w-1.5 h-1.5 rounded-full bg-current"></i>{{ rule.is_active ? '启用' : '停用' }}
                </span>
                <!-- 2026-09-22 隔离改造: 显示规则归属账号 -->
                <span v-if="rule.account_id" class="px-1.5 py-0.5 rounded-full text-[10px] font-semibold" style="background: var(--t-brand-dim); color: var(--t-brand);">
                  账号: {{ accName(accountStore.accounts.find(a => a.account_id === rule.account_id)) }}
                </span>
                <span v-else class="px-1.5 py-0.5 rounded-full text-[10px] font-semibold bg-text-muted/10 text-text-muted">全部账号</span>
              </div>
              <div class="text-xs mt-1" style="color: var(--t-text-3);">
                来源: {{ !(rule.source_chat_ids || []).length ? '全部消息' : (rule.source_chat_ids || []).length + ' 个指定群组' }}
                | 匹配: {{ matchTypeLabel(rule.match_type) }} | 动作: {{ actionLabel(rule.type) }}{{ rule.type === 'dm' ? dmSummary(rule) : '' }}
                <span v-if="rule.keywords"> | 关键词: {{ rule.keywords }}</span>
              </div>
            </div>
            <div class="flex items-center gap-1">
              <t-switch :model-value="rule.is_active" @update:model-value="(v: boolean) => toggleRule(rule.rule_id, v)" />
              <button type="button" @click="editRule(rule)" class="ui-icon-btn" title="编辑" aria-label="编辑规则">
                <Edit3 class="w-4 h-4 text-text-muted" />
              </button>
              <button type="button" @click="deleteRule(rule.rule_id)" class="ui-icon-btn ui-icon-btn--danger" title="删除" aria-label="删除规则">
                <Trash2 class="w-4 h-4 text-[var(--t-err)]" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加/编辑规则弹窗: 统一全站 Modal(移动端底部抽屉) -->
    <Modal v-model="showAddRule" :title="editingRule ? '编辑规则' : '添加规则'" class="max-w-lg">
      <div class="space-y-4">
          <!-- R112 表单分组① -->
          <div class="flex items-center gap-2 -mt-1"><span class="text-[10.5px] font-semibold text-text-muted uppercase tracking-wider">基础信息</span><span class="flex-1 h-px bg-border/60"></span></div>
          <div>
            <label class="block text-sm text-text-secondary mb-1">规则名称</label>
            <input v-model="ruleForm.name" class="ui-input" />
          </div>
          <div>
            <label class="block text-sm text-text-secondary mb-2">源群组（监控来源）</label>
            <label class="flex items-start gap-2.5 p-3 rounded-lg border border-border cursor-pointer">
              <input v-model="monitorAllMessages" type="checkbox" class="mt-0.5 accent-[var(--t-brand)]" />
              <span class="min-w-0">
                <span class="block text-sm font-medium">监控全部消息</span>
                <span class="block text-xs text-text-muted mt-0.5">开启后不限制来源群组，监听该账号收到的所有消息</span>
              </span>
            </label>
            <template v-if="!monitorAllMessages">
              <div @click="showSourceSelector = true"
                   class="flex items-center gap-2 p-3 mt-2 rounded-lg border border-border cursor-pointer hover:border-[color:var(--t-brand)]/30 transition-colors">
                <Users class="w-4 h-4 text-text-muted" />
                <span v-if="!selectedSourceGroups.length" class="text-text-muted text-sm">点击选择源群组</span>
                <span v-else class="text-sm">已选 {{ selectedSourceGroups.length }} 个群组</span>
              </div>
              <div v-if="selectedSourceGroups.length" class="flex flex-wrap gap-1.5 mt-2">
                <span v-for="g in selectedSourceGroups" :key="g.group_id"
                      class="px-2 py-0.5 rounded-full text-xs bg-[color:var(--t-brand-dim)] text-[var(--t-brand)] border border-[color:var(--t-brand-dim)]">{{ g.title }}</span>
              </div>
              <p v-if="!selectedSourceGroups.length" class="text-xs text-[var(--t-warn)] mt-1">请至少选择一个来源群组，或开启“监控全部消息”</p>
            </template>
          </div>
          <!-- R112 表单分组② -->
          <div class="flex items-center gap-2 pt-1"><span class="text-[10.5px] font-semibold text-text-muted uppercase tracking-wider">匹配条件与动作</span><span class="flex-1 h-px bg-border/60"></span></div>
          <div>
            <label class="block text-sm text-text-secondary mb-1">匹配类型</label>
            <select v-model="ruleForm.match_type" class="ui-input">
              <option value="contains">包含关键词</option>
              <option value="exact">完全匹配</option>
              <option value="regex">正则匹配</option>
              <option value="from_user">来自指定用户</option>
            </select>
          </div>
          <div>
            <label class="block text-sm text-text-secondary mb-1">关键词（每行一个）</label>
            <textarea v-model="ruleForm.keywordsInput" rows="3" class="ui-input resize-none" />
          </div>
          <div>
            <label class="block text-sm text-text-secondary mb-1">动作</label>
            <select v-model="ruleForm.type" class="ui-input">
              <option value="reply">自动回复</option>
              <option value="forward">转发消息</option>
              <option value="dm">私聊触达</option>
            </select>
          </div>
          <div v-if="ruleForm.type === 'reply'">
            <label class="block text-sm text-text-secondary mb-1">回复内容</label>
            <textarea v-model="ruleForm.reply_content" rows="2" class="ui-input resize-none" />
          </div>
          <!-- 私聊触达: 命中后由监控号私聊触发人, 子动作可任意组合 -->
          <div v-if="ruleForm.type === 'dm'" class="space-y-3 p-3 rounded-lg border border-border">
            <label class="flex items-center gap-2 text-sm text-text-secondary cursor-pointer select-none">
              <input type="checkbox" v-model="ruleForm.dmContentEnabled" class="accent-[var(--t-ok)]" />
              发送私聊消息
            </label>
            <div v-if="ruleForm.dmContentEnabled">
              <textarea v-model="ruleForm.dm_content" rows="2" placeholder="私聊发送给触发人的文本"
                        class="ui-input resize-none" />
            </div>
            <label class="flex items-center gap-2 text-sm text-text-secondary cursor-pointer select-none">
              <input type="checkbox" v-model="ruleForm.dmContactEnabled" class="accent-[var(--t-ok)]" />
              发送联系人名片
            </label>
            <div v-if="ruleForm.dmContactEnabled" class="space-y-2">
              <div class="flex items-center gap-2">
                <select v-model="contactPickAccount" class="ui-input !w-auto flex-1 !min-h-[36px] !px-2 !py-1.5 !text-xs">
                  <option value="">选择账号…</option>
                  <option v-for="a in accountStore.accounts" :key="a.account_id" :value="a.account_id">{{ accName(a) }}</option>
                </select>
                <t-button type="button" class="ui-t-button--primary ui-t-button whitespace-nowrap"
                        :disabled="!contactPickAccount || contactLoading"
                        @click="loadContacts">
                  {{ contactLoading ? '获取中…' : '获取联系人' }}
                </t-button>
              </div>
              <select v-if="contactList.length" v-model="contactPick" class="ui-input !min-h-[36px] !px-2 !py-1.5 !text-xs" @change="applyContact">
                <option value="">选择联系人（自动填入手机号和姓名）…</option>
                <option v-for="ct in contactList" :key="ct.tg_user_id" :value="ct.tg_user_id">
                  {{ ct.first_name }} {{ ct.last_name || '' }} · {{ ct.phone }}
                </option>
              </select>
              <p v-if="contactPick && !ruleForm.dm_contact_phone" class="text-xs text-[var(--t-warn)]">该联系人资料未返回手机号，Telegram 名片发送必须提供手机号；请手动补充，或选择有手机号的联系人。</p>
              <div class="grid grid-cols-2 gap-2">
                <input v-model="ruleForm.dm_contact_phone" placeholder="电话号码（自动带入；缺失时需补充）" class="ui-input col-span-2" />
                <input v-model="ruleForm.dm_contact_first_name" placeholder="名(必填)" class="ui-input" />
                <input v-model="ruleForm.dm_contact_last_name" placeholder="姓(可空)" class="ui-input" />
              </div>
            </div>
            <label class="flex items-center gap-2 text-sm text-text-secondary cursor-pointer select-none">
              <input type="checkbox" v-model="ruleForm.dmCallEnabled" class="accent-[var(--t-ok)]" />
              弹语音通话
            </label>
            <div v-if="ruleForm.dmCallEnabled" class="flex items-center gap-2 text-sm text-text-muted">
              响铃
              <input type="number" v-model.number="ruleForm.dm_call_seconds" min="1" max="60"
                     class="ui-input !w-20 !min-h-[36px] !px-2 !py-1.5 !text-center" />
              秒后自动挂断(对方显示未接来电)
            </div>
            <p class="text-[11px] leading-relaxed" style="color: var(--t-text-3);">
              同一用户对同一规则只触发一次；同一监控号对同一用户 10 分钟内最多拨打一次，频繁拨打有封号风险
            </p>
          </div>
          <div v-if="ruleForm.type === 'forward'">
            <label class="block text-sm text-text-secondary mb-1">目标群组（转发目的地）</label>
            <div @click="showTargetSelector = true"
                 class="flex items-center gap-2 p-3 rounded-lg border border-border cursor-pointer hover:border-[color:var(--t-brand)]/30 transition-colors">
              <Users class="w-4 h-4 text-text-muted" />
              <span v-if="!selectedTargetGroups.length" class="text-text-muted text-sm">点击选择目标群组</span>
              <span v-else class="text-sm">已选 {{ selectedTargetGroups.length }} 个群组</span>
            </div>
            <div v-if="selectedTargetGroups.length" class="flex flex-wrap gap-1.5 mt-2">
              <span v-for="g in selectedTargetGroups" :key="g.group_id"
                    class="px-2 py-0.5 rounded-full text-xs bg-[color:var(--t-ok-dim)] text-[var(--t-ok)] border border-[color:var(--t-ok)]/20">{{ g.title }}</span>
            </div>
          </div>
        </div>
        <div class="flex justify-end gap-3 mt-6">
          <t-button @click="showAddRule = false" class="ui-t-button--secondary ui-t-button">取消</t-button>
          <t-button @click="saveRule" class="ui-t-button--primary ui-t-button" :loading="ruleSaving" :disabled="ruleSaving">保存</t-button>
        </div>
    </Modal>

    <!-- 群组选择器 -->
    <GroupSelector
      v-model="showSourceSelector"
      :groups="groups"
      :account-id="monAccId"
      mode="multiple"
      @select="(gs: Group[]) => { selectedSourceGroups = gs; if (gs.length) monitorAllMessages = false }"
      @refresh="fetchGroups"
    />
    <GroupSelector
      v-model="showTargetSelector"
      :groups="groups"
      :account-id="monAccId"
      mode="multiple"
      @select="(gs: Group[]) => selectedTargetGroups = gs"
      @refresh="fetchGroups"
    />
  </div>
  <LiveTaskFeed v-if="monAccId" :account-id="monAccId" task-type="broadcast" class="mt-6" />

</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAccountStore } from '@/stores/account'
import { useAuthStore } from '@/stores/auth'
import { monitorApi } from '@/api/monitor'
import { groupApi } from '@/api/group'
import { accountApi } from '@/api/account'
import EmptyState from '@/components/ui/EmptyState.vue'
import Modal from '@/components/ui/Modal.vue'
import { toast } from '@/utils/toast'
import { Eye, Power, Edit3, Trash2, Users } from 'lucide-vue-next'
import GroupSelector from '@/components/selectors/GroupSelector.vue'
import type { Group } from '@/types'
import LiveTaskFeed from '@/components/LiveTaskFeed.vue'
import { ask } from '@/utils/confirm'

const route = useRoute()
const router = useRouter()
const accountStore = useAccountStore()
const authStore = useAuthStore()
const currentAccount = computed(() => accountStore.currentAccount)
const accName = (a: any) => a?.first_name || a?.remark || a?.phone || (a?.account_id || '').slice(0, 8)
// R109: 监控按账号管理, 下拉选择管理哪个号
const monAccId = ref('')
watch(currentAccount, (acc: any) => { if (acc?.account_id && !monAccId.value) monAccId.value = acc.account_id }, { immediate: true })
watch(monAccId, () => {
  groupsController?.abort()
  rulesController?.abort()
  // 切换执行账号时先清除上一账号的群组，避免新请求返回前短暂展示旧账号数据。
  groupsRequestId++
  groups.value = []
  fetchGroups()
  fetchRules()
})
const groups = ref<Group[]>([])
const showSourceSelector = ref(false)
const showTargetSelector = ref(false)
const selectedSourceGroups = ref<Group[]>([])
// 空 source_chat_ids 在后端表示不限制来源；编辑旧规则时兼容该语义。
const monitorAllMessages = ref(true)
const selectedTargetGroups = ref<Group[]>([])

async function fetchGroups() {
  const accountId = monAccId.value
  groupsController?.abort()
  const controller = new AbortController()
  groupsController = controller
  const requestId = ++groupsRequestId
  if (!accountId) { groups.value = []; groupsController = null; return }
  try {
    const res = await groupApi.list(accountId, undefined, { signal: controller.signal })
    if (!monitorPageActive || requestId !== groupsRequestId || accountId !== monAccId.value) return
    const raw = res as any
    groups.value = Array.isArray(raw.list) ? raw.list : []
  } catch (e) {
    if (!controller.signal.aborted && monitorPageActive && requestId === groupsRequestId) groups.value = []
  } finally {
    if (groupsController === controller) groupsController = null
  }
}

const rules = ref<any[]>([])
const isMonitoring = ref(false) // 精读注: 预留字段, 当前由 rules.length+isAccountOnline 推导运行态
const anyRuleActive = computed(() => rules.value.some(r => r.is_active))
const showAddRule = ref(false)
const editingRule = ref<any>(null)
const emptyRuleForm = () => ({
  name: '',
  match_type: 'contains',
  keywordsInput: '',
  type: 'reply',
  reply_content: '',
  dmContentEnabled: false,
  dm_content: '',
  dmContactEnabled: false,
  dm_contact_phone: '',
  dm_contact_first_name: '',
  dm_contact_last_name: '',
  dmCallEnabled: false,
  dm_call_seconds: 5,
})
const ruleForm = ref(emptyRuleForm())

// 名片联系人选择器: 借账号实时拉 TG 联系人, 选中后自动填电话/名/姓
const contactPickAccount = ref('')
const contactList = ref<Array<{ tg_user_id: number; first_name: string; last_name?: string; username?: string; phone?: string }>>([])
const contactPick = ref<number | ''>('')
const contactLoading = ref(false)
let contactRequestId = 0
let monitorPageActive = true
const ruleActionsPending = ref(new Set<string>())
const ruleSaving = ref(false)
let groupsRequestId = 0
let rulesRequestId = 0
let groupsController: AbortController | null = null
let rulesController: AbortController | null = null
let contactsController: AbortController | null = null
async function loadContacts() {
  if (!contactPickAccount.value) return
  const accountId = contactPickAccount.value
  contactsController?.abort()
  const controller = new AbortController()
  contactsController = controller
  const requestId = ++contactRequestId
  contactLoading.value = true
  contactList.value = []
  contactPick.value = ''
  try {
    const res: any = await accountApi.getContacts(accountId, { signal: controller.signal })
    if (!monitorPageActive || requestId !== contactRequestId || accountId !== contactPickAccount.value) return
    contactList.value = (res?.list || res || []).filter((c: any) => String(c.phone || '').trim())
    if (!contactList.value.length) toast({ type: 'info', title: '暂无可发送名片的联系人', message: 'Telegram 未向该账号返回带手机号的联系人；名片发送必须有手机号，可换账号获取或手动填写。' })
  } catch (err: any) {
    if (!monitorPageActive || requestId !== contactRequestId || accountId !== contactPickAccount.value) return
    toast({ type: 'error', title: '获取联系人失败', message: err?.message || '请稍后重试' })
  } finally {
    if (contactsController === controller) contactsController = null
    if (monitorPageActive && requestId === contactRequestId) contactLoading.value = false
  }
}
function applyContact() {
  const ct = contactList.value.find(c => c.tg_user_id === contactPick.value)
  if (!ct) return
  // 切换联系人时覆盖旧值，避免复用上一个联系人的号码。
  ruleForm.value.dm_contact_phone = ct.phone || ''
  ruleForm.value.dm_contact_first_name = ct.first_name || ct.username || ''
  ruleForm.value.dm_contact_last_name = ct.last_name || ''
}

async function fetchRules() {
  rulesController?.abort()
  const controller = new AbortController()
  rulesController = controller
  const requestId = ++rulesRequestId
  try {
    const res = await monitorApi.listRules({ signal: controller.signal })
    if (!monitorPageActive || requestId !== rulesRequestId) return
    rules.value = res.list || []
  } catch (e: any) {
    if (controller.signal.aborted || !monitorPageActive || requestId !== rulesRequestId) return
    toast({ type: 'error', title: '加载规则失败', message: e?.message || '请稍后重试' })
  } finally {
    if (rulesController === controller) rulesController = null
  }
}


async function toggleRule(id: string, val: boolean) {
  if (ruleActionsPending.value.has(id)) return
  ruleActionsPending.value = new Set(ruleActionsPending.value).add(id)
  try {
    await monitorApi.toggleRule(id, { is_active: val })
    if (monitorPageActive) await fetchRules()
  } catch {
    if (monitorPageActive) {
      toast({ type: 'error', title: '操作失败', message: '规则状态更新失败，请重试' })
      await fetchRules() // 回滚本地乐观变更
    }
  } finally {
    if (monitorPageActive) {
      const next = new Set(ruleActionsPending.value)
      next.delete(id)
      ruleActionsPending.value = next
    }
  }
}

async function deleteRule(id: string) {
  if (ruleActionsPending.value.has(id) || !await ask('确定删除此规则吗？') || !monitorPageActive) return
  ruleActionsPending.value = new Set(ruleActionsPending.value).add(id)
  try {
    await monitorApi.deleteRule(id)
    if (monitorPageActive) await fetchRules()
  } catch {
    if (monitorPageActive) toast({ type: 'error', title: '删除失败', message: '请稍后重试' })
  } finally {
    if (monitorPageActive) {
      const next = new Set(ruleActionsPending.value)
      next.delete(id)
      ruleActionsPending.value = next
    }
  }
}

function editRule(rule: any) {
  editingRule.value = rule
  ruleForm.value = {
    name: rule.name,
    match_type: rule.match_type,
    keywordsInput: (rule.keywords || []).join('\n'),
    type: rule.type || rule.action,
    reply_content: rule.reply_content || '',
    dmContentEnabled: !!rule.dm_content,
    dm_content: rule.dm_content || '',
    dmContactEnabled: !!rule.dm_contact,
    dm_contact_phone: rule.dm_contact_phone || '',
    dm_contact_first_name: rule.dm_contact_first_name || '',
    dm_contact_last_name: rule.dm_contact_last_name || '',
    dmCallEnabled: !!rule.dm_call,
    dm_call_seconds: rule.dm_call_seconds || 5,
  }
  // 恢复选中的源群组和目标群组
  monitorAllMessages.value = !(rule.source_chat_ids || []).length
  selectedSourceGroups.value = (rule.source_chat_ids || []).map((id: string) => {
    const found = groups.value.find(g => g.group_id === id || String(g.tg_chat_id) === id)
    return found || { group_id: id, title: id, tg_chat_id: id }
  })
  selectedTargetGroups.value = (rule.target_chat_ids || []).map((id: string) => {
    const found = groups.value.find(g => g.group_id === id || String(g.tg_chat_id) === id)
    return found || { group_id: id, title: id, tg_chat_id: id }
  })
  showAddRule.value = true
}

// P0-28: 打开添加弹窗前必须清空 editingRule(见模板注释)
function openAddRule() {
  editingRule.value = null
  showAddRule.value = true
}

async function saveRule() {
  if (ruleSaving.value) return
  if (ruleForm.value.type === 'forward' && !selectedTargetGroups.value.length) {
    toast({ type: 'warning', title: '请选择转发目标', message: '转发规则至少需要一个目标群组' })
    return
  }
  if (!monitorAllMessages.value && !selectedSourceGroups.value.length) {
    toast({ type: 'warning', title: '请选择监控来源', message: '请至少选择一个源群组，或开启“监控全部消息”' })
    return
  }
  // dm 规则: 子动作开关勾选但内容为空时按未勾选处理, 交给后端兜底校验
  const isDM = ruleForm.value.type === 'dm'
  const dmContent = isDM && ruleForm.value.dmContentEnabled ? ruleForm.value.dm_content.trim() : ''
  const dmContact = isDM && ruleForm.value.dmContactEnabled
  const dmCall = isDM && ruleForm.value.dmCallEnabled
  const data = {
    name: ruleForm.value.name,
    type: ruleForm.value.type as "reply" | "forward" | "dm",
    match_type: ruleForm.value.match_type as "contains" | "exact" | "regex" | "from_user",
    keywords: ruleForm.value.keywordsInput.split('\n').map(k => k.trim()).filter(Boolean),
    reply_content: ruleForm.value.reply_content,
    dm_content: dmContent,
    dm_call: dmCall,
    dm_call_seconds: ruleForm.value.dm_call_seconds || 5,
    dm_contact: dmContact,
    dm_contact_phone: dmContact ? ruleForm.value.dm_contact_phone.trim() : '',
    dm_contact_first_name: dmContact ? ruleForm.value.dm_contact_first_name.trim() : '',
    dm_contact_last_name: dmContact ? ruleForm.value.dm_contact_last_name.trim() : '',
    source_chat_ids: monitorAllMessages.value ? [] : selectedSourceGroups.value.map(g => String(g.tg_chat_id || g.group_id)),
    target_chat_ids: ruleForm.value.type === 'forward' ? selectedTargetGroups.value.map(g => String(g.tg_chat_id || g.group_id)) : [],
    enabled: true,
    // R109: 规则归属下拉所选账号(原绑 currentAccount=全局切换器, 已删除)
    account_id: monAccId.value || '',
  }
  ruleSaving.value = true
  try {
    if (editingRule.value) {
      await monitorApi.updateRule(editingRule.value.rule_id, data)
    } else {
      await monitorApi.createRule(data)
    }
    if (!monitorPageActive) return
    // 先记录是否编辑态(editingRule 随后会被清空), 保证提示文案准确
    const wasEditing = !!editingRule.value
    showAddRule.value = false
    editingRule.value = null
    ruleForm.value = emptyRuleForm()
    selectedSourceGroups.value = []
    monitorAllMessages.value = true
    selectedTargetGroups.value = []
    await fetchRules()
    toast({ type: 'success', title: wasEditing ? '规则已更新' : '监控规则已创建', message: '规则即时生效' })
  } catch (e: any) {
    if (monitorPageActive) toast({ type: 'error', title: '保存失败', message: e?.message || '请稍后重试' })
  } finally {
    if (monitorPageActive) ruleSaving.value = false
  }
}

function matchTypeLabel(t: string) {
  const map: Record<string, string> = { contains: '包含', exact: '完全匹配', regex: '正则', from_user: '指定用户' }
  return map[t] || t
}

function actionLabel(a: string) {
  const map: Record<string, string> = { reply: '自动回复', forward: '转发', dm: '私聊触达', alert: '告警' }
  return map[a] || a
}

// 规则列表里展示 dm 子动作摘要, 如 "(消息+通话+名片)"
function dmSummary(rule: any): string {
  const parts: string[] = []
  if (rule.dm_content) parts.push('消息')
  if (rule.dm_call) parts.push('通话')
  if (rule.dm_contact) parts.push('名片')
  return parts.length ? `(${parts.join('+')})` : ''
}

onMounted(() => {
  monitorPageActive = true
  if (!authStore.tenant?.plan_features?.monitor) {
    toast({ type: 'warning', title: '套餐限制', message: '当前套餐不支持消息监控，请联系客服升级' })
    router.push('/dashboard')
    return
  }
  fetchRules()
})
onUnmounted(() => {
  monitorPageActive = false
  contactsController?.abort()
  groupsController?.abort()
  rulesController?.abort()
  contactRequestId++
  groupsRequestId++
  rulesRequestId++
})
</script>
