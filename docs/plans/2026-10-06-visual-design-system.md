# Visual Design System — Implementation Plan

> **For agentic workers:** executed by `/orchestrate` (orchestrator + `designer` → `worker` → `reviewer` lanes, one branch and worktree per lane). Each task is one lane. Global constraints, gates and git rules are those of `docs/plans/2026-10-05-platform-core-and-staffing.md` and are not repeated.

**Goal:** every page of the web app, in light and dark mode, uses one palette, two typefaces and one icon set, meets WCAG AA, and reads as deliberately designed (spec §1, §5).

**Spec:** `docs/specs/2026-10-06-visual-design-system.md` — every task cites its sections.

**Order:** Task 1 (foundation) lands first; it owns `index.css`, `index.html`, `package.json` and `components/`. Tasks 2-5 then run in parallel, one feature directory each, against the landed components. Task 6 last.

## Review Focus

1. **No literal colours** — the `no-literal-colors` test is the gate; a reviewer greps too. Pinned in Task 1.
2. **Both modes on every page** — a page that looks right in light and wrong in dark is a defect, not polish. Reviewers switch modes on each page they review.
3. **Copy and selectors unchanged** — `npm run e2e` is the arbiter; the only e2e edit is the dark-mode test in Task 1. Pinned in every task.
4. **Contrast is measured** — every token pair in `docs/design/P6-L1.md` has a number; the reviewer re-measures three at random.
5. **No one-off styles** — a feature file may compose tokens and components; it may not define a new recipe. Pinned in Tasks 2-5.

## File Structure

```
apps/web/index.html                              theme boot script (same-origin file, CSP), color-scheme
apps/web/package.json                            lucide-react, @fontsource-variable/dm-sans
apps/web/src/index.css                           fonts, tokens for both modes, base, ui-table, skeleton
apps/web/src/lib/theme.ts (+ test)               read/write/apply `unicourse:theme`
apps/web/src/components/theme-switch.tsx         System / Light / Dark control
apps/web/src/components/card.tsx                 section/course card (cover band, title, meta)
apps/web/src/components/header-search.tsx        search field that navigates to the catalog with q
apps/web/src/components/*.tsx                    restyled shared components, icons
apps/web/src/no-literal-colors.test.ts           gate
apps/web/src/features/auth/*, routes/sign-in.tsx, routes/invite.$token.tsx, routes/index.tsx   Task 2
apps/web/src/features/my-sections/*, features/catalog/*, routes/$slug/{me,catalog,index}.tsx   Task 3
apps/web/src/features/courses/*, routes/$slug/courses*, routes/$slug/sections.$id.tsx          Task 4
apps/web/src/features/admin/*, routes/$slug/admin*                                             Task 5
apps/web/e2e/theme.spec.ts                       Task 1 (dark mode persists across reload)
```

---

## Phase 6 — Visual design system

### Task 1 (P6-L1, D): foundation — tokens, fonts, icons, theme switch, shared components

Spec §3, §4, §6 (Shell, Shared), §7, §8. Designer writes `docs/design/P6-L1.md`: the full token table for both modes with measured ratios, the derived shades and their sources, the type scale, the icon per navigation item and per shared component, the shell layout, the restyled recipe of each shared component (states included), the skeleton recipe, the theme-switch markup, and the per-page notes Tasks 2-5 build from (one section per page of spec §6 with the composition of components, the icons used and any copy change with its selector check). Sub-lanes a/b if the designer judges the worker budget needs it.

- [ ] Add the three packages; import the two fonts in `index.css`; set `--font-sans`, `--font-serif`.
- [ ] Tokens for both modes per the design; `color-scheme`; `[data-theme]` and `prefers-color-scheme` rules; reduced-motion rule.
- [ ] `lib/theme.ts` with a unit test: `read()`, `write(choice)`, `apply(choice)`, `boot()`; the inline boot script in `index.html` is the same logic, minified by hand, with a comment pointing at `theme.ts`.
- [ ] `theme-switch.tsx`: three-option control, labelled, keyboard operable, in the header.
- [ ] Restyle every file in `components/`: button (`on-accent` token, icon slot), badge (warning tone via tokens), dialog (header/body/footer rhythm), empty-state (icon slot), error-banner, form-field, load-more, loading (skeleton), page-header, status-message (Lucide check), table (`ui-table` for both modes). Existing component tests stay green; add state tests only where behaviour changed.
- [ ] `app-shell.tsx`: Udemy-style top bar: wordmark, text nav, header search to the catalog, university switch, theme switch, sign-out, hairline.
- [ ] `card.tsx`: the shared card (spec §6) with a unit test.
- [ ] `no-literal-colors.test.ts` per spec §8; make it pass by fixing `badge.tsx` and `button.tsx`.
- [ ] `e2e/theme.spec.ts`: sign in, choose Dark, reload, `html[data-theme="dark"]`.
- [ ] Gates: lint, typecheck, test, build; e2e. Commit.

### Task 2 (P6-L2, W): entry pages

Spec §6 (Entry), §5.2. Scope: `features/auth/*`, `routes/sign-in.tsx`, `routes/invite.$token.tsx`, `routes/index.tsx`. Build each page as `docs/design/P6-L1.md` describes; replace the Unicode divider; provider buttons with icons; no copy change. Gates: lint, typecheck, test, build; the auth unit tests green.

### Task 3 (P6-L3, W): my sections and catalog

Spec §6 (My sections, Catalog), §8 (D60). Scope: `features/my-sections/*`, `features/catalog/*`, `routes/$slug/me.tsx`, `routes/$slug/catalog.tsx`, `routes/$slug/index.tsx`, and the e2e selectors for these two pages only. The list and the catalog become card grids on the shared card; the week grid keeps its DOM (regions, list items, clash text); the catalog reads `q` from the URL so the header search lands on filtered results. Gates as Task 2, plus `lib/schedule.test.ts` untouched and green, and `npm run e2e`.

### Task 4 (P6-L4, W; sub-lanes a: lists and detail, b: dialogs and editors): courses

Spec §6 (Courses). Scope: `features/courses/*`, `routes/$slug/courses.tsx`, `routes/$slug/courses/*`, `routes/$slug/sections.$id.tsx`. 4a: course list, section list, section detail, roster. 4b: course, section, import, remove dialogs, staff editor, meetings editor. Every dialog uses the shared dialog rhythm; import preview badges per §3.1. Gates as Task 2.

### Task 5 (P6-L5, W): admin

Spec §6 (Admin). Scope: `features/admin/*`, `routes/$slug/admin.tsx`, `routes/$slug/admin/*`. Tab list restyled; `admin-layout.test.tsx` asserts `aria-current` instead of classes; tables and dialogs through shared components. Gates as Task 2.

### Task 6 (P6-L6, orchestrator): evidence and instructions

Spec §1, §9. Orchestrator: production build served by the API, every page of §6 in both modes screenshotted, contrast of three random pairs re-measured, full web gate and e2e; then `CLAUDE.md` gains a "UI conventions" section (palette link, fonts, icons, tokens, dark mode rule, the literal-colour gate, where the design document is) so later sessions and lanes follow it; PLAN.md updated. No lane.
