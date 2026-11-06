# 按钮体系专项审计（Phase 0）

审计分支：`main`
审计基准提交：`main`（本次收敛前复核）
范围：`web/src/**/*.vue`、按钮相关全局 CSS、`main.ts` 导入顺序。
方法：读取仓库文件并按模板标签/类名进行静态检索；计数是源码文本出现次数，不等于浏览器实际渲染数量。未执行浏览器视觉验收。

## 1. 结论

当前至少存在四条并行按钮样式路径：

1. TDesign 原生按钮：`<t-button>` / `.t-button`
2. 项目通用按钮类：`.ui-btn`
3. 弹窗专用按钮类：`.modal-btn-primary`、`.modal-btn-ghost`、`.modal-btn-danger`
4. 旧视觉按钮：`.glow-btn`、`.btn-success`、`.btn-error`

此外，`.ui-icon-btn` 是独立图标操作按钮样式；原生 `<button>` 还会被全局规则影响。当前不能将所有路径机械合并：TDesign 组件保留 loading/disabled/事件语义，原生 button 可能承载表单提交、菜单、弹窗和图标操作，必须按语义迁移。

## 2. 关键样式定义位置

| 文件 | 定义/影响 |
|---|---|
| `styles/ui-kit.css` | `.ui-btn` 全套主/次/危险/警告/ghost/text/尺寸；`.ui-icon-btn`；基础 `.t-button` 几何；`.t-button.ui-t-button` 统一视觉入口及 primary/secondary/text/size 变体 |
| `styles/modal-system.css` | 三种 `.modal-btn-*`；弹窗 footer 中原生 button 和 TDesign 按钮的尺寸覆盖 |
| `styles/tdesign-skin.css` | 对所有 `.t-button` 使用 `!important` 设置背景、边框、颜色；primary/base 又叠加渐变 |
| `styles/tdesign-overrides.css` | Dialog footer、modal footer、legacy modal 的 button 覆盖 |
| `styles/ui-refined.css` | `.glow-btn`、`.btn-success`、`.btn-error` |
| `styles/ui-polish.css` | 全局 `button` 与 `.glow-btn` 的 hover/active/transition 等通用规则 |
| `App.vue` | 再次对全局 `button` 设置 transition 与 active transform |

## 3. 已确认的调用点（源码静态计数）

计数只统计对应文件中的精确类名文本或标签文本；动态 class、组合类和子组件内部实现仍需在迁移时复核。

### `.ui-btn` 高使用页面

| 文件 | `.ui-btn` 文本出现 |
|---|---:|
| `views/accounts/List.vue` | 60 |
| `views/workspace/Tasks.vue` | 33 |
| `views/admin/codes/List.vue` | 33 |
| `views/workspace/Broadcast.vue` | 32 |
| `views/settings/Index.vue` | 32 |
| `views/workspace/Join.vue` | 22 |
| `views/workspace/Scrape.vue` | 21 |
| `views/tasks/Detail.vue` | 21 |
| `views/scrape/Links.vue` | 15 |
| `views/sessions/Index.vue` | 16 |
| `views/raffle/Index.vue` | 15 |
| `views/workspace/Monitor.vue` | 10 |
| `views/auth/Login.vue` | 4 |
| `views/admin/support/Config.vue` | 6 |
| `views/admin/tenants/Detail.vue` | 6 |
| `views/admin/tenants/List.vue` | 3 |
| `views/admin/probe/Config.vue` | 2 |
| `views/auth/Register.vue` | 2 |
| `views/more/Index.vue` | 3 |
| `views/NotFound.vue` | 2 |

### 弹窗按钮类

已确认调用：
- `views/Layout.vue`：三组取消/确认按钮，合计 6 处类名引用。
- `components/ui/ConfirmDialog.vue`：ghost、danger、primary 动态类组合。
- `components/ui/ConfirmHost.vue`：ghost 与 danger/primary 动态切换。
- `views/workspace/Broadcast.vue`：至少 1 处 primary 类引用。

### TDesign 与原生按钮共存

抽查的业务页面中，TDesign 与原生 button 并非互斥。例如：
- `views/accounts/List.vue`：13 个 `<t-button>` 标签文本、38 个 `<button>` 标签文本。
- `views/workspace/Broadcast.vue`：3 个 `<t-button>`、15 个原生 `<button>`。
- `views/settings/Index.vue`：5 个 `<t-button>`、12 个原生 `<button>`。
- `views/groups/Discover.vue`：16 个 `<t-button>`、3 个原生 `<button>`。
- `views/sessions/Index.vue`：7 个 `<t-button>`、1 个原生 `<button>`。
- `views/admin/tenants/List.vue`：2 个 `<t-button>`、7 个原生 `<button>`。

这说明不能通过全局替换标签来统一按钮。原生 button 还需要区分普通操作、表单提交、菜单触发、弹窗操作和图标按钮。

## 4. 主要冲突与风险

1. `ui-kit.css` 定义 `.ui-btn` 和 TDesign 按钮样式，但 `tdesign-skin.css` 又对所有 `.t-button` 使用 `!important`，形成两层视觉规则。
2. `.t-button.ui-t-button` 已有实际调用（例如 `views/groups/Discover.vue`），是当前项目 TDesign 按钮统一视觉入口；此前审计把“是否有调用”列为待确认项，现已纠正。
3. `modal-system.css` 独立定义弹窗按钮几何和颜色，和 `.ui-btn` / `.ui-t-button` 的语义、尺寸体系重复。公共确认弹窗、Layout 账号登录弹窗和 Broadcast 预览弹窗已迁移到 `.ui-t-button`；其他业务弹窗仍保留兼容类，后续按调用点逐页迁移。
4. `ui-polish.css` 对通用 `button` 和 `.glow-btn` 加交互效果，`App.vue` 再对所有 button 加 transition/active，可能影响本应由组件自身控制的按钮。
5. `ui-refined.css` 中 `.glow-btn` 是历史视觉入口；当前至少在 `workspace/Join.vue`、`workspace/Scrape.vue` 被引用，不能直接删除。
6. `.ui-icon-btn` 应继续作为图标按钮语义单独治理，不应强行套普通文字按钮的 padding/宽度。

## 5. 建议的统一目标

- **基础控件主入口：** TDesign Button。需要保持 TDesign 的 loading、disabled、事件、表单行为。
- **项目统一视觉入口：** TDesign Button + `.ui-t-button`，通过 `ui-t-button--primary/secondary/danger/text` 与 `ui-t-button--sm` 表达语义；禁止新建平行 Button 视觉类。
- **原生 button：** 仅用于菜单触发、特殊交互、原生表单语义或基础库不适配的场景；不再用全局 button 规则赋予完整视觉皮肤。
- **弹窗按钮：** 公共 `ConfirmDialog.vue` / `ConfirmHost.vue`、`Layout.vue` 的账号登录弹窗、`workspace/Broadcast.vue` 的预览弹窗已迁移到 TDesign Button + `.ui-t-button`；保留 Modal footer 布局规则。旧 `.modal-btn-*` CSS 定义暂时保留，待全仓库确认调用归零后再删除。
- **图标按钮：** 保留独立尺寸/命中区域/aria-label 规范，继续使用 `.ui-icon-btn`，并核查所有调用。
- **旧类：** `.ui-btn`、`.modal-btn-*`、`.glow-btn` 暂时兼容；每迁移一个页面就检查调用数，归零后才能删除定义。

## 6. 本部分完成状态

- [x] 核对当前 main 与真实样式导入顺序
- [x] 找出主要按钮 CSS 定义入口
- [x] 抽查业务页、弹窗和管理端按钮调用
- [x] 明确不可直接批量替换的业务语义风险
- [ ] 全量调用点机器统计（包括脚本动态 class 和非 Vue 模板字符串）
- [x] 确认现有 Button API：TDesign Button + `.ui-t-button`；迁移公共 ConfirmDialog/ConfirmHost
- [ ] 执行 typecheck、test、design-token check、build
- [ ] 桌面/移动端三主题真实浏览器视觉验收

**本轮收敛：公共确认弹窗已统一使用 TDesign Button + `.ui-t-button`；`tdesign-skin.css` 已对 `.ui-t-button` 让出全局 `!important` 皮肤覆盖，新增 danger 语义变体。旧业务调用保持兼容。尚未完成 `.ui-btn` / `.glow-btn` 等全仓库旧类迁移、全量动态调用统计、自动化测试与桌面/移动端视觉验收；不得宣称所有按钮已完全统一。**
