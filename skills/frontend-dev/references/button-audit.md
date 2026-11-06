# 按钮体系专项审计（全量 Vue 模板收敛）

审计分支：`main`
审计基准：本次按钮迁移后的最新 `main`
范围：`web/src/**/*.vue`（共 56 个 Vue 文件，含 App、34 个 views、21 个 components），以及已核对的按钮基础样式入口。
方法：逐文件静态检索 `.ui-btn`、`<button>`、`<t-button>`；静态计数不等于浏览器实际渲染数量。未执行真实浏览器视觉验收。

## 1. 本轮结论

- 56 个 Vue 文件中，`.ui-btn` 调用静态检索结果为 **0**。
- 之前剩余的 15 个页面调用已迁移：Scrape、任务详情、链接采集、会话、抽奖、消息监控、登录、注册、管理端客服配置、租户详情、租户列表、探测配置、公告管理、更多、404 页面。
- 已迁移普通动作按钮统一使用 TDesign `<t-button>` + `.ui-t-button`，语义变体为 `ui-t-button--primary/secondary/danger/text`，小尺寸使用 `ui-t-button--sm`。
- 图标按钮继续使用 `.ui-icon-btn`；导航、账号/选项选择、分段切换、优先级选择、菜单项、文件选择和原生表单特殊行为保留其专用语义，不为追求标签统一而强行替换。
- `.ui-btn` 的 CSS 定义暂时保留为兼容层；在完成非 Vue 脚本、模板字符串及外部嵌入页面的引用扫描前，不直接删除定义，避免遗漏动态调用造成回归。

## 2. 统一约定

| 场景 | 统一入口 |
|---|---|
| 普通主操作 | `<t-button class="ui-t-button ui-t-button--primary">` |
| 次要/取消操作 | `<t-button variant="outline" class="ui-t-button ui-t-button--secondary">` |
| 危险操作 | `<t-button theme="danger" class="ui-t-button ui-t-button--danger">` |
| 文字操作 | `<t-button variant="text" class="ui-t-button ui-t-button--text">` |
| 小尺寸 | 追加 `ui-t-button--sm`，并按现有 TDesign API 设置 size |
| 图标操作 | `.ui-icon-btn`，必须有可访问名称 |
| 分段/选项/导航 | 使用既有 segmented、导航或选择控件，不套通用动作按钮 |

禁止新增平行按钮视觉类。保留 TDesign 的 loading、disabled、事件、表单 type 和业务条件。

## 3. 迁移覆盖

本轮迁移的剩余页面：

- `views/workspace/Scrape.vue`：创建采集、清空记录、导出 CSV/Excel、弹窗取消。
- `views/tasks/Detail.vue`：重新加载、暂停/恢复/取消、重试失败目标、成员刷新、清除显示。
- `views/scrape/Links.vue`：导出链接、复制链接等普通动作。
- `views/sessions/Index.vue`：批量删除、导入协议号、刷新、编辑资料、头像操作。
- `views/raffle/Index.vue`：配置保存、全选/全不选、清空与刷新。
- `views/workspace/Monitor.vue`：添加规则、加载联系人、弹窗取消与保存。
- `views/auth/Login.vue`、`views/auth/Register.vue`：登录、续费登录、注册提交。
- `views/admin/support/Config.vue`：清除二维码、重置。
- `views/admin/tenants/Detail.vue`、`views/admin/tenants/List.vue`：返回、弹窗取消等普通动作。
- `views/admin/probe/Config.vue`：保存配置。
- `views/admin/announcements/List.vue`：弹窗取消；公告优先级选择仍保留原生选择按钮。
- `views/more/Index.vue`、`views/NotFound.vue`：退出、返回。

此前已迁移的账号管理、卡密管理、群发、系统设置、任务列表、进群邀请、公共确认弹窗等继续沿用统一入口。

## 4. 样式职责

- `styles/ui-kit.css`：维护 `.ui-t-button` 及 primary/secondary/danger/text/size 变体；`.ui-icon-btn` 独立负责图标按钮。
- `styles/tdesign-skin.css`：全局 TDesign 皮肤需让出 `.ui-t-button` 的语义变体，不得用全局 `!important` 覆盖。
- `styles/modal-system.css`：只负责 Modal footer 布局、定位和尺寸，不再定义独立的弹窗按钮皮肤。
- `styles/tdesign-overrides.css`：只保留必要的第三方组件适配。
- `styles/ui-refined.css` / `styles/ui-polish.css`：历史 `.glow-btn` 皮肤已清理；`.btn-success`、`.btn-error` 仍需结合 `useAsyncBtn` 的真实引用决定是否退役。
- `App.vue` 与全局原生 button 规则仍需避免覆盖 TDesign 组件自身状态。

## 5. 尚未完成的边界

1. `.ui-btn` 在 56 个 Vue 文件中的调用已清零，但 `ui-kit.css` 的旧 CSS 定义仍保留作兼容层。
2. 非 Vue 的 JS/TS 动态 class、模板字符串、测试 fixture、静态 HTML 和外部注入内容尚未完成全仓引用扫描；确认无引用后再删除兼容 CSS。
3. 尚未执行 `npm run typecheck`、`npm run test`、`npm run check:design-tokens`、`npm run build`。
4. 尚未完成桌面端/移动端及 normal/eye/dark 三主题真实浏览器验收。

## 6. 本轮验收记录

- [x] 盘点并迁移剩余页面普通动作按钮。
- [x] 56 个 Vue 文件静态扫描：`.ui-btn` 调用为 0。
- [x] 抽查迁移页面 TDesign 开始/结束标签计数匹配。
- [x] 保留图标按钮、分段控件、导航、菜单及选择类特殊交互。
- [ ] 非 Vue 动态引用全仓扫描。
- [ ] 移除旧 `.ui-btn` CSS 兼容定义（须先完成上一项）。
- [ ] typecheck、test、design-token check、build。
- [ ] 桌面/移动端三主题真实浏览器验收。

CI：本轮最新 run `42163898801` 对应提交在 runner 执行步骤前失败；所有 jobs 的 steps 均为空/未生成，不能据此判断代码测试通过或失败。需要先修复 runner/CI 执行环境，再取得有效测试结果。
