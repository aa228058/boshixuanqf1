# UI/UX 全量审计台账（第一阶段）

- 基线分支：`ui-ux-only`
- 基线提交：`07042761470276c31088e95704565daac15303b5`
- 审计工作分支：`audit/uiux-full-scan`
- 范围：用户提供的 UI/UX 清单 #1–#180
- 原则：先核验后整改；不触碰业务主分支；不以静态源码推断代替运行时验收。

## 状态定义

- **已确认**：当前分支源码直接支持问题描述；不代表运行时影响已经量化。
- **待跑**：源码可定位，但需要浏览器/设备实测才能判断影响。
- **待 grep**：需要全仓引用/调用关系检索后才能定性。
- **待核**：尚未完成源码复核。
- **误报/已修复**：现有源码不支持原问题描述，或当前代码已具备防护。
- **触及层**：纯展示（模板/CSS/文案/图标）或触及逻辑（计算、状态、事件、store/API）。
- **验收**：lint/build 不能替代逻辑回归；逻辑项必须有手动或 E2E 步骤。

## A. 用户补充的源码候选（27 项逐项核验）

| ID | 页面/组件（完整路径） | 问题描述 | 严重度 | 归类 | 源码状态 | 触及层 | 验收方式 | 整改提交/备注 |
|---|---|---|---|---|---|---|---|---|
| #4 | `src/components/ui/CountUp.vue` | 清洗货币符号后动画只写数字，`¥` 不恢复 | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 单测 + dashboard 手动 | 已提交修复 `51740bc`：动画过程保留数字前缀/后缀；待构建和货币值回归 |
| #9 | `src/components/MobileTabBar.vue` | Tab 激活态依赖包含 accountId 的完整路径，切号/嵌套路由可能失活 | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 多账号切换与任务详情 E2E | 已提交 `ded9030`：按 workspace 功能段匹配，不再绑定具体账号 ID；新增路由 helper 单测 |
| #10 | `src/components/MobileTabBar.vue` | 当前账号缺失时使用 `none` 生成工作区路由，无有效 fallback | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 无账号点群发/任务 Tab，验证跳转账号页且账号 Tab 高亮 | 已提交 `ded9030`：无账号时群发/任务均回退 `/accounts`；新增路由 helper 单测 |
| #20 | `src/components/ui/Breadcrumb.vue` | 模板固定渲染“首页”，管理端传入仪表盘后重复层级 | 🟡 | 细节/横切 | 已确认 | 触及逻辑 | 租户端、管理端面包屑逐页检查 | 已提交修复 `e7be58b` / `042fcc1`：当 items 首项已是主题首页时不重复插入；首项不是首页时保留首页入口 |
| #26 | `src/components/ui/ConfirmActions.vue` | 移动端 `column-reverse` 导致视觉顺序与 DOM/焦点顺序不一致 | 🟡 | 细节/可访问性 | 已确认 | 纯展示 | 移动端键盘焦点与视觉顺序手测 | 已提交 `49d0202`：改为 column，使取消在上、确认在下，顺序与 DOM 一致 |
| #27 | `src/components/ui/PullDownRefresh.vue` | 无滚动父级时未检查 window/page scrollTop，页面滚动到中段也允许下拉刷新 | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 页面滚动到顶部/中段分别下拉验证 | 已提交 `7200d8d` / `7a89b48`：无滚动父时检查 window/document scrollTop，只允许页面顶部触发 |
| #34 | `src/components/ui/SupportFloat.vue` | touch 拖拽结束后合成 click 可能在 50ms 复位后触发展开 | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 拖拽后松手不展开；停顿后轻点可展开 | 已提交 `7029378` / `bfd6777`：记录最后拖动时间，400ms 内抑制合成 click |
| #36 | `src/components/ui/EmptyState.vue` | `title` 传入 TDesign `description`，真描述另行渲染 | 🟠 | 真缺陷 | 已确认 | 纯展示 | 组件用例/页面手测 | 已提交修复 `5827ff5`：title 放入 image slot 标题区，description 交给 TDesign description；待页面视觉回归 |
| #38 | `src/components/ui/ProgressRing.vue` | 百分比文字 round，圆弧保留小数，显示不一致 | 🟡 | 细节 | 已确认 | 纯展示 | 99.5、0.5、100 等边界值单测 | 已提交修复 `ec0f05b` / `553dba7`：弧与文本共用 0.1% 精度的夹紧值，背景轨道 stroke-width 改为绑定真实 stroke prop；待边界测试 |
| #57 | `src/views/accounts/List.vue` | 移动端账号操作按钮最小高度 36px | 🟡 | 细节/可访问性 | 已确认 | 纯展示 | 320/375/430 宽度触控检查 | 已提交 `b14ff6d`：操作按钮和更多按钮均提升到 44px；待检查窄屏换行与卡片高度 |
| #58 | `src/views/accounts/List.vue` | `loginCodeFresh()` / `codeAge()` 未读取响应式时间依赖 | 🟠 | 真缺陷候选 | 已确认（源码） | 触及逻辑 | 验证验证码年龄是否随时间刷新及 10 分钟到期隐藏 | 已提交修复 `dea6832`：两个函数统一读取 `nowTs.value`；待手动回归 |
| #68 | `src/views/accounts/List.vue` | `statusPoll` 与 `useAutoRefresh` 都触发账号刷新，可能重复轮询 | 🟠 | 横切/性能 | 已确认（双触发存在） | 触及逻辑 | 网络面板统计 30 秒请求次数、切后台恢复 | 已提交修复 `f7b392b`：合并为单一 30 秒强制刷新入口，统一刷新账号/统计/任务摘要，并保留切后台暂停、回前台立即刷新；待网络面板回归 |
| #99 | `src/views/workspace/Scrape.vue` | 导出 Excel 的独立 HTML 使用 CSS 变量 `var(--t-line)` / `var(--t-space-1-5)` | 🟡 | 真缺陷 | 已确认 | 纯展示 | 下载 XLS 并检查边框与内边距 | 已提交 `ee3c1c9`：替换为 Excel 可识别的固定边框色与 px padding |
| #100 | `src/views/workspace/Scrape.vue` | CSV/XLS 导出创建 Object URL 后未 revoke | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 连续导出后检查 URL 生命周期 | 已提交 `ee3c1c9`：两个下载均在触发后 1 秒 revoke URL；待浏览器下载回归 |
| #109 | `src/views/raffle/Index.vue` | 清空记录使用原生 `confirm()` | 🟡 | 细节/横切 | 已确认 | 触及逻辑 | 桌面/移动端确认交互回归 | 已提交 `17db125`：改用全局 `ask()` 确认弹窗；取消不触发删除 |
| #130 | `src/views/admin/dashboard/Index.vue` | 演示数据角标位于 `main` 外，绝对定位缺少正确定位上下文 | 🟡 | 真缺陷 | 已确认 | 纯展示 | 管理端 dashboard 实机检查角标位置 | 已提交 `3170fa6`：角标移入 relative main 容器 |
| #131 | `src/views/admin/dashboard/Index.vue` | 使用 `bg-warning` / `text-warning`，可能没有对应 Tailwind token | 🟡 | 待核 | 待 grep | 纯展示 | Tailwind 配置/构建 CSS 检查 | 需核对完整 token 定义与生成结果 |
| #132 | `src/views/admin/dashboard/Index.vue` | created_at 的 format 元数据未被 TDesign columns 消费 | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 管理端 dashboard 日期列检查 | 已提交 `c1441e8`：改用 `#created_at` 插槽明确格式化日期，移除无效 format 字段 |
| #138 | `src/views/admin/tenants/List.vue` | 注册时间列未格式化，原始日期值直接交给 TDesign | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 租户列表日期列检查 | 已提交 `e98aecc`：增加 `#created_at` 插槽，按 zh-CN 日期显示 |
| #141 | `src/views/admin/codes/List.vue` | 使用 text-yellow/red/green 硬编码语义色 | 🟡 | 误报（当前文件） | 误报 | 纯展示 | 如有其他匹配文件再补证据 | 当前文件全文未检出所述 class |
| #142 | `src/views/admin/codes/List.vue` | 批量操作浮条固定距底 92px | 🟡 | 细节 | 已确认 | 纯展示 | 管理端移动端实机检查遮挡 | 已提交 `9c48b53`：移动端改为距底 20px + safe-area；待检查是否遮挡底部系统手势区 |
| #145 | `src/views/admin/support/Config.vue` | 二维码预览使用 `object-cover`，可能裁切二维码边缘 | 🟠 | 真缺陷 | 已确认 | 纯展示 | 不同长宽比二维码实机扫码 | 已提交 `0f6ccca`：改为 object-contain，并提供白底与内边距；待扫码验证 |
| #152 | `src/views/admin/system/Monitor.vue` | CPU 渐变条 `from-[var(--t-*)] to-[var(--t-*)]` 是否生成/显示异常 | 🟡 | 待验证 | 待跑 | 纯展示 | 构建产物 + 管理端监控页实机检查 | 当前源码是合法的 gradient-stop 写法；静态源码不能证明运行时可见 |
| #159 | `src/views/admin/codes/List.vue`、`src/views/admin/audit/Logs.vue` 等 | 卡密表格的 plan_id/created_at format 元数据未消费；审计日志虽有 format 字段但已有同名插槽 | 🟠 | 横切 | 部分已确认、部分误报 | 触及逻辑 | 管理端表格日期/套餐显示检查 | 已提交 `d434078`：卡密表新增 plan_id/created_at 插槽并删除无效 format；`cb53076`：审计日志删除冗余 format 元数据（其插槽已负责格式化） |
| #175 | `src/styles/tailwind.css` | `:not([class*="t-"])` 会排除含 `t-` 的普通类（如 `text-sm`），使其失去该 focus-visible 样式 | 🟠 | 横切 | 已确认 | 纯展示 | 检查按钮/链接键盘 focus-visible | 已提交 `75a27ec`：改用 class token 边界匹配，仅排除以 `t-` 开头的 TDesign 类 |
| #121 | `src/views/NotFound.vue` | 无站内历史记录时仍调用 `router.back()` | 🟡 | 真缺陷 | 已确认 | 触及逻辑 | 直接打开 404 后点击返回 | 已提交 `bd1e02e`：优先检查 router history state；无站内上一页则回 dashboard |
| #51 | `src/views/dashboard/Index.vue` | ECharts 读取不存在的 `--text-secondary`，fallback 也不是 CSS 颜色值 | 🟡 | 真缺陷 | 已确认 | 触及逻辑 | 三主题下检查图例/轴标签颜色 | 已提交 `8644b38`：改读 `--color-text-secondary`，fallback 使用 `var(--t-text-3)`；待图表回归 |

> 注：用户原始 27 项为审计候选，并非全部自动认定为真缺陷。源码复核发现 #34、#141 至少在当前版本不支持原问题描述；#27 是有意的整页 fallback；#131、#138、#159 仍需补证据。保留编号便于追踪，不强行计入“已确认真缺陷”数量。

## B. 运行时待验证（8 项）

| ID | 路径/范围 | 待验证点 | 状态 | 触及层 | 验收方式 |
|---|---|---|---|---|---|
| #174 | Toast 组件/样式 | 淡出动画中途被移除是否闪断 | 待跑 | 触及逻辑 | 连续触发 toast 并慢放录屏 |
| #176 | `src/components/ui/Modal.vue` | 移动端打开带自动 focus 的弹窗时键盘/transform 是否抖动 | 待跑 | 触及逻辑 | iOS/Android 真机录屏 |
| #180 | `src/views/Layout.vue`、`src/views/admin/Layout.vue`、`src/components/MobileTabBar.vue` | 租户侧栏 `md:flex` 与 TabBar `md:hidden` 同以 768px 为切换点；管理端抽屉/按钮同以 `lg`（1024px）切换，源码断点成对一致，暂未发现冲突 | 源码已复核，待视口确认 | 待跑/疑似误报 | 纯展示 | 640/700/767/768/769/1023/1024px 视口实测，重点看 768px 边界 |
| #152 | `src/views/admin/system/Monitor.vue` | CPU 渐变条最终是否可见 | 待跑 | 纯展示 | 构建后页面实测 |
| #51 | `src/views/dashboard/Index.vue` | 深色主题 ECharts 图例/轴标签是否回退黑色 | 待跑 | 触及逻辑 | 三主题切换检查 |
| #65 | `src/views/dashboard/Index.vue` | 当前 KPI 区是 `grid-cols-2 lg:grid-cols-4`，卡片 `min-w-0`，源码未见 stat-strip 横向滚动容器；暂不支持原问题描述 | 源码已复核，待视口确认 | 待验证/疑似误报 | 纯展示 | 1920px 实测是否有页面级横向溢出 |
| #61 | `src/views/accounts/List.vue` | 选择多账号时底部批量操作浮条可能遮挡列表末尾 | 🟠 | 真缺陷候选 | 已确认（布局空间不足） | 纯展示 | 移动端选 5+ 账号滚到列表末尾 | 已提交 `47504ae`：选中账号时列表主容器增加移动端底部滚动留白；待实机验证 |
| #84 | `src/views/workspace/Broadcast.vue` | 已选目标 chips 预览区 max-height 仅 96px，20 个目标时可视区域较窄 | 🟡 | 细节 | 已确认（源码） | 纯展示 | 20 个目标实测 | 已提交 `0f8a8c0`：预览区 max-height 提升至 160px；待检查移动端表单高度与滚动体验 |

## C. 死代码/技术债候选（独立清理批次）

| ID | 路径 | 候选项 | 当前结论 | 状态 | 验收方式 |
|---|---|---|---|---|---|
| #12 | `src/components/selectors/GroupSelector.vue` | 私聊（dm）分支是否不可达 | 误报：当前组件存在可见 dm Tab、加载、过滤、全选与选择流程 | 已核验 | 保留功能，不清理 |
| #39 | `src/components/ui/LottieAnim.vue`、`src/assets/lottie/empty-tasks.json` | 组件零引用、资产 367,199 bytes | 已确认并已清理 | 已清理 | 删除组件、资产及 lottie-web 依赖；待构建验证 |
| #122 | updates 页面 | 版本文案 v68.9 过期 | 内容治理，不作为 UI bug | 待核 | 页面内容核验 |
| #178 | 全局文案/i18n | 术语不统一 | 规范建设，不作为单点 UI bug | 待核 | 术语表/全局 grep |

## D. 尚未核验的其余清单

#1–#180 中未在上表单列的条目统一暂列 **待核**，不代表不存在问题或已通过。后续按组件层、租户端页面、管理端页面、横切样式四组继续补齐，所有条目都必须记录完整 `src/` 路径、源码状态、触及层和验收方式。

## E. 当前源码复核中的重要更正

1. `src/components/ui/SupportFloat.vue` 的 `toggle()` 已在 `isDragging` 为 true 时 return；#34“拖拽后 click 未拦截”不能直接认定成立，仍需检查 click 与延迟复位的时序。
2. `src/components/selectors/GroupSelector.vue` 当前确实包含 dm Tab、加载、过滤、全选及选择流程；#12“私聊分支不可达”与当前源码不符，不应按死代码删除。
3. `src/components/ui/PullDownRefresh.vue` 无滚动父时返回 true 的代码注释明确写为整页场景 fallback；需要验证误触，不宜直接当 bug。
4. `src/views/admin/codes/List.vue` 当前全文未检出 #141 所述 `text-yellow/red/green` 类。
5. `src/views/workspace/Scrape.vue` 两处 Object URL 均未看到 revoke；#100 可直接进入整改候选。
6. 已逐一检查当前分支全部 58 个 Vue SFC，未发现 `LottieAnim` 组件调用；`empty-tasks.json` 为 367,199 bytes，仅被 `LottieAnim.vue` 的 glob 收集。#39 可进入独立清理批次，但删除前仍需确认非 Vue 动态引用与构建依赖。
7. `src/views/accounts/List.vue` 的 `statusPoll` 与 `useAutoRefresh` 同时每 30 秒刷新账号；`useAutoRefresh` 已包含回前台刷新，因此重复轮询已在试点中移除。
8. #58 的 `nowTs` 响应式时钟本来就存在，但 `loginCodeFresh()` 与 `codeAge()` 没有读取它；试点已改为读取 `nowTs.value`，使验证码到期和年龄文案随时钟更新。

## F. 下一步执行计划

1. 补齐全仓 grep：LottieAnim 引用、GroupSelector 调用与 dm mode、会话页死代码、Tailwind token、管理表格 format、四个管理页关联。
2. 完成 #1–#180 全量核验和分类统计。
3. 以 `src/views/accounts/List.vue` 为第一试点，集中处理确认的问题，并逐项附逻辑回归步骤。
4. 通过试点后批量修复组件层、横切层、租户端、管理端。
5. 每批执行 lint、typecheck、build、可用单测；涉及路由/状态/轮询/导出生命周期的改动增加手动或 E2E 验收。

**本台账是第一版审计记录，不表示已完成 180 项核验，也不表示已开始修改业务页面。**


## G. Accounts 页面试点（第一轮整改已提交）

- 文件：`src/views/accounts/List.vue`
- 提交：`dea68325054a92b44a6e9fcc956fddba5ae87748`
- 本批只修改账号列表页面，不改变账号 API、任务调度或业务状态。

| 对应问题 | 修改 | 触及层 | 回归步骤 |
|---|---|---|---|
| #58 | `loginCodeFresh()` 与 `codeAge()` 改为读取已有响应式 `nowTs.value`，不再各自调用非响应式 `Date.now()` | 触及逻辑 | 登录验证码在 10 分钟内显示；等待跨分钟后年龄文案更新；超过 10 分钟后验证码隐藏 |
| #68 | 删除页面独立的 `statusPoll` 与重复 `visibilitychange` 监听；保留 `useAutoRefresh` 统一每 30 秒刷新账号、统计和任务摘要，并在回前台刷新 | 触及逻辑 | Network 面板确认每个 30 秒周期只有一条账号列表刷新；切后台不轮询；回前台只补一次刷新 |
| #57 | 暂未改动 36px 按钮高度，待完成移动端布局实测后决定是否调整，避免未经验证导致账号卡片挤压 | 暂不整改 | 320/375/430px 视口检查触控尺寸与卡片布局 |

### 本批验证状态

- [x] 源码修改已提交到独立工作分支
- [x] 确认 `useAutoRefresh` 自带页面可见性暂停与回前台刷新
- [ ] 前端 lint / typecheck / build
- [ ] 账号验证码时间变化手动回归
- [ ] 网络面板轮询次数回归
- [ ] 桌面 / 移动端三主题回归

**注意：GitHub 源码提交不等于构建或浏览器回归已通过。当前环境尚未执行这些验证。**


## H. 组件层集中整改（第二批源码修改）

| 问题 | 文件 | 修改 | 提交 | 触及层 | 验收 |
|---|---|---|---|---|---|
| #4 | `src/components/ui/CountUp.vue` | 动画数字使用输入值的前缀/后缀格式，保留 `¥` 等货币符号 | `51740bc` | 触及逻辑 | 待运行测试：`¥0 → ¥120`、纯数字、单位后缀 |
| #20 | `src/components/ui/Breadcrumb.vue` | 仅当 items 首项已是当前主题首页时才隐藏固定首页入口，避免重复且保留其他路径的首页导航 | `e7be58b`, `042fcc1` | 触及逻辑 | 待租户端/管理端路由逐页检查 |
| #36 | `src/components/ui/EmptyState.vue` | title 移入 image slot 的标题区域，description 交给 TDesign description | `5827ff5` | 纯展示 | 待空态页面截图回归 |
| #38 | `src/components/ui/ProgressRing.vue` | 进度弧与文字统一使用夹紧到 0–100、精度 0.1 的值 | `ec0f05b` | 纯展示 | 待 0、0.5、99.5、100、越界值测试 |
| #7 | `src/components/LiveTaskFeed.vue` | 倒计时期间用 Clock3，其他状态才显示旋转 Loader2 | `665a62c` | 纯展示 | 待等待发送状态手动检查 |

### 当前验证状态

- [x] 5 个组件源码修改已提交
- [ ] `npm run typecheck`
- [ ] `npm run test`
- [ ] `npm run build`
- [ ] 桌面/移动端三主题视觉回归

此批尚无 CI 或本地构建结果；代码已提交不等于验证通过。


## I. Accounts 轮询方案复核更正

第一次合并尝试直接移除 `statusPoll`，但进一步检查 `src/stores/account.ts` 后发现：`useAutoRefresh` 调用的是非强制 `fetchAccounts()`，会受 Store 60 秒新鲜期限制。单纯移除强制轮询会把账号状态刷新降到最多约 60 秒一次，因此该方案不保留。

最终提交 `f7b392b` 改为单一 `refreshAccountView()`：
- 每 30 秒统一调用 `fetchAccounts(true)`、`loadStats()`、`loadTaskSummary()`；
- 页面隐藏时不刷新；
- 回到前台立即刷新；
- 使用 `refreshInFlight` 防止请求尚未完成时重复启动下一轮；
- 卸载时清理 interval 与 visibilitychange 监听。

该修正保留原有 30 秒账号强制刷新语义，同时消除重复定时器。仍需在浏览器 Network 面板验证实际请求节奏。


## J. 横切与页面细节集中修复（第三批源码修改）

| 问题 | 文件 | 修改 | 提交 | 触及层 | 验收 |
|---|---|---|---|---|---|
| #51 | `src/views/dashboard/Index.vue` | ECharts 图例改读实际存在的 `--color-text-secondary`，fallback 改为有效 CSS var | `8644b38` | 触及逻辑 | 待三主题图表验证 |
| #99/#100 | `src/views/workspace/Scrape.vue` | Excel 导出 HTML 不再依赖应用 CSS 变量；CSV/XLS 下载 URL 均延迟释放 | `ee3c1c9` | 展示 + 逻辑 | 待下载文件检查及连续导出回归 |
| #109 | `src/views/raffle/Index.vue` | 清空参与记录改用全局 `ask()`，不再调用原生 confirm | `17db125` | 触及逻辑 | 待桌面/移动端确认、取消和删除回归 |
| #121 | `src/views/NotFound.vue` | 仅存在站内 history state 时返回；直接打开 404 时回 dashboard | `bd1e02e` | 触及逻辑 | 待直接访问 404、站内跳转后访问两种场景回归 |
| #142 | `src/views/admin/codes/List.vue` | 管理端批量操作浮条距底从 92px 改为 20px + safe-area | `9c48b53` | 纯展示 | 待移动端底部遮挡检查 |
| #145 | `src/views/admin/support/Config.vue` | 二维码预览改 object-contain，避免裁切二维码定位点 | `0f6ccca` | 纯展示 | 待不同尺寸二维码扫码测试 |
| #175 | `src/styles/tailwind.css` | focus-visible 排除规则改为 class token 边界，避免误伤 `text-*` 等普通工具类 | `75a27ec` | 纯展示 | 待键盘导航检查按钮、链接与 TDesign 控件 |

**以上修改均在审计工作分支；尚未通过本地 lint/typecheck/test/build，也没有浏览器实测结果。**


## K. 管理端表格与语义色修复（第四批源码修改）

| 问题 | 文件 | 修改 | 提交 | 触及层 | 验收 |
|---|---|---|---|---|---|
| #131 | `src/views/admin/dashboard/Index.vue` | 两处 `bg-warning/15 text-warning` 改为已有 amber token | `2a44778` | 纯展示 | 待管理端主题视觉回归 |
| #132 | `src/views/admin/dashboard/Index.vue` | created_at 列改用 `#created_at` 插槽格式化日期，移除无效 format 元数据 | `c1441e8` | 触及逻辑 | 待桌面表格日期检查 |
| #138 | `src/views/admin/tenants/List.vue` | 增加 created_at 插槽，按 zh-CN 日期展示 | `e98aecc` | 触及逻辑 | 待桌面表格日期检查 |
| #159 | `src/views/admin/codes/List.vue` | 增加 plan_id、created_at 插槽；删除 TDesign 不会消费的 format 元数据 | `d434078` | 触及逻辑 | 待套餐名与创建日期检查 |
| #159 | `src/views/admin/audit/Logs.vue` | 删除未被使用的 format 属性；同名插槽原本已正确渲染，属于技术债清理而非用户可见 bug | `cb53076` | 纯展示/清理 | 确认审计日志三列仍由插槽格式化 |

以上管理端修改未执行构建和浏览器回归。


## L. MobileTabBar 路由问题修复

- 文件：`src/components/MobileTabBar.vue`
- 新增：`src/components/mobileTabRuntime.ts`
- 新增测试：`src/components/mobileTabRuntime.test.ts`
- 组件提交：`ded9030`；路由 helper 与测试提交：`07d2755`、`cf5a386`

修改内容：
1. Tab 激活态不再用 `startsWith(t.path)` 比较完整动态路径，而是按 workspace 功能段匹配，切换账号后当前群发/任务 Tab 仍保持高亮。
2. 当前没有选中账号时，群发和任务 Tab 不再拼出 `/workspace/none/...`，统一跳转到 `/accounts`。
3. 增加 helper 单测覆盖无账号 fallback、切换账号、任务详情嵌套路由和固定 Tab 匹配。

验收状态：
- [x] 源码修改已提交
- [x] 新增 Vitest 用例
- [ ] 执行 `npm run test -- src/components/mobileTabRuntime.test.ts`
- [ ] 移动端实际切换账号与无账号点击回归
- [ ] 三主题视觉回归

注意：新增单测尚未执行，不能标记为通过。


## M. 死代码独立清理批次

本批按“清理，不算修 bug”的原则单独提交：

| 对象 | 依据 | 清理内容 | 状态 |
|---|---|---|---|
| `src/views/sessions/Index.vue` | `src/router/index.ts` 的 `/sessions` 路由明确 redirect 到 `/accounts`；页面不再被路由加载，源码搜索未找到其他引用 | 删除孤儿页面文件 | 已提交 |
| `src/components/ui/LottieAnim.vue` | 当前分支全部 58 个 Vue SFC 未发现引用；源码搜索无引用结果 | 删除未使用组件 | 已提交 |
| `src/assets/lottie/empty-tasks.json` | 367,199 bytes，仅由已删除的 LottieAnim glob 收集 | 删除未使用动画资源 | 已提交 |
| `lottie-web` | 源码仅由已删除的 LottieAnim 使用，package-lock 中为独立直接依赖 | 从 package.json 和 package-lock.json 移除 | 已提交 |
| GroupSelector dm 分支 | 组件当前确有 dm Tab、loadDmUsers、过滤、全选、选择逻辑 | 不删除，维持功能 | 误报排除 |

清理提交：
- `210a1d2` 删除 sessions 孤儿页面
- `2785822` 删除 LottieAnim 组件
- `6f52fe7` 删除 367KB 动画 JSON
- `391b12f` / `e7dff0c` 移除 lottie-web 直接依赖并同步锁文件

验收仍待 `npm ci`、`npm run typecheck`、`npm run test`、`npm run build`。本地环境无法连接 GitHub，尚未检出代码执行构建；不能把清理标记为构建通过。


## N. 组件层补充修复

| 问题 | 文件 | 修改 | 提交 | 触及层 | 验收 |
|---|---|---|---|---|---|
| #26 | `src/components/ui/ConfirmActions.vue` | 移动端按钮改为 column，保持 DOM 顺序与视觉/键盘焦点顺序一致 | `49d0202` | 纯展示/可访问性 | 待移动端焦点顺序检查 |
| #27 | `src/components/ui/PullDownRefresh.vue` | 无滚动容器时检查页面 scrollTop，只有整页位于顶部才能触发下拉 | `7200d8d`, `7a89b48` | 触及逻辑 | 待页面顶部/中段触摸回归 |
| #34 | `src/components/ui/SupportFloat.vue` | 记录最后拖动时间，拖拽结束后的 400ms 内抑制合成 click，避免拖动后意外展开 | `7029378`, `bfd6777` | 触及逻辑 | 待 iOS/Android 拖拽后松手回归 |

以上逻辑项均需手动回归；当前尚未执行浏览器/真机验证。


## O. 账号卡片移动端触控尺寸

- 文件：`src/views/accounts/List.vue`
- 提交：`b14ff6d`
- 将移动端账号卡片的操作按钮和“更多”按钮最小高度从 36px 提升至 44px。
- 验收：待 320/375/430px 视口检查按钮可点击区域、卡片高度和是否出现横向溢出。


## P. 管理端角标与群发目标预览

| 问题 | 文件 | 修改 | 提交 | 触及层 | 验收 |
|---|---|---|---|---|---|
| #130 | `src/views/admin/dashboard/Index.vue` | 演示数据角标移入 `main.relative`，定位锚点明确 | `3170fa6` | 纯展示 | 待管理端 dashboard 实机定位检查 |
| #84 | `src/views/workspace/Broadcast.vue` | 已选目标 chips 预览最大高度从 96px 提升至 160px | `0f8a8c0` | 纯展示 | 待 20 个目标、窄屏和页面滚动回归 |


## Q. 账号批量操作浮条遮挡修复

- 文件：`src/views/accounts/List.vue`
- 提交：`47504ae`
- 多选时为账号列表主容器增加移动端底部留白（桌面端不增加），让列表末尾能够滚动到浮动批量操作栏上方。
- 验收：移动端选中 5 个以上账号，滚动至列表最后一项，确认最后一行和操作栏不重叠。


补充：#38 组件复核时发现背景轨道曾使用字面量 `stroke-width="stroke"`，而非 Vue 属性绑定。已在 `553dba7` 改为 `:stroke-width="stroke"`，确保轨道粗细与两段进度弧一致。

## R. SupportFloat 剪贴板降级修复

- 文件：`src/components/ui/SupportFloat.vue`
- 提交：`9aec6528cf49999a7120a1255e76bc51b8843c54`
- 问题：原实现对 `navigator.clipboard` 使用可选链后直接串接 `.then().catch()`。当 Clipboard API 不存在时，整个表达式短路为 `undefined`，传统复制兜底不会执行；同时旧兜底不检查 `execCommand('copy')` 的返回值，可能错误提示复制成功。
- 修改：
  1. 将复制流程改为 async/await，显式区分 Clipboard API 可用与不可用。
  2. Clipboard API 缺失或权限拒绝时，尝试传统 textarea 复制。
  3. 传统复制检查返回值并在 finally 中清理临时 textarea。
  4. 只有复制确实成功才提示成功；所有路径失败时提示用户手动复制。
- 归类：真缺陷；触及逻辑。
- 回归步骤：
  - HTTPS/支持 Clipboard API：点击复制，确认剪贴板内容正确且显示成功。
  - 禁用 Clipboard API：确认自动走 textarea 降级且内容正确。
  - 模拟 Clipboard API 拒绝权限：确认降级成功。
  - 同时让降级复制失败：确认显示失败提示，不显示“已复制”。
- 验收状态：源码修复已提交；尚未运行 lint/typecheck/build，也未进行浏览器权限场景回归。

## S. 组件层与目标选择器补充核验

| 文件 | 问题与修改 | 提交 | 触及层 | 回归步骤 | 状态 |
|---|---|---|---|---|---|
| `src/components/ui/ProgressRing.vue` | 复核发现总进度弧仍把 `stroke-width="stroke"` 当作字面字符串，SVG 无法将其解析为数值；已改为 `:stroke-width="stroke"`，使底轨、总进度弧、成功弧统一使用传入粗细 | `ccfe359975278054c865524bd80f883eb052a73d` | 纯展示 | 分别传入 stroke=2/4/8，检查三段线宽一致；验证 0%、部分进度、100% | 源码已修，待组件/浏览器回归 |
| `src/components/selectors/TargetSelector.vue` | 手动目标校验注释规定 `-100` 群 ID 后至少 10 位数字，但两个校验正则都写成恰好 10 位，较长的有效 chat ID 会被误判无效；已统一改为至少 10 位 | `9b83a31abb4dadcc1e2b666cb8c002ca9bf07355` | 触及逻辑 | 输入 `-1001234567890` 与更长 ID 均应识别；少于 10 位仍应提示无效；确认手动输入和无效计数一致 | 源码已修，待输入回归 |

以上两项是源码直接确认的问题，不代表已通过浏览器测试。

## T. Toast 默认时长与应用配置统一

- 文件：`src/utils/toast.ts`
- 提交：`d2a7f7d8f5eabbff6dc3920c510b606056331673`
- 发现：`src/App.vue` 的全局 `<Toaster>` 声明默认时长为 2500ms，旁边注释也明确写明 Toast 2.5 秒自动消失；但 `useToast().add()` 每次都传入 `t.duration || 4000`，实际把全局默认覆盖成 4 秒。
- 修改：默认时长统一为 2500ms，并使用 nullish coalescing 保留调用方显式传入的 `0` 等值，不再误用默认值覆盖。
- 归类：真缺陷/横切体验不一致；触及逻辑。
- 回归步骤：不传 duration 时，Sonner 定时器应使用 2.5 秒；显式传入 5000/0 时检查 Sonner 定时器是否保留调用值。
- **未解决的关联项**：`src/styles/ui-kit.css` 原先 CSS 在 2.3 秒启动、300ms 后结束，而 Sonner 在 2.5 秒移除节点；WAAPI 同样在 2.3 秒启动、250ms 后结束，存在退场尚未完成就被移除的窗口。已在后续 V 项将两种退场都提前到 2.2 秒启动。因此自定义 duration 目前仍可能被固定 2.2 秒视觉退场提前隐藏；本次仅保证默认 2.5 秒场景的退场时序，不宣称自定义时长已完整支持。该交互需单独实测并调整动画调度。
- 验收状态：默认值源码已修；淡出时序及自定义 duration 尚未完成浏览器回归。

## U. 发送日历提示文案与键盘可访问性

- 文件：`src/components/SendCalendar.vue`
- 提交：`abc1d899ab0da2515fca1008666c53f3bc3c4fde`、`5e7cf7ec2a723f7317b58590a95f894e3264ce8b`
- 发现：日期格子的 `cell.day` 是 boolean，却被当成日期数字写入 tooltip，因此鼠标悬停会显示“true 成功发送 N 条”。
- 修改：tooltip 改用 `cell.date` 并明确“日”；月份切换按钮补充可访问名称，月份标题使用 `aria-live="polite"`，便于读屏感知月份变化；热力图图例改为与实际单元格使用的 20/40/60/80% 四档一致。
- 归类：真 UI 文案缺陷 + 可访问性细节；触及层为纯展示/模板。
- 回归步骤：悬停任意日期应显示“X 日成功发送 N 条”；键盘聚焦月份切换按钮应读出上/下月用途；切换后读屏应播报月份变化。
- 验收状态：源码已修；待浏览器和屏幕阅读器回归。

## V. Toast 退场动画提前结束，避免节点移除截断

- 文件：`src/App.vue`、`src/styles/ui-kit.css`
- 提交：`fbc63bae56bb01f4a6026d4191d869f2d62dd21f`、`09c685a6a2ff63097533c2abb2c837079fb0a4be`
- 发现：默认 Toast 生命周期为 2500ms，但 CSS 淡出从 2300ms 开始、持续 300ms；WAAPI 从 2300ms 开始、持续 250ms。Sonner 可能在 2500ms 移除节点，导致两条淡出动画未完成便被截断。
- 修改：CSS 与 WAAPI 统一提前至 2200ms 开始，淡出在约 2450ms 完成，为 2500ms 节点移除留出约 50ms 缓冲。
- 归类：真 UI 动效时序缺陷；触及层为样式 + 逻辑。
- 回归步骤：连续触发成功/失败/警告 toast，录屏逐帧检查退场是否完整；检查快速连续触发时旧 toast 不闪回、不残留。
- 限制：当前退场动画仍按默认 2.5 秒时长固定调度；显式传入更长或更短的自定义 duration 可能与固定动画时刻不一致，需后续统一动态时长策略。
- 验收状态：源码已修；尚未浏览器慢放回归。

## W. 活动任务进度包含失败目标

- 文件：`src/components/LiveTaskFeed.vue`
- 提交：`f8f709dd182b47460652a6718d759267dec445bb`、`a26902f927751c85047b83e35d352aaadbb5b71b`、`1dc0bd692b7e0c0b18ad314cfdfc293957fe9814`
- 发现：任务终态判断已使用 `done + failed` 判断所有目标是否处理完毕，但进度条百分比和计数只使用 `done`。有失败目标时，任务已处理完仍可能显示低于 100% 的进度。
- 修改：新增 `processed = done + failed`（限制在 0 到 total），活动进度条、计数和最近任务总结统一显示已处理目标数。
- 归类：真 UI 状态表达缺陷；触及逻辑。
- 回归步骤：total=10、done=8、failed=2 时显示 10/10 和 100%；done=6、failed=1 时显示 7/10 和 70%；无 failed 字段时按 0 处理；total=0 时不除零。
- 验收状态：源码已修；待单测/任务实时页面回归。

## X. 两个选择器的行级控件可访问性

- 文件：`src/components/selectors/GroupSelector.vue`、`src/components/selectors/TargetSelector.vue`
- 提交：`f080273bbdb417195002df9f958f11eb6a7d11bc`、`66361a6df37a00dc49aa43233bd700655b3222dd`
- 发现：私聊列表和目标群列表的卡片外层声明为可聚焦的 `role="button"`，但没有 Enter/Space 键盘处理；真正可操作的是内部复选框，其中私聊复选框无可访问名称、目标群复选框仅有笼统的“选择”名称。
- 修改：移除两个列表中没有键盘行为的伪按钮角色/Tab 停靠点；给私聊复选框补充联系人名称，给群组复选框补充群组标题作为 `aria-label`，让键盘和读屏用户聚焦实际选择控件。
- 归类：可访问性缺陷；纯展示/模板。
- 回归步骤：Tab 应聚焦私聊复选框；读屏分别播报“选择私聊对象 + 联系人名称”和“选择群组 + 群组名称”；Space 可切换复选框；鼠标点击卡片仍可切换。
- 验收状态：源码已修；待键盘与屏幕阅读器回归。

## Y. StatCard 遗留展示映射清理

- 文件：`src/components/charts/StatCard.vue`、`src/views/dashboard/Index.vue`
- 提交：`918dbb544393e364787c7f8942f028169020f045`、`769b0fe1f96c3e655b8fd186e891848cc375f09e`
- 发现：StatCard 当前模板只展示 label/value/trend，但脚本仍保留未被模板使用的图标映射、背景色映射和文字色映射；Dashboard 也保留了一个未使用的 StatCard import。
- 修改：删除未使用的映射计算与图标依赖，移除 Dashboard 死 import；保留 icon/color props 声明，避免影响仍可能存在的调用方属性传递。
- 归类：死代码/技术债，不计作用户可见 bug。
- 验收：typecheck/build；若其他页面仍使用 StatCard，确认 label/value/trend 正常。
- 状态：源码已清理；待构建验证。

## Z. 进度组件与发送日历读屏信息补齐

| 文件 | 发现与修改 | 提交 | 归类/触及层 | 回归验收 |
|---|---|---|---|---|
| `src/components/SendCalendar.vue` | 日期格此前只有鼠标 `title`，触屏/读屏用户无法获知日期与发送量；为每个有效日期格增加图片语义及完整 `aria-label`（年月日、成功发送数量） | `7127c13e9a1dffda1c1fde484571bdc23ae7378` | 可访问性缺陷；模板 | 用 NVDA/VoiceOver 浏览日历，逐格应读出日期和发送数量；空白补位格不应被读出 |
| `src/components/ui/ProgressRing.vue` | 成功弧原先仅做上限限制，负数成功值会生成负 dash 长度；现在成功值同时夹紧 0 到当前总进度，并为环形整体增加“任务进度 X%”读屏名称 | `2de0fa1dcd8cfa76820243f631e46fafcc2e873c` | 边界显示缺陷 + 可访问性；逻辑/模板 | success=-1/0/超过 percent、percent=-10/50/120、NaN 时检查弧长有效；读屏应播报当前百分比 |

- 状态：源码已修，未执行 typecheck/build；读屏与不同进度边界需回归。

## AA. 设置页交互与确认弹窗统一

- 文件：`src/views/settings/Index.vue`
- 提交：`e2e85ce66fd7b399d4e4c456f00362db8657c786`
- 发现：
  1. 套餐升级弹窗内的套餐卡片使用 `role="button"` 和 `tabindex="0"`，但没有 Enter/Space 键盘激活行为，也没有向读屏暴露选中状态。
  2. 删除文本分组仍使用浏览器原生 `window.confirm`，与全站统一确认弹窗不一致。
- 修改：套餐卡片增加 Enter/Space 激活和 `aria-pressed`；删除文本分组改用项目统一 `ask` 确认工具，明确危险操作标题和删除按钮文案。
- 归类：真可访问性缺陷 + UI 一致性；触及逻辑。
- 回归步骤：Tab 聚焦套餐卡片后 Enter/Space 均可选中，读屏播报 pressed 状态；删除文本分组时展示统一危险确认弹窗，取消不发请求，确认后删除并刷新。
- 验收状态：源码已修；待键盘、读屏和删除流程回归。

## AB. 任务列表筛选状态读屏反馈

- 文件：`src/views/workspace/Tasks.vue`
- 提交：`51f10b73671d6aa5cbd0055b78742a76e022cbc2`
- 发现：任务类型与创建时间筛选使用按钮和视觉高亮表达当前选择，但没有向辅助技术提供选中状态。
- 修改：两组筛选按钮增加动态 `aria-pressed`，与现有视觉 active 状态保持一致。
- 归类：可访问性缺陷；模板。
- 回归步骤：切换任务类型/日期范围后，读屏应播报当前按钮已按下；每组始终只有一个按钮处于 pressed 状态。
- 验收状态：源码已修；待读屏与键盘回归。


## AC. 管理端客服配置读取失败保护

- 文件：`src/views/admin/support/Config.vue`
- 提交：`377cbaeae4ee2405c09fcb1e3bd8dae55dec84be`
- 发现：页面初始化读取客服配置失败时，原逻辑静默将表单重置为空值，但没有阻止保存。管理员可能在接口故障期间误保存空表单，覆盖已有客服联系方式。
- 修改：增加配置读取中、读取成功和读取失败状态；只有接口成功返回(包括首次部署时成功返回空配置)才允许保存；读取失败时显示明确错误和重试入口；保存过程中禁用重复提交并显示 loading。
- 同时修正联系方式类型标题的 `for` 指向不存在元素的问题，改为普通分组标题，实际按钮组保留 `role=group` 与 `aria-label`。
- 归类：配置数据保护缺陷 + 表单可访问性；触及逻辑/模板。
- 回归步骤：模拟 GET `/admin/support` 网络失败，确认出现错误提示且保存按钮不可用；恢复接口后点击重新读取，确认成功后才可保存；首次部署接口成功返回空值时，仍可填写并保存；连续点击保存只发出一次请求。
- 验收状态：源码已提交；未运行 typecheck/build，也未执行网络故障场景浏览器回归。

## AD. 管理端风控探测配置读取失败保护

- 文件：`src/views/admin/probe/Config.vue`
- 提交：`7e19ff1d8852b5e602ae720fb8d81087dd015c65`
- 发现：初始 `getProbe()` 失败时页面只依赖拦截器提示，表单保留空默认值，保存按钮仍可用；管理员可能在读取失败后误保存空值，覆盖全局探测配置。
- 修改：增加读取中/成功/失败状态；读取成功前禁用保存；失败时显示错误与重新读取入口；保存时防止重复提交，保存后等待重新读取完成。
- 归类：配置数据保护缺陷；触及逻辑/模板。
- 回归步骤：模拟 GET `/admin/probe` 失败，确认保存不可用；恢复接口后重新读取，确认配置值恢复且可保存；保存后确认页面重新拉取最新 `updated_at` 和字段值。
- 验收状态：源码已提交；未运行 typecheck/build，也未进行网络故障场景回归。

## AE. 管理端公告优先级选择状态

- 文件：`src/views/admin/announcements/List.vue`
- 提交：`d00946ecd05468b32d040b8b88f8561288592a89`
- 发现：普通/紧急优先级通过视觉样式表达当前选项，但按钮没有 `aria-pressed`，读屏用户无法得知当前选择；标题 `label for` 也指向不存在的表单控件。
- 修改：两个选项按钮增加动态 `aria-pressed`，将不对应表单控件的标题改为普通分组标题。
- 归类：可访问性缺陷；模板。
- 回归步骤：键盘切换普通/紧急后，读屏应播报对应按钮 pressed 状态；每次只能有一个选项处于 pressed。
- 验收状态：源码已提交；待键盘/读屏回归，未运行 typecheck/build。

## AF. 注册成功后阻止重复提交 + 表单标签关联

- 文件：`src/views/auth/Register.vue`
- 提交：`86fa9e7c27a24acc4bf3a155bf72dcd8359baa0b`
- 发现：注册成功后按钮重新启用，自动跳转登录页前的 3 秒窗口内仍可重复提交注册请求；同时可见 label 未通过 `for/id` 与输入框关联，主要依赖 aria-label。
- 修改：注册成功状态禁用提交按钮，处理函数增加 loading/success 双重幂等保护；用户名、昵称、密码、确认密码、卡密输入框补齐稳定 id 与 label 关联；错误/成功提示增加 alert/status live region。
- 归类：重复提交防护 + 表单可访问性；触及逻辑/模板。
- 回归步骤：注册成功后快速连续点击提交，确认不会发出第二次注册请求；检查五个字段点击标签可聚焦对应输入框；读屏能及时播报错误和成功状态。
- 验收状态：源码已提交；未运行 typecheck/build，未做真实注册接口回归。


## AG. 登录表单校验反馈与读屏播报

- 文件：`src/views/admin/Login.vue`、`src/views/auth/Login.vue`
- 提交：`5a5610bda9cce8c6f866ecb2c901e7194b654d48`、`70cd1b3e8653a1512ad73a15103b03cc968c6bf2`
- 发现：管理端 `validateForm()` 会写入 `errors.password`，但模板没有渲染该字段错误；密码为空时只显示用户名错误，用户无法知道密码也必填。两个登录页的异步登录失败提示都没有 live region，读屏用户可能不会及时获知结果。
- 修改：管理端补显示密码字段错误；用户名/密码字段错误使用 alert 语义；租户端和管理端登录失败提示增加 assertive live region。
- 归类：表单反馈缺陷 + 可访问性；模板。
- 回归步骤：管理端分别留空用户名、密码、两者同时留空，确认对应字段错误都显示；输入错误凭证，确认错误提示可被读屏播报；租户端登录失败时确认错误消息即时播报。
- 验收状态：源码已提交；未进行浏览器/读屏回归，未运行 typecheck/build。


## AH. 管理端租户列表移动端日期兜底与创建表单标签

- 文件：`src/views/admin/tenants/List.vue`
- 提交：`6412d3d5e7ce03098473c787fe8f9161c5c2a2b9`
- 发现：
  1. 桌面端注册时间列对空 `created_at` 显示 `-`，但移动端卡片直接格式化日期；当 API 返回空值时会出现 `Invalid Date`，两种布局表现不一致。
  2. 创建租户弹窗的三个可见 label 未通过 `for/id` 关联输入框，点击标签无法聚焦对应输入框。
- 修改：
  1. 移动端注册时间增加与桌面端一致的空值兜底，空值显示 `-`。
  2. 为租户名称、登录用户名、登录密码补齐稳定 input id 和 label for。
- 归类：移动端数据展示边界缺陷 + 表单可访问性；模板。
- 回归步骤：模拟租户 `created_at=null`，桌面和移动端均显示 `-`；点击三个表单标签分别聚焦对应输入框；正常日期仍按 zh-CN 格式显示。
- 验收状态：源码已提交；尚未运行 typecheck/build，也未进行浏览器/读屏回归。


## AI. 加群任务创建表单标签关联与分组语义

- 文件：`src/views/workspace/Join.vue`
- 提交：`976e3fa75d339b7a3dde86fecf51a1f5c8054ddc`
- 发现：执行账号下拉框、多账号分配方式、邀请目标群、邀请来源群、群链接输入框均有可见 label，但未通过 `for/id` 关联实际控件；多账号列表标题也使用了不对应具体控件的 label。
- 修改：
  1. 为上述五个控件补齐稳定 id，并将可见 label 的 for 指向对应控件。
  2. 多账号列表标题改为普通文本，列表容器增加 `role="group"` 与明确的 `aria-label`，避免把分组标题伪装成单个控件标签。
- 归类：表单可访问性缺陷；模板。
- 回归步骤：点击每个可见字段标签应聚焦对应控件；Tab/读屏浏览多账号复选框时应播报“选择执行账号”分组和每个账号名称；多选和提交行为保持不变。
- 验收状态：源码已提交；尚未运行 typecheck/build，也未做浏览器/读屏回归。


## AJ. 账号管理弹窗表单标签关联

- 文件：`src/views/accounts/List.vue`
- 提交：`e25068cd8c4ca09245691a42f367911f719de4c4`
- 发现：Session 导入、备注编辑、代理编辑、群转移目标账号、克隆资料来源用户名等可见标签未关联对应控件；群转移间隔标题容易被误认为单个输入框标签，但实际对应两个输入框。
- 修改：
  1. 为 Session 文本区、Session 文件选择、备注、代理、转移目标账号、克隆来源用户名补齐 id/for。
  2. 将双输入框的间隔标题改为普通分组说明，两个输入框继续保留各自明确的 aria-label（最小间隔/最大间隔）。
- 归类：表单可访问性缺陷；模板。
- 回归步骤：点击标签应聚焦对应控件；读屏分别识别最小/最大间隔；Session 导入、代理测试、群转移、克隆资料原有行为不变。
- 验收状态：源码已提交；尚未运行 typecheck/build，也未做浏览器/读屏回归。


## AK. 群发任务消息、媒体类型与定时启动标签关联

- 文件：`src/views/workspace/Broadcast.vue`
- 提交：`60277be2b88d325c5bea07c13039d78c72c698af`
- 发现：消息内容、附带媒体类型、单次任务定时启动均有可见标签，但没有显式关联表单控件。
- 修改：为消息文本域、媒体类型选择器、定时启动日期时间选择器补齐稳定 id，并将对应 label 的 for 指向控件。
- 归类：表单可访问性缺陷；模板。
- 回归步骤：点击三个标签应聚焦/激活对应控件；键盘操作可正常输入消息、选择媒体类型和设置定时启动；创建群发任务流程不变。
- 验收状态：源码已提交；TDesign 控件对 id/label 关联的实际表现尚待浏览器与读屏验证，未运行 typecheck/build。


## AL. 群组发现/导入弹窗表单标签与间隔说明

- 文件：`src/views/groups/Discover.vue`
- 提交：`19c2098f6d4a4df1af5dbf5baddc89c7df1a8910`、`bf6c9ae6e660d094e1021a058f20d86b63b03b9d`
- 发现：导入文件、解析账号控件缺少显式标签关联；群链接粘贴框没有可见字段标签；批量加群的“每账号分配”标签未关联数字输入；加群间隔标题横跨两个输入框，不应伪装成某一个输入框的 label。
- 修改：补齐文件/账号/链接输入的 id 与 label for；为每账号分配输入建立 label 关联；将双间隔输入的共同说明改为普通分组说明，两个输入框保留各自 aria-label。
- 归类：表单可访问性缺陷；模板。
- 回归步骤：点击文件、解析账号、粘贴群链接、每账号分配标签分别定位对应控件；读屏能区分最小/最大间隔；导入与批量加群行为保持不变。
- 验收状态：源码已提交；尚未运行 typecheck/build，也未做浏览器/读屏回归。


## AM. 管理端配置保护、登录标签、租户详情导航与 Toast 生命周期复核

### 已确认并整改

| 文件 | 发现 | 修改提交 | 触及层 | 回归步骤 |
|---|---|---|---|---|
| `src/views/admin/plans/Index.vue` | 套餐列表读取失败/返回空列表时，`plans` 为空但“保存全部”仍可点击；`ok === plans.length` 在 0===0 时会显示“全部套餐已保存”，造成未保存却成功的假反馈 | `a2c9c04675564ddd8f9b1f64e366ac4c24232306` | 触及逻辑/展示 | 模拟 listPlans 失败、空数组、正常返回；失败/空列表时保存按钮禁用且不出现成功提示；正常加载后保存全部档位 |
| `src/views/admin/Login.vue` | 管理员登录用户名/密码使用普通 div 作为可见标题，没有关联输入控件 | `4ad798de18d8ae5d5faecafd3b5474da70ccaee0` | 模板 | 点击用户名/密码标签聚焦对应输入框；NVDA/VoiceOver 应读出字段名称；Enter 登录行为不变 |
| `src/views/admin/tenants/Detail.vue` | 详情页返回按钮直接调用 history.back，直接打开深链接时缺少稳定的站内回退目的地；修改套餐弹窗两个字段的 label 未关联控件 | `82f3941ced214230a618a78373674b72d4273a68` | 模板/导航逻辑 | 新标签页直接打开租户详情，点击返回应进入租户列表；点击套餐/到期时间标签应聚焦相应控件 |

### Toast 自定义时长缺陷（已修复源码，待运行验证）

- 文件：`src/App.vue`、`src/styles/ui-kit.css`、`src/utils/toast.ts`
- 发现：Toast 工具允许调用方传入 `duration`，Sonner 也接收该时长；但 App.vue 的 WAAPI 和 ui-kit.css 的 `toast-auto-hide` 都硬编码在 2200ms 触发淡出。自定义 5 秒提示仍会在 2.2 秒提前消失，短时长提示也可能出现动画与 Sonner 移除时点不一致。
- 修改：移除固定 2200ms 的 JS 退场计时器和 CSS 强制 auto-hide；WAAPI 只负责进场，Toast 退场统一等待 Sonner 按实际 duration 将 `data-mounted` 设为 false，再由现有 `island-suck-out` 动画播放。
- 提交：`1f4efa57aeca3c880c8648b7c38dc4a5ccc0ab0f`、`40c5162da25799ab866f2c221225d283ff2f6cd2`
- 回归步骤：分别触发默认 2500ms、显式 5000ms、显式 800ms 的 success/error/info toast；录屏核对实际显示时长由 Sonner 控制、退场动画完整；快速连续触发时确认无残影、无提前消失。
- 验收状态：源码已修；未运行浏览器回归、typecheck 或 build。不可标记为运行时通过。

### 本批核验补充

- `src/styles/modal-system.css` 确实消费 `--modal-kb`，因此 Modal runtime 的 visualViewport 偏移变量不是死代码；#176 仍需 iOS/Android 真机验证。
- `src/views/settings/Index.vue` 套餐卡片的 `role=button` 已有 Enter/Space handler 和 `aria-pressed`，该处不是未处理的伪按钮。
- 管理端套餐矩阵使用横向滚动容器与 `min-w-[720px]`，不应仅凭表格最小宽度将其判为页面横向溢出。
- 本节只记录本轮已复核的文件和问题；180 项全量逐条状态仍在继续核验，未完成项不得推定为通过。


## AN. 管理端表单与数据读取状态补齐

| 文件 | 发现 | 修改提交 | 触及层 | 回归步骤 |
|---|---|---|---|---|
| `src/views/admin/codes/List.vue` | 桌面端/移动端卡密选择复选框没有可访问名称；移动端创建时间未对空值兜底，而桌面端已有 `-` 兜底 | `e115bd21dec89e2a88e6efccb963fdaa2551d115` | 模板 | 读屏应分别播报“选择卡密 + 卡密值”；created_at 为 null 时移动端显示 `-`，正常日期格式不变 |
| `src/views/admin/support/Config.vue` | 联系账号/链接、显示名称可见 label 未通过 for/id 关联输入框（仅有 aria-label） | `e3ccbe1a5cf6d720fed0df3d90907588c88d371b` | 模板 | 点击可见标签聚焦对应输入框；读屏播报与可见标签一致 |
| `src/views/admin/probe/Config.vue` | 三个探测配置字段的可见 label 未关联实际输入控件 | `cc37ecd265b2022f08c243e845fc12c0c8320517` | 模板 | 点击测试号、私有群链接、公开群链接标签分别聚焦对应输入框；原保存/读取流程不变 |
| `src/views/admin/tenants/Detail.vue` | 租户详情 GET 失败时 `tenant` 保持 null，模板没有 loading/error 分支，页面主体空白，仅有 toast；直接进入页面也缺少加载反馈 | `9b656ec9ad63f1bb4c2174d05d88d96fd2d5beda` | 触及逻辑/模板 | 正常加载显示详情；网络失败显示错误与重新加载/返回入口；恢复网络后重新加载显示数据；无效 ID 不发起详情请求 |

### 本批验证状态

- [x] 四处源码修改已提交到 `audit/uiux-full-scan`
- [ ] 前端 lint / typecheck / build
- [ ] 管理端三主题视觉回归
- [ ] 卡密选择读屏回归
- [ ] 配置页标签点击/读屏回归
- [ ] 租户详情 GET 失败与重试浏览器回归

**以上均为源码级整改，不代表浏览器或构建验收已通过。**
