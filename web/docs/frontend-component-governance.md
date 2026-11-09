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


### Batch B2 — button adapter foundation

- Added `components/ui/UiButton.vue` as the canonical project-owned TDesign button adapter.
- The adapter maps semantic tones (`primary`, `secondary`, `danger`, `text`) to TDesign theme/variant values and applies the shared `.ui-t-button` design tokens/classes.
- Native TDesign attributes and events (including `type`, `disabled`, `loading`, click handlers and other attrs) are forwarded to the underlying button.
- Migrated the shared confirmation action component to `UiButton`, making both confirmation entry points use the same button implementation.
- This is the migration foundation, not a claim that every existing page button has already been migrated. Remaining direct TDesign/native button call sites require inventory and page-by-page conversion.


### Batch B3 — first page migration wave

Migrated all TDesign button tags to the shared `UiButton.vue` adapter in these seven page files:

- `views/groups/Discover.vue` (14 button instances)
- `views/sessions/Index.vue` (7)
- `views/tasks/Detail.vue` (8)
- `views/workspace/Broadcast.vue` (13)
- `views/accounts/List.vue` (35)
- `views/settings/Index.vue` (13)
- `views/admin/tenants/List.vue` (5)

Total: 95 TDesign button instances now use the canonical adapter. The adapter supports existing `theme`, `variant`, and size attributes while allowing new code to use semantic `tone` values. Native `button` elements remain native where they implement tabs, toggles, icon-only controls, or other specialized semantics; they are not blindly converted to generic buttons.

### Batch C — forms and data display
- Normalize input/select/checkbox/switch/date-picker wrappers and validation behavior.
- Normalize table, pagination, badges, tooltips and empty/loading states.

### Batch D — stylesheet consolidation
- Reduce overlapping TDesign overrides.
- Retire obsolete selectors only after call-site and cascade verification.
- Preserve theme parity and responsive behavior.

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
