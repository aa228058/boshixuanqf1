<template>
  <Modal v-model="confirmState.visible" :title="confirmState.title" :z-index="3000">
    <ConfirmPrompt :message="confirmState.message" :detail="confirmState.detail" />
    <template #footer>
      <t-button
        type="button"
        variant="outline"
        class="ui-t-button ui-t-button--secondary"
        @click="settle(false)"
      >
        {{ confirmState.cancelText }}
      </t-button>
      <t-button
        type="button"
        :theme="confirmState.danger ? 'danger' : 'primary'"
        :class="['ui-t-button', confirmState.danger ? 'ui-t-button--danger' : 'ui-t-button--primary']"
        @click="settle(true)"
      >
        {{ confirmState.confirmText }}
      </t-button>
    </template>
  </Modal>
</template>

<script setup lang="ts">
import ConfirmPrompt from './ConfirmPrompt.vue'
import Modal from './Modal.vue'
import { confirmState, settle } from '@/utils/confirm'
</script>
