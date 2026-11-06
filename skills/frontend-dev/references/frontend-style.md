# tg-sender 项目前端 UI 与编码约定

> 本文服务于 `skills/frontend-dev/SKILL.md`。它描述当前仓库的适配原则，不是脱离仓库的通用设计系统。发生冲突时先检查当前代码和调用点。

## 1. 技术栈事实

当前 `web/package.json` 使用 Vue 3、TypeScript、Vite、Vue Router、Pinia、TDesign Vue Next、lucide-vue-next、vue-final-modal、vue-sonner、Tailwind CSS 4、Vitest 和 vue-tsc。

- UI 组件：TDesign Vue Next，按需注册；新增组件先检查 `src/main.ts`。
- 图标：lucide-vue-next，沿用项目现有图标尺寸和封装。
- 弹窗：自定义 Modal/vue-final-modal 与 TDesign Dialog 不是同一实现，不能混用样式假设。
- 状态：Pinia；路由：Vue Router。
- 样式：`src/styles/` 下的令牌、主题、组件基类、TDesign 覆盖、弹窗样式及 Tailwind 工具类共同构成现有系统。

## 2. 样式层级与修改方法

当前全局样式入口包含 tokens、tailwind、animations、theme、ui-refined、ui-polish、tdesign-skin、tdesign-theme、ui-kit、tdesign-overrides、modal-system 等文件。**真实导入顺序必须以当前 `src/main.ts` 为准。** 不要把文件注释中的“唯一真相源”当作已验证事实。

修改前：
1. 查找现有 token 和公共 class。
2. 查找已有组件封装及相同业务页面用法。
3. 确定规则应由 TDesign 默认样式、项目公共类、scoped 样式还是弹窗专用层负责。
4. 确认 CSS cascade、优先级、主题变量和调用范围。
5. 在正确层级修复，避免重复覆盖。

不要在未分析调用点时互换 `.ui-btn`、`.t-button`、`.modal-btn-primary`、`.glow-btn` 等类。它们的宽度、颜色、布局职责可能不同。发现多套按钮冲突时，先列出调用位置和覆盖关系，再做最小统一；不要再新增一套。

## 3. 主题

当前主题工具 `src/utils/theme.ts` 包含 normal、eye、dark 三种模式，具体 html class 和 token 以当前实现为准。

新增样式优先使用项目已有 `--t-*` 等 token（先确认变量存在及语义）。三种主题均需检查：
- 页面背景、卡片、弹窗和浮层
- 主/次/弱化文字及边框
- 品牌色、成功/警告/错误状态
- 输入框、禁用态、focus ring
- 遮罩、Toast、下拉菜单和 tooltip

不要用系统 prefers-color-scheme 覆盖用户在应用内选择的主题。

## 4. 按钮、表单和控件

对同语义操作检查尺寸、高度、内外边距、字号、字重、圆角、边框、图标尺寸/间距、垂直对齐及 hover/active/focus/disabled/loading。

优先复用现有组件和类。表单保持可见标签、校验、错误提示、提交中、防重复提交和成功反馈一致。图标按钮有 aria-label 或等效可访问名称。非提交按钮明确 type=button，提交按钮明确 type=submit。

## 5. 弹窗与浮层

自定义 Modal 当前样式策略包含桌面居中、移动端底部抽屉的差异；这不是所有 Dialog/Popover 的通用规则。

桌面检查：视口居中、max-width/max-height、内容内部滚动、footer、overlay、z-index、父级 transform/overflow、Esc、焦点管理和背景滚动锁定。

移动检查：抽屉/全屏策略、安全区、软键盘、VisualViewport、最大高度、底部导航遮挡、footer 可见性、内容滚动和背景滚动锁定。

不要只加 `position: fixed` 或 `!important` 修定位；先检查 Modal.vue 传入类名、vue-final-modal 包装层和实际 DOM 结构。

## 6. 双端验收

桌面：页面标题、筛选区、工具栏、表格和分页对齐；弹窗居中；按钮和输入控件高度一致；侧栏、顶栏和主体不覆盖。

移动：无非预期横向滚动；主要操作可触达；底部导航不遮挡内容；长链接/群名/消息可换行或查看完整；表格有合理的卡片/横向滚动/隐藏列策略；软键盘出现时输入框和操作仍可见。

响应式修改至少检查一个桌面宽度和一个手机宽度；不能实测时明确标注。

## 7. Vue 与 TypeScript

新代码优先 `<script setup lang="ts">`；Props/Emits/API 类型明确；复杂派生状态使用 computed；列表使用稳定 key；Pinia 解构保持响应性；复用现有 API、通知、主题和路由机制；清理 Observer、事件、timer 和订阅。不为抽象而过度拆分组件。

## 8. 页面改动验收清单

- [ ] 无重复 HTML 属性（尤其 Vue 模板重复 class）。
- [ ] 按钮语义、类型、尺寸和状态正确。
- [ ] 图标按钮有可访问名称，图标无错位。
- [ ] 表单必要状态完整。
- [ ] 弹窗定位、滚动、遮罩和关闭行为正确。
- [ ] normal/eye/dark 主题下文字和背景均可辨认。
- [ ] 桌面无错位，手机无溢出/遮挡。
- [ ] 异步请求、事件监听、Observer 和 timer 有清理。
- [ ] 未改变 API、权限、数据语义和业务行为。
- [ ] 执行并记录 typecheck、test、design-token check、build（以当前脚本为准）。
