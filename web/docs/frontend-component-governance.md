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
| Shared button/input primitives | TDesign components plus project adapters in `ui-kit.css` | Prefer existing primitives and variants; no page-local global button systems. |
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
