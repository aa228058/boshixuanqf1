<template>
  <div role="status" aria-live="polite" :aria-label="text || '加载中'" class="flex flex-col items-center justify-center py-12 gap-4">
    <div aria-hidden="true" class="loading-spinner w-8 h-8 rounded-full border-[4px] border-border border-t-cyan animate-spin" />
    <span v-if="text" class="text-sm text-text-secondary font-medium">{{ text }}</span>
  </div>
</template>

<script setup lang="ts">
defineProps<{ text?: string }>()
</script>

<style scoped>
@media (prefers-reduced-motion: reduce) {
  .loading-spinner { animation: none; }
}
</style>
