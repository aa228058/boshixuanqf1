<template>
  <Modal v-model="visible" title="消息通知">
    <div class="space-y-1 max-h-[500px] overflow-y-auto">
      <button
        type="button"
        v-for="n in notifications"
        :key="n.id"
        @click="markRead(n.id)"
        class="w-full text-left p-3 rounded-lg cursor-pointer transition-colors"
        :class="n.read ? 'opacity-60' : 'bg-cyan/5 border border-cyan/10'"
        :aria-label="`标记通知为已读：${n.title}`">
        <div class="flex items-start gap-3">
          <component :is="typeIcon(n.type)" class="w-5 h-5 shrink-0 mt-0.5" :class="typeColor(n.type)" />
          <div class="flex-1 min-w-0">
            <p class="text-sm font-medium">{{ notificationTitle(n.type) }}</p>
            <p class="text-xs text-text-secondary mt-0.5">{{ notificationMessage(n.type) }}</p>
            <p class="text-xs text-text-muted mt-1">{{ formatRelativeTime(n.created_at) }}</p>
          </div>
          <div v-if="!n.read" class="w-2 h-2 rounded-full bg-cyan shrink-0 mt-1.5" />
        </div>
      </button>
      <EmptyState
        v-if="!notifications.length"
        icon="Bell"
        title="暂无通知"
        description="有新消息时会在这里显示"
      />
    </div>
    <template #footer>
      <button type="button" v-if="unreadCount > 0" @click="markAllRead" class="text-sm text-cyan hover:text-cyan-bright transition-colors">
        全部标记已读
      </button>
    </template>
  </Modal>
</template>

<script setup lang="ts">
import { computed, type Component } from 'vue'
import Modal from './Modal.vue'
import EmptyState from './EmptyState.vue'
import { useNotificationStore } from '@/stores/notification'
import { formatRelativeTime } from '@/utils/format'
import { CheckCircle, AlertTriangle, XCircle, Info, Bell } from 'lucide-vue-next'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const visible = computed({ get: () => props.modelValue, set: v => emit('update:modelValue', v) })
const store = useNotificationStore()
// 只保留需要用户处理的事件；任务成功、普通采集完成和泛系统消息不进入通知中心。
const actionableTypes = new Set(['task_failed', 'account_risk', 'account_offline', 'plan_expiring'])
const notifications = computed(() => store.notifications.filter(n => actionableTypes.has(n.type)))
const unreadCount = computed(() => notifications.value.filter(n => !n.read).length)

function typeIcon(type: string): Component {
  const map: Record<string, unknown> = {
    task_completed: CheckCircle, task_failed: XCircle,
    account_risk: AlertTriangle, account_offline: XCircle,
    plan_expiring: Info, system: Info, scrape_completed: CheckCircle
  }
  return map[type] || Bell
}

function typeColor(type: string) {
  const map: Record<string, string> = {
    task_completed: 'text-emerald', task_failed: 'text-rose',
    account_risk: 'text-amber', account_offline: 'text-rose',
    plan_expiring: 'text-amber', system: 'text-cyan', scrape_completed: 'text-emerald'
  }
  return map[type] || 'text-text-secondary'
}

function notificationTitle(type: string): string {
  const labels: Record<string, string> = {
    task_failed: '任务执行失败',
    account_risk: '账号风险提醒',
    account_offline: '账号连接中断',
    plan_expiring: '套餐即将到期',
  }
  return labels[type] || '系统提醒'
}

function notificationMessage(type: string): string {
  const messages: Record<string, string> = {
    task_failed: '任务未能完成，请前往任务列表查看失败原因。',
    account_risk: '账号状态可能受到限制，请前往账号管理检查。',
    account_offline: '账号当前已断开连接，可在账号管理中重新连接。',
    plan_expiring: '套餐有效期即将结束，请前往设置查看到期时间。',
  }
  return messages[type] || '请打开相关页面查看详情。'
}

function markRead(id: string) { store.markRead(id) }
function markAllRead() { notifications.value.forEach(n => store.markRead(n.id)) }
</script>
