<template>
  <Modal :model-value="confirmState.visible" :title="confirmState.title" :z-index="3000" @update:model-value="handleVisibilityChange">
    <ConfirmPrompt :message="confirmState.message" :detail="confirmState.detail" />
    <template #footer>
      <ConfirmActions
        :danger="confirmState.danger"
        :confirm-text="confirmState.confirmText"
        :cancel-text="confirmState.cancelText"
        @cancel="settle(false)"
        @confirm="settle(true)"
      />
    </template>
  </Modal>
</template>

<script setup lang="ts">
import ConfirmPrompt from './ConfirmPrompt.vue'
import ConfirmActions from './ConfirmActions.vue'
import Modal from './Modal.vue'
import { confirmState, settle } from '@/utils/confirm'

// Overlay click and Escape are dismissals too; resolve the pending ask(false)
// instead of only hiding the modal and leaving its Promise pending forever.
function handleVisibilityChange(visible: boolean) {
  if (!visible && confirmState.visible) settle(false)
}
</script>
