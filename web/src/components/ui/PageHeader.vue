<template>
  <!-- 统一页头: 与更多页同一层级 — eyebrow(可选) + 标题 + 副文案 + 右侧操作区 -->
  <header class="flex items-end justify-between gap-3 min-w-0">
    <div class="min-w-0">
      <div v-if="eyebrow" class="text-[11px] font-extrabold tracking-[0.12em] uppercase" style="color: var(--t-text-3);">{{ eyebrow }}</div>
      <h1 class="text-display font-extrabold tracking-tight leading-tight truncate" :class="eyebrow ? 'mt-1' : ''" style="color: var(--t-text-1);">{{ title }}</h1>
      <p v-if="subtitle" class="text-caption mt-0.5 truncate" style="color: var(--t-text-3);">{{ subtitle }}</p>
    </div>
    <div v-if="$slots.actions" class="flex items-center gap-2 shrink-0">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  subtitle?: string
  eyebrow?: string
}>()
</script>
