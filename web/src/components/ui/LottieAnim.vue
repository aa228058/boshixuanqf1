<template>
  <div ref="el" :style="{ width, height }" />
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import lottie from 'lottie-web'

// 构建时静态分析 src/assets/lottie/ 下全部 JSON(Vite 正确解析 @ 别名与哈希)
// 禁止运行时 import(变量路径) 或 lottie path 模式(内置 XHR 在部分 WebView 读 responseText 抛 InvalidStateError)
const localModules = import.meta.glob('@/assets/lottie/*.json')

const props = withDefaults(defineProps<{
  src: string
  width?: string
  height?: string
  loop?: boolean
  autoplay?: boolean
}>(), { width: '100%', height: '100%', loop: true, autoplay: true })

const el = ref<HTMLDivElement>()
let anim: any = null
let disposed = false
let loadId = 0
let controller: AbortController | null = null

async function loadAnimation() {
  const id = ++loadId
  controller?.abort()
  controller = null
  anim?.destroy()
  anim = null
  const box = el.value
  if (!box || disposed) return
  try {
    let data: any = null
    if (props.src.startsWith('@/')) {
      // 本地资源: glob 构建时解析, key 为 '/src/assets/...' 形式
      const key = '/src/' + props.src.slice(2)
      const loader = (localModules as Record<string, () => Promise<any>>)[key]
      if (loader) data = (await loader()).default
    } else {
      // public / 远程: fetch 拿对象再传给 lottie, 绕开其内置 XHR
      controller = new AbortController()
      const res = await fetch(props.src, { signal: controller.signal })
      if (res.ok) data = await res.json()
    }
    if (disposed || id !== loadId) return
    if (!data) { box.style.display = 'none'; return }
    box.style.display = ''
    anim = lottie.loadAnimation({
      container: box,
      renderer: 'svg',
      loop: props.loop,
      autoplay: props.autoplay,
      animationData: data,
    })
  } catch (e) {
    if (disposed || id !== loadId || (e instanceof Error && e.name === 'AbortError')) return
    console.warn('[LottieAnim] 加载失败:', e)
    box.style.display = 'none'
  } finally {
    if (id === loadId) controller = null
  }
}

onMounted(() => { void loadAnimation() })
watch(() => props.src, () => { void loadAnimation() })

onUnmounted(() => {
  disposed = true
  loadId++
  controller?.abort()
  controller = null
  anim?.destroy()
  anim = null
})
</script>
