# 按钮体系专项审计（Phase 0）

审计分支：`main`
审计基准提交：`main`（本次收敛前复核）
范围：`web/src/**/*.vue`、按钮相关全局 CSS、`main.ts` 导入顺序。
方法：读取仓库文件并按模板标签/类名进行静态检索；计数是源码文本出现次数，不等于浏览器实际渲染数量。未执行浏览器视觉验收。

## 1. 结论

当前至少存在四条并行按钮样式路径：

1. TDesign 原生按钮：`<t-button>` / `.t-button`
2. 项目通用按钮类：`.ui-btn`
3. 历史兼容按钮类：`.ui-btn`（仍有业务调用）
4. 少量旧状态类：`.btn-success`、`.btn-error`（需结合 `useAsyncBtn` 继续确认调用）

此外，`.ui-icon-btn` 是独立图标操作按钮样式；原生 `<button>` 还会被全局规则影响。当前不能将所有路径机械合并：TDesign 组件保留 loading/disabled/事件语义，原生 button 可能承载表单提交、菜单、弹窗和图标操作，必须按语义迁移。

## 2. 关键样式定义位置

| 文件 | 定义/影响 |
|---|---|
| `styles/ui-kit.css` | `.ui-btn` 全套主/次/危险/警告/ghost/text/尺寸；`.ui-icon-btn`；基础 `.t-button` 几何；`.t-button.ui-t-button` 统一视觉入口及 primary/secondary/text/size 变体 |
| `styles/modal-system.css` | Modal footer 的布局、定位与尺寸规则；旧 `.modal-btn-*` 按钮皮肤已删除 |
| `styles/tdesign-skin.css` | 对所有 `.t-button` 使用 `!important` 设置背景、边框、颜色；primary/base 又叠加渐变 |
| `styles/tdesign-overrides.css` | Dialog footer、modal footer、legacy modal 的 button 覆盖 |
| `styles/ui-refined.css` | `.btn-success`、`.btn-error` 状态类；旧 `.glow-btn` 皮肤已删除 |
| `styles/ui-polish.css` | 原生 `button` 的通用 hover/active/transition 规则；旧 `.glow-btn` 特殊规则已删除 |
| `App.vue` | 再次对全局 `button` 设置 transition 与 active transform |

## 3. 已确认的调用点（源码静态计数）

计数只统计对应文件中的精确类名文本或标签文本；动态 class、组合类和子组件内部实现仍需在迁移时复核。

### `.ui-btn` 高使用页面

| 文件 | `.ui-btn` 文本出现 |
|---|---:|
| `views/accounts/List.vue` | 0（本轮已迁移） |
| `views/admin/codes/List.vue` | 0（本轮已迁移） |
| `views/workspace/Broadcast.vue` | 0（本轮已迁移） |
| `views/settings/Index.vue` | 0（本轮已迁移） |
| `views/workspace/Join.vue` | 0（本轮已迁移） |
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

- `views/workspace/Join.vue`：新建任务、加载重试、任务重试失败/停止/详情、弹窗取消与创建操作迁移为 TDesign Button + `.ui-t-button`；链接/邀请模式切换保留 `.ui-segmented__item`，无效链接展开、文件选择和通知开关等特殊交互保留原生控件。该页 `.ui-btn` 调用已清零。\n\n### 已完成迁移的业务页

- `views/workspace/Tasks.vue`：桌面表格与移动端卡片中的停止、重试、仅重试失败目标、每账号统计操作，统一迁移为 TDesign Button + `.ui-t-button`；保留 `@click.stop`、状态条件和业务方法。类型/时间范围筛选改用独立 `.ui-segmented__item`，不再借用通用 `.ui-btn`。该页 `.ui-btn` 调用已清零，图标刷新/清空操作继续使用独立 `.ui-icon-btn`。
- `views/admin/codes/List.vue`：桌面表格操作、移动端卡片操作、生成/详情弹窗 footer、批量操作栏中的普通操作按钮统一迁移为 TDesign Button + `.ui-t-button`；保留状态条件、原事件处理与批量操作确认逻辑；清空已选卡密继续使用 `.ui-icon-btn`。该页 `.ui-btn` 调用已清零。
- `views/workspace/Broadcast.vue`：新建任务、任务卡片操作、重试/停止/详情、目标名称展开、批量导入、变量预览、媒体清除等普通动作按钮统一迁移到 TDesign Button + `.ui-t-button`；发送模式切换迁移到 `.ui-segmented__item`。页面 tabs、执行账号选择与消息变量快捷插入仍保留原生 button，因为它们是导航/选择/插入交互而非通用动作按钮。该页 `.ui-btn` 调用已清零。
- `views/accounts/List.vue`：多选入口/退出、批量启用与转备用、克隆资料、账号卡片常规操作、重新登录验证码入口、添加账号/分组/备注/转移弹窗取消按钮、分页及批量资料/克隆/检测提交按钮迁移到 TDesign Button + `.ui-t-button`；保留禁用条件、事件修饰符、响应式显隐和原业务方法。独立图标按钮、更多菜单内部菜单项、分段登录模式切换继续保留原生语义。该页 `.ui-btn` 调用已清零。

- `views/settings/Index.vue`：套餐升级/激活、通知保存与测试邮件、AI 配置保存/连通测试、数据导出、模板管理/保存/添加、客服复制及升级弹窗操作迁移为 TDesign Button + `.ui-t-button`；SMTP 服务商切换保留 `.ui-segmented__item`，顶部 Tab 保留 segmented 导航，通知事件开关与删除图标按钮保留专用原生交互。该页 `.ui-btn` 调用已清零。\n\n## 弹窗按钮类

历史 `.modal-btn-*` 调用已迁移并清零：
- `views/Layout.vue`：账号登录弹窗按钮已迁移。
- `components/ui/ConfirmDialog.vue` / `ConfirmHost.vue`：公共确认按钮已迁移。
- `views/workspace/Broadcast.vue`：预览弹窗按钮已迁移。
- `styles/modal-system.css`：旧按钮皮肤定义已删除，保留 Modal footer 布局与尺寸规则。

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
3. `modal-system.css` 独立定义弹窗按钮几何和颜色，和 `.ui-btn` / `.ui-t-button` 的语义、尺寸体系重复。公共确认弹窗、Layout 账号登录弹窗和 Broadcast 预览弹窗已迁移到 `.ui-t-button`；相关公共与业务弹窗按钮均已迁移；旧类定义已清理。
4. `ui-polish.css` 对通用 `button` 和 `.glow-btn` 加交互效果，`App.vue` 再对所有 button 加 transition/active，可能影响本应由组件自身控制的按钮。
5. `.glow-btn` 的业务调用已从 `workspace/Join.vue`、`workspace/Scrape.vue` 移除，`ui-refined.css` / `ui-polish.css` 中对应特殊皮肤规则也已删除。
6. `.ui-icon-btn` 应继续作为图标按钮语义单独治理，不应强行套普通文字按钮的 padding/宽度。

## 5. 建议的统一目标

- **基础控件主入口：** TDesign Button。需要保持 TDesign 的 loading、disabled、事件、表单行为。
- **项目统一视觉入口：** TDesign Button + `.ui-t-button`，通过 `ui-t-button--primary/secondary/danger/text` 与 `ui-t-button--sm` 表达语义；禁止新建平行 Button 视觉类。
- **原生 button：** 仅用于菜单触发、特殊交互、原生表单语义或基础库不适配的场景；不再用全局 button 规则赋予完整视觉皮肤。
- **弹窗按钮：** 公共 `ConfirmDialog.vue` / `ConfirmHost.vue`、`Layout.vue` 的账号登录弹窗、`workspace/Broadcast.vue` 的预览弹窗已迁移到 TDesign Button + `.ui-t-button`；保留 Modal footer 布局规则。旧 `.modal-btn-*` CSS 定义已删除；保留 Modal footer 的桌面横排与移动端布局规则。
- **图标按钮：** 保留独立尺寸/命中区域/aria-label 规范，继续使用 `.ui-icon-btn`，并核查所有调用。
- **旧类：** `.ui-btn` 仍有大量业务调用，是下一阶段的主要收敛对象；`.btn-success` / `.btn-error` 暂保留，待确认 `useAsyncBtn` 的真实使用后再决定是否清理。

## 6. 本部分完成状态

- [x] 核对当前 main 与真实样式导入顺序
- [x] 找出主要按钮 CSS 定义入口
- [x] 抽查业务页、弹窗和管理端按钮调用
- [x] 明确不可直接批量替换的业务语义风险
- [x] 对已迁移的公共弹窗、Join、Scrape、认证、设置、会话、任务页执行重复 class 属性静态检查
- [ ] `.ui-btn` 全仓调用迁移与动态调用统计（包括脚本动态 class 和非 Vue 模板字符串）
- [x] 确认现有 Button API：TDesign Button + `.ui-t-button`；迁移公共 ConfirmDialog/ConfirmHost
- [ ] 执行 typecheck、test、design-token check、build
- [ ] 桌面/移动端三主题真实浏览器视觉验收

**本轮收敛：任务页桌面/移动端操作按钮已统一到 TDesign Button + `.ui-t-button`，筛选按钮改用专用 segmented 样式；任务页 `.ui-btn` 调用清零。卡密管理页的桌面表格、移动端卡片、弹窗 footer 与批量操作按钮也已统一到 TDesign Button + `.ui-t-button`，该页 `.ui-btn` 调用清零。群发页普通动作按钮统一到 TDesign Button + `.ui-t-button`，发送模式切换使用 `.ui-segmented__item`，该页 `.ui-btn` 调用清零。账号管理页通用动作按钮、弹窗取消、分页及批量操作按钮迁移到 TDesign Button + `.ui-t-button`，独立图标按钮、菜单项和分段切换保留专用语义，该页 `.ui-btn` 调用清零。系统设置页普通操作按钮迁移到 TDesign Button + `.ui-t-button`，SMTP 服务商切换和顶部 Tab 保留 segmented 导航，通知事件开关与图标删除保留专用语义，该页 `.ui-btn` 调用清零。公共确认弹窗已统一使用 TDesign Button + `.ui-t-button`；`tdesign-skin.css` 已对 `.ui-t-button` 让出全局 `!important` 皮肤覆盖，新增 danger 语义变体。旧业务调用保持兼容。`workspace/Join.vue` 和 `workspace/Scrape.vue` 已修复重复 class 属性，并移除这两页创建/采集按钮上冗余的 `.glow-btn`。本轮已清理 `.modal-btn-*` 与 `.glow-btn` 的业务调用和对应 CSS 皮肤；修复 Join/Scrape、管理登录/探测配置、用户登录/注册、会话、设置、任务页的重复 class 属性。`.ui-btn` 仍是大量业务页的独立按钮皮肤，尚未迁移，不能宣称全仓按钮体系已完全统一。尚未执行 typecheck、test、design-token check、build 与桌面/移动端视觉验收。**
