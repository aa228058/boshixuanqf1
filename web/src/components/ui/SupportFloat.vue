<template>
  <!-- R182.6: 未配置客服联系方式时不渲染悬浮球——空悬浮按钮无功能还遮挡内容 -->
  <div
    v-if="hasContact"
    ref="wrapper"
    class="fixed z-30"
    :style="{ right: pos.right + 'px', bottom: pos.bottom + 'px' }"
  >
    <Transition name="support">
      <div v-if="expanded" class="mb-3 glass rounded-[var(--t-r-md)] p-4 w-72 elev-pop border border-border">
        <div class="flex items-center justify-between mb-3">
          <div class="flex items-center gap-2">
            <div class="w-7 h-7 rounded-lg bg-cyan/15 flex items-center justify-center">
              <component :is="contactIcon" class="w-3.5 h-3.5 text-cyan" />
            </div>
            <h4 class="font-medium text-sm">{{ config?.display_name || '联系客服' }}</h4>
          </div>
          <button type="button" aria-label="关闭客服面板" @click="expanded = false" class="w-6 h-6 rounded-md hover:bg-milk-01 flex items-center justify-center text-text-muted hover:text-text-primary transition-colors">
            <X class="w-3.5 h-3.5" />
          </button>
        </div>
        <div class="space-y-3">
          <div class="flex items-center gap-2 p-2.5 rounded-lg bg-ink-01 border border-border/50">
            <component :is="contactIcon" class="w-4 h-4 text-cyan shrink-0" />
            <span class="text-sm text-text-secondary truncate">{{ config?.contact_value }}</span>
            <button type="button" @click="copyContact" class="px-2 py-1 rounded-md bg-cyan/10 text-cyan hover:bg-cyan/20 text-xs font-medium ml-auto shrink-0 transition-colors">
              {{ copied ? '已复制' : '复制' }}
            </button>
          </div>
          <div v-if="config?.qr_code_url" class="flex justify-center p-2">
            <img
              :src="safeQrUrl(config.qr_code_url)"
              alt="客服二维码"
              class="w-36 h-36 rounded-[var(--t-r-md)] border border-milk-01"
              @error="(e) => { const t = e.target as HTMLElement; if (t) t.style.display = 'none' }"
            />
          </div>
          <div class="border-t border-border/50 pt-3">
            <router-link to="/settings" class="text-xs text-cyan hover:text-cyan-bright transition-colors block text-center py-1">
              已有卡密？点击激活套餐
            </router-link>
          </div>
        </div>
      </div>
    </Transition>

    <!-- 悬浮按钮 -->
    <button type="button"
      ref="dragBtn"
      @mousedown="startDrag"
      @touchstart="startDrag"
      @click="toggle"
      :aria-expanded="expanded"
      aria-label="联系客服"
      class="relative w-11 h-11 rounded-full flex items-center justify-center cursor-move select-none touch-none group"
    >
      <!-- 脉冲光环 -->
      <span class="absolute inset-0 rounded-full bg-cyan/20 animate-ping-slow" />
      <span class="absolute inset-0 rounded-full bg-cyan/10 animate-ping-slow delay-500" />
      <!-- 按钮主体 -->
      <span class="absolute inset-0 rounded-full bg-cyan/75 backdrop-blur-sm shadow-md shadow-cyan/20 group-hover:bg-cyan/90 group-hover:shadow-cyan/30 group-hover:scale-105 transition-[transform,opacity,border-color,background-color] duration-300" />
      <MessageCircle class="relative w-5 h-5 text-white" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from 'vue'
import type { Component } from 'vue'
import { MessageCircle, X, MessageSquare, Mail, Phone, Globe } from 'lucide-vue-next'
import { settingsApi } from '@/api/settings'
import { useAuthStore } from '@/stores/auth'
import { toast } from '@/utils/toast'
import type { SupportConfig } from '@/types'

const expanded = ref(false)
const config = ref<SupportConfig | null>(null)
const copied = ref(false)
let copyTimer: ReturnType<typeof setTimeout> | null = null
let dragResetTimer: ReturnType<typeof setTimeout> | null = null
let disposed = false
const authStore = useAuthStore()

const pos = ref({ right: 20, bottom: 20 })
const isDragging = ref(false)
const dragStart = ref({ x: 0, y: 0, right: 0, bottom: 0 })

const wrapper = ref<HTMLElement | null>(null)
const dragBtn = ref<HTMLElement | null>(null)

function startDrag(e: MouseEvent | TouchEvent) {
  if ('touches' in e && e.touches.length !== 1) {
    stopDrag()
    return
  }
  if (dragResetTimer) clearTimeout(dragResetTimer)
  dragResetTimer = null
  isDragging.value = false
  const clientX = 'touches' in e ? e.touches[0].clientX : e.clientX
  const clientY = 'touches' in e ? e.touches[0].clientY : e.clientY
  dragStart.value = {
    x: clientX,
    y: clientY,
    right: pos.value.right,
    bottom: pos.value.bottom,
  }

  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
  document.addEventListener('touchmove', onDrag, { passive: false })
  document.addEventListener('touchend', stopDrag)
  document.addEventListener('touchcancel', stopDrag)
}

function onDrag(e: MouseEvent | TouchEvent) {
  let clientX: number
  let clientY: number
  if ('touches' in e) {
    // Some browsers dispatch a final touchmove with no active touches while
    // a system gesture is being cancelled. Never index an empty TouchList.
    const touch = e.touches[0] || e.changedTouches[0]
    if (!touch) {
      stopDrag()
      return
    }
    clientX = touch.clientX
    clientY = touch.clientY
  } else {
    clientX = e.clientX
    clientY = e.clientY
  }
  const dx = dragStart.value.x - clientX
  const dy = dragStart.value.y - clientY

  if (Math.abs(dx) > 3 || Math.abs(dy) > 3) {
    isDragging.value = true
  }

  if (isDragging.value) {
    e.preventDefault()
    pos.value.right = Math.max(0, Math.min(window.innerWidth - 44, dragStart.value.right + dx))
    pos.value.bottom = Math.max(0, Math.min(window.innerHeight - 44, dragStart.value.bottom + dy))
  }
}

function stopDrag() {
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
  document.removeEventListener('touchmove', onDrag)
  document.removeEventListener('touchend', stopDrag)
  document.removeEventListener('touchcancel', stopDrag)
  if (dragResetTimer) clearTimeout(dragResetTimer)
  dragResetTimer = setTimeout(() => {
    isDragging.value = false
    dragResetTimer = null
  }, 50)
}

function toggle() {
  if (isDragging.value) return
  expanded.value = !expanded.value
}

const contactIcon = computed<Component>(() => {
  const m: Record<string, Component> = {
    telegram: MessageSquare,
    email: Mail,
    phone: Phone,
    website: Globe,
  }
  return m[config.value?.contact_type || ''] || MessageCircle
})

// R182.6: 客服联系方式为空 → 不渲染悬浮球(空按钮碍眼且无功能)
const hasContact = computed(() => !!config.value?.contact_value)

// P1-32: 原 navigator.clipboard 无 ?.——非 HTTPS 环境 clipboard 为 undefined,
// 直接抛 TypeError 触发全局红条。加 ?. + execCommand 兜底。
function fallbackCopy(text: string): boolean {
  const ta = document.createElement('textarea')
  ta.value = text
  ta.setAttribute('readonly', '')
  ta.style.position = 'fixed'
  ta.style.opacity = '0'
  document.body.appendChild(ta)
  ta.select()
  let succeeded = false
  try { succeeded = document.execCommand('copy') } catch { /* 浏览器可能禁用旧式复制接口 */ }
  document.body.removeChild(ta)
  return succeeded
}

function showCopyResult(succeeded: boolean) {
  if (copyTimer) clearTimeout(copyTimer)
  copied.value = succeeded
  toast({
    type: succeeded ? 'success' : 'error',
    title: succeeded ? '已复制到剪贴板' : '复制失败，请手动复制',
  })
  if (succeeded) copyTimer = setTimeout(() => { copied.value = false }, 2000)
}

async function copyContact() {
  const value = config.value?.contact_value
  if (!value) return

  try {
    if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(value)
      showCopyResult(true)
      return
    }
  } catch {
    // 权限拒绝或非安全上下文时尝试兼容性回退。
  }

  showCopyResult(fallbackCopy(value))
}

// P1-25: 原 safeQrUrl 只判 http 前缀——客服二维码存 base64(data:image/...)时
// 被拼成 /api/v1data:image/... 坏图。data: 开头的直接返回。
function safeQrUrl(url: string): string {
  if (url.startsWith('data:')) return url
  if (url.startsWith('http')) return url
  return `${import.meta.env.VITE_API_BASE_URL || ''}${url}`
}

onMounted(async () => {
  try {
    const res = await settingsApi.getSupport()
    if (!disposed) config.value = res
  } catch (e) {
    // ignore
  }
})

onUnmounted(() => {
  disposed = true
  if (copyTimer) clearTimeout(copyTimer)
  if (dragResetTimer) clearTimeout(dragResetTimer)
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
  document.removeEventListener('touchmove', onDrag)
  document.removeEventListener('touchend', stopDrag)
  document.removeEventListener('touchcancel', stopDrag)
})
</script>

<style scoped>
.support-enter-active, .support-leave-active {
  transition: opacity 0.3s cubic-bezier(0.32, 0.72, 0, 1), transform 0.3s cubic-bezier(0.32, 0.72, 0, 1);
}
.support-enter-from, .support-leave-to {
  opacity: 0;
  transform: translateY(8px) scale(0.96);
}

/* P1-09: 原动画 0.3s infinite——客服按钮每秒闪 3 次以上, 违反无意义动画原则。
   放缓到 2s 呼吸节奏; delay-500(0.8s) 同步为 2s 周期的相位差。 */
@keyframes ping-slow {
  0% { transform: scale(1); opacity: 0.4; }
  50% { opacity: 0.15; }
  100% { transform: scale(2.2); opacity: 0; }
}
/* P1-09: 0.3s infinite → 2s 呼吸节奏(客服按钮不再高频闪烁) */
.animate-ping-slow {
  animation: ping-slow 2s cubic-bezier(0.32, 0.72, 0, 1) infinite;
}
.delay-500 {
  animation-delay: 0.6s;
}
</style>
