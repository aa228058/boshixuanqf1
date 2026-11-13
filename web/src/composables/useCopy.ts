import { ref, onUnmounted } from 'vue'

// ===== 复制反馈(交互层: 原地变勾, 不弹 Toast 打扰) =====
// 用法: const { copied, copy } = useCopy()
//   <button @click="copy(apiKey)">{{ copied ? '已复制 ✓' : '复制' }}</button>
export function copyWithExecCommand(text: string, doc: Document = document): boolean {
  let textarea: HTMLTextAreaElement | null = null
  try {
    textarea = doc.createElement('textarea')
    textarea.value = text
    textarea.setAttribute('readonly', '')
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    doc.body.appendChild(textarea)
    textarea.select()
    return doc.execCommand('copy')
  } catch {
    return false
  } finally {
    // execCommand/select may throw; never leave a hidden textarea in the page.
    if (textarea?.parentNode) textarea.parentNode.removeChild(textarea)
  }
}

export function useCopy(resetMs = 1600) {
  const copied = ref(false)
  let timer: ReturnType<typeof setTimeout> | null = null
  let disposed = false

  function showCopied() {
    if (disposed) return
    copied.value = true
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => {
      copied.value = false
      timer = null
    }, Math.max(0, resetMs))
  }

  async function copy(text: string): Promise<boolean> {
    try {
      await navigator.clipboard.writeText(text)
      showCopied()
      return !disposed
    } catch {
      // 剪贴板 API 不可用(非安全上下文)时退回 execCommand
      const succeeded = copyWithExecCommand(text)
      if (!succeeded) return false
      showCopied()
      return !disposed
    }
  }

  onUnmounted(() => {
    disposed = true
    if (timer) clearTimeout(timer)
    timer = null
  })

  return { copied, copy }
}
