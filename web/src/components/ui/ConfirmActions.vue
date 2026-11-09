<template>
  <div class="confirm-actions">
    <UiButton
      tone="secondary"
      @click="emit('cancel')"
    >
      {{ cancelText }}
    </UiButton>
    <UiButton
      :tone="danger ? 'danger' : 'primary'"
      @click="emit('confirm')"
    >
      {{ confirmText }}
    </UiButton>
  </div>
</template>

<script setup lang="ts">
import UiButton from './UiButton.vue'

withDefaults(defineProps<{
  danger?: boolean
  confirmText?: string
  cancelText?: string
}>(), {
  danger: false,
  confirmText: '确认',
  cancelText: '取消',
})

const emit = defineEmits<{
  (event: 'confirm'): void
  (event: 'cancel'): void
}>()
</script>

<style scoped>
.confirm-actions {
  display: flex;
  width: 100%;
  justify-content: flex-end;
  gap: var(--t-space-2);
}
@media (max-width: 768px) {
  .confirm-actions {
    flex-direction: column-reverse;
  }
  .confirm-actions :deep(.t-button) {
    width: 100%;
    min-height: 44px;
  }
}
</style>
