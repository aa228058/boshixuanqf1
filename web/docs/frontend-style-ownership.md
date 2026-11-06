# Frontend stylesheet ownership and acceptance

This document records the current stylesheet contract. It is intentionally conservative: it does not delete legacy selectors until their runtime usage and visual effect have been verified.

## Actual global import order

From `web/src/main.ts`:

1. TDesign vendor CSS and Vue Final Modal vendor CSS
2. `tokens.css` — semantic design-token source
3. `tailwind.css` — Tailwind utilities, compatibility aliases and legacy component styles
4. `animations.css` — reusable keyframes and animation utility classes
5. `theme-stripe.css`, `theme-dark.css` — theme-specific legacy selectors and theme overrides
6. `ui-refined.css` — legacy visual compatibility and generic refinements
7. `ui-polish.css` — interaction transitions, focus treatment, list motion and WebKit scrollbar baseline
8. `tdesign-skin.css` — theme-aware TDesign skin
9. `tdesign-theme.css` — TDesign semantic variable bridge
10. `ui-kit.css` — shared project component classes and button variants
11. `tdesign-overrides.css` — targeted TDesign behavior/layout overrides, including native dialog mobile layout
12. `modal-system.css` — final Vue Final Modal geometry and footer layout

Import order is part of runtime behavior. Do not reorder these files based only on their names.

## Ownership contract

| Concern | Owner | Allowed elsewhere |
|---|---|---|
| Semantic colors, spacing, typography, radius, elevation tokens | `tokens.css` | Themes may assign semantic token values; components consume tokens |
| Tailwind utilities and old utility aliases | `tailwind.css` | Do not create a second utility framework |
| Shared project component classes (cards, segmented controls, icon buttons, shared TDesign button variants) | `ui-kit.css` | Pages compose classes; avoid redefining their base geometry |
| TDesign semantic variable mapping | `tdesign-theme.css` | No page-level global TDesign token overrides |
| TDesign skin / theme-specific appearance | `tdesign-skin.css` | Special behavior overrides only in `tdesign-overrides.css` |
| TDesign DOM/layout exceptions (notably native `t-dialog`) | `tdesign-overrides.css` | Do not put Vue Final Modal geometry here |
| Vue Final Modal positioning, desktop centering, mobile bottom sheet, keyboard offset | `modal-system.css` | Component supplies structure/state; do not duplicate global geometry in scoped CSS |
| Interaction-only motion and generic focus/hover feedback | `ui-polish.css` | Theme files may adjust colors only when required |
| Reusable keyframes and animation utility classes | `animations.css` | Do not redefine same keyframe names in page styles |
| Legacy selectors pending migration | `theme-*.css`, `ui-refined.css`, relevant sections in `tailwind.css` | New code must not extend the legacy layer |

## Conflict resolution

1. User-visible product behavior and accessibility win over cosmetic consistency.
2. Existing TDesign / Vue Final Modal component APIs and runtime behavior win over generic design-pattern suggestions.
3. Token semantics win over hard-coded page colors. Theme files may map tokens; they must not introduce a competing token vocabulary.
4. `ui-kit.css` owns reusable component geometry. TDesign-specific behavior exceptions belong in `tdesign-overrides.css`.
5. `modal-system.css` is the final authority for Vue Final Modal geometry. Do not replace desktop centering with bottom-sheet rules or vice versa.
6. When two rules conflict, inspect selector specificity and actual import order before editing. Do not solve conflicts by adding another broad `!important` rule.
7. Do not delete a legacy selector until repository-wide usage has been searched and its visual/runtime effect has been verified.

## Known legacy overlap — do not silently remove

- `theme-stripe.css` and `theme-dark.css` contain theme-specific component selectors that overlap conceptually with the token and TDesign skin layers.
- `tdesign-skin.css` contains broad TDesign skin selectors, including button and input surfaces; `ui-kit.css` adds shared geometry and opt-in `.ui-t-button` variants.
- `tdesign-overrides.css` contains native TDesign dialog layout and mobile drawer behavior; `modal-system.css` targets Vue Final Modal classes. These are different modal implementations and must not be merged as if they were one.
- `tailwind.css` still contains compatibility aliases and legacy component styling, so it is not a utilities-only file yet.
- `ui-refined.css` is a compatibility layer, not the canonical component system.
- `ui-polish.css` is not the last stylesheet imported; its interaction rules are followed by TDesign and modal-specific layers.

## Change acceptance checklist

- [ ] No new UI framework, CSS framework, icon system, or duplicate component base.
- [ ] New component styling consumes existing semantic tokens.
- [ ] No broad global selector added where a component-scoped selector is sufficient.
- [ ] No duplicate declarations added to compensate for unknown import order.
- [ ] Desktop modal remains centered; mobile modal behavior remains intentionally differentiated.
- [ ] Normal, eye, and dark themes retain readable surfaces, text, borders, focus and status colors.
- [ ] Reduced-motion preference remains respected.
- [ ] Keyboard focus remains visible; icon-only controls retain accessible names.
- [ ] Run `npm run typecheck`, `npm run test`, `npm run check:design-tokens`, and `npm run build` from `web/`.
- [ ] Review Git diff and verify CI against the exact commit. Static checks do not replace desktop/mobile browser verification.

## Safe migration sequence

1. Inventory selector usage and visual ownership.
2. Move or consolidate one component family at a time (buttons, fields, cards, badges, dialogs).
3. Remove old selectors only after proving no remaining callers and no theme regressions.
4. Run all four frontend checks and review all three themes at desktop and mobile breakpoints.
5. Keep legacy files until their remaining selectors are migrated; then remove imports and files in a separate verified change.
