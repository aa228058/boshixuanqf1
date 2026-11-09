<template>
  <div class="flex items-start gap-3">
    <AlertTriangle class="w-6 h-6 text-amber shrink-0 mt-0.5" />
    <div class="min-w-0">
      <p class="text-text-primary">{{ message }}</p>
      <p v-if="detail" class="text-sm text-text-secondary mt-2">{{ detail }}</p>
    </div>
  </div>
  <template v-if="showActions">
    <div class="confirm-prompt__actions">
      <t-button
        type="button"
        variant="outline"
        class="ui-t-button ui-t-button--secondary"
        @click="emit('cancel')"
      >
        {{ cancelText }}
      </t-button>
      <t-button
        type="button"
        :theme="danger ? 'danger' : 'primary'"
        :class="['ui-t-button', danger ? 'ui-t-button--danger' : 'ui-t-button--primary']"
        @click="emit('confirm')"
      >
        {{ confirmText }}
      </t-button>
    </div>
  </template>
</template>

<script setup lang="ts">
import { AlertTriangle } from 'lucide-vue-next'

withDefaults(defineProps<{
  message: string
  detail?: string
  danger?: boolean
  confirmText?: string
  cancelText?: string
  showActions?: boolean
}>(), {
  danger: false,
  confirmText: '确认',
  cancelText: '取消',
  showActions: true,
})

const emit = defineEmits<{
  (event: 'confirm'): void
  (event: 'cancel'): void
}>()
</script>

<style scoped>
.confirm-prompt__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--t-space-2);
  margin-top: var(--t-space-5);
}
@media (max-width: 768px) {
  .confirm-prompt__actions {
    flex-direction: column-reverse;
  }
  .confirm-prompt__actions :deep(.t-button) {
    width: 100%;
    min-height: 44px;
  }
}
</style>
