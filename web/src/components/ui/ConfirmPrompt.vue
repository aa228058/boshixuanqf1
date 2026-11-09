<template>
  <div class="flex items-start gap-3">
    <AlertTriangle class="w-6 h-6 text-amber shrink-0 mt-0.5" />
    <div class="min-w-0">
      <p class="text-text-primary">{{ message }}</p>
      <p v-if="detail" class="text-sm text-text-secondary mt-2">{{ detail }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { AlertTriangle } from 'lucide-vue-next'

defineProps<{
  message: string
  detail?: string
}>()
</script>
