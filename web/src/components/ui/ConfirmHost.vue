<template>
  <Modal v-model="confirmState.visible" :title="confirmState.title" :z-index="3000">
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
</script>
