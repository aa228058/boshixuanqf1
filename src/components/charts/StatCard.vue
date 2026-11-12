<template>
  <!-- R87-M7: KPI 卡改 demo 范式"纯文字三层"(label/大数字/趋势), 去掉彩色图标块 —— 色彩纪律: 数字才是主角 -->
  <div class="glass rounded-[var(--t-r-lg)] p-3 sm:p-4 min-w-0 transition-[transform,border-color] duration-300 lift">
    <p class="text-[10.5px] font-medium text-text-muted truncate">{{ label }}</p>
    <p class="text-title leading-tight font-bold text-text-primary tracking-tight mt-1.5"><CountUp :value="value" /></p>
    <p v-if="trend" class="text-[10.5px] mt-1 font-medium" :class="trendClass">{{ trend }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import CountUp from '@/components/ui/CountUp.vue'

const props = defineProps<{
  label: string
  value: string
  icon: string
  color: string
  trend?: string
}>()

const trendClass = computed(() => {
  if (!props.trend) return ''
  return props.trend.startsWith('+') ? 'text-emerald' : 'text-text-muted'
})
</script>
