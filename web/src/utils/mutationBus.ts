// 全局变更总线：任何非 GET 请求成功后广播，相关 store 自动失效重刷。
// 解决"每个页面自己记得刷新"的漏刷问题——新增功能不用手动接刷新。

export type InvalidateFn = () => void

type MutationSubscription = { pattern: RegExp; fn: InvalidateFn }
const registry = new Set<MutationSubscription>()

/** 注册一个 URL 模式与对应的失效回调；返回取消订阅函数。 */
export function onMutation(pattern: RegExp, fn: InvalidateFn): () => void {
  const subscription = { pattern, fn }
  registry.add(subscription)
  return () => {
    registry.delete(subscription)
  }
}

/** 广播变更事件（request.ts 内部调用） */
export function emitMutation(method: string, url: string): void {
  if (method.toUpperCase() === 'GET') return
  // 快照遍历：回调可以安全地在广播期间取消自身或其他订阅。
  for (const { pattern, fn } of [...registry]) {
    // global/sticky 正则的 test() 会修改 lastIndex；每次匹配都从头开始，
    // 避免同一订阅在连续请求中交替漏报。
    pattern.lastIndex = 0
    const matched = pattern.test(url)
    pattern.lastIndex = 0
    if (matched) {
      try { fn() } catch { /* 单个订阅者失败不影响其他 */ }
    }
  }
}
