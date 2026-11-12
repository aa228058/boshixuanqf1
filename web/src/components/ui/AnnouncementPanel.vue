<template>
  <Modal v-model="visible" title="公告通知">
    <div class="space-y-3 max-h-[500px] overflow-y-auto">
      <div
        v-for="a in announcements"
        :key="a.announcement_id"
        class="p-4 rounded-lg border"
        :class="a.priority === 'urgent' ? 'border-rose/30 bg-rose/5' : 'border-border bg-bg-card'"
      >
        <div class="flex items-center gap-2 mb-2">
          <span v-if="a.priority === 'urgent'" class="text-xs px-1.5 py-0.5 rounded bg-rose/10 text-rose font-medium">紧急</span>
          <span class="font-medium text-sm">{{ a.title }}</span>
          <span class="text-xs text-text-secondary ml-auto">{{ formatDate(a.created_at) }}</span>
        </div>
        <p class="text-sm text-text-secondary whitespace-pre-wrap max-w-[65ch] mx-auto text-center">{{ a.content }}</p>
      </div>
      <Loading v-if="loading" text="正在加载公告…" />
      <EmptyState
        v-else-if="loadError"
        icon="AlertTriangle"
        title="公告加载失败"
        description="暂时无法获取公告，请稍后重试或重新打开公告面板。"
      />
      <EmptyState
        v-else-if="!loading && !announcements.length"
        icon="Bell"
        title="暂无公告"
        description="管理员还没有发布公告"
      />
    </div>
  </Modal>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import Modal from './Modal.vue'
import EmptyState from './EmptyState.vue'
import Loading from './Loading.vue'
import { settingsApi } from '@/api/settings'
import { formatDate } from '@/utils/format'
import type { Announcement } from '@/types'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const visible = computed({ get: () => props.modelValue, set: v => emit('update:modelValue', v) })
const announcements = ref<Announcement[]>([])
const loading = ref(true)
const loadError = ref(false)
let disposed = false

async function loadAnnouncements() {
  loading.value = true
  loadError.value = false
  try {
    const res = await settingsApi.getAnnouncements()
    if (!disposed) announcements.value = Array.isArray(res) ? res : []
  } catch {
    if (!disposed) loadError.value = true
  } finally {
    if (!disposed) loading.value = false
  }
}

watch(() => props.modelValue, visible => {
  if (visible) void loadAnnouncements()
}, { immediate: true })
onUnmounted(() => { disposed = true })
</script>
