# Frontend component governance

## Goal

Reduce the number of competing implementations for the same interaction without breaking page behavior. This is a migration plan and review standard, not permission to delete files based on naming or age.

## Current baseline (main, reviewed 2026-11-10)

- The application is Vue 3 + TypeScript.
- TDesign Vue Next is registered selectively in `web/src/main.ts`.
- `vue-final-modal` is installed and used by the project `Modal.vue` wrapper.
- `vue-sonner` is installed for toast notifications.
- Project-owned UI SFCs currently include `Modal.vue`, `ConfirmDialog.vue`, `ConfirmHost.vue`, `EmptyState.vue`, `Loading.vue`, `Skeleton.vue`, `Breadcrumb.vue`, and other specialized components.
- Global CSS is layered across tokens, Tailwind, theme files, UI refinement/polish, TDesign skin/theme/overrides, UI kit, and modal-system rules.

These facts show multiple layers, but do not by themselves prove that any particular file is dead or redundant.

## Canonical ownership (target)

| Concern | Canonical owner | Migration rule |
|---|---|---|
| Design tokens | `styles/tokens.css` | New component styling must use tokens; do not add parallel palettes. |
| Shared button/input primitives | TDesign components plus semantic classes in `ui-kit.css` | Prefer existing primitives and variants; add a Vue wrapper only for repeated project-specific behavior. |
| Modal/dialog shell | `components/ui/Modal.vue` | All project-owned modal surfaces should use the wrapper unless a documented exception requires a native TDesign dialog. |
| Confirm interaction | `utils/confirm.ts` + `components/ui/ConfirmHost.vue` | Prefer the centralized promise-based confirmation flow; keep `ConfirmDialog.vue` only for verified declarative use cases during migration. |
| Toast | `vue-sonner` | Do not introduce another toast/event-bus implementation. |
| Empty/loading/skeleton states | Existing project UI components | Consolidate APIs only after call-site inventory; do not replace specialized progress components blindly. |
| TDesign compatibility | `tdesign-skin.css`, `tdesign-theme.css`, `tdesign-overrides.css` | Each file must have a documented, non-overlapping responsibility; consolidate after selector ownership audit. |
| Modal positioning | `modal-system.css` | Keep positioning out of generic polish and theme layers. |
| Motion/interaction polish | `ui-polish.css` | Keep visual motion and feedback here; avoid defining component base appearance. |

## Required audit before removal

A component or stylesheet can be removed only after all checks pass:

1. Search static imports, lazy imports, Vue template tags, app-level registration, route-level imports, and string-based/dynamic component resolution.
2. Check whether a component is used by tests, stories, fixtures, or runtime configuration.
3. Check CSS selectors against templates and dynamically generated class names. A selector with no static match is not automatically dead.
4. Check whether a stylesheet is imported directly or indirectly and inspect cascade/order dependencies.
5. Migrate every verified call site to the canonical implementation.
6. Run typecheck, unit tests, design-token audit, production build, and browser regression checks.
7. Delete only in a separate, reviewable change with before/after evidence.

The inventory script `npm run audit:components` reports direct SFC imports and global CSS import order. Its "no direct import" output is a manual-review lead, never a deletion instruction.

## Migration batches

### Batch A — inventory and contracts
- Generate component import inventory.
- Document current global CSS import order.
- Identify repeated component basenames and overlapping CSS selector ownership.
- Establish canonical props/events/slots and accessibility expectations.

### Batch B — interaction primitives
- Normalize button variants and sizes.
- Normalize modal/dialog/drawer behavior, desktop centering, mobile viewport sizing, focus management, Escape/backdrop behavior, and scroll locking.
- Consolidate confirmation flows without changing business semantics.


### Batch B1 — confirmation content convergence

- Added `components/ui/ConfirmPrompt.vue` for shared confirmation message/icon/detail content and `components/ui/ConfirmActions.vue` for shared cancel/confirm buttons, variants, and mobile touch-target layout.
- Updated both `ConfirmDialog.vue` (declarative + exposed `open()` API) and `ConfirmHost.vue` (promise-based `ask()` API) to reuse that shared content.
- Kept the two invocation contracts separate intentionally: changing them into one API would risk breaking existing call sites and promise settlement semantics.
- Moved the shared button presentation into `ConfirmActions.vue`; the two hosts still own their distinct event handlers (local v-model close vs promise settlement).
- No legacy component was deleted in this batch.


### Batch B2 — button entrypoint correction

- Re-read `skills/frontend-dev/SKILL.md`, `skills/frontend-dev/references/architecture-governance.md`, and `skills/frontend-dev/references/button-audit.md`.
- The repository's established ordinary-action button entrypoint is TDesign `<t-button>` with the `.ui-t-button` semantic classes in `styles/ui-kit.css`; the button audit documents zero remaining `.ui-btn` usages on its audited baseline.
- A temporary `UiButton.vue` adapter and page migrations were introduced in this branch, but this created an unnecessary component wrapper contrary to the repo's established usage. They have been reverted; the redundant wrapper file has been deleted.
- Confirmation action buttons now use the established TDesign entrypoint directly, with explicit `type="button"` and existing semantic classes.
- No claim is made that all current pages were migrated in this PR; page button markup is back on the established main baseline.

### Batch C — forms and data display
- Keep TDesign Input, Textarea, InputNumber, Select, Checkbox, Switch, DatePicker and Table as the canonical primitives; native HTML controls remain valid where native semantics or specialized behavior is required.
- Do not create generic wrappers around each TDesign component merely to make tags look uniform. Add a project wrapper only when it centralizes repeated validation, accessibility, layout or domain behavior used by multiple call sites.
- Extend `npm run audit:components` to report TDesign and native form-control usage separately, so mixed implementations can be reviewed by page and semantic role before migration.
- Extend the same inventory to count TDesign pagination and native table structure (`table`, `thead`, `tbody`, `tr`) separately. These are source-level indicators only; review each call site for semantics, responsive behavior, sorting, selection, and accessibility before migrating.
- Normalize table, pagination, badges, tooltips and empty/loading states only after call-site and state-contract inventory.

### Batch D — stylesheet consolidation
- Reduce overlapping TDesign overrides.
- Retire obsolete selectors only after call-site and cascade verification.
- Preserve theme parity and responsive behavior.

### Batch D1 — dark-theme button cascade correction

- `theme-dark.css` applied a generic `.t-button--variant-base` fallback after the TDesign skin and without excluding the project's semantic `.ui-t-button` buttons.
- That selector could override the intended semantic primary/danger appearance in dark theme because it used a more specific selector and `!important`.
- Scoped the dark-theme fallback to TDesign buttons that are not marked `.ui-t-button`, matching the existing exclusion in `tdesign-skin.css`.
- Corrected the `tdesign-overrides.css` header comment: it loads after `ui-kit.css`, while `modal-system.css` is imported after it and owns project Modal positioning.

### Batch D2 — CSS ownership and control inventory

The actual `main.ts` import order is:

1. Third-party TDesign and Vue Final Modal styles
2. `tokens.css`, `tailwind.css`, `animations.css`
3. `theme-stripe.css`, `theme-dark.css`
4. `ui-refined.css`, `ui-polish.css`
5. `tdesign-skin.css`, `tdesign-theme.css`
6. `ui-kit.css`, `tdesign-overrides.css`
7. `modal-system.css`

The layers are not interchangeable: `tdesign-theme.css` maps TDesign semantic variables; `tdesign-skin.css` provides generic third-party component skin; `ui-kit.css` owns project semantic classes such as `.ui-t-button` and `.ui-input`; `tdesign-overrides.css` handles third-party responsive/DOM-specific adjustments; `modal-system.css` owns project Modal geometry. Theme files should override tokens or narrowly scoped theme exceptions, not redefine generic component ownership.

- Extended `npm run audit:components` to count TDesign and native Input, Textarea, InputNumber, Select, Checkbox, Switch, DatePicker and Table usages separately.
- This is a source-level inventory; it does not classify whether each native control is a valid semantic exception. Do not replace native controls without reviewing their form behavior and call sites.
- Removed the obsolete `UiButton` audit category after deleting that temporary wrapper; the inventory now reports only the established TDesign/native entrypoints and project semantic classes.

### Batch D3 — session-page dialogs and state inventory

- Migrated the session management page's two TDesign `<t-dialog>` forms (session import and profile edit) to the established `Modal.vue` primitive.
- Preserved the 560px/480px desktop widths, import/save loading states, explicit cancel actions, delayed successful-import close, and existing form behavior.
- Removed the page's viewport-width watcher; the shared Modal now owns desktop centering and mobile bottom-sheet geometry.
- Added page-scoped footer actions: right-aligned on desktop, full-width stacked controls on mobile, with 44px minimum touch height.
- Extended the inventory to report TDesign Dialog/Empty/Loading usage separately from project Modal/EmptyState/Loading/Skeleton usage.
- `EmptyState` remains the established wrapper around TDesign Empty and is used across multiple pages. `Skeleton` is used for content-shaped loading placeholders. The standalone `Loading.vue` has no direct import found in the audited page/component set; it is not deleted until dynamic/runtime references and any external contract are ruled out.
- The app-level TDesign Dialog registration is intentionally retained until the complete source inventory confirms no dynamic or non-template use.

### Batch D4 — session page confirmation and initial loading

- Replaced the session page's native `window.confirm()` bulk-delete prompt with the shared promise-based `ask()` confirmation host, preserving the destructive intent while using the project's accessible, theme-aware modal system.
- Fixed the initial empty-list flash: while the first account request is pending and there are no cached accounts, the page now shows the shared Loading component; only after loading completes does it show EmptyState. Refreshes with existing rows keep the table visible.
- Added `role="status"` and `aria-live="polite"` to the shared Loading root and marked its decorative spinner as hidden from assistive technology.

### Batch D5 — confirmation usage inventory

- Extended the component inventory to report native `window.confirm()` and `window.alert()` separately from shared `ask()` calls.
- This gives the next cleanup pass a measurable way to find browser-native prompts that bypass the project's themed confirmation host; source-level counts still require call-site review before migration.

### Batch D6 — Modal initial-open stack registration

- Fixed an edge case in `Modal.vue`: its modal-stack watcher previously ran only on subsequent `modelValue` changes. If a parent mounted a Modal already open, it was absent from the active stack and received no managed order.
- The watcher now runs immediately, registering an initially open modal and preserving the same cleanup/compaction behavior on close and unmount.
- Extended the inventory to distinguish TDesign `<Dialog>`, dynamic `resolveComponent("Dialog")`, render-function `h(Dialog)`, and global `app.use(Dialog)` registration from lowercase `<t-dialog>`. This is evidence gathering only; global registration remains until every usage path is reviewed.

### Batch D8 — stacked modal keyboard viewport lifecycle

- Found a shared-state race in `Modal.vue`: each instance independently attached `visualViewport` listeners but all wrote the document-level `--modal-kb` variable. Closing or unmounting one modal could reset the offset while another modal remained open.
- Replaced per-instance listeners with a module-level registry keyed by modal instance. The first open modal attaches one shared listener; additional open modals register without duplicating listeners; closing one only unregisters that instance; the last close/unmount detaches listeners and resets the CSS variable.
- The modal stack order registry remains separate from the keyboard registry because they represent different lifecycles.
- No dedicated Modal unit/spec file exists in the current frontend tree; CI typecheck/unit tests and Browser UI audit are the next verification gates. This does not replace manual nested-modal testing on desktop/mobile.

### Batch D7 — first-render loading state correction

- Follow-up verification found that the sessions page initialized `loading` to `false`, while the first `refresh()` runs from the mount lifecycle. The page could therefore render EmptyState for one frame before the request started.
- Changed the initial value to `true`; `refresh()` still owns the request lifecycle and resets it in `finally`. This closes the first-render gap rather than only fixing the post-request state branch.

### Batch C1 — admin table pagination behavior

- Audited the admin audit-log, activation-code, and tenant tables plus the workspace task table.
- Found a real defect in `admin/codes/List.vue`: the desktop TDesign table pagination callback only changed the local page number and never requested the corresponding server page. The mobile pagination buttons did call `fetchCodes()`, so desktop and mobile behavior diverged.
- Added a single `onPageChange` handler that updates the page and fetches the new server page; it ignores no-op page changes. The existing status-filter watcher still resets to page 1 and fetches, while the mobile previous/next controls remain unchanged.
- Audit-log pagination already fetches on desktop and mobile. Tenant pagination already uses a shared page-change handler and resets pagination when search/status filters change. Workspace task pagination is driven by the page watcher and URL state.
- This batch intentionally does not replace TDesign tables or native semantic summary tables with wrappers. Source-level inventory is not enough to justify changing table semantics, responsive layouts, or data-loading contracts.

### Batch C2 — join-task account/group selector race

- Audited account, broadcast, and join-task form controls and their account/group watchers.
- Found an asynchronous race in `workspace/Join.vue`: changing the execution-account selector starts an asynchronous group load, but an earlier account's slower response could arrive later and overwrite the group options for the newly selected account.
- Added a request-generation guard and an account-ID check before applying loaded groups. Changing the selector invalidates any previous in-flight result.
- Removed the top-level current-account watcher's redundant group reload/reset. The modal's explicit `phoneAccId` selector is the source of truth for invite-group options; changing the global account should not clear the user's explicit modal selection or reload a different account's groups.
- The current-account watcher still refreshes the task list. The dedicated `phoneAccId` watcher owns group-option clearing and loading.
- Native file inputs, checkboxes, datetime-local inputs, and selects in these workflows remain intentional where they provide browser-native file selection, checkbox semantics, or native mobile picker behavior; no blanket conversion to TDesign was made.

### Batch C3 — CSS duplicate-selector and cascade review

- Reviewed `theme-dark.css`, `tdesign-theme.css`, `tdesign-skin.css`, `tdesign-overrides.css`, `ui-kit.css`, and `modal-system.css`.
- Repeated selector text is not automatically duplicate styling: `.ui-segmented__item` appears in its base rule and a narrow-screen media query; compact TDesign/native button rules repeat in mobile media queries to enlarge touch targets; `.modal-content` and `.modal-card` repeat across desktop/mobile geometry and keyboard-offset scopes.
- The repeated dark-theme glass selector is intentionally scoped to separate mobile blur removal and desktop solid-surface overrides. Consolidating these blocks without preserving media-query scope would change behavior.
- No safe dead-rule deletion was established in this pass. Keep the current cascade ownership: theme files own theme variables/fallbacks, `ui-kit.css` owns semantic project primitives, TDesign skin/overrides own vendor adjustments, and `modal-system.css` owns modal geometry.
- The inventory's duplicate-selector report should be treated as a review queue, not an auto-fix list. Future CSS consolidation must compare selector, at-rule context, specificity, source order, and computed property overlap.

### Batch E1 — account selection and broadcast draft isolation

- Account-list “select all” previously replaced the entire selected-ID set with the current filtered result, or cleared every selection when the current result was fully selected. This made multi-filter/multi-page batch selection surprising: selecting another search result could silently discard IDs selected under a previous search.
- “Select all” and “clear all” now operate only on IDs matching the current filter and preserve selections outside that filter. Existing cross-page selection remains supported.
- Broadcast message drafts were stored under one global `broadcast_msg_draft` key. Switching the active account could therefore restore another account's unsent message, and separate tenants sharing a browser could share the same draft.
- Draft keys now include tenant and active account identifiers. On account switch, the current draft is persisted to the old scope before restoring the new scope; pending debounce work is cancelled safely, and unmount flushes the current draft.
- This batch changes only client-side draft storage and selection presentation; task payloads, API contracts, and send semantics are unchanged.

### Batch E2 — account selection cleanup and selector request isolation

- Account-list multi-selection could retain IDs after an account was deleted or removed from the refreshed account list. Those stale IDs could still be submitted by bulk actions. Added pruning against the complete account-store ID set while deliberately preserving selections outside the current filter and page.
- Broadcast's merged group list used `Math.random()` as a fallback key for records without group ID/chat ID/username. Every computed reevaluation could therefore remount those rows, disrupting focus and scroll position. Replaced it with a deterministic fallback based on account, title/name, and source index.
- `TargetSelector.vue` previously allowed a slow contact/group-cache response for the old `accountId` to overwrite the selector after the user switched account while the modal remained open. Added a request-generation guard to selector opening and contact loading; closing the selector invalidates outstanding work, and changing account reloads its context without clearing already selected targets.
- These changes are frontend state/lifecycle safeguards. They do not alter task payloads or server contracts.

### Batch E3 — GroupSelector account-scoped cache lifecycle

- `GroupSelector.vue` previously copied incoming groups only when the new array was non-empty. When the parent cleared its group list or switched accounts, the selector could continue showing the previous account's cached groups.
- The selector now treats `props.groups` as the authoritative snapshot, including an empty array. On account changes it invalidates in-flight selector/DM loads, clears account-scoped local data and DM cache state, and reloads the new account when the modal is open.
- Cache loading now checks both request generation and current account/modal state before writing results. Closing and reopening also invalidates stale results.
- Private-message contacts are cached per currently selected account rather than once for the component lifetime; late responses from a previous account are ignored.
- Manual refresh captures its initiating account ID and request generation. A late poll from a previous account/modal session can no longer replace the active list or overwrite its loading/sync status; the original sync lock is still released when its operation ends.
- This is a client-side account-context correction. Selection payload format and backend APIs are unchanged.

### Batch E4 — Broadcast creation contract and async state isolation

- The broadcast form previously sent `is_loop` on a normal `type: broadcast` request. The generic handler dispatches that request to `CreateBroadcast`, whose DTO does not include loop fields; the flag was silently discarded. Loop mode now uses the dedicated `type: scheduled_broadcast` contract and supplies the nested message payload, account/target IDs, and interval fields expected by `CreateScheduledBroadcast`.
- The scheduled-broadcast handler does not accept scheduled start timestamps or media attachments, so the form now blocks those unsupported combinations instead of silently dropping them. Variation/obfuscation controls are hidden with an explicit explanation in loop mode.
- Regular broadcast's executor already reads `auto_vary` and `obfuscate` from task payload, but the HTTP request DTO previously dropped both fields. The handler DTO and `broadcastExtras` now preserve those options.
- Submission now has a business-layer duplicate-submit guard, snapshots/deduplicates account and target IDs, validates send and loop intervals, rejects past scheduled timestamps, and prevents submitting while media upload is pending.
- Media upload responses are generation-guarded: clearing or replacing media while an upload is in flight prevents an older response from restoring the cleared file.
- Group coverage loading, account group-cache loading, and task-list fetching now guard against stale responses after modal close or account switching.
- Successful form reset now clears one-time targets, imported text, message, media, scheduling and loop state while intentionally retaining the selected execution-account pool. Failed creation leaves entered values intact for correction and retry.

### Batch E5 — Selector large-list rendering and refresh isolation

- TargetSelector's manual group refresh now captures account ID + request generation, ignores stale results after account switch/modal close, waits for another in-flight sync without releasing its lock, and releases only a lock acquired by its own operation.
- Removed a permanently disabled `if (false)` cache-refresh branch from TargetSelector; manual refresh now has one explicit sync path.
- TargetSelector group/contact lists and GroupSelector group list render the first 200 matching rows, with an explicit incremental “load more” action. Search resets the visible window. Bulk select/deselect still applies to the entire filtered result set, not just rendered rows, so progressive rendering does not change selection semantics. GroupSelector's collected private-contact list uses the same progressive rendering so large member collections do not mount thousands of checkbox rows at once.
- The selector changes reduce initial DOM work for large accounts while retaining access to all matches through search and incremental loading.

### Batch E6 — Task list request isolation and account batch profile snapshot

- Workspace task list fetches now use a monotonically increasing request generation. Responses from older page/filter/range requests are ignored, and only the latest request may clear the loading state or replace the visible rows.
- Task list errors now populate the existing `loadError` state instead of silently retaining stale rows without an error state.
- Account batch profile updates snapshot selected account IDs and profile fields at submit time, reject empty submissions, and only remove the submitted IDs from the live selection after success. This prevents selection changes during an in-flight operation from changing the request's target set or clearing newly selected accounts.
- Existing clone-profile operation already has a duplicate-submit guard; no redundant wrapper was added.

### Batch E — page migration and regression
- Migrate pages in coherent groups.
- Run automated checks and real-browser desktop/mobile visual review.
- Record remaining intentional exceptions.

## Acceptance criteria

- Each shared interaction has one documented canonical entry point.
- No page adds a competing global primitive for an existing shared interaction.
- Public props/events/slots and keyboard behavior are documented for shared wrappers.
- No component or CSS file is removed solely because an automated search reports no direct reference.
- `npm run typecheck`, `npm test`, `npm run check:design-tokens`, and `npm run build` pass.
- Desktop and mobile browser checks cover dialogs, buttons, forms, tables, empty/loading states, themes, and overflow.


### Batch E7 — task detail action and request lifecycle

- Task detail pause, resume, cancel, and failed-target retry controls now share a single in-flight action guard. While one action is pending, all competing action buttons are disabled and the active action displays its loading state; this prevents rapid repeated clicks from issuing duplicate mutations.
- Consolidated pause/resume/cancel request handling so success refresh, error feedback, and lock release follow one path.
- Added a monotonically increasing task-detail fetch request ID. Only the newest active request may update task data, page errors, or loading state; responses arriving after unmount are ignored.
- Guarded the task-detail SSE ticket exchange, EventSource callbacks, watchdog, and reconnect scheduler with the page lifecycle. If the page unmounts while ticket acquisition is pending, the late response cannot create a detached EventSource or schedule another reconnect.
- This is a client-side duplicate-submit guard, not a replacement for backend idempotency or state-transition validation. Server-side protection and manual browser verification remain required.


### Batch E8 — task log pagination and live snapshot reconciliation

- Task-detail log fetches now track request generations per page; an older response for the same page cannot overwrite a newer refresh, and responses after page unmount are ignored.
- A page-1 live refresh merges persisted rows into the current timeline without resetting the user's deepest loaded history page.
- The pagination cursor's hasMore state is updated only by the currently deepest loaded page, so a late page-1 refresh cannot undo the end-of-history result from page 2+.
- Response normalization now handles nested { data: { list } } in addition to array, { list }, and { data } forms.
- Clearing the displayed log list invalidates in-flight page requests; this remains a local display action and does not delete persisted audit/send logs.
- Manual verification is still required for overlapping page loads, SSE arrival during refresh, and reaching the oldest page.


### Batch E9 — task detail completion and member-request lifecycle

- The 100%-progress convergence fallback now owns a single tracked timeout. New progress/status changes replace or cancel the previous timeout, and leaving the page clears it; the callback also checks that the page is still active before fetching task state.
- Member-list requests now use a generation counter and page-lifecycle guard. A late response from an older refresh or an unmounted detail page cannot replace the visible member list, error, or loading state.
- Account and group name-map requests check the page lifecycle immediately after their awaited API calls, preventing post-unmount state writes during the sequential detail-page bootstrap.
- This batch is frontend lifecycle hardening only; it does not change task execution, log persistence, or backend APIs.


### Batch E10 — task log fallback identity

- Reviewed the task-log deduplication key used when the API does not provide `log_id`.
- Replaced delimiter-joined fields with JSON-encoded field tuples and included `target_name`, preventing log message text containing the delimiter from creating accidental key collisions.
- Logs without a stable server ID can still be indistinguishable if every fallback field is identical. The durable fix for that case is a unique backend log identifier; this frontend fallback does not pretend to manufacture one.


### Batch E11 — task detail end-to-end reliability

- Removed the detail page's second subscription to the tenant-wide `useTaskEvents` stream. The page already owns a task-scoped SSE endpoint that forwards this task's logs, status, wait countdown, and progress; processing both streams duplicated work and allowed the global callback to append logs without the task stream's lifecycle/cap behavior. The task-scoped stream is now the single source for detail-page realtime events.
- Bound every EventSource callback and the watchdog to the exact EventSource instance. A late `error` callback from an old connection can no longer close a newer connection after reconnect; watchdog expiry also only closes the connection it was created for.
- Added task-scoped `task_wait` handling to the dedicated stream and clear the countdown when it reaches zero or the task leaves running state.
- Progress display is clamped to 0–100%, and equal-timestamp logs receive a deterministic `log_id` tie-breaker.
- Reordered page bootstrap so task/log loading is followed immediately by the task-scoped stream, visibility recovery, and disconnect polling. Account/group name dictionaries load independently and cannot delay live updates. Initial task-fetch failure no longer prevents the stream and retry path from starting.
- Action and failed-target retry completions now suppress stale toasts/state updates after navigation; disconnected polling is silent to avoid repeated error-toast storms.
- Replaced task-detail history paging's mutable OFFSET continuation with a keyset cursor (`before_log_id`, ordered by `executed_at, log_id`). New live inserts no longer shift the next historical page and cause skipped/duplicated rows. Existing page-number requests remain backward compatible for callers that do not supply a cursor.
- Added migration `086_task_logs_cursor_index.sql` with a tenant/task/time/log-ID composite index matching the keyset predicate and sort order.
- Clear-display now resets the local cursor chain and invalidates outstanding page requests; it still never deletes server-side logs.
- Live log arrivals follow the timeline only when the reader is already near the bottom. Loading older cursor pages preserves the current reading position by compensating for the prepended row height; background refreshes no longer yank a user away from the history they are reading.
- Live-event deduplication now uses a maintained key set instead of scanning the entire loaded timeline for every event, avoiding quadratic lookup work as a long-running task accumulates logs.
- Ticket acquisition has a 15-second abort deadline; a stalled ticket request can no longer hold the connection lock indefinitely and prevent retries.
- Remaining verification: CI must compile/test the backend cursor query and frontend changes; manually verify reconnect, wait countdown, and historical paging while a task is actively generating logs in desktop and mobile browsers.


## 全仓前端剩余问题复核（2026-11-10）

### 复核基线与覆盖

- 复核分支：`refactor/frontend-component-governance`。
- 复核基线：`d08037df1c37230c1ea65d0ae5195deff5ffb03f`；后续测试夹具修复提交：`8287fb51d19a36c4b54480b36532bdc45c4f19c1`。
- 对仓库树进行全量枚举：`web/src` 下 58 个 Vue SFC，其中 34 个页面/布局视图、23 个组件；全局样式目录 12 个 CSS 文件；前端现有 6 个测试文件。
- 页面覆盖包括工作区（群发、进群、监控、采集、任务）、账号、群组发现、会话、任务详情、仪表盘、设置、帮助、更新、验证器、登录注册、更多/404，以及管理端布局、登录、公告、审计、卡密、仪表盘、套餐、探测、客服、系统监控和租户列表/详情。
- 公共组件覆盖 AppShell、移动导航、实时任务/日历/图表、选择器、Modal/Confirm、Empty/Loading/Skeleton、Breadcrumb、PullDownRefresh 等。
- 样式覆盖 tokens、Tailwind、两套主题、UI refined/polish/UI kit、TDesign skin/theme/overrides、modal-system、animations。
- 本次是以源码、组件调用关系、治理文档、GitHub Actions 结果为依据的仓库复核；未连接生产环境，未执行真实设备手工视觉验收。

### 已确认收敛（本轮不再列为待修）

- 普通动作按钮已按仓库既定 TDesign + `.ui-t-button` 入口收敛；`.ui-btn` 模板调用静态扫描为 0。图标按钮、导航、分段控件和原生语义控件保留各自用途。
- 项目业务 Modal 的桌面居中/移动端几何由 Modal 与 modal-system 管理；会话页两个业务 Dialog 已迁移到 Modal；确认内容和动作复用 ConfirmPrompt/ConfirmActions。
- dark 主题 TDesign 基础按钮兜底规则已排除语义按钮，避免覆盖主/危险色。
- 空状态图标映射、Loading 可访问状态、会话页首屏加载态已修。
- 管理端卡密桌面分页、Join 执行账号群组异步竞态、TargetSelector/GroupSelector 账号切换缓存、账号批量选择与草稿隔离、Broadcast DTO 选项透传和循环模式不支持项提示、任务列表请求竞态、任务详情操作/SSE/日志历史分页等已完成相应代码整改。
- CSS 重复选择器复核未证明存在可安全删除的死规则；不得把相同选择器文本直接等同于重复/无效 CSS。

### 当前剩余问题清单

| 编号 | 级别 | 位置/范围 | 当前证据与影响 | 下一步 |
|---|---|---|---|---|
| R1 | P1（验收阻塞） | 全部页面与公共浮层 | 自动 Browser UI audit 已通过，但这不是人工视觉验收，无法证明桌面/移动端三主题下的居中、遮挡、焦点、键盘和滚动行为均正确 | 使用真实浏览器逐模块验收 normal/eye/dark，桌面和移动视口；记录截图与复现步骤 |
| R2 | P2 | `styles/ui-refined.css`、`styles/ui-polish.css`、`useAsyncBtn` 及调用方 | `.btn-success`、`.btn-error` 仍保留；现有搜索不足以证明动态 class、组合 class 或运行时注入没有调用 | 追踪 composable 返回值、全部调用方和模板绑定；有完整零调用证据后再决定迁移/删除 |
| R3 | P2 | `components/ui/Modal.vue` | 有模块级 VisualViewport/弹窗栈生命周期修复，但没有专门 Modal 单测；浏览器自动审计不能替代多弹窗键盘/焦点测试 | 测试桌面居中、窄屏底部策略、软键盘、嵌套弹窗、Esc、遮罩、滚动锁定和卸载 |
| R4 | P2 | `views/tasks/Detail.vue` | 任务详情的 SSE 重连、等待倒计时、实时日志、历史游标和滚动锚定有代码防护；需要运行中的任务才能验证事件交错和断线恢复 | 用持续产生日志的测试任务验证断线重连、旧 EventSource 回调、翻页时插入新日志和到达最旧页 |
| R5 | P2 | `components/selectors/TargetSelector.vue`、`GroupSelector.vue` | 大列表已限制首屏渲染 200 项并支持增量加载；全量筛选/全选语义由代码保留，但极大数据量下的滚动、搜索和交互响应未实测 | 用数千/数万群组/联系人数据验证搜索、全选、反选、分页渲染、刷新与切换账号 |
| R6 | P2 | `views/workspace/Broadcast.vue`、`Join.vue`、`Scrape.vue`、`Tasks.vue` 等关键业务表单 | 发现并修复过多项请求竞态和表单状态问题；全流程浏览器回归仍未完成 | 逐项走通创建、校验、取消、失败重试、账号切换、媒体上传中提交、循环模式限制和任务刷新 |
| R7 | P3（代码与自动化验证已通过） | `web/src/main.ts` 全局错误探针 | 已改为仅显示通用错误提示，不再把异常堆栈、服务器响应、内部路径、资源 URL 或查询参数展示给租户；同类提示 3 秒内去重；横幅增加 `role=alert`/`aria-live`，点击关闭同步清理计时器；最新提交 `eb87ad7` 的 CI 与 Browser UI audit 均通过 | 代码与自动化门禁关闭；人工验证横幅显示、读屏播报、关闭与自动消失仍并入 R1 |
| R8 | P2（已关闭） | `internal/repository/task_integration_test.go` | 测试夹具缺少 `task_logs` 表的问题已通过最小表结构和清理逻辑修复；提交 `470421c` 上 CI build、Go tests 已通过 | 已由 CI 验证通过，关闭 |

### 验证状态

- 基线 SHA `d08037d`：Frontend Typecheck + Unit Tests、frontend、integration、sql-audit、secrets 均成功；Browser UI audit 成功。
- 同一 SHA 的 `build` Job 中 `gofmt`、`go vet`、`gosec` 成功，但 `go test` 因 R8 的测试夹具缺表失败；日志已定位到具体失败测试和 SQL relation，不是笼统的 Runner 启动故障。
- R8 修复先提交于 `8287fb5`，并在后续 `470421c` 的 CI 中通过；CI 的 `build` Job（含 Go tests）、integration、secrets、sql-audit、Frontend Typecheck + Unit Tests、frontend 全部成功。
- Browser UI audit 在 `470421c` 上成功，生成 Playwright 报告 artifact（`tg-sender-playwright-report`）；该结果不代表人工浏览器验收，也不代表所有主题/设备组合都被覆盖。
- `580903e` 收敛全局错误横幅：展示内容脱敏、同类错误 3 秒去重、ARIA alert 语义和关闭时清理计时器；后续最新提交 `eb87ad7` 的 CI 与 Browser UI audit 均已成功，R7 的代码与自动化验证条件满足。
- 尚未删除任何组件、样式文件或依赖；没有证据支持的删除候选继续保留。
- 未合并 PR、未部署生产。

### 关闭标准

R1、R3、R4、R5、R6 必须有桌面/移动真实浏览器记录；R2 必须有动态引用追踪结果；R8 必须在修复提交上取得 CI 全绿。以上条件完成前，本轮前端全仓问题清单保持“复核已完成、验收未关闭”状态。


### 复核进展更新（2026-11-10）

- `470421c` 的 CI 全部 Job 成功：integration、secrets、Frontend Typecheck + Unit Tests、frontend、sql-audit、build；Go 测试夹具问题 R8 关闭。
- `470421c` 的 Browser UI audit Job 成功，Playwright 报告已上传为 `tg-sender-playwright-report` artifact。自动化成功仍不等同于人工视觉验收。
- R7 已在后续提交 `580903e` 修复：横幅不再把 stack、内部路径、服务器响应和资源 URL 展示给用户；相同类别提示 3 秒内去重；增加 ARIA alert 属性，并在手动关闭时清除计时器。等待该提交 CI 验证。
- R2 暂不关闭：`.btn-success`/`.btn-error` CSS 仍存在；GitHub Code Search 返回 0 结果但标记 `incomplete_results=true`，且搜索索引针对默认分支，不能证明 PR 分支及运行时拼接无引用。需要本地完整仓库检索与动态调用追踪。
- 最新分支 HEAD：`eb87ad76bc7d3db948f6cb0bcbecfe0ba94de8e3`。该 SHA 的 CI 与 Browser UI audit 均为 completed/success；Playwright 自动化结果不等同于人工视觉验收。
- R2 继续保持未关闭：本次确认分支树中不存在独立的 `useAsyncBtn` 源文件，且 CSS 中仍保留 `.btn-success`/`.btn-error` 规则；尚未取得完整的模板动态绑定/运行时注入零引用证据，因此不删除旧规则。
- 当前仍需人工/真实浏览器验收：R1、R3、R4、R5、R6；R7 的人工可访问性检查并入 R1。


### Modal runtime lifecycle test update（2026-11-10）

- 在 `103d20bcaf176fe0a86a4c03c52b7ae0cf5a71fc` 中，将 Modal 的弹窗栈和共享 VisualViewport 键盘保护生命周期抽取到 `components/ui/modalRuntime.ts`，Modal.vue 改为调用统一 runtime。
- 新增 `modalRuntime.test.ts` 的 3 项 Vitest 测试：弹窗 z-index/关闭后顺序压缩、多弹窗共享 viewport 监听且最后一个关闭时统一清理、重复注册及小视口偏移边界。
- 该提交的 CI（包括 Frontend Typecheck + Unit Tests、frontend、integration、sql-audit、secrets、build）与 Browser UI audit 均 completed/success。
- R3 仍不关闭：单元测试和自动化审计不能替代真实浏览器的焦点陷阱、Esc/遮罩、嵌套弹窗、软键盘与滚动锁定验收；这些继续并入 R1 的桌面/移动端手工验收。
- 测试文件数量由此前复核时的 6 个增加为至少 7 个；后续统计以分支当前文件树为准。

### Task log runtime test update（2026-11-10）

- 在 `d0ab3dd513ebfe90d102c560227b4c15bf807385` 中，将任务日志稳定身份键、HTTP/SSE 合并去重、按时间与 log_id 稳定排序、历史页游标状态抽取到 `views/tasks/taskLogRuntime.ts`。
- 新增 `taskLogRuntime.test.ts` 的 5 项测试：持久化 ID 去重、无 ID 结构化身份避免分隔符碰撞、实时与历史日志合并、同时间戳稳定排序、分页游标边界。
- `Detail.vue` 的历史页加载现在复用同一合并函数；保留请求 ID 防止旧分页响应覆盖新状态，并继续使用后端 keyset cursor。
- 提交 `d0ab3dd` 的 CI 全部 Job completed/success（build、integration、frontend、Frontend Typecheck + Unit Tests、sql-audit、secrets）；Browser UI audit completed/success。
- R4 仍不关闭：这些测试覆盖纯运行时数据逻辑，不模拟真实 EventSource 生命周期、断线重连和浏览器滚动锚定。需要在持续产生日志的任务中完成桌面/移动浏览器验收。


### Workspace request and form snapshot hardening（2026-11-10）

- 在 `0416993` 中，Scrape 任务列表请求捕获 accountId 与 request generation；账号切换、页面卸载或后续请求启动后，旧响应/错误/finally 不再覆盖当前列表或 loading 状态。
- 在 `b640049` 中，Broadcast 目标预检加入单飞保护，并在请求前快照目标 ID；只有当前选择仍与快照完全一致时才应用预检结果，避免旧预检响应删除用户后来新增的目标。
- 在 `ca3ceff` 中，Join 提交增加重复提交保护；邀请任务捕获账号/目标群/来源群快照；多账号批量进群在首个 await 前快照去重后的账号、链接、分配策略与间隔，整个批次使用同一配置，避免用户中途编辑表单造成批次配置漂移。
- `ca3ceff41654c8e1728fb878f9d7d0e76a40c8bc` 的 CI 全部 6 个 Job（build、integration、frontend、Frontend Typecheck + Unit Tests、sql-audit、secrets）均 completed/success；Browser UI audit completed/success。
- R6 仍不关闭：自动化门禁通过不能替代关键业务流程人工回归；仍需覆盖账号切换、预检期间增删目标、Join 连续点击/中途编辑、失败重试与任务刷新等桌面/移动端场景。

- 后续复核发现 Join 任务列表仍缺少请求代次与页面卸载保护：15 秒轮询、SSE 状态刷新和手动刷新可能交错。已在 `edbe987` 中增加 request generation、当前账号快照校验和卸载失效保护；仅最新有效请求可以更新列表/错误/loading。该修复等待最新 SHA 自动化验证。

- 后续异步边界复核追加：
  - Scrape 成员详情请求按 taskId 维护 request generation，并校验账号仍一致、任务仍在当前列表；账号切换时清空成员详情/展开状态并使旧请求失效，避免迟到响应写回旧数据。
  - Scrape 清理记录在确认后再次校验账号，删除任务 ID 在确认前快照化；执行期间切换账号不会清空新账号的 UI 状态。
  - Scrape 创建任务加入单飞保护并快照账号、群组、模式和过滤配置；创建完成后仅在账号/群组仍匹配时关闭当前创建上下文，避免清空用户新选择。
  - Join 邀请群组加载现在显式处理失败，只允许当前账号请求写入错误状态，避免未处理 Promise rejection。
- 上述 Scrape/Join 追加修复分别落在 `78cbc2d`、`2557789`、`cb174b8`；最新 PR head 为 `cb174b8d03f773b197bd048f3df6cf0b56fd601f`。等待该 head 的 CI 与 Browser UI audit 结果后再记录最终门禁状态。
- 群组缓存层再补一处并发保护：`groupStore.refresh(accountId)` 为每个账号维护请求代次与失效 epoch；只允许最新且未被 mutation/invalidate 作废的请求写入 sessionStorage 缓存，防止较早请求迟到后覆盖较新群列表，或在群组写操作清缓存后把旧数据重新填回。修复提交：`f2b8f75`、`429f4c7`。


### Legacy success/error button class audit automation（2026-11-10）

- 新增 `web/scripts/audit-legacy-button-classes.mjs`，扫描 `web/src` 下 Vue/TS/JS 与 CSS/SCSS 文件，分别列出 `.btn-success`、`.btn-error` 的样式定义、源码静态引用、`useAsyncBtn` 引用和疑似动态 class 表达式。
- 新增 `npm run audit:legacy-css`，并接入 CI 的 frontend job，避免仅凭人工抽查或 GitHub 默认分支搜索判断旧样式是否可删。
- 对 PR 分支源码的逐项抽查中，已检查全部 Vue SFC、主要业务组件、选择器、composables 和相关样式；目前未发现这两个 class 的模板/组件调用点，CSS 定义仍在 `ui-refined.css`。完整结果以本次 CI 输出为准。
- 该扫描器是保守审计工具，不会自动删除 CSS；动态拼接、外部运行时注入和第三方集成仍需人工审查。R2 在取得 CI 扫描结果并复核所有候选后再决定是否删除，暂不关闭。


### Selector dead-code and large-list cleanup（2026-11-10）

- `GroupSelector.vue` 中 `autoSync()` 没有任何调用点；其整套重复同步/轮询实现已移除，保留当前实际使用的 `loadAccountGroups()`、`onRefresh()` 与共享同步锁流程。同步失败计数变量和未使用的轮询计数也一并删除。
- `TargetSelector.vue` 中未接入界面的 `syncAllAccounts()`、`syncAllLoading`，以及只写不读的 `fromCache` 状态已移除；现有按账号读取联系人缓存、按当前账号刷新群组的行为保持不变。
- 两个选择器继续采用最多首批渲染 200 条、搜索过滤后增量加载；GroupSelector 的全选/反选继续复用 Set-based `toggleSelectionByKey`，其 10,000 项语义压力测试已存在。
- 这批改动是已确认无调用的局部死代码清理，不删除任何样式、API 或用户可见功能。大数据量真实浏览器滚动/搜索体验仍需 R5 手工压力验收。


### R2 legacy button selectors removal（2026-11-10）

- 最新全仓审计在 124 个 Vue/TS/JS/CSS/SCSS 文件中得到：两个旧选择器 CSS 定义各 1 处；源码静态引用 0；`useAsyncBtn` 只在旧 CSS 注释中出现 1 次；动态 class 候选仅 1 处，位于任务日志错误状态的 Tailwind 令牌 class，与旧按钮选择器无关。
- 因为已经同时检查了静态引用、动态 class 候选和旧 helper 标记，现从 `ui-refined.css` 删除这两条无调用规则及过时注释，不影响当前按钮体系。
- R2 的这两个遗留选择器已完成代码清理；删除后的 CI 再次扫描必须确认定义、静态引用和 helper 标记均为 0，之后关闭该项。


### Batch D7 — inventory becomes a CI-visible guard

- Added `npm run audit:components` to the frontend CI job after the design-token and legacy-button audits.
- Every PR CI run now records the current direct component import map, possible unreferenced SFC candidates, duplicate component filenames, global CSS import order, button/form-control usage, confirmation API usage, and Modal/Empty/Loading/Dialog usage.
- This remains an informational inventory: no component is deleted and no build is failed merely because a component has no direct import. Dynamic/runtime references and template semantics still require review before removal.
- The standalone project `components/ui/Loading.vue` remains a removal candidate only; no deletion is included in this batch because a zero direct-import count alone is not enough evidence to rule out runtime references.


### Batch D8 — final dead-component sweep and stylesheet ownership inventory

- The CI component inventory for the pre-cleanup tree found exactly one component with no direct import: `components/AppShell.vue`. Repository code search also returned no `AppShell` references; there is no Vue auto-component registration in `vite.config.ts`. Removed this orphan SFC rather than keeping an unreachable second application shell beside the routed `views/Layout.vue` shell.
- Removed the unused `WifiOff` import from `App.vue`; the network banner uses text only and did not render that icon.
- The same inventory confirmed `components/ui/Loading.vue` is used by `views/sessions/Index.vue`; it is retained. All other UI components in the inventory have direct import sites. No duplicate component basenames were found.
- CSS ownership is now recorded from the actual `main.ts` import order: tokens/theme files own semantic values and theme surfaces; Tailwind is the utility layer; `ui-kit.css` owns shared component primitives; TDesign skin/theme/overrides own vendor adaptation; `modal-system.css` owns Vue Final Modal geometry; `animations.css` owns reusable motion primitives. `ui-polish.css` and `ui-refined.css` remain legacy refinement layers with overlapping selectors. They are intentionally not bulk-deleted: selector overlap alone does not prove identical cascade behavior. This is the remaining stylesheet consolidation candidate and requires computed-style/browser regression before removing rules.
- No behavior or route changes were made by this cleanup. The component inventory is a CI review aid and deliberately does not auto-delete dynamic/runtime components.


### Batch F1 — mobile action touch-target consistency

- The component inventory reports 190 TDesign `<t-button>` matches and 106 native `<button>` matches. The shared `.ui-t-button` class is already widely adopted (446 source matches), while `.ui-native-button` currently has no source call sites. These are source-level counts, not proof that every native button should be replaced: file inputs, selector rows, modal close controls, and other semantic actions have different behavior.
- Standardized the compact shared TDesign/native button variants to a 44px minimum height at the mobile breakpoint. This preserves their visual compact variant on desktop while keeping touch targets consistent on phones.
- Enlarged the VueFinalModal close control to a 44px square on mobile without changing its icon, event handler, desktop geometry, or modal positioning.
- Kept the broader migration of 106 native button call sites out of this change. It requires classifying each control as an action, icon affordance, selector row, or structural control before changing semantics or styling.
- Verification target: CI, frontend typecheck/unit tests, build, and Browser UI audit. Automated screenshots/viewport assertions do not replace manual review on real iOS/Android devices.


### Batch F2 — animation ownership consolidation

- `main.ts` imports `tailwind.css` before `animations.css`. The later animation layer already defined the effective `.animate-float`, `.animate-shimmer`, and `.animate-fade-in-scale` utilities, shadowing the earlier Tailwind-layer copies.
- Removed only the shadowed `float`/`shimmer` keyframes and utility rules plus the unused-by-cascade `fadeInScale` keyframe/utility from `tailwind.css`. The canonical implementations remain in `animations.css`.
- Kept Tailwind's distinct `pulse-ring` and other utility animations; this is not a blanket animation cleanup.
- This is cascade-equivalent under the verified import order. CI and Browser UI audit are required to catch build/runtime regressions.


### Batch F3 — notification settings form-control alignment

- Migrated the five notification/SMTP text controls in `views/settings/Index.vue` from repeated one-off Tailwind field styling to the shared `.ui-input` primitive. Grid placement, flex sizing, password/number/email types, and model bindings are preserved.
- Added explicit accessible names to the email, SMTP username, password, host, and port controls; placeholders are retained as visual hints rather than the only accessible name.
- Added a consistent disabled affordance to `.ui-input`. Keyboard focus continues to use the existing global soft focus ring; a second outline was deliberately avoided to prevent a double-ring effect.
- This batch intentionally does not convert checkboxes, hidden file inputs, or switches into text-field styles. Native controls retain their semantics.
- Verification target: frontend typecheck/unit tests, build, and Browser UI audit. Manual keyboard and mobile visual QA remain distinct acceptance steps.


### Batch F4 — Join/Scrape/Monitor form-control alignment

- Migrated the remaining Join task form's account and distribution selects, plus the multiline group-link field, to the shared `.ui-input` primitive. Existing v-model bindings, select options, textarea row sizing intent, and link-import behavior remain unchanged.
- Migrated Scrape's history-message numeric limit to `.ui-input`; its bounded numeric semantics and 36px compact layout are preserved.
- Migrated Monitor's page-level account selector to `.ui-input`, matching the rule editor's existing selects and text fields. Added an explicit accessible name.
- Left checkboxes and the compact native recent-days picker native: these controls have native checkbox semantics and the picker intentionally invokes the mobile system selection UI. They should not be forced into text-field styling.
- Rechecked `modal-system.css` and `Modal.vue`: desktop wrapper uses centered flex alignment and a bounded 560px content width; mobile uses a bottom-anchored dynamic-viewport container, a constrained drawer height, and a flex-column card with the body as the scroll container and footer fixed outside that scroll region. No geometry rewrite was justified by the source audit; real-device keyboard/safe-area behavior remains a separate visual acceptance item.
- Verification target: frontend typecheck/unit tests, production build, and Browser UI audit. Automated checks do not constitute manual desktop/mobile visual sign-off.
