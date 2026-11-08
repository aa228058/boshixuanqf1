# 前端重建：后端对接资产盘点（第一版）

> 基准分支：main  
> 基准提交：aeb2dcf4cc63e6324ae1a85e7b80a4258f8e8b13  
> 状态：初步盘点，尚未完成逐接口请求/响应契约核验  
> 原则：本阶段只提取和核对业务能力，不删除旧前端、不修改后端 API、不触碰生产数据。

## 1. 系统边界

- 后端：Go + Echo，API 前缀 `/api/v1`；租户 API 使用 JWT、租户隔离、套餐上下文和限流中间件。
- 前端：`web/` 下 Vue 3 + TypeScript + Vite + Vue Router + Pinia。
- 管理后台：独立 `/admin` API 与独立路由区域。
- 实时通信：租户 SSE `GET /api/v1/events`，先通过 `POST /api/v1/events/ticket` 获取短期 ticket；任务另有 `GET /tasks/:id/progress/stream`。
- 文件能力：账号 session 导入、群组库导入、媒体上传、头像上传、导出下载。
- 静态前端由 Go 服务托管 `web/dist`。因此重建不能擅自改变构建产物路径、API 前缀、部署容器边界。

## 2. 已确认的后端能力域

| 能力域 | 已确认 API 路由 | 前端页面/入口候选 | 备注 |
|---|---|---|---|
| 认证与租户 | `/auth/register`、`/auth/login`、`/auth/renew`、`/auth/refresh`、`/auth/logout`、`/auth/me`、`/tenant/profile`、`/tenant/activate` | 登录、注册、会话恢复、设置 | refresh cookie / token 生命周期必须保留 |
| 实时事件 | `/events`、`/events/ticket` | 全局通知、任务状态、实时活动 | SSE 需短期 ticket；不要在重写时改成普通轮询而丢实时事件 |
| 首页仪表盘 | `/dashboard/stats`、`/dashboard/activities`、`/dashboard/send-calendar`、`/dashboard/risk-events` | 首页、发送趋势、日历、风险提示 | 与参考图首页/日历直接对应 |
| 账号管理 | `/accounts`、`/accounts/stats`、账号详情、连接/重连/断开、验证码/2FA、session 导入、资料/头像/代理、账号池批量操作、探测、配额、Telegram dialogs/contacts、群转移、离群 | 账号列表、账号详情、导入流程、账号工作区 | 账号授权是多阶段流程，不能只按普通 CRUD 重写 |
| 群组库 | `/group-library`、`/group-library/join`、`/group-library/search`、`/group-library/import`、`/group-library/import-jobs`、导入任务详情/确认 | 群组库、导入预览、导入进度、群组加入 | 平台共享群库；导入先筛选/确认，再异步解析 |
| 账号群组 | `/groups`、同步/全量同步、详情、成员、全局成员、成员采集 | 工作区群组、成员采集、群组详情 | 与平台群组库不是同一资源域 |
| 发送/任务 | `/tasks`、目标预检、任务详情/编辑、暂停/恢复/重试/取消/删除、批量取消/删除、日志、账号统计、进度、进度 SSE | 任务列表、任务详情、群发创建流程 | 状态机、目标预检、失败重试语义需要单独固化 |
| 链接采集 | `/scrape-links`、导出、开关、状态；`/scrape/:id/export` | 链接采集、采集结果导出 | 与群组链接导入解析区分 |
| 消息监控 | `/monitor/rules` CRUD/toggle、消息、账号 start/stop | 工作区消息监控 | 套餐功能；消息目前可能由 SSE 实时推送 |
| AI 入群验证 | `/ai-config`、测试、`/verifier/toggle`、`/verification-logs` | 验证配置、验证记录 | 需要核对套餐限制和密钥脱敏 |
| 抽奖助手 | `/raffle/status`、toggle/reply/ping/results、清空结果、账号 raffle 设置 | 抽奖助手、账号功能设置 | 功能入口是否保留需结合产品决策，不在接口盘点阶段删减 |
| 租户设置 | 通知配置、邮件测试、套餐、公告/已读、消息模板、账号分组、数据导出、客服信息 | 设置、通知中心、模板、账号分组 | 导出是文件响应，不应按 JSON 请求处理 |
| 媒体 | `POST /media` | 群发素材/媒体上传 | 单独 52M 请求体限制 |
| 管理员后台 | `/admin/auth/*`、stats/system、tenants、plans、codes、support、probe、announcements、audit-logs | 管理员登录、租户/套餐/卡密/公告/审计/系统监控 | 与租户 API 的认证和权限完全分离 |
| 公共信息/健康 | `GET /support`、`GET /health` | 登录页客服信息、部署探活 | health 不属于用户业务页面 |

## 3. 当前前端 API 封装目录

已确认存在以下封装文件：

- `web/src/api/auth.ts`
- `web/src/api/account.ts`
- `web/src/api/admin.ts`
- `web/src/api/contact.ts`
- `web/src/api/dashboard.ts`
- `web/src/api/group.ts`
- `web/src/api/groups.ts`
- `web/src/api/media.ts`
- `web/src/api/monitor.ts`
- `web/src/api/scrape.ts`
- `web/src/api/settings.ts`
- `web/src/api/task.ts`
- `web/src/api/verifier.ts`

其中已经发现的待核验项：

1. `api/group.ts` 与 `api/groups.ts` 职责命名容易混淆：前者是账号维度的 `/groups`，后者是平台共享 `/group-library`。重建时应按领域命名，避免继续产生含糊的 group/groups。
2. `api/scrape.ts` 当前导出空对象，文件内保留未使用类型导入，属于疑似残留封装。
3. `api/account.ts` 中仍暴露 `/accounts/:id/checkup` 与 `/accounts/:id/appeal` 调用，但后端路由注释明确表示能力体检/SpamBot 申诉已移除；这是前后端契约漂移，必须追查调用点并在重建契约中标记为不可用，不能照搬。
4. `api/account.ts` 同时存在 `accountApi.updateProfile` 和独立导出的 `updateProfile`，且请求字段范围不同；需要追查页面调用后统一。
5. `api/task.ts` 中 `ScrapedLink` 附近存在自相矛盾的“重复定义已删”注释，提示历史审计注释可能过期；需要以代码和调用为准。
6. `api/monitor.ts` 的消息查询封装含 `rule_id` 参数，但后端当前 handler 需要核对是否支持该过滤条件。
7. `api/admin.ts` 的 CSV 导出绕过通用 JSON 客户端，重建必须保留文件下载语义。
8. `api/groups.ts` 的导入流程有 preview/review、confirm、后台 job 查询几个阶段，不能简化成一次上传即完成。

## 4. 当前前端页面与导航资产

租户侧路由（以 router 为准）：

- 登录、注册、仪表盘、更多、账号管理、设置、群组发现、验证记录、更新日志、帮助、抽奖助手。
- 账号工作区：群发、加群、成员采集、链接采集、消息监控、任务列表、任务详情。
- 旧地址 `/sessions`、`/tasks`、`/scrape` 等部分重定向到账号管理，说明当前导航结构经历过合并/迁移。

管理员侧：

- 管理员登录、仪表盘、套餐、租户列表/详情、卡密、客服配置、探测配置、公告、系统监控、审计日志。

## 5. 技术依赖与样式治理风险（初步）

`web/package.json` 当前主要运行依赖包括 Vue 3、Pinia、Vue Router、TDesign Vue Next、ECharts/vue-echarts、vue-final-modal、vue-sonner、Lottie、canvas-confetti、lucide-vue-next；开发链包括 Vite、TypeScript、Vitest、vue-tsc、Tailwind CSS 4。

`web/src/main.ts` 当前全局导入多套样式层：tokens、Tailwind、animations、theme-stripe、theme-dark、ui-refined、ui-polish、tdesign-skin、tdesign-theme、ui-kit、tdesign-overrides、modal-system。当前还初始化 stripe/dark 双主题。必须进一步追踪每层 CSS 的选择器覆盖、页面引用和实际视觉效果，不能仅按文件名直接删。

## 6. 重建时必须冻结的后端契约

- API base path：`/api/v1`；管理员 API 使用 `/admin`。
- 租户身份：JWT + refresh session；租户隔离由后端中间件执行。
- 管理员身份：独立 AdminAuth 与角色限制。
- SSE：先取短期 ticket，再连接事件流；任务进度流单独核验事件格式。
- 上传：群库导入 21M、媒体 52M；session/头像使用 multipart 或对应请求格式。
- 导出：CSV/成员导出为原始文件响应，不得经过默认 JSON 解码。
- 套餐功能：群组加入、成员采集、监控、AI 验证、抽奖、模板等存在后端功能门禁；前端只负责展示能力和解释拒绝原因，不能自行当作权限来源。
- 长流程：账号授权、群库导入、任务执行均需保留进度/错误/可恢复状态。

## 7. 尚未完成的核验清单

- [ ] 完整提取所有 Echo 路由（逐条方法、路径、handler、middleware、套餐门禁）。
- [ ] 为每个 API 获取请求结构、响应结构、分页规则、错误码和文件语义。
- [ ] 将前端 API 封装逐条映射到后端路由，标记缺失、孤儿、重复、参数不一致。
- [ ] 搜索全部页面调用点，确认 API 使用频率和真实业务入口。
- [ ] 盘点 SSE 事件名称、payload、重连策略和任务进度事件。
- [ ] 盘点所有上传/下载接口的字段名、MIME、大小限制和响应头。
- [ ] 核对租户与管理员的认证恢复、权限、套餐降级/过期行为。
- [ ] 输出最终接口矩阵和重建页面数据依赖表。
- [ ] 在接口矩阵完成前，不删除旧页面、不清理旧 CSS、不开始大规模重写。

## 8. 本阶段安全边界

本分支只新增盘点文档。不得修改生产分支、数据库、部署文件、Docker 配置、环境变量、数据卷或生产服务。前端重建应在独立分支推进，旧版本通过 Git 历史保留。
