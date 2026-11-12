<template>
  <span>{{ display }}</span>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'

// 数字滚动组件: 数值变化时从旧值平滑滚到新值, 800ms 缓动
const props = withDefaults(defineProps<{
  value: number | string
  duration?: number
}>(), { duration: 800 })

const display = ref('0')
let raf = 0

function animateTo(target: number) {
  const start = parseFloat(String(display.value).replace(/[^\d.-]/g, '')) || 0
  if (start === target) { display.value = String(target); return }
  cancelAnimationFrame(raf)
  const t0 = performance.now()
  const tick = (now: number) => {
    const p = Math.min(1, (now - t0) / props.duration)
    const eased = 1 - Math.pow(1 - p, 3) // easeOutCubic
    display.value = String(Math.round(start + (target - start) * eased))
    if (p < 1) raf = requestAnimationFrame(tick)
  }
  raf = requestAnimationFrame(tick)
}

// P1-27/P2-54: 原 Number('¥0')=NaN → 今日收入恒显示 0。统一剥掉货币前缀/
// 千分位/单位后缀再 parseFloat, 任何带单位字符串都得数字。
function parseCountValue(v: unknown): number {
  if (typeof v === 'number') return Number.isFinite(v) ? v : 0
  // 先去掉千分位，再提取完整数值，保留正负号、小数和科学计数法。
  const normalized = String(v ?? '').replace(/,(?=\\d{3}(?:\\D|$))/g, '')
  const match = normalized.match(/[+-]?(?:\\d+\\.?\\d*|\\.\\d+)(?:e[+-]?\\d+)?/i)
  if (!match) return 0
  const n = Number(match[0])
  return Number.isFinite(n) ? n : 0
}

onMounted(() => animateTo(parseCountValue(props.value)))
watch(() => props.value, (v) => animateTo(parseCountValue(v)))
</script>
