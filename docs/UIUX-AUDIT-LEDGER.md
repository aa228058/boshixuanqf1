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
| #9 | `src/components/MobileTabBar.vue` | `startsWith(t.path)` 将账号 ID 纳入匹配，切换账号后 Tab 可能失活 | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 多账号切换 E2E | 路径匹配需按路由段/名称设计 |
| #10 | `src/components/MobileTabBar.vue` | 当前账号缺失时使用 `none` 生成工作区路由，无有效 fallback | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 无账号点群发 Tab，验证跳转账号页 | 需确认路由守卫目标 |
| #20 | `src/components/ui/Breadcrumb.vue` | 模板固定渲染“首页”，管理端传入仪表盘后重复层级 | 🟡 | 细节/横切 | 已确认 | 触及逻辑 | 租户端、管理端面包屑逐页检查 | 已提交修复 `e7be58b` / `042fcc1`：当 items 首项已是主题首页时不重复插入；首项不是首页时保留首页入口 |
| #26 | `src/components/ui/ConfirmActions.vue` | 移动端使用 `column-reverse`，按钮视觉顺序与 DOM/焦点顺序不一致 | 🟡 | 细节 | 已确认 | 纯展示 | 移动端键盘焦点与视觉顺序手测 | 需确定产品期望按钮顺序 |
| #27 | `src/components/ui/PullDownRefresh.vue` | 无滚动父级时 `canPull()` 返回 true | 🟡 | 待验证 | 已确认（行为） | 触及逻辑 | 页面级滚动容器/无滚动容器两种场景实测 | 源码注释明确这是整页场景的有意 fallback；是否误触需实测 |
| #34 | `src/components/ui/SupportFloat.vue` | 拖拽后 click 未拦截 | 🟡 | 误报/待复核 | 误报（当前代码有防护） | 触及逻辑 | 拖拽后松手、轻点分别手测 | `toggle()` 已检查 `isDragging`；仍需手测事件时序 |
| #36 | `src/components/ui/EmptyState.vue` | `title` 传入 TDesign `description`，真描述另行渲染 | 🟠 | 真缺陷 | 已确认 | 纯展示 | 组件用例/页面手测 | 已提交修复 `5827ff5`：title 放入 image slot 标题区，description 交给 TDesign description；待页面视觉回归 |
| #38 | `src/components/ui/ProgressRing.vue` | 百分比文字 round，圆弧保留小数，显示不一致 | 🟡 | 细节 | 已确认 | 纯展示 | 99.5、0.5、100 等边界值单测 | 已提交修复 `ec0f05b`：弧与文本共用 0.1% 精度的夹紧值；待边界测试 |
| #57 | `src/views/accounts/List.vue` | 移动端账号操作按钮最小高度 36px | 🟡 | 细节 | 已确认 | 纯展示 | 320/375/430 宽度触控检查 | 36px 源码已定位；按 44px 目标评估布局空间 |
| #58 | `src/views/accounts/List.vue` | `loginCodeFresh()` / `codeAge()` 未读取响应式时间依赖 | 🟠 | 真缺陷候选 | 已确认（源码） | 触及逻辑 | 验证验证码年龄是否随时间刷新及 10 分钟到期隐藏 | 已提交修复 `dea6832`：两个函数统一读取 `nowTs.value`；待手动回归 |
| #68 | `src/views/accounts/List.vue` | `statusPoll` 与 `useAutoRefresh` 都触发账号刷新，可能重复轮询 | 🟠 | 横切/性能 | 已确认（双触发存在） | 触及逻辑 | 网络面板统计 30 秒请求次数、切后台恢复 | 已提交修复 `f7b392b`：合并为单一 30 秒强制刷新入口，统一刷新账号/统计/任务摘要，并保留切后台暂停、回前台立即刷新；待网络面板回归 |
| #99 | `src/views/workspace/Scrape.vue` | 导出 Excel 的独立 HTML 使用 CSS 变量 `var(--t-line)` / `var(--t-space-1-5)` | 🟡 | 真缺陷 | 已确认 | 纯展示 | 下载 XLS 并检查边框与内边距 | 已提交 `ee3c1c9`：替换为 Excel 可识别的固定边框色与 px padding |
| #100 | `src/views/workspace/Scrape.vue` | CSV/XLS 导出创建 Object URL 后未 revoke | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 连续导出后检查 URL 生命周期 | 已提交 `ee3c1c9`：两个下载均在触发后 1 秒 revoke URL；待浏览器下载回归 |
| #109 | `src/views/raffle/Index.vue` | 清空记录使用原生 `confirm()` | 🟡 | 细节/横切 | 已确认 | 触及逻辑 | 桌面/移动端确认交互回归 | 已提交 `17db125`：改用全局 `ask()` 确认弹窗；取消不触发删除 |
| #130 | `src/views/admin/dashboard/Index.vue` | 演示数据角标位于 `main` 外，定位上下文可能错误 | 🟡 | 真缺陷候选 | 已确认（结构） | 纯展示 | 管理端 dashboard 实机检查角标位置 | 顶层 span 在 main 前，需确认外层定位上下文 |
| #131 | `src/views/admin/dashboard/Index.vue` | 使用 `bg-warning` / `text-warning`，可能没有对应 Tailwind token | 🟡 | 待核 | 待 grep | 纯展示 | Tailwind 配置/构建 CSS 检查 | 需核对完整 token 定义与生成结果 |
| #132 | `src/views/admin/dashboard/Index.vue` | columns 的 `format` 函数未传给 TDesign columns | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 管理端表格日期列手测 | 映射只保留 colKey/title |
| #138 | `src/views/admin/tenants/List.vue` | 表格列格式化函数可能未被 TDesign 消费 | 🟠 | 真缺陷候选 | 待核 | 触及逻辑 | 检查实际列定义及渲染 | 与 #132 可能同根因，需避免重复计数 |
| #141 | `src/views/admin/codes/List.vue` | 使用 text-yellow/red/green 硬编码语义色 | 🟡 | 误报（当前文件） | 误报 | 纯展示 | 如有其他匹配文件再补证据 | 当前文件全文未检出所述 class |
| #142 | `src/views/admin/codes/List.vue` | 批量操作浮条固定距底 92px | 🟡 | 细节 | 已确认 | 纯展示 | 管理端移动端实机检查遮挡 | 已提交 `9c48b53`：移动端改为距底 20px + safe-area；待检查是否遮挡底部系统手势区 |
| #145 | `src/views/admin/support/Config.vue` | 二维码预览使用 `object-cover`，可能裁切二维码边缘 | 🟠 | 真缺陷 | 已确认 | 纯展示 | 不同长宽比二维码实机扫码 | 已提交 `0f6ccca`：改为 object-contain，并提供白底与内边距；待扫码验证 |
| #152 | `src/views/admin/system/Monitor.vue` | CPU 渐变条 `from-[var(--t-*)] to-[var(--t-*)]` 是否生成/显示异常 | 🟡 | 待验证 | 待跑 | 纯展示 | 构建产物 + 管理端监控页实机检查 | 当前源码是合法的 gradient-stop 写法；静态源码不能证明运行时可见 |
| #159 | `src/views/admin/*`（4 个页面，待逐一列出） | 与 #132/#138 相同的表格 format 映射问题 | 🟠 | 横切 | 待核 | 触及逻辑 | 全部管理表格页面逐项核验 | 先定位四页及共同映射代码，避免重复登记 |
| #175 | `src/styles/tailwind.css` | `:not([class*="t-"])` 会排除含 `t-` 的普通类（如 `text-sm`），使其失去该 focus-visible 样式 | 🟠 | 横切 | 已确认 | 纯展示 | 检查按钮/链接键盘 focus-visible | 已提交 `75a27ec`：改用 class token 边界匹配，仅排除以 `t-` 开头的 TDesign 类 |
| #121 | `src/views/NotFound.vue` | 无站内历史记录时仍调用 `router.back()` | 🟡 | 真缺陷 | 已确认 | 触及逻辑 | 直接打开 404 后点击返回 | 已提交 `bd1e02e`：优先检查 router history state；无站内上一页则回 dashboard |
| #51 | `src/views/dashboard/Index.vue` | ECharts 读取不存在的 `--text-secondary`，fallback 也不是 CSS 颜色值 | 🟡 | 真缺陷 | 已确认 | 触及逻辑 | 三主题下检查图例/轴标签颜色 | 已提交 `8644b38`：改读 `--color-text-secondary`，fallback 使用 `var(--t-text-3)`；待图表回归 |

> 注：用户原始 27 项为审计候选，并非全部自动认定为真缺陷。源码复核发现 #34、#141 至少在当前版本不支持原问题描述；#27 是有意的整页 fallback；#131、#138、#159 仍需补证据。保留编号便于追踪，不强行计入“已确认真缺陷”数量。

## B. 运行时待验证（8 项）

| ID | 路径/范围 | 待验证点 | 状态 | 触及层 | 验收方式 |
|---|---|---|---|---|---|
| #174 | Toast 组件/样式 | 淡出动画中途被移除是否闪断 | 待跑 | 触及逻辑 | 连续触发 toast 并慢放录屏 |
| #176 | `src/components/ui/Modal.vue` | 移动端打开带自动 focus 的弹窗时键盘/transform 是否抖动 | 待跑 | 触及逻辑 | iOS/Android 真机录屏 |
| #180 | 全局响应式布局 | 640–768px 是否出现侧栏、抽屉、TabBar 冲突 | 待跑 | 纯展示 | 640/700/768px 视口实测 |
| #152 | `src/views/admin/system/Monitor.vue` | CPU 渐变条最终是否可见 | 待跑 | 纯展示 | 构建后页面实测 |
| #51 | `src/views/dashboard/Index.vue` | 深色主题 ECharts 图例/轴标签是否回退黑色 | 待跑 | 触及逻辑 | 三主题切换检查 |
| #65 | `src/views/dashboard/Index.vue`（待定位） | 1920px 桌面 stat-strip 是否仍需横向滚动 | 待跑 | 纯展示 | 1920px 实测 |
| #61 | `src/views/accounts/List.vue` | 选择 5+ 账号时底部浮条是否遮挡最后几行 | 待跑 | 纯展示 | 移动端长列表滚动实测 |
| #84 | 目标选择器（待定位） | 选择 20 个目标时 chips 是否被 96px 高度截断 | 待跑 | 纯展示 | 20 个目标实测 |

## C. 死代码/技术债候选（独立清理批次）

| ID | 路径 | 候选项 | 当前结论 | 状态 | 验收方式 |
|---|---|---|---|---|---|
| #12 | `src/components/selectors/GroupSelector.vue` | 私聊（dm）分支是否不可达 | 当前源码存在 dm Tab、加载、过滤和选择逻辑，不能认定死代码 | 待 grep / 误报候选 | 全仓引用与 props.mode 使用检索 |
| #39 | `src/components/ui/LottieAnim.vue`、`src/assets/lottie/empty-tasks.json` | 组件零引用、资产 367,199 bytes | 已确认（58 个 Vue SFC 均无组件引用；GitHub 源码搜索无引用结果；组件自身 glob 导入该资产） | 待清理批次 | 删除前再核对非 Vue 动态引用；清理后构建验证 |
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
