<template>
  <div class="confirm-actions">
    <t-button
      type="button"
      theme="default"
      variant="outline"
      class="ui-t-button ui-t-button--secondary ui-t-button--sm"
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

<script setup lang="ts">
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
