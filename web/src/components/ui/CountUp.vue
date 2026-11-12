<template>
  <span>{{ display }}</span>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import { formatCountValue, parseCountValue } from './countUp'

// 数字滚动组件: 数值变化时从旧值平滑滚到新值, 800ms 缓动
const props = withDefaults(defineProps<{
  value: number | string
  duration?: number
}>(), { duration: 800 })

const display = ref('0')
let currentValue = 0
let raf = 0

function animateTo(target: number) {
  const start = currentValue
  cancelAnimationFrame(raf)
  if (start === target) { currentValue = target; display.value = formatCountValue(props.value, target); return }
  const duration = Number.isFinite(props.duration) ? Math.max(0, props.duration) : 800
  if (duration === 0) { currentValue = target; display.value = formatCountValue(props.value, target); return }
  const precisionOf = (value: number) => {
    const fraction = String(value).toLowerCase().split('e')[0].split('.')[1] || ''
    return Math.min(6, fraction.length)
  }
  const precision = Math.max(precisionOf(start), precisionOf(target))
  const t0 = performance.now()
  const tick = (now: number) => {
    const p = Math.min(1, (now - t0) / duration)
    const eased = 1 - Math.pow(1 - p, 3) // easeOutCubic
    const current = start + (target - start) * eased
    currentValue = current
    display.value = formatCountValue(props.value, Number(current.toFixed(precision)))
    if (p < 1) raf = requestAnimationFrame(tick)
    else { currentValue = target; display.value = formatCountValue(props.value, target) }
  }
  raf = requestAnimationFrame(tick)
}

onMounted(() => animateTo(parseCountValue(props.value)))
watch(() => [props.value, props.duration], () => animateTo(parseCountValue(props.value)))
onUnmounted(() => cancelAnimationFrame(raf))
</script>
