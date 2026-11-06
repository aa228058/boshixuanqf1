<template>
  <Modal v-model="confirmState.visible" :title="confirmState.title" :z-index="3000">
    <div class="flex items-start gap-3">
      <AlertTriangle class="w-6 h-6 text-amber shrink-0 mt-0.5" />
      <div>
        <p class="text-text-primary">{{ confirmState.message }}</p>
        <p v-if="confirmState.detail" class="text-sm text-text-secondary mt-2">{{ confirmState.detail }}</p>
      </div>
    </div>
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
import { AlertTriangle } from 'lucide-vue-next'
import Modal from './Modal.vue'
import { confirmState, settle } from '@/utils/confirm'
</script>
