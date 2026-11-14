import { ref, watch } from 'vue'
import { accountApi } from '@/api/account'

export interface ProbeResult {
  items: any[]
  verdict: string
  checked_at: string
}

/**
 * 账号探测（在线/死号检测）弹窗的状态与动作。
 * 从 accounts/List.vue 抽出：轮询式请求 + 过期响应丢弃逻辑自成一类，
 * 与列表本身的筛选/分页无关。
 */
export function useAccountProbe() {
  const showProbe = ref(false)
  const probeAccount = ref<any>(null)
  const probeResult = ref<ProbeResult | null>(null)
  const probeRunning = ref(false)
  const probeCooldownText = ref('') // 冷却中提示(后端 429 返回)

  let probeRequestId = 0
  let probeController: AbortController | null = null

  watch(showProbe, (visible) => {
    if (!visible) {
      probeController?.abort()
      probeController = null
      probeRequestId++
      probeRunning.value = false
    }
  })

  async function openProbe(acc: any) {
    probeController?.abort()
    const controller = new AbortController()
    probeController = controller
    const requestId = ++probeRequestId
    const accountId = acc?.account_id
    probeAccount.value = acc
    probeResult.value = null
    probeCooldownText.value = ''
    showProbe.value = true
    if (!accountId) return
    // 先展示上次结果(不触发检测)；切换账号或关闭弹窗后丢弃过期响应。
    try {
      const cached = await accountApi.getProbe(accountId, { signal: controller.signal })
      if (requestId === probeRequestId && showProbe.value && probeAccount.value?.account_id === accountId) {
        probeResult.value = (cached as ProbeResult) || null
      }
    } catch { /* 无历史结果, 忽略 */ }
    finally { if (probeController === controller) probeController = null }
  }

  async function runProbe() {
    const accountId = probeAccount.value?.account_id
    if (!accountId || probeRunning.value) return
    const requestId = probeRequestId
    probeRunning.value = true
    probeCooldownText.value = ''
    try {
      const result = await accountApi.runProbe(accountId)
      if (requestId === probeRequestId && showProbe.value && probeAccount.value?.account_id === accountId) {
        probeResult.value = (result as ProbeResult) || null
      }
    } catch (e: any) {
      // 冷却中: 429 带文案; 其他错误拦截器已 toast
      const msg = e?.response?.data?.message || e?.message || ''
      if (requestId === probeRequestId && showProbe.value && probeAccount.value?.account_id === accountId && msg.includes('冷却')) {
        probeCooldownText.value = msg
      }
    } finally {
      if (requestId === probeRequestId) probeRunning.value = false
    }
  }

  /** 页面卸载时调用：中止在途请求并让迟到的响应作废。 */
  function disposeProbe() {
    probeController?.abort()
    probeController = null
    probeRequestId++
    probeRunning.value = false
  }

  return { showProbe, probeAccount, probeResult, probeRunning, probeCooldownText, openProbe, runProbe, disposeProbe }
}
