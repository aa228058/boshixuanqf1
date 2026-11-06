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
- 已完成 Vue 模板以及 composables/utils 脚本的旧类引用扫描，未发现动态 `.ui-btn` 引用；旧 `.ui-btn` CSS 定义和 segmented 旧变体已从 `ui-kit.css` 删除。

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

1. `.ui-btn` Vue 模板调用为 0；composables/utils 脚本未发现旧类动态引用；旧 `.ui-btn` CSS 规则已删除。外部注入 HTML 或仓库外运行时拼接内容不在本轮静态扫描边界内。
2. `.btn-success`、`.btn-error` 状态类仍暂留在 `ui-refined.css`，避免未核实 `useAsyncBtn` 或其他状态反馈调用前误删；需单独追踪其来源与调用后再决定退役。
3. 尚未执行 `npm run typecheck`、`npm run test`、`npm run check:design-tokens`、`npm run build`。
4. 尚未完成桌面端/移动端及 normal/eye/dark 三主题真实浏览器验收。

## 6. 本轮验收记录

- [x] 盘点并迁移剩余页面普通动作按钮。
- [x] 56 个 Vue 文件静态扫描：`.ui-btn` 调用为 0。
- [x] 抽查迁移页面 TDesign 开始/结束标签计数匹配。
- [x] 保留图标按钮、分段控件、导航、菜单及选择类特殊交互。
- [x] 扫描 Vue 模板及 composables/utils 脚本中的旧类引用。
- [x] 删除 `ui-kit.css` 中旧 `.ui-btn` 与 segmented 遗留规则。
- [ ] typecheck、test、design-token check、build。
- [ ] 桌面/移动端三主题真实浏览器验收。

CI：本轮最新 run `42163898801` 对应提交在 runner 执行步骤前失败；所有 jobs 的 steps 均为空/未生成，不能据此判断代码测试通过或失败。需要先修复 runner/CI 执行环境，再取得有效测试结果。

## 7. CI 执行环境复核（2026-11-06）

- 最新 main：`daacf1e441aae6ba6fdfa44d858157e458edd71a`。
- 最新 push run：`42164631209`，对应 HEAD 与 main 一致；运行从创建到结束约 4 秒。
- 6 个失败 Job（Frontend Typecheck + Unit Tests、frontend、integration、secrets、sql-audit、build）均为 `completed/failure`，返回的 `steps=null`、`logs_url=null`，没有可供诊断的执行日志。
- `.github/workflows/ci.yml` 中这 6 个 Job 全部配置为 `runs-on: ubuntu-latest`，不是 self-hosted；仓库唯一 self-hosted 配置位于只读的 `.github/workflows/server-preflight.yml`，标签为 `[self-hosted, Linux, X64]`。因此不能把 CI 主工作流故障简单归因为自托管 Runner 标签不匹配。
- GitHub Code Search 对 `.btn-success`、`.btn-error`、`useAsyncBtn` 均返回 0，但响应标记 `incomplete_results=true`，不能作为全仓无引用的证明。状态类 CSS 暂不删除。
- 当前证据指向 Job 启动/Actions 执行环境阶段异常，而非某条测试命令失败；具体原因尚不能从现有 API 返回中确认。仓库管理员需在 Settings → Actions 检查 Actions 是否启用、策略/并发限制与 runner 分配；同时检查账户/组织 Actions 分钟额度及 GitHub Actions 服务状态。取得带 steps 和日志的有效 run 后，再判断代码测试结果。
- 未修改 CI workflow 来掩盖问题；未触发生产部署。
