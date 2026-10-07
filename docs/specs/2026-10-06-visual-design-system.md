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

One coolors.co trending palette, used as is, chosen by the owner on 2026-10-06: **Olive Garden Feast**, `https://coolors.co/palette/606c38-283618-fefae0-dda15e-bc6c25`.

| Swatch | Hex | Light mode role | Dark mode role |
|---|---|---|---|
| Dark moss green | `#606C38` | accent: primary buttons (white text, 5.7:1), links, active navigation, focus ring | surface tint for the raised table header; secondary accent |
| Pakistan green | `#283618` | ink for body text and headings (13.5:1 on white) | canvas: page background |
| Cornsilk | `#FEFAE0` | canvas: page background | ink: body text (12.2:1 on the dark canvas) |
| Earth yellow | `#DDA15E` | warning and highlight backgrounds with dark text only (2.3:1 on white, never as text) | accent: links, active navigation, primary button fill with dark text (5.7:1 on the dark canvas) |
| Tiger's eye | `#BC6C25` | accent hover on buttons; badge backgrounds with dark text; never body text on white (4.0:1) | accent hover; borders that must carry meaning (3.3:1 on the dark canvas) |

Derived shades are allowed only where the five swatches cannot meet §7, and each one is written down in the design document with the swatch it is derived from and its measured ratio: a light-mode muted text (a Pakistan green tint, ≥ 6:1 on white and canvas), hairlines and a strong line for input borders (Pakistan green at low alpha over the surface, the strong one ≥ 3:1), an accent-soft background for active navigation and badges (Dark moss green at low alpha over the surface), a dark-mode surface one step lighter than the canvas, a dark-mode muted text and lines (Cornsilk at reduced alpha). Pure white `#FFFFFF` is the light-mode surface; there is no pure black anywhere.

Status colours are not brand colours and stay out of the palette: danger (red), success (a cooler green that reads apart from the olive accent), warning (built on Earth yellow), each as a text shade for its mode and a soft background, each ≥ 4.5:1 on the surface it sits on. They are used only to mean error, success or caution, never for decoration.

### 3.2 Typography

Two open-source families from Google Fonts, chosen by the owner on 2026-10-06, self-hosted through fontsource packages so the app makes no request to Google at runtime:

- **DM Serif Display** for the wordmark and page titles (`h1`) only. It ships in one weight (400) with an italic; use the regular only. Tight tracking, sizes from 24 px up so its high contrast stays legible.
- **DM Sans** (variable) for everything else: body, tables, forms, buttons, section headings (`h2`, `h3`). Weights 400, 500 and 600 only; optical size left to the font.

Scale: 16 px body (`1rem`), 14 px for tables, forms and secondary text, 13 px minimum anywhere (captions, badges), 24 px page titles, 16 px section headings. Line height 1.5 for body, 1.2 for titles. Tabular numerals (`font-variant-numeric: tabular-nums`) on tables, class codes, times and counts. System fonts remain the fallback stack.

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
- The theme switch lives in the header, as a labelled control with three options, System, Light and Dark (Lucide `monitor`, `sun`, `moon`), and stores the choice in `localStorage` under `unicourse:theme`. A small same-origin script file (`/theme-boot.js`, loaded blocking in the head; the API's content security policy forbids inline scripts) applies the stored choice before the first paint. A `theme.ts` module in `lib/` owns the read, write and apply logic and has a unit test; no React context is needed.
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

- Allowed new packages, and no others: `lucide-react` (runtime), `@fontsource-variable/dm-sans` and `@fontsource/dm-serif-display` (runtime, bundled by Vite). D3 is amended for these three (D50).
- Tailwind stays the styling layer; `index.css` is the single place tokens, fonts and the two base recipes (`ui-table`, skeleton) live. No new CSS files per feature.
- Routes, copy, roles, API calls and test identifiers do not change. The retired class assertions in `admin-layout.test.tsx` become `aria-current="page"` assertions.
- A unit test (`src/no-literal-colors.test.ts`) reads every `.tsx` and `.ts` file under `src/` except tests and `index.css` and fails on `#` hex colours, `bg-white`, `text-white`, `text-black`, arbitrary named colours such as `bg-[white]`, Tailwind palette utilities (grey, slate, zinc, neutral, stone and the hue scales), colour functions and `dark:` variants, stacked or not.

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
