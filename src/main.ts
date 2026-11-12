import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createVfm } from 'vue-final-modal'
import router from './router'
import App from './App.vue'
// 第三方组件样式先加载，项目设计令牌与覆盖层后加载，避免 TDesign 默认按钮覆盖统一规范。
import 'tdesign-vue-next/es/style/index.css'
import 'vue-final-modal/style.css'
import './styles/tokens.css'
import './styles/tailwind.css'
import './styles/animations.css'
import './styles/theme-stripe.css'
import './styles/theme-dark.css'
import './styles/ui-refined.css'
import './styles/ui-polish.css'
import './styles/tdesign-skin.css'
import './styles/tdesign-theme.css'
import './styles/ui-kit.css'
import './styles/tdesign-overrides.css'
import './styles/modal-system.css'
// 按需注册: 全项目仅用到以下 17 种组件(统计自全部模板标签),
// 比 app.use(全量 TDesign) 减少 ~70% 组件库 JS 体积, 首屏明显变快.
// 新增 t- 组件时在此处补注册, 否则运行时报"组件未注册".
import {
  Button,
  Input,
  InputNumber,
  Textarea,
  Select,
  Checkbox,
  Switch,
  Table,
  Pagination,
  Tag,
  Tooltip,
  Loading,
  Progress,
  Empty,
  Breadcrumb,
  BreadcrumbItem,
  DatePicker,
  // 注意: 不要在这里加 PullDownRefresh——它是 TDesign 移动端组件, 桌面版
  // tdesign-vue-next 不导出, import 会让 vite 构建失败。下拉刷新用手写组件
  // (下方 import + app.component 注册), 见 components/ui/PullDownRefresh.vue。
} from 'tdesign-vue-next'
import { initTheme } from '@/utils/theme'
import PullDownRefresh from '@/components/ui/PullDownRefresh.vue'

// 两套主题: stripe(浅色) / dark(深色), 默认浅色
initTheme()

const app = createApp(App)
app.use(createPinia())
app.use(createVfm())
app.use(Button)
  .use(Input)
  .use(InputNumber)
  .use(Textarea)
  .use(Select)
  .use(Checkbox)
  .use(Switch)
  .use(Table)
  .use(Pagination)
  .use(Tag)
  .use(Tooltip)
  .use(Loading)
  .use(Progress)
  .use(Empty)
  .use(Breadcrumb)
  .use(BreadcrumbItem)
  .use(DatePicker)
  // P0-01: 下拉刷新——5 个工作区页面 <pull-down-refresh v-model @refresh> 的真正实现
  // (桌面版 tdesign-vue-next 无此组件导出, 手写版见 components/ui/PullDownRefresh.vue)
app.component('PullDownRefresh', PullDownRefresh)
app.use(router)

// ===== 全局错误探针: 运行时错误红条置顶显示(诊断用) =====
// 8 秒自动消失 + 点击立即关闭, 避免一次报错永久驻留污染所有页面观感
let errBannerTimer: ReturnType<typeof setTimeout> | undefined
let lastBannerMessage = ''
let lastBannerAt = 0

function showErrBanner(msg: string) {
  // 业务性 403(套餐不含此功能)不算系统错误: 只展示友好提示。
  const friendly = /plan feature not available|feature not available|套餐/i.test(msg)
  const resourceFailure = /资源加载失败/.test(msg)
  const crossOriginFailure = /Script error/.test(msg)
  // 不把异常堆栈、服务器响应、内部路径、资源 URL 或查询参数显示给租户。
  const displayMessage = friendly
    ? 'ℹ 当前套餐不含此功能 —— 请联系管理员升级套餐后使用'
    : resourceFailure
      ? '⚠ 页面资源加载失败，请刷新页面；如果问题持续，请联系管理员。'
      : crossOriginFailure
        ? '⚠ 页面脚本运行异常，请刷新页面；如果问题持续，请联系管理员。'
        : '⚠ 页面遇到异常，请刷新页面；如果问题持续，请联系管理员。'

  // 同类错误短时间重复触发时不反复重置横幅/计时器，避免高频错误刷屏。
  const now = Date.now()
  if (displayMessage === lastBannerMessage && now - lastBannerAt < 3000) return
  lastBannerMessage = displayMessage
  lastBannerAt = now

  if (!document.getElementById('err-banner-css')) {
    const st = document.createElement('style')
    st.id = 'err-banner-css'
    st.textContent = '#err-banner.err-banner-friendly{background:var(--t-warn)}#err-banner{background:var(--t-err)}'
    document.head.appendChild(st)
  }
  let el = document.getElementById('err-banner')
  if (!el) {
    el = document.createElement('div')
    el.id = 'err-banner'
    el.setAttribute('role', 'alert')
    el.setAttribute('aria-live', 'assertive')
    el.setAttribute('aria-atomic', 'true')
    el.style.cssText = 'position:fixed;top:0;left:0;right:0;z-index:var(--t-z-toast);color:var(--t-brand-ink);font-size:var(--t-font-xs);line-height:var(--t-leading-normal);padding:var(--t-space-2) 36px var(--t-space-2) var(--t-space-3);white-space:pre-wrap;word-break:break-all;max-height:45vh;overflow:auto;box-shadow:var(--t-shadow-md);cursor:pointer;'
    el.title = '点击关闭'
    el.onclick = () => {
      el?.remove()
      if (errBannerTimer) clearTimeout(errBannerTimer)
      errBannerTimer = undefined
    }
    document.body.appendChild(el)
  }
  el.className = friendly ? 'err-banner-friendly' : ''
  el.textContent = displayMessage
  if (errBannerTimer) clearTimeout(errBannerTimer)
  errBannerTimer = setTimeout(() => {
    el?.remove()
    errBannerTimer = undefined
  }, 8000)
}
// 捕获阶段也监听: 资源加载失败(script/img 404)和跨域错误只有 capture 阶段能拿到 target
window.addEventListener('error', (e) => {
  const t = e.target as HTMLElement | null
  if (t && (t as any) !== window && (t.tagName === 'SCRIPT' || t.tagName === 'IMG' || t.tagName === 'LINK')) {
    showErrBanner(`资源加载失败: <${t.tagName.toLowerCase()}> ${t.getAttribute('src') || t.getAttribute('href') || ''}`)
    return
  }
  // 跨域脚本错误只有 "Script error." 无栈: 尽力带上位置信息定位文件
  if (e.message === 'Script error.') {
    showErrBanner(`Script error (跨域或无栈) @ ${e.filename || 'unknown'}:${e.lineno || 0}:${e.colno || 0}`)
  }
}, true)
// P2-01 审计修复(修正版): 审计指认 capture 监听未过滤 ResizeObserver——实际
// capture 只弹资源错误/Script error, 安全。真正无过滤的是这两处: ECharts 自动
// 重排等高频触发的 ResizeObserver loop 错误会经 errorHandler/unhandledrejection
// 刷红条, 与 114 行冒泡监听对齐加同款过滤。
app.config.errorHandler = (err, vm, info) => {
  // R124 错误定位器: 打出出错组件名(script setup 的 __name)——报错 chunk 位置
  // 无法精确映射时, 组件名是最快的定位途径
  const compName = (vm as any)?.$options?.__name || (vm as any)?.$?.type?.__name || (vm as any)?.$.type?.name || 'unknown'
  console.error(`[errorHandler] 组件=${compName}`, err, info)
  const msg = String((err as Error)?.stack || err) + ' @' + info
  if (/ResizeObserver loop/.test(msg)) return
  showErrBanner(`[组件:${compName}] ` + msg)
}
window.addEventListener('unhandledrejection', (e) => {
  console.error('[unhandledrejection]', e.reason)
  const msg = String((e.reason as Error)?.stack || e.reason)
  if (/ResizeObserver loop/.test(msg)) return
  showErrBanner(msg)
})
window.addEventListener('error', (e) => {
  // 无害错误过滤(2026-10-20): ResizeObserver 循环通知(echarts autoresize 触发)与
  // 纯 "Script error."(浏览器跨域安全限制, 无栈可查)不打扰租户——管理端 dashboard 因此不再爆红
  if (!e.message) return
  if (e.message.includes('ResizeObserver')) return
  if (e.message === 'Script error.') return
  showErrBanner(e.message)
})

app.mount('#app')
