// useElasticScroll —— R127 iOS 橡皮筋回弹(安卓补体感)
// 效果: 滚动到顶/底后继续滑动, 容器阻尼跟手(0.35x), 松手弹性回弹。
// 用法: 在滚动容器的父级调用 useElasticScroll(scrollElRef)
import { onMounted, onUnmounted, type Ref } from 'vue'

export function useElasticScroll(elRef: Ref<HTMLElement | null>) {
  let startY = 0
  let startScrollTop = 0
  let pulling = false
  let resetTimer: ReturnType<typeof setTimeout> | null = null
  let originalTransition = ''
  let originalTransform = ''
  let originalOverscrollBehavior = ''

  const clearResetTimer = () => {
    if (resetTimer) clearTimeout(resetTimer)
    resetTimer = null
  }

  const restoreStyles = (el: HTMLElement) => {
    el.style.transition = originalTransition
    el.style.transform = originalTransform
    el.style.overscrollBehavior = originalOverscrollBehavior
    pulling = false
  }

  const onTouchStart = (e: TouchEvent) => {
    const el = elRef.value
    if (!el) return
    const touch = e.touches[0]
    if (!touch) return
    clearResetTimer()
    startY = touch.clientY
    startScrollTop = el.scrollTop
    pulling = false
    originalTransition = el.style.transition
    originalTransform = el.style.transform
    originalOverscrollBehavior = el.style.overscrollBehavior
  }

  const onTouchMove = (e: TouchEvent) => {
    const el = elRef.value
    if (!el) return
    const deltaY = e.touches[0].clientY - startY
    const atTop = el.scrollTop <= 0 && deltaY > 0 && startScrollTop <= 0
    const atBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 1 && deltaY < 0
    if (atTop || atBottom) {
      if (!pulling) { pulling = true; el.style.transition = 'none' }
      // 阻尼: 只跟手 1/3, 行程越大阻力越明显(平方衰减)
      const damped = deltaY * 0.35 * (1 - Math.min(Math.abs(deltaY) / 800, 0.6))
      el.style.transform = `translateY(${damped}px)`
      el.style.overscrollBehavior = 'none'
    } else if (pulling) {
      pulling = false
      el.style.transform = ''
    }
  }

  const onTouchEnd = () => {
    const el = elRef.value
    if (!el || !pulling) return
    pulling = false
    el.style.transition = 'transform 0.38s cubic-bezier(0.22, 1, 0.36, 1)'
    el.style.transform = originalTransform
    el.style.overscrollBehavior = originalOverscrollBehavior
    clearResetTimer()
    resetTimer = setTimeout(() => {
      if (elRef.value === el) el.style.transition = originalTransition
      resetTimer = null
    }, 400)
  }

  onMounted(() => {
    const el = elRef.value
    if (!el) return
    el.addEventListener('touchstart', onTouchStart, { passive: true })
    el.addEventListener('touchmove', onTouchMove, { passive: true })
    el.addEventListener('touchend', onTouchEnd)
    el.addEventListener('touchcancel', onTouchEnd)
  })
  onUnmounted(() => {
    const el = elRef.value
    if (!el) return
    el.removeEventListener('touchstart', onTouchStart)
    el.removeEventListener('touchmove', onTouchMove)
    el.removeEventListener('touchend', onTouchEnd)
    el.removeEventListener('touchcancel', onTouchEnd)
    clearResetTimer()
    restoreStyles(el)
  })
}
