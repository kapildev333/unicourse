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

Chosen by the owner on 2026-10-07: the look of Udemy's web app, which supersedes the coolors.co constraint (D57). The palette is Udemy's public brand colours, named here by role; the five swatches are used as is and everything else is derived and measured in the design document.

| Swatch | Hex | Light mode role | Dark mode role |
|---|---|---|---|
| Purple | `#A435F0` | accent: primary buttons (white text, 4.8:1), links at rest (4.6:1 on the canvas), active states | a lighter tint (derived) for links; the primary button is that tint with charcoal text, since no one purple is both a readable link and a fill white text passes on (design §0) |
| Deep purple | `#5624D0` | hover on links and buttons | hover on links (as a lighter tint, derived) |
| Charcoal | `#1C1D1F` | ink: body text and headings (16.5:1 on white) | canvas: page background |
| Grey | `#6A6F73` | muted text (5.1:1 on white), placeholder text | muted text as a lighter tint (derived) |
| Mist | `#F7F9FA` | canvas: page background; table header; card hover | ink: body text on the dark canvas |

Surfaces are white in light mode; the dark surface is a step above the dark canvas (derived, about `#2D2F31`). Hairlines are Udemy's `#D1D7DC` in light mode and a derived tint in dark. Derived shades are allowed only where these cannot meet §7 and each is written down with its source and measured ratio: a dark-mode link purple (a tint of Deep purple at or above 4.5:1 on the dark surface), accent-soft backgrounds for active navigation and badges (Purple at low alpha over the surface), hover shades. Purple is the one accent and appears only on actions, links, the current location and the card's cover band (a pale tint standing in for Udemy's cover image), never elsewhere as decoration; this is how the look stays deliberate. The focus ring is the ink colour in both modes (a purple ring on a purple button measures 1:1).

Status colours stay outside the brand: danger (red), success (green), warning (amber), each as a text shade for its mode and a soft background, each ≥ 4.5:1 on the surface it sits on, used only for error, success or caution.

### 3.2 Typography

One open-source family from Google Fonts, chosen by the owner on 2026-10-07 as the stand-in for Udemy Sans: **DM Sans** (variable), self-hosted through its fontsource package so the app makes no request to Google at runtime. No serif anywhere.

- Weights 400 (body), 500 (labels, nav) and 700 (page titles, card titles, buttons, section headings, table headers). Page titles are bold and large; this is the one place the type is loud.
- Scale: 16 px body (`1rem`), 14 px for tables, forms and secondary text, 13 px minimum anywhere, 32 px page titles (`h1`, weight 700, line height 1.2, tracking -0.01em), 20 px section headings (`h2`), 16 px card titles (700). Line height 1.5 for body.
- DM Sans has no tabular figures (D56); numbers set proportionally.

### 3.3 Icons

**Lucide** (`lucide-react`), 16 px inside text and buttons, 20 px in navigation and empty states, stroke width 1.75, `currentColor`, always `aria-hidden` next to a visible label. An icon-only button carries an `aria-label` and a native `title`. No emoji, no Unicode symbols as icons (the `✓` and `·` in use today are replaced), no second icon set.

### 3.4 Shape, depth, spacing

- Radius: 4 px for controls, cards, tables and dialogs; badges 4 px; nothing larger, nothing fully rounded except avatars. Primary and secondary buttons are 40 px tall (48 px for the sign-in submit), bold, with a 1 px charcoal border on the secondary.
- Depth: 1 px borders in the hairline token. Cards get a 1 px border and, on hover, a soft shadow (the only hover shadow). The dialog has a shadow. No gradients, no blur, no glass.
- Spacing on a 4 px grid; 8 px between related controls, 16 px inside cards, 24 px between page sections, 32 px page padding.
- Density: 48 px table rows, 40 px inputs, content left-aligned in a 84 rem container with 24 px gutters; card grids of 3 columns at 1024 px and up, 2 at 768 px, 1 below.

### 3.5 Motion

Colour and opacity transitions of 150 ms on hover and focus, nothing else animates. `prefers-reduced-motion: reduce` turns transitions off.

## 4. Theme mechanics

- The semantic tokens that exist today (`canvas`, `surface`, `line`, `line-strong`, `ink`, `ink-muted`, `accent`, `accent-hover`, `accent-soft`, `focus`, `danger`, `danger-hover`, `danger-soft`, `success`, `success-soft`) keep their names and gain `warning` and `warning-soft`. They are defined once as CSS custom properties; Tailwind utilities reference the variables, so a component written as `bg-surface text-ink` is correct in both modes with no `dark:` variant. Feature code never uses a `dark:` variant, a literal hex, `bg-white`, `text-white`, `text-black` or a Tailwind grey; `text-white` on a primary button becomes a token (`on-accent`).
- Dark mode is a second set of values for the same variables, applied when `<html data-theme="dark">`, and when `data-theme` is absent and `prefers-color-scheme: dark`. `color-scheme` is set on `html` for both modes so native controls, scrollbars and the `<dialog>` backdrop follow.
- The theme switch lives in the header, as a labelled control with three options, System, Light and Dark (Lucide `monitor`, `sun`, `moon`), and stores the choice in `localStorage` under `unicourse:theme`. A small same-origin script file (`/theme-boot.js`, loaded blocking in the head; the API's content security policy forbids inline scripts) applies the stored choice before the first paint. A `theme.ts` module in `lib/` owns the read, write and apply logic and has a unit test; no React context is needed.
- Tests run in light mode; the one theme unit test covers the apply logic, and one end-to-end test switches to Dark, reloads, and asserts `data-theme="dark"` survives.

## 5. What "not AI-generated" means here

Checked by the reviewer against each page:

1. No gradients, glow, blur, glassmorphism or oversized rounded corners. Purple is the single accent and sits only on actions, links, the current location and the card cover band; never on other backgrounds, never as decoration.
2. No decorative hero, no feature cards with icons, no illustration placeholders, no marketing voice. The sign-in page is a form on a quiet canvas with the wordmark, not a split-screen pitch. Cards show data the product has (code, title, instructors, meetings); a card's cover block is a flat colour band with the course code, never a stock image.
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
| Entry | `/` (redirect), `/sign-in`, magic-link sent, `/invite/{token}` accept, "no access" | Centred white card on the canvas with the bold wordmark; 48 px bold purple submit; provider buttons full width with a charcoal border and Lucide icons; the divider done with tokens. |
| Shell | header, main navigation, search, university switch, theme switch, sign-out, skip link | Udemy's top bar: white, 72 px, hairline below; bold wordmark; nav items as text links with the current one bold and underlined in purple; a search field in the middle (rounded, `line-strong` border for 3:1, Lucide `search`, named and placeholdered "Find a course") that submits to the Catalog with `q`; theme switch, university switch and sign-out on the right; the signed-in email lives in the sign-out button's tooltip, not in the bar. |
| My sections | `/{slug}/me` list and schedule, term picker | The list becomes a card grid: cover band with the class code, bold title, instructor line, meeting lines, role badge. The schedule keeps its week grid (regions per weekday, clash lines) restyled with the tokens. |
| Catalog | `/{slug}/catalog` | Filters as one row of labelled controls above a card grid with the same card as My sections plus the clash note; "Showing n" and load-more below. The `q` search param arrives from the header search. |
| Courses | `/{slug}/courses`, `/{slug}/courses/sections`, `/{slug}/sections/{id}` detail with staff editor, roster, meetings editor, course / section / import / remove dialogs | Admin tables keep rows (48 px, bold header text, hairlines); section detail gets a title band (code, title, term, instructors) above tabbed-looking sections for staff, meetings and roster; dialogs share one header/body/footer rhythm; import preview rows use the warning and danger badges. |
| Admin | `/{slug}/admin` tabs: departments, terms, members, invitations, audit log | Tabs as a bold underlined tab list; tables as above; audit timestamps in muted text. |
| Shared | button, badge, dialog, empty state, error banner, form field, load more, loading, page header, status message, table, card (new) | Each restyled once; pages consume them. Loading is a skeleton. The card is the one new shared component. |

## 7. Accessibility

WCAG 2.1 AA in both modes: 4.5:1 for text, 3:1 for input borders, focus rings and icons that carry meaning, visible focus on every control (2 px ring in the focus token with 2 px offset), 44 px targets on touch widths, no information by colour alone (the clash state, badges and status messages all carry text), `prefers-reduced-motion` honoured, `prefers-color-scheme` honoured by default. Existing landmarks, roles, names and the skip link stay as they are.

## 8. Constraints and dependencies

- Allowed new packages, and no others: `lucide-react` and `@fontsource-variable/dm-sans` (runtime, bundled by Vite). D3 is amended for these two (D50).
- Tailwind stays the styling layer; `index.css` is the single place tokens, fonts and the two base recipes (`ui-table`, skeleton) live. No new CSS files per feature.
- Routes, copy, roles and API calls do not change. The header search is the one new behaviour: it navigates to `/{slug}/catalog?q=` with the typed text, which the Catalog already filters by. Where a page's pattern changes from table rows to cards (Catalog, My sections list), the end-to-end suite may swap selectors (`row`/`cell` to `listitem`/text) for the same assertions, and the page's own unit tests may do the same, and nothing else (D60, D66). The retired class assertions in `admin-layout.test.tsx` become `aria-current="page"` assertions.
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
