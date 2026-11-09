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
