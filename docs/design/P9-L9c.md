# P9-L9c - Phase 9 closing audit, group c: grading screens, states, dark mode, phone width

Lane P9-L9c. Re-scores screens 60-66 and 70-73 after Phase 9, plus the cross-cutting dark mode (l) and phone width (m) checks, against the rubric in `docs/design/P8-L2.md` §4 and the "before" in `docs/design/P8-L3c.md`. No product code. Legend for fixes: impact H/M/L, effort S/M/L.

## 0. Method

- Read: P8-L2 §0, §1, §3, §4 (i) (j) (k) (l) (m) and (g)/(h) where a screen uses them; Spec 5 §1, §3, §4.14-§4.18, §5.4, §5.5; P8-L3c; taste-skill `design-taste-frontend/SKILL.md` §4 and §9; lane logs P9-L5, P9-L5-review, P9-L1c (branch `lane/P9-L1c-shared-followups`).
- Captures: AFTER set `scratchpad/after-l6/`, cited by base name. Read (23): light-1440 of 60, 61, 62, 63, 64, 65, 66, 70, 71; light-390 of 60, 62, 63, 65, 66; dark-1440 of 60, 63, 65, 72, 73; dark-390 of 60; other groups' phone captures 44, 46, 52 light-390 (the three longest dialog and table pages).
- Code read only to name causes: `index.css`, `components/table.tsx`, `loading.tsx`, `badge.tsx`, `features/grades/gradebook.tsx` and `finalize-dialog.tsx` on `lane/P9-L6-entry-search-separators` (the AFTER build; it contains L1c, and its `index.css` tokens equal the worktree's).
- Measured: `sips -g pixelWidth` on every `*-390.png` in both sets; pixel colours sampled from the PNGs with a stdlib decoder (`scratchpad/l9c/px.py`); contrast with the P6-L1 §1.3 functions (`lum`, `ratio`) on the `index.css` hex values (`scratchpad/l9c/measure.py`, `python3 -I`).
- Not observable in a still (kept from the before, marked "by design"): cell saving/saved, keyboard movement, focus return, skeleton suppression under 1 s, live-region announcements.

## 1. Measurements

### 1.1 Phone width (S5-68)

| Set | `*-390.png` files | Width 390 | Wider |
|---|---|---|---|
| BEFORE | 67 | 46 | 21 files: 21, 23 (755); 43, 44 (693); 45, 46, 47 (555); 52 (561); 53 (679); 60, 61, 62, 63 (542) |
| AFTER | 71 (adds 72, 73) | **71** | none |

The P8-L3c note said 550 for 60-63; the BEFORE files measure 542.

### 1.2 Contrast (S5-66, S5-67), after tokens from `index.css`

| Pair | Hex | Ratio | Minimum |
|---|---|---|---|
| dark `ink-muted` on `surface` | #a6a8aa on #2d2f31 | **5.63** | 4.5 |
| dark `ink-muted` on `canvas` | #a6a8aa on #1c1d1f | 7.07 | 4.5 |
| dark `line-control` on `surface` | #888a8b on #2d2f31 | **3.88** | 3.0 |
| dark `line-control` on `canvas` | #888a8b on #1c1d1f | 4.86 | 3.0 |
| light `line-control` on `surface` | #1c1d1f on #ffffff | 16.87 | 3.0 |
| dark `ink-muted` on `accent-soft` | #a6a8aa on #453057 | 4.85 | 4.5 |
| dark `success` on `success-soft` (72) | #8fd4b8 on #1c342e | 7.77 | 4.5 |
| dark `warning` on `warning-soft` (73) | #ddbc93 on #42301b | 7.00 | 4.5 |
| light `success` on `success-soft` | #1d6b50 on #e8f0ee | 5.54 | 4.5 |
| light `warning` on `warning-soft` | #865213 on #f4e8db | 5.39 | 4.5 |

All pass. Dark `ink-muted` and `line-strong` equal the P6-L1 derivations (`mix(mist, dsurf, .6)` = #a6a8aa, `.45` = #888a8b); `line-control` reuses #888a8b in dark. Sampled 60-dark-1440 at a secondary button edge: #888a8b, so the token is what renders.

## 2. Per screen

### 2.1 Gradebook (60, 73)

| (i) criterion | Before | After | Reason |
|---|---|---|---|
| 1 frozen names, header title/max/weight | 4 | 5 | Names now read "Amara Okafor" over a muted email (C1 via P9-L8); header stack title, "100 pts · 40%", Published, right-aligned to the input |
| 2 cell states | 3 | 4 | Empty published cell shows a muted "Missing" caption (60 Lucas Moreau, 73 every Extra Credit cell); saving/saved by design, not captured |
| 3 out-of-range inline | 4 | 4 | By design, not captured |
| 4 weight total warns with text | 4 | 5 | 73: "Weights total 110%" plus warning Badge "Must total 100% to finalize", readable in dark (7.00:1) |
| 5 finalize disabled-with-reason, preview, confirm | 4 | 5 | 73: Finalize disabled with "Assessment weights must total 100%." under it; 62 preview |
| 6 keyboard | 5 | 5 | By design |
| 7 phone | 1 | 4 | 60-390 is 390 wide; Student capped and truncated ("amara.okafor@northfield..."), Midterm scores beside it, table scrolls in its own box. Loses one point: the "Letters:" text wraps inside the footer cell and its last count drops to a second line ("... B- 2" then "1") and is clipped |

S5 items:
- S5-52 sticky Student cap: **delivered** (60-390, 60-dark-390).
- S5-53 Missing caption and "(1 missing)": **delivered** ("81.75% (1 missing)" in muted text).
- S5-54 header right-aligned, Student narrow: **partly**. Header stacks sit on the input's right edge; at 1440 the Student column ends at x about 363 and Midterm's right edge is at 657, so the space now sits inside the score columns (L1c: not fixable without fixed layout).
- S5-55 footer one tone, zero letters dropped: **delivered**. Sampled 60-light Mean row #ffffff at x 80 and 700; 60-dark #2d2f31 at both; "Letters: A 1 · B 1 · B- 2 ..." has no zero counts.
- S5-69 / C1 names: **delivered** (names over emails).
- S5-70 separators: **delivered** in the description ("CS101 · Section 001 · Introduction...").

Remaining findings:
1. **Stub hairline under the last student** (60, 63, 64, 73, both themes): a 290 px line under the Student cell only (60-light pixel (200,822) #d1d7dc, (700,822) #ffffff). Cause: `index.css` `.ui-table tbody tr:last-child td { border-bottom: 0; }` skips the row-header `th`. Fix in `index.css`: change the selector to `.ui-table tbody tr:last-child > *`, and add `.ui-table tfoot tr:first-child > * { border-top: 1px solid var(--color-line); }` so the footer keeps one full-width rule. Serves (i)1 and taste §9.C (no stray lines). M/S. Every table with a `th scope="row"` gains it.
2. **Letters tally inside a table cell** (60/63-390 clipped, 1440 floats mid-row): move it out of `tfoot` into a muted `<p className="text-sm text-ink-muted">` under the `Table`, rendered as `flex flex-wrap gap-x-3` spans ("A 1", "B 1", ...) with no middle dots. Serves (i)7, H8 and taste §9.F (dot rationing). M/S.
3. **Row height jumps** where "Missing" shows (Lucas row 70 px against 49 px; 73 has five 71 px rows): put the caption beside the input instead of under it (`flex items-center justify-end gap-2`, caption first, `text-xs text-ink-muted`), keeping 48 px rows. Serves (i)2 and taste §9.C. L/S.

### 2.2 Assessment dialog (61)

| (h) criterion | Before | After | Reason |
|---|---|---|---|
| 1 rhythm, Cancel first | 5 | 5 | Delete left, Cancel, Save changes |
| 2 label above, error below, sized fields | 5 | 5 | Max points and Weight about 128 px, Due date about 192 px; helpers under each |
| 3 destructive confirm | 4 | 4 | Delete separate and red text; confirm step by design, not captured |
| 5 focus | 4 | 4 | Ink ring on Name at open; return by design |
| 6 phone | n/s | n/s | 61-light-390 not opened; it measures 390 wide |

S5 items: S5-58 **delivered** (widths and "Other assessments total 60%.").

Remaining findings: none above L. The Due date still shows the native "yyyy-mm-dd"; the helper "Optional." covers meaning (L, no change proposed).

### 2.3 Finalize preview (62)

| Criterion | Before | After | Reason |
|---|---|---|---|
| (h)1 rhythm | 5 | 5 | Consequence sentence, summary, table, Cancel then "Finalize 8 grades" |
| (i)5 preview and confirm | 4 | 5 | Lucas Moreau sorted first with warning Badge "1 missing"; "Keep F" select replaces the Letter column; "Showing 8 of 8. Preview only..." under a bordered scroll area |
| (h)6 phone | 1 | **3** | Dialog fits 390 and the footer is reachable, but the Grade select is cut at the dialog edge ("Keep" only), Reason is off-screen, the Student column is not sticky inside the dialog, the "1 missing" Badge wraps to two lines, and the title breaks the code ("CS101-" / "001") |

S5 items: S5-59 **delivered**; S5-52 inside the dialog **partly** (cap and truncate present, `sticky` absent, so it scrolls away).

Remaining findings:
1. **Phone table** (H, lifts (h)6): `finalize-dialog.tsx` passes `stack` to `<Table>` and gives each cell `data-label` (Student on the row `th`, Percent, Grade, Reason). Below 640 px each student becomes a block with the select full width; above 640 px nothing changes. Land after P9-L9b F1 (stack CSS moved to `@layer utilities`, label-value grid), or the first cell loses its bold head. Remove `capName` from the dialog then (stack mode needs no cap). M/S.
2. **Two-tone Student column** (1440 and 390): sampled 62-light-1440 (310,330) #f7f9fa against (600,330) #ffffff. Cause: `.ui-table th` sets `background: var(--color-canvas)` for every `th`, including body row headers. Fix in `index.css`: move the background into `.ui-table thead th { background: var(--color-canvas); }` and leave body `th` on the row's surface. The gradebook's explicit `bg-surface` then becomes redundant but harmless. Serves H4 and (l)2. M/S.
3. Title: wrap the section code in `<span className="whitespace-nowrap">` so "CS101-001" never breaks. L/S.

### 2.4 Gradebook finalized and TA (63, 64)

| (i) criterion | Before | After | Reason |
|---|---|---|---|
| 1 headers | 4 | 4 | Instructor 5; TA header now three lines but "Published" is still an `accent` Badge (purple as status) and the TA header row is 66 px against 80 px |
| 2 cell states | 2 | 4 | "Finalized" Badge by the description, "Running %" and "Running letter" headers, the note says scores no longer change final grades, Missing visible |
| 5 finalize/reopen | 4 | 5 | "Reopen grades" in the primary slot (secondary style), Final "B 3.00 Change" |
| 7 phone | 1 | 4 | 63-390 is 390 wide with scores beside names; rows are 70 px against 49 px in 60-390, likely the off-screen Final cell wrapping (inference, the column is not visible) |

S5 items: S5-56 **delivered** (Badge is `neutral`, not `success`: `Badge` has no success tone, and D87/B11 says neutral for states); S5-57 **delivered** (TA note "Score changes do not change them until an instructor reopens grades."; three-line header); B11 on 64 **not visible** (P9-L7 not landed).

Remaining findings:
1. TA "Published": `Badge tone="neutral"` (B11, already approved, P9-L7). L/S.
2. Final cell: `whitespace-nowrap` on the Final `td` so phone rows stay 48 px; check 63-390 after. L/S.

### 2.5 My grades (65)

| (j) criterion | Before | After | Reason |
|---|---|---|---|
| 1 GPA and credits labelled | 3 | 4 | "Graded credits 8.0" with "Counts finalized sections only. 11.0 credits in progress."; card at content width. Term GPA stays in the term heading, not beside Cumulative |
| 2 final or running labelled | 5 | 5 | "B / 3.00 grade points · 83.00%" against "Current D+ / 69.20% so far" |
| 3 assessments expandable | 4 | 4 | "Published scores (2)" disclosure with a Lucide chevron; contents not captured |
| 4 hidden grades say why | 2 | **2** | Nothing says some work is not yet published (C7 deferred) |
| 5 number emphasis | 4 | 4 | Large bold letter, muted points and percent; tabular figures unavailable (L1c: DM Sans has no `tnum`) |
| 6 phone | 5 | 5 | Cards stack, nothing truncates, 390 wide |

S5 items: S5-60 **delivered**; S5-61 **partly** (card bottoms align, "Published scores" rows at the same y in a row; summary at content width; Term GPA not beside Cumulative, P9-L5 left it in the heading because e2e reads it there); S5-71 **delivered** (Lucide chevrons on both disclosures); C7 **not built**.

Remaining findings:
1. **Hidden work** (H, lifts (j)4 to 4 without the API): on running cards add a muted line under "% so far": "Counts published scores only." (`text-sm text-ink-muted`). With C7 later, replace it with "{n} assessments not yet published". M/S.

### 2.6 Grade scale (66, 72)

| Criterion | Before | After | Reason |
|---|---|---|---|
| (g)1 title, primary in a fixed spot | 2 | 4 | "Save scale" in the header action slot; presets and Add row as a toolbar; no row count |
| (g)2 aligned, numbers right | 2 | 4 | Letter about 80 px, Minimum percent and Grade points about 112 px right-aligned, Range column "90 to 92.99"; columns still spread over 1296 px (300 px gap after Letter) |
| (g)3 row actions | 3 | 4 | One quiet "Remove" with Lucide `Trash2`, muted (B6 delivered by L1c); per-row accessible name not visible in a still |
| (b)1 current location | 1 | 5 | "Admin" underlined in the header (66, 72-dark) |
| (g)7 phone | n/s | **3** | 390 wide, but Grade points, Range and Remove are off-screen inside the table's scroll box and nothing is frozen; 4 fields plus an action is the "cards" case of (m)4 |

S5 items: S5-62 **delivered**; S5-63 **partly** (Save placement delivered; "Unsaved changes" not in any capture, and per P9-L5-review M2 it renders through `StatusMessage`, which is green with a check, so an unsaved draft reads as success); B6 **delivered**; S5-06 (current location) **delivered**.

Remaining findings:
1. **Phone table** (H, lifts (g)7 and (m)4): `scale-editor.tsx` passes `stack` to `<Table>`, `data-label` on Letter, Minimum percent, Grade points and Range cells, Remove last (full width by the stack CSS). Same dependency on P9-L9b F1 as 2.3 #1. M/S.
2. **Unsaved draft in green** (M, H19/Spec 3 status meaning): replace the `StatusMessage` for "Unsaved changes" with `<p role="status" className="text-sm text-ink-muted">`; clear the "Grade scale saved" notice on the next edit so both never show. M/S.
3. 72: the success line repeats the description ("Final grades already recorded keep their letters."); shorten to "Grade scale saved." H8. L/S. Copy: the e2e reads status "Grade scale saved" (P9-L5-review), which still matches.

### 2.7 Loading and error states (70, 71, 72, 73)

| (k) criterion | Before | After | Reason |
|---|---|---|---|
| 1 skeleton shaped like the result | 2 | 5 | 70: real header row "Code, Title, Department" over 5 rows of bars per column; header and filter render first |
| 2 error plain with retry | 3 | 5 | 71: "Couldn't load sections" / "Something went wrong on our side. Try again in a moment." / Try again |
| 3 empty copy | 4 | 4 | Not re-read (12 is group a) |
| 4 filtered-empty distinct | 4 | 4 | Not re-read (32 is group a) |
| 5 never together | 5 | 5 | One state each in 70, 71 |
| 6 announced once | 4 | 4 | By design (`role="status"`, `role="alert"`) |

S5 items: S5-64 **delivered** (table variant on 70); S5-65 **delivered** (71); note c 1.7 #5 **delivered** ("New course" enabled during loading, 70); S5-67 **delivered** (72 success and 73 warning captured in both themes).

Remaining findings: none in my scope. Seen in 71 (owned by group b): the Term select shows "Academic Year 2026 (Jan 1, 2026", then an en dash (U+2013) separator and "D", cut by the 288 px select (`features/courses/section-list.tsx:255`). Fix: a wider select (`sm:w-auto`) so the range is not cut. The en dash in ranges is P9-L9a's owner question Q1 (recommendation keep; the e2e asserts it in times), so this audit does not propose changing it. L/S.

## 3. Dark mode (l)

Captures: 60, 63, 65, 72, 73 dark-1440, 60 dark-390.

| (l) criterion | Before | After | Reason |
|---|---|---|---|
| 1 text 4.5, borders 3.0 | 4 (by eye) | 5 | Measured: `ink-muted` on `surface` 5.63, `line-control` on `surface` 3.88 (§1.2) |
| 2 elevation by surface steps | 5 | 5 | Canvas #1c1d1f, table and cards #2d2f31; footer now one tone (sampled) |
| 3 links legible | 5 | 5 | Assessment names, back links, disclosure links in the lilac tint |
| 4 no pure black, no light island | 5 | 5 | None in six captures |
| 5 status colours | 4 | 5 | Now observable: 72 success 7.77:1, 73 warning Badge 7.00:1, both soft backgrounds distinct from surface |
| 6 badges adapt | 4 | 5 | Neutral "Finalized" outlined (63), warning Badge (73), accent-soft current theme button |

App-wide: P9-L9a §7 (01, 20, 21, 30) and P9-L9b (41, 46, 52, 54 dark) report no criterion below 4, so the (l) row holds at the scores above for the whole app. S5-66 **delivered**: secondary buttons now draw a mid-grey #888a8b border, quieter than the lilac primary (60-dark, 73-dark toolbar). S5-55 **delivered** in dark. Remaining: the 2.1 #1 stub line and 2.3 #2 row-header background apply in dark too (same CSS).

## 4. Phone width (m)

| (m) criterion | Before | After | Reason |
|---|---|---|---|
| 1 no horizontal page scroll | 1 | 5 | 71 of 71 phone captures are 390 wide (§1.1) |
| 2 targets 44 px | 3 | n/s | `@media (pointer: coarse)` raises buttons, inputs and nav links to 2.75 rem; the captures run with a fine pointer, so buttons show 40 px and the rule cannot be seen (same call as P9-L9a §0) |
| 3 nav and search one tap | 4 | 5 | Header is two rows plus search, ends at y about 157 (60-390) against 205 before |
| 4 tables collapse by rule | 1 | **3** | Stacked rows on 44, 46, 52 (cards with label-value lines); gradebook scrolls with a frozen name column. Grade scale (66) and the Finalize table (62) neither stack nor freeze |
| 5 dialogs fit | 2 | 4 | 44, 46, 62 fit with margins and a reachable footer; 62's Grade select is clipped |
| 6 type readable | 5 | 5 | Nothing below 13 px seen |

Spec 5 §1 verdict on width: **met**. No 390 capture, light or dark, is wider than 390 (dialogs have no dark-390 by design).

Seen in other groups' phones (for P9-L9b): 46's stacked import preview labels rows with the raw CSV keys "email" and "displayName" and right-aligns "Line 2" under its label; 46 meeting times use an en dash (U+2013) between start and end ("10:00", "11:20"). Not scored here.

## 5. Taste-skill pass (group c)

§9 AI tells:
- 9.A no pure black, no glow, no gradient text: pass in every capture.
- 9.B H1 32 px bold, hierarchy by weight: pass.
- 9.C "no awkward gaps": fails mildly in three places: the gradebook stub line (2.1 #1), row-height jumps under "Missing" (2.1 #3), the 300 px gap after Letter on the grade scale (accepted, table width 100%).
- 9.D realistic data: pass since C1 (Amara Okafor, Mei-Ling Chou); the "-muyfdo34" email suffix is the capture seed, not product copy.
- 9.F middle-dot rationing (one per line): the Finalize summary line carries 8 dots and the Letters tally 6. S5-70 chose " · " to replace em dashes in labels; the tallies are the overuse. Fix in 2.1 #2 (spans with gaps); the Finalize summary uses the same spans after "Preview:". **Owner question (low):** keep " · " as the house separator for two-part labels (taste allows one per line) and ban it in counts lists, yes or no?
- 9.G em and en dashes: none visible in group c captures; en dashes remain in ranges (71 term, 46 times), covered by P9-L9a Q1.

§4 bias checks:
- 4.2 one accent: purple only on actions, links, current location, checkboxes; one exception left, the TA "Published" Badge (B11 pending).
- 4.4 one radius, cards only for grouped data: holds (DIR items unchanged).
- 4.5 states: skeleton, error, empty, success, warning all distinct; one duplicate intent in 72 (success line restates the description, 2.6 #3) and one success-styled non-success (2.6 #2).
- 4.6 label above, helper below: holds (61).
- Typography D56: `index.css` sets `font-variant-numeric: tabular-nums` on `html`, but L1c found DM Sans has no `tnum`, so the declaration does nothing. Delete it (L/S) or keep it as a no-op for a future font; worker's call is not needed, delete per D56.

## 6. Still below 4

| Screen | Criteria below 4 | The single fix that lifts each |
|---|---|---|
| 60 Gradebook | none | (polish: 2.1 #1 stub line) |
| 61 Assessment dialog | none (phone n/s) | |
| 62 Finalize preview | (h)6 phone 3 | `<Table stack>` with `data-label` cells in `finalize-dialog.tsx` (2.3 #1) |
| 63, 64 Finalized, TA | none | |
| 65 My grades | (j)4 hidden grades 2 | "Counts published scores only." under each running percent; C7 later (2.5 #1) |
| 66 Grade scale | (g)7 phone 3 | `<Table stack>` with `data-label` cells in `scale-editor.tsx` (2.6 #1) |
| 70-73 States | none | |
| Dark mode (l) | none | |
| Phone width (m) | (m)4 tables collapse 3 | the two `stack` fixes above (62, 66) |

Ranked remaining work for a follow-up lane (all inside Spec 3 tokens and components, no copy that e2e asserts except where noted): 2.6 #1, 2.3 #1, 2.5 #1, 2.3 #2 and 2.1 #1 (one `index.css` change each, every table), 2.6 #2, 2.1 #2, 2.1 #3, 2.4 #1 (P9-L7), 2.3 #3, 2.4 #2, 2.6 #3, §5 `tabular-nums` deletion.
