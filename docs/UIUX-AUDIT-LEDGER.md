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
| #4 | `src/components/ui/CountUp.vue` | 清洗货币符号后动画只写数字，`¥` 不恢复 | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 单测 + dashboard 手动 | `animateTo` 写入 String(number)，需保留前后缀 |
| #9 | `src/components/MobileTabBar.vue` | `startsWith(t.path)` 将账号 ID 纳入匹配，切换账号后 Tab 可能失活 | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 多账号切换 E2E | 路径匹配需按路由段/名称设计 |
| #10 | `src/components/MobileTabBar.vue` | 当前账号缺失时使用 `none` 生成工作区路由，无有效 fallback | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 无账号点群发 Tab，验证跳转账号页 | 需确认路由守卫目标 |
| #20 | `src/components/ui/Breadcrumb.vue` | 模板固定渲染“首页”，管理端传入仪表盘后重复层级 | 🟡 | 细节/横切 | 已确认 | 触及逻辑 | 租户端、管理端面包屑逐页检查 | 根因在组件 API/模板层 |
| #26 | `src/components/ui/ConfirmActions.vue` | 移动端使用 `column-reverse`，按钮视觉顺序与 DOM/焦点顺序不一致 | 🟡 | 细节 | 已确认 | 纯展示 | 移动端键盘焦点与视觉顺序手测 | 需确定产品期望按钮顺序 |
| #27 | `src/components/ui/PullDownRefresh.vue` | 无滚动父级时 `canPull()` 返回 true | 🟡 | 待验证 | 已确认（行为） | 触及逻辑 | 页面级滚动容器/无滚动容器两种场景实测 | 源码注释明确这是整页场景的有意 fallback；是否误触需实测 |
| #34 | `src/components/ui/SupportFloat.vue` | 拖拽后 click 未拦截 | 🟡 | 误报/待复核 | 误报（当前代码有防护） | 触及逻辑 | 拖拽后松手、轻点分别手测 | `toggle()` 已检查 `isDragging`；需仅验证时序 |
| #36 | `src/components/ui/EmptyState.vue` | `title` 传入 TDesign `description`，真描述另行渲染 | 🟠 | 真缺陷 | 已确认 | 纯展示 | 组件用例/页面手测 | 组件 prop 语义需修正 |
| #38 | `src/components/ui/ProgressRing.vue` | 百分比文字 round，圆弧保留小数，显示不一致 | 🟡 | 细节 | 已确认 | 纯展示 | 99.5、0.5、100 等边界值单测 | 文本与弧统一精度策略 |
| #57 | `src/views/accounts/List.vue` | 移动端账号操作按钮最小高度 36px | 🟡 | 细节 | 已确认 | 纯展示 | 320/375/430 宽度触控检查 | 36px 源码已定位；按 44px 目标评估布局空间 |
| #58 | `src/views/accounts/List.vue` | `codeAge()` 使用 `Date.now()`，没有响应式时间依赖 | 🟠 | 真缺陷候选 | 已确认（源码） | 触及逻辑 | 验证验证码年龄是否随时间刷新 | 页面有 `nowTs`，但函数未读取它 |
| #68 | `src/views/accounts/List.vue` | `statusPoll` 与 `useAutoRefresh` 都触发账号刷新，可能重复轮询 | 🟠 | 横切/性能 | 已确认（双触发存在） | 触及逻辑 | 网络面板统计 30 秒请求次数、切后台恢复 | 需检查 store 缓存/去重语义后决定是否合并 |
| #99 | `src/views/workspace/Scrape.vue` | 导出 Excel 的独立 HTML 使用 CSS 变量 `var(--t-line)` | 🟡 | 真缺陷 | 已确认 | 纯展示 | 下载 XLS 并检查边框 | 独立 Blob 文档不继承应用主题变量 |
| #100 | `src/views/workspace/Scrape.vue` | CSV/XLS 导出创建 Object URL 后未 revoke | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 连续导出后检查 URL 生命周期 | 两处 createObjectURL 均未释放 |
| #109 | `src/views/raffle/Index.vue` | 清空记录使用原生 `confirm()` | 🟡 | 细节/横切 | 已确认 | 触及逻辑 | 桌面/移动端确认交互回归 | 改用项目统一确认组件/ask |
| #130 | `src/views/admin/dashboard/Index.vue` | 演示数据角标位于 `main` 外，定位上下文可能错误 | 🟡 | 真缺陷候选 | 已确认（结构） | 纯展示 | 管理端 dashboard 实机检查角标位置 | 顶层 span 在 main 前，需确认外层定位上下文 |
| #131 | `src/views/admin/dashboard/Index.vue` | 使用 `bg-warning` / `text-warning`，可能没有对应 Tailwind token | 🟡 | 待核 | 待 grep | 纯展示 | Tailwind 配置/构建 CSS 检查 | 需核对完整 token 定义与生成结果 |
| #132 | `src/views/admin/dashboard/Index.vue` | columns 的 `format` 函数未传给 TDesign columns | 🟠 | 真缺陷 | 已确认 | 触及逻辑 | 管理端表格日期列手测 | 映射只保留 colKey/title |
| #138 | `src/views/admin/tenants/List.vue` | 表格列格式化函数可能未被 TDesign 消费 | 🟠 | 真缺陷候选 | 待核 | 触及逻辑 | 检查实际列定义及渲染 | 与 #132 可能同根因，需避免重复计数 |
| #141 | `src/views/admin/codes/List.vue` | 使用 text-yellow/red/green 硬编码语义色 | 🟡 | 误报（当前文件） | 误报 | 纯展示 | 如有其他匹配文件再补证据 | 当前文件全文未检出所述 class |
| #142 | `src/views/admin/codes/List.vue` | 批量操作浮条固定距底 92px | 🟡 | 细节 | 已确认 | 纯展示 | 管理端移动端实机检查遮挡 | 该页无租户 TabBar；92px 可能造成不必要空隙 |
| #145 | `src/views/admin/support/Config.vue` | 二维码预览使用 `object-cover`，可能裁切二维码边缘 | 🟠 | 真缺陷候选 | 已确认（样式） | 纯展示 | 不同长宽比二维码实机扫码 | 图片预览当前 64px 盒子 + object-cover |
| #152 | `src/views/admin/system/Monitor.vue` | CPU 渐变条 `from-[var()]` 颜色是否生成/显示异常 | 🟡 | 待验证 | 待跑 | 纯展示 | 构建产物 + 管理端监控页实机检查 | 当前源码使用 Tailwind gradient stop，不能仅凭字符串断定缺少 color |
| #159 | `src/views/admin/*`（4 个页面，待逐一列出） | 与 #132/#138 相同的表格 format 映射问题 | 🟠 | 横切 | 待核 | 触及逻辑 | 全部管理表格页面逐项核验 | 先定位四页及共同映射代码，避免重复登记 |
| #175 | `src/styles/tailwind.css` | `:not([class*="t-"])` 选择器可能误排除包含 `t-` 的普通类 | 🟠 | 横切 | 已确认（选择器存在） | 纯展示 | 检查匹配元素及 focus-visible 回归 | 需确认实际命中范围后调整选择器 |
| #121 | `src/views/NotFound.vue` | 无历史记录时仍调用 `router.back()` | 🟡 | 真缺陷 | 已确认 | 触及逻辑 | 直接打开 404 后点击返回 | 应提供安全 fallback |
| #51 | `src/views/dashboard/Index.vue` | ECharts 读取 `--text-secondary`，当前 tokens 暴露的是 `--color-text-secondary` | 🟡 | 真缺陷候选 | 已确认（变量名差异） | 触及逻辑 | 三主题下检查图例/轴标签颜色 | 读取不到时 fallback 字符串可能不是有效 CSS 色值 |

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
| #39 | `src/components/ui/LottieAnim.vue`、`src/assets/lottie/*` | 组件零引用、资产约 367KB | 目录存在不等于零引用，尚未完成全仓引用检索 | 待 grep | 全仓 grep + 构建依赖图 |
| #122 | updates 页面 | 版本文案 v68.9 过期 | 内容治理，不作为 UI bug | 待核 | 页面内容核验 |
| #178 | 全局文案/i18n | 术语不统一 | 规范建设，不作为单点 UI bug | 待核 | 术语表/全局 grep |

## D. 尚未核验的其余清单

#1–#180 中未在上表单列的条目统一暂列 **待核**，不代表不存在问题或已通过。后续按组件层、租户端页面、管理端页面、横切样式四组继续补齐，所有条目都必须记录完整 `src/` 路径、源码状态、触及层和验收方式。

## E. 当前源码复核中的重要更正

1. `src/components/ui/SupportFloat.vue` 的 `toggle()` 已在 `isDragging` 为 true 时 return；#34“拖拽后 click 未拦截”不能直接认定成立，仍需检查 click 与 50ms 延迟复位的时序。
2. `src/components/selectors/GroupSelector.vue` 当前确实包含 dm Tab、加载、过滤、全选及选择流程；#12“私聊分支不可达”与当前源码不符，必须查调用参数后再定性。
3. `src/components/ui/PullDownRefresh.vue` 无滚动父时返回 true 的代码注释明确写为整页场景 fallback；需要验证误触，不宜直接当 bug。
4. `src/views/admin/codes/List.vue` 当前全文未检出 #141 所述 `text-yellow/red/green` 类。
5. `src/views/workspace/Scrape.vue` 两处 Object URL 均未看到 revoke；#100 可直接进入整改候选。
6. `src/views/accounts/List.vue` 的 30 秒 `statusPoll` 与 30 秒 `useAutoRefresh` 同时存在；先确认 store 去重/缓存行为，再决定如何合并，避免误删刷新保障。

## F. 下一步执行计划

1. 补齐全仓 grep：LottieAnim 引用、GroupSelector 调用与 dm mode、会话页死代码、Tailwind token、管理表格 format、四个管理页关联。
2. 完成 #1–#180 全量核验和分类统计。
3. 以 `src/views/accounts/List.vue` 为第一试点，集中处理确认的问题，并逐项附逻辑回归步骤。
4. 通过试点后批量修复组件层、横切层、租户端、管理端。
5. 每批执行 lint、typecheck、build、可用单测；涉及路由/状态/轮询/导出生命周期的改动增加手动或 E2E 验收。

**本台账是第一版审计记录，不表示已完成 180 项核验，也不表示已开始修改业务页面。**
