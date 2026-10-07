# Spec 3: Visual design system - light and dark

Status: decided by the orchestrator under D8 delegation, owner to confirm · Date: 2026-10-06 · Builds on Spec 1 and Spec 2 (web only; no API change)

## 1. Intent

Every screen of the web app looks like one deliberate, professional product that a university would pay for, in light and in dark mode, and reads as designed by a person: one palette, two typefaces, one icon set, one density, real copy. Nothing in the data model, the API, the routes or the user-facing copy changes; this spec restyles what exists.

Owner's brief (2026-10-06): "Make all the UI very professional, should have light and dark mode, every page should look good, website should not look AI slop or AI generated; use a coolors.co trending palette (choose one, stick to it), Google Fonts, Lucide icons; leave instructions so any next session is aware."

**Success criterion.** With the API serving the production build, a reviewer opens every page in §6 twice, once in light and once in dark mode, and finds: the palette, typefaces and icons of §3 and nothing else; every text and control pair at or above the WCAG AA ratios in §7; the theme switch in the header offering System, Light and Dark, the choice surviving a reload and a sign-out, and no flash of the wrong theme on load; the full web gate (lint, typecheck, unit tests, build) and the existing end-to-end suite green with no edit to the end-to-end suite other than the one test this spec adds; and a unit test that fails when a component uses a literal colour. The orchestrator records screenshots of every page in both modes as the evidence.

## 2. Out of scope

A marketing or landing site · a logo beyond a text wordmark · charts · a component library (shadcn, Radix, MUI) · CSS-in-JS · animation beyond 150 ms colour and opacity transitions · a mobile layout below 768 px beyond "nothing overflows and nothing is unreachable" · a high-contrast theme · user-level theme storage on the server (the choice is per browser) · translations · any change to copy that an end-to-end test or a unit test asserts, except the class assertions in `admin-layout.test.tsx` that this spec retires.

## 3. Visual system

### 3.1 Palette

One coolors.co trending palette, used as is: **Deep Blue Sea**, `https://coolors.co/palette/134074-13315c-0b2545-8da9c4-eef4ed`.

| Swatch | Hex | Light mode role | Dark mode role |
|---|---|---|---|
| Yale blue | `#134074` | accent: primary buttons, links, active navigation, focus ring base | surface of the primary button (white text, 10.4:1), raised table header |
| Berkeley blue | `#13315C` | ink for headings (12.9:1 on white) | surface: cards, tables, dialogs, inputs |
| Oxford blue | `#0B2545` | ink for body text (15.4:1 on white, 13.8:1 on canvas) | canvas: page background |
| Powder blue | `#8DA9C4` | hairlines, table rules, disabled text, decorative only (2.4:1, never for text) | muted text (5.3:1 on surface), links (or a derived lighter tint when 7:1 is wanted), input borders |
| Mint cream | `#EEF4ED` | canvas: page background | ink: body text (11.6:1 on surface, 13.8:1 on canvas) |

Derived shades are allowed only where the five swatches cannot meet §7, and each one is written down in the design document with the swatch it is derived from and its measured ratio: a light-mode muted text (a Berkeley blue tint around `#4A5D78`, ≥ 6:1 on white and canvas), a light-mode strong line for input borders (≥ 3:1 on white), an accent-soft background for active navigation and badges (an Oxford blue tint at low alpha over the surface), hover shades of the accent, and a dark-mode line between surface and canvas. Pure white `#FFFFFF` is the light-mode surface; there is no pure black anywhere.

Status colours are not brand colours and stay out of the palette: danger (red), success (green), warning (amber), each as a text shade for its mode and a soft background, each ≥ 4.5:1 on the surface it sits on. They are the only colours outside the palette and they are used only to mean error, success or caution, never for decoration.

### 3.2 Typography

Two open-source families from Google Fonts, self-hosted through `@fontsource-variable` packages so the app makes no request to Google at runtime:

- **Source Serif 4** (variable) for the wordmark and page titles (`h1`) only. Weight 600, tight tracking, optical size left to the font.
- **Source Sans 3** (variable) for everything else: body, tables, forms, buttons, section headings (`h2`, `h3`). Weights 400, 500 and 600 only.

Scale: 16 px body (`1rem`), 14 px for tables, forms and secondary text, 13 px minimum anywhere (captions, badges), 20/24 px page titles, 16 px section headings. Line height 1.5 for body, 1.25 for headings. Tabular numerals (`font-variant-numeric: tabular-nums`) on tables, class codes, times and counts. System fonts remain the fallback stack.

### 3.3 Icons

**Lucide** (`lucide-react`), 16 px inside text and buttons, 20 px in navigation and empty states, stroke width 1.75, `currentColor`, always `aria-hidden` next to a visible label. An icon-only button carries an `aria-label` and a native `title`. No emoji, no Unicode symbols as icons (the `✓` and `·` in use today are replaced), no second icon set.

### 3.4 Shape, depth, spacing

- Radius: 6 px for controls and badges, 8 px for cards, tables and dialogs; nothing larger, nothing fully rounded except role badges and avatars.
- Depth: borders, not shadows. The only shadows are the dialog and the theme menu. No gradients, no blur, no glass.
- Spacing on a 4 px grid; 8 px between related controls, 16 px inside cards, 24 px between page sections, 32 px page padding.
- Density is that of a tool, not a landing page: 40 px table rows, 36 px inputs and buttons, content left-aligned in a 72 rem container.

### 3.5 Motion

Colour and opacity transitions of 150 ms on hover and focus, nothing else animates. `prefers-reduced-motion: reduce` turns transitions off.

## 4. Theme mechanics

- The semantic tokens that exist today (`canvas`, `surface`, `line`, `line-strong`, `ink`, `ink-muted`, `accent`, `accent-hover`, `accent-soft`, `focus`, `danger`, `danger-hover`, `danger-soft`, `success`, `success-soft`) keep their names and gain `warning` and `warning-soft`. They are defined once as CSS custom properties; Tailwind utilities reference the variables, so a component written as `bg-surface text-ink` is correct in both modes with no `dark:` variant. Feature code never uses a `dark:` variant, a literal hex, `bg-white`, `text-white`, `text-black` or a Tailwind grey; `text-white` on a primary button becomes a token (`on-accent`).
- Dark mode is a second set of values for the same variables, applied when `<html data-theme="dark">`, and when `data-theme` is absent and `prefers-color-scheme: dark`. `color-scheme` is set on `html` for both modes so native controls, scrollbars and the `<dialog>` backdrop follow.
- The theme switch lives in the header, as a labelled control with three options, System, Light and Dark (Lucide `monitor`, `sun`, `moon`), and stores the choice in `localStorage` under `unicourse:theme`. A small inline script in `index.html` applies the stored choice before the first paint. A `theme.ts` module in `lib/` owns the read, write and apply logic and has a unit test; no React context is needed.
- Tests run in light mode; the one theme unit test covers the apply logic, and one end-to-end test switches to Dark, reloads, and asserts `data-theme="dark"` survives.

## 5. What "not AI-generated" means here

Checked by the reviewer against each page:

1. No gradients, glow, blur, glassmorphism, oversized rounded corners, or purple and indigo anywhere.
2. No decorative hero, no three-column feature cards with icons, no illustration placeholders, no marketing voice. The sign-in page is a form on a quiet canvas with the wordmark, not a split-screen pitch.
3. Hierarchy comes from size, weight and space, not from colour: one accent, used for actions and the current location only.
4. Every empty state names what is missing in the product's own words and offers the one action that fixes it, or none if the viewer cannot act.
5. Every table has a caption, aligned columns, tabular numbers, right-aligned numeric and action columns, and a row hover that is visible in both modes.
6. Every interactive element has hover, focus-visible and disabled states; links in running text are underlined.
7. Icons sit next to words. An icon-only control is a toolbar action with an accessible name, never a navigation item.
8. Copy stays the existing copy. The design document lists any wording it changes, and the orchestrator checks each against the test selectors before the lane starts.
9. Consistency over novelty: the same control looks the same on every page; a page introduces no one-off style.

## 6. Page inventory

Every route the app has; each is restyled, none is added or removed.

| Area | Pages | Notes for the design document |
|---|---|---|
| Entry | `/` (redirect), `/sign-in`, magic-link sent, `/invite/{token}` accept, "no access" | Centred card on canvas with the serif wordmark; provider buttons with Lucide icons; the divider between OIDC and email done with tokens, not `✓`/`·`. |
| Shell | header, main navigation, university switch, theme switch, sign-out, skip link | Header on surface with a hairline; wordmark in serif; nav items with icons; the current item marked by weight and the accent, not only colour. |
| My sections | `/{slug}/me` list and schedule, term picker | The week grid keeps its structure (`region` per weekday, clash lines); blocks get surface, hairline, tabular times; the clash state keeps text first, border second, in both modes. |
| Catalog | `/{slug}/catalog` | Filters as one row of labelled controls; results table per §5.5. |
| Courses | `/{slug}/courses`, `/{slug}/courses/sections`, `/{slug}/sections/{id}` detail with staff editor, roster, meetings editor, course / section / import / remove dialogs | Dialog header, body and footer rhythm shared by every dialog; import preview rows use the warning and danger badges from §3.1; staff editor rows align with the table rules. |
| Admin | `/{slug}/admin` tabs: departments, terms, members, invitations, audit log | Tabs as an underlined tab list; audit log rows monospace-free, tabular timestamps. |
| Shared | button, badge, dialog, empty state, error banner, form field, load more, loading, page header, status message, table | Each restyled once; pages consume them. Loading becomes a skeleton row set or an inline spinner with the same `role="status"`. |

## 7. Accessibility

WCAG 2.1 AA in both modes: 4.5:1 for text, 3:1 for input borders, focus rings and icons that carry meaning, visible focus on every control (2 px ring in the focus token with 2 px offset), 44 px targets on touch widths, no information by colour alone (the clash state, badges and status messages all carry text), `prefers-reduced-motion` honoured, `prefers-color-scheme` honoured by default. Existing landmarks, roles, names and the skip link stay as they are.

## 8. Constraints and dependencies

- Allowed new packages, and no others: `lucide-react` (runtime), `@fontsource-variable/source-sans-3` and `@fontsource-variable/source-serif-4` (runtime, bundled by Vite). D3 is amended for these three (D50).
- Tailwind stays the styling layer; `index.css` is the single place tokens, fonts and the two base recipes (`ui-table`, skeleton) live. No new CSS files per feature.
- Routes, copy, roles, API calls and test identifiers do not change. The retired class assertions in `admin-layout.test.tsx` become `aria-current="page"` assertions.
- A unit test (`src/no-literal-colors.test.ts`) reads every `.tsx` and `.ts` file under `src/` except tests and `index.css` and fails on `#` hex colours, `bg-white`, `text-white`, `text-black`, Tailwind grey/slate/zinc/neutral/stone utilities and `dark:` variants.

## 9. Gates

`npm run lint`, `npm run typecheck`, `npm test`, `npm run build` green; `npm run e2e` green against the real API and database; the orchestrator's browser pass of every page in §6 in both modes with screenshots; contrast ratios of every token pair in the design document measured, not asserted.

## 10. Deferred

| Item | Trigger to pick it up |
|---|---|
| Theme stored on the user account | a user signs in from more than one device and asks for it |
| Mobile navigation (drawer) below 768 px | a student uses the schedule on a phone in testing |
| High-contrast theme | an accessibility request from a university |
| Print stylesheet for the schedule and roster | a staff member asks to print |
| Automated contrast test (axe) in CI | the token set changes a second time |
