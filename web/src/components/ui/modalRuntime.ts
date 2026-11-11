import type { Ref } from 'vue'

type ModalRuntimeEnvironment = {
  window?: Pick<Window, 'innerHeight' | 'visualViewport'>
  document?: Pick<Document, 'documentElement'>
}

/**
 * Owns the shared modal stack and mobile keyboard viewport guard.
 * The factory keeps the runtime isolated so its lifecycle can be unit-tested
 * without a browser or a mounted Vue component.
 */
export function createModalRuntime(
  getEnvironment: () => ModalRuntimeEnvironment = () => ({
    window: typeof window !== 'undefined' ? window : undefined,
    document: typeof document !== 'undefined' ? document : undefined,
  }),
) {
  let modalCounter = 0
  const activeModalOrders = new Map<symbol, Ref<number>>()
  const activeKeyboardModalIds = new Set<symbol>()
  const keyboardVar = '--modal-kb'
  let keyboardViewport: VisualViewport | null = null

  function compactModalOrders() {
    const active = [...activeModalOrders.entries()]
    active.forEach(([, order], index) => { order.value = index + 1 })
    modalCounter = active.length
  }

  function registerModal(id: symbol, order: Ref<number>) {
    if (!activeModalOrders.has(id)) {
      modalCounter += 1
      activeModalOrders.set(id, order)
    }
    order.value = modalCounter
  }

  function unregisterModal(id: symbol, order?: Ref<number>) {
    activeModalOrders.delete(id)
    compactModalOrders()
    if (order) order.value = 0
  }

  function getZIndex(order: number, customZIndex?: number) {
    return customZIndex ?? (2500 + order * 20)
  }

  function syncSharedKeyboardOffset() {
    const { window: browserWindow, document: browserDocument } = getEnvironment()
    const viewport = keyboardViewport
    if (!viewport || !browserWindow || !browserDocument) return
    const keyboardHeight = Math.max(0, browserWindow.innerHeight - viewport.height - viewport.offsetTop)
    browserDocument.documentElement.style.setProperty(keyboardVar, keyboardHeight > 80 ? `${keyboardHeight}px` : '0px')
  }

  function activateKeyboardGuard(id: symbol) {
    const browserWindow = getEnvironment().window
    if (!browserWindow?.visualViewport || activeKeyboardModalIds.has(id)) return
    activeKeyboardModalIds.add(id)
    if (activeKeyboardModalIds.size === 1) {
      keyboardViewport = browserWindow.visualViewport
      keyboardViewport.addEventListener('resize', syncSharedKeyboardOffset)
      keyboardViewport.addEventListener('scroll', syncSharedKeyboardOffset)
    }
    syncSharedKeyboardOffset()
  }

  function deactivateKeyboardGuard(id: symbol) {
    if (!activeKeyboardModalIds.delete(id) || activeKeyboardModalIds.size > 0) return
    keyboardViewport?.removeEventListener('resize', syncSharedKeyboardOffset)
    keyboardViewport?.removeEventListener('scroll', syncSharedKeyboardOffset)
    keyboardViewport = null
    getEnvironment().document?.documentElement.style.setProperty(keyboardVar, '0px')
  }

  return {
    registerModal,
    unregisterModal,
    getZIndex,
    activateKeyboardGuard,
    deactivateKeyboardGuard,
  }
}

export const modalRuntime = createModalRuntime()
