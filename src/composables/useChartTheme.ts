// ECharts 使用 Canvas 渲染，不能直接消费 CSS 变量。
// 从当前三主题的真实设计 Token 读取颜色，并在主题 class 变化时刷新。
import { ref } from 'vue'

function readVar(name: string, fallback: string): string {
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return v || fallback
}

export function useChartTheme() {
  const palette = ref<string[]>([
    '#5b62f6', '#15966b', '#d58a22', '#d95757', '#4a51e0', '#5d6678',
  ])
  const textPrimary = ref('#192234')
  const textMuted = ref('#8a93a3')
  const border = ref('#e8ebf2')
  const surface = ref('#ffffff')

  function refresh() {
    palette.value = [
      readVar('--t-brand', '#5b62f6'),
      readVar('--t-ok', '#15966b'),
      readVar('--t-warn', '#d58a22'),
      readVar('--t-err', '#d95757'),
      readVar('--t-brand-hover', '#4a51e0'),
      readVar('--t-text-2', '#5d6678'),
    ]
    textPrimary.value = readVar('--t-text-1', '#192234')
    textMuted.value = readVar('--t-text-3', '#8a93a3')
    border.value = readVar('--t-line', '#e8ebf2')
    surface.value = readVar('--t-card', '#ffffff')
  }

  let observer: MutationObserver | null = null
  function start() {
    refresh()
    observer?.disconnect()
    observer = new MutationObserver(refresh)
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  }
  function stop() {
    observer?.disconnect()
    observer = null
  }

  return { palette, textPrimary, textMuted, border, surface, start, stop }
}
