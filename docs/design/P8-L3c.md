# P8-L3c - Benchmark: grading screens, loading and error states, dark mode, phone width

Lane P8-L3c, Phase 8 (D75-D78). Compares our captures against the reference pack (P8-L2) and the redesign-skill audit (D78). No product code. Feeds P8-L3d.

Legend for improvements: impact H/M/L, effort S/M/L, category A (inside Spec 3 tokens and components), B (owner direction question), C (product or API gap).

## 0. Method

- Read: CLAUDE.md UI conventions; Spec 3 §3, §4, §5, §7, §10; Spec 4 §6; P7-L2 §0 and §3; pack §1, §2 (i)-(m), §3, §4 (i) (j) (k) (l) (m) and (h) for dialogs, §5; redesign-skill Design Audit (Interactivity and States) and Fix Priority. Glanced at `components/loading.tsx`, `table.tsx`, `error-banner.tsx`, `empty-state.tsx` and the sticky classes in `features/grades/gradebook.tsx` to name causes.
- Images read (21): light-1440 of 60, 61, 62, 63, 64, 65, 66, 70, 71; dark-1440 of 01, 20, 30, 45, 52, 60; light-390 of 01, 20, 21, 30, 45, 52, 60, 65, plus 62-light-390; light-1440 of 12 and 32 for empty states.
- Measured with `sips`: the pixel width of every light-390 capture (a full-page capture wider than 390 means the document scrolls sideways).
- Skipped: all Dribbble preview images (the pack's descriptions were enough), dark-390 captures, 10, 11, 13 empty states, 61/63/64/66 at 390. No contrast was measured with a tool; contrast remarks are by eye and flagged as such.
- DIR rows (H21 tabular figures, H23 Lucide, H24 DM Sans/600 weight, H25 purple, card border, radius, press motion) are not scored; they appear only in §4.

## 1. Grading screens and states

### 1.1 Gradebook (instructor, before finalizing)

Captures: 60-grades-gradebook-instructor-light-1440, -dark-1440, -light-390.

Scores (pack §4 (i)):
1. Frozen names, header title/max/weight - 4: headers show "Midterm Exam / 100 pts · 40% / Published"; student column is sticky; no due date shown in the capture (none set).
2. Cell edit/saving/saved/error states - 3: built per P7-L2 §3.2 (Saving…, Saved, error text, sr-only live region), but no cell state visible in the capture; an empty cell (lucas.moreau, Final Exam) looks identical to an unscored-by-design cell.
3. Out-of-range flagged inline by outline plus text - 4: `aria-invalid` border plus message under the input (by design; not captured).
4. Weight total not 100 warns with text - 4: "Weights total 100%" plain text; warning Badge "Must total 100% to finalize" exists by design (not captured).
5. Finalize states why disabled, previews, confirms - 4: muted reason line by design; preview dialog in 62.
6. Keyboard between cells - 5: Enter/arrows/Escape and paste per P7-L2 §3.2.
7. Phone: horizontal scroll with frozen name column, no breakage - 1: at 390 the sticky Student column fills the whole table width, so no score, Percent or Letter column is visible, and the document is 550 px wide (page scrolls sideways).

Works:
- Header cell carries title, max points, weight and publish state in one stack: richer than the Revelski gradebook shot (10832693), which shows neither max nor weight.
- Running Percent right-aligned and Letter beside it, server-computed; stats footer with mean and median.
- One primary action (Finalize grades) top right; the three secondary tools grouped on the toolbar row; the explanatory note under it states the missing-score rule in plain words.
- Dark mode holds: inputs, checkboxes and the lilac primary read cleanly (60-dark).

Weaker than references:
- Google Classroom (pack (i)) colours and labels a missing cell; ours shows a blank input, and lucas.moreau reads "81.75% B-" while the Finalize preview (62) turns the same student into "32.70% F". The running percent hides the drop until finalize.
- Revelski 10832693 rows are about 26 px; ours are 48 px with a two-line name block where both lines are the same email (people have no display name), so 8 students fill the screen.
- Canvas (pack (i)) and Gaurav Mathur 11064516 keep the number column tight to its header; ours centres each assessment header over a wide column while the input sits right, leaving a 400 px gap after Student at 1440.
- Vass 9226089 (pack (m)): frozen first column is narrow and the value columns scroll beside it; ours freezes a column wider than the phone.
- Footer: the sticky Mean/Median header cell uses `bg-canvas` while the rest of the footer is surface, a visible two-tone strip in both themes; "Letters: A 1 · A- 0 · B+ 0 ..." lists every zero-count letter (H8 noise).

Improvements:
1. Sticky student cell (`gradebook.tsx` `th scope="row"` and the header `th`): cap at `max-w-48` with `truncate` on the name line below the `sm` breakpoint (full text in `title`), so score columns show beside it at 390. H/S/A.
2. Student cell: when the person has no display name, render the email once (bold) and drop the muted second line; same in the Finalize dialog. H/S/A. Real names need the API: H/M/C.
3. Missing score cell: render a muted "Missing" caption under an empty input when the assessment is published, and add "(1 missing)" after Percent as the Finalize dialog already does, so the 81.75% vs 32.70% surprise is visible before finalize. H/S/A.
4. Assessment header: right-align the header stack (`text-right`) to the input edge and let Student take `w-auto`; the column then sits next to its numbers. M/S/A.
5. Footer: give the sticky footer `th` `bg-surface` like body rows; in "Letters:" omit letters with count 0. M/S/A.
6. Page description: replace the em dash in "Section 001 (em dash) Introduction..." with a comma (H22; same pattern in Section detail 45). L/S/A.

### 1.2 Assessment dialog

Captures: 61-grades-assessment-dialog-instructor-light-1440.

Scores (pack §4 (h), relevant rows):
1. Header/body/footer rhythm, Cancel first - 5: title, four fields, footer "Delete assessment" left, Cancel, Save changes.
2. Label above, error below, no placeholder-as-label, Save not disabled - 5: helper text under Max points, Weight, Due date.
3. Destructive confirm - 4: Delete swaps the body to a confirm with count of scores lost (by design); entry point is a ghost-danger text button, separated left.
5. Focus trap and return - 4: focus ring visible on Name at open (ink ring); trap per `Dialog`.
6. Phone fit - not judged (390 capture skipped).

Works:
- Helper copy explains Max points and Weight in domain words, including the "must total 100" rule (H2, H5).
- Destructive action is visually separate from Save and never purple (pack §3 item 6).
- Dialog width about 450 px suits four short fields.

Weaker than references:
- Ben Low 7486471 and P7-L2 §3.4 size numeric fields to their content; ours stretch Max points and Weight to full width (the design asked `w-32`), so "100" sits in a 400 px box.
- The due date shows the native "yyyy-mm-dd" placeholder; Spec 3 "Optional." helper covers meaning, but the field gives no example date.

Improvements:
1. Max points and Weight inputs: apply `w-32` (Due date `w-48`) as P7-L2 §3.4 specified; check the class is not overridden by `inputClass` order. M/S/A.
2. Weight helper: append the current remainder, e.g. "Other assessments total 60%." so the author can see what 100 needs. M/S/A.

### 1.3 Finalize preview dialog

Captures: 62-grades-finalize-preview-instructor-light-1440, -light-390.

Scores (pack §4 (i) 5 and (h)):
- Finalize previews letters and confirms - 4: summary "8 students · 1 with missing scores · Preview: A 1 · B 1 ...", per-row Percent/Letter/Grade, verb button "Finalize 8 grades".
- Rhythm, Cancel first - 5.
- Phone fit - 1: at 390 only the Student column shows; Percent, Letter, Grade and Reason are hidden behind it; the backdrop covers only the 390 px viewport while the page is 550 px wide.

Works:
- The consequence sentence names both effects (missing counts as 0, every assessment gets published) before the table (pack §3 item 12).
- "(1 missing)" next to 32.70% is the only place the missing score is visible.
- Count in the confirm label ("Finalize 8 grades"), as Primer ConfirmationDialog recommends.

Weaker than references:
- Primer ConfirmationDialog keeps body short; ours scrolls the table inside the dialog with the last row cut mid-height and no sign that more rows follow.
- "Letter" and "Computed: B" repeat the same value per row; Google Classroom's draft/return list carries one value per student.
- The Student cell has a white sticky background against a grey row band (two-tone row), same cause as 1.1 item 5.

Improvements:
1. Grade select: change the default option label from "Computed: {letter}" to "Keep {letter}" and drop the separate Letter column (the select shows it). M/S/A. Copy change: check e2e selectors first.
2. Rows with missing scores: sort them first or add `Badge tone="warning"` "1 missing" in the Percent cell so they are reviewed before confirm. M/S/A.
3. Same sticky-width fix as 1.1 item 1 inside the dialog table, so Grade and Reason are reachable at 390. H/S/A.
4. Scroll area: add a bottom border on the scroll container and a "Showing 8 of 8" muted line under it, so a clipped last row reads as scrollable. L/S/A.

### 1.4 Gradebook finalized (instructor) and TA view

Captures: 63-grades-gradebook-finalized-instructor-light-1440, 64-grades-gradebook-ta-light-1440.

Scores (pack §4 (i), deltas from 1.1):
- 1 headers - 4 (instructor), 4 (TA): TA header is shorter (no ghost-button name, Published as Badge), so header height differs between roles.
- 2 cell states - 2: after finalizing, score inputs look exactly as before although the note says changes do not alter final grades.
- 5 finalize/reopen - 4: "Reopen grades" secondary replaces Finalize; Final column "B 3.00 Change".

Works:
- Final column pairs a bold letter with muted grade points (pack (j) 5 emphasis pattern).
- The note line swaps to "Final grades are recorded. Changing a score now does not change them; reopen grades to recompute."
- TA sees Import/Export only, no New assessment, no Change: capabilities are legible from the toolbar.

Weaker than references:
- Canvas and Google Classroom (pack (i), §3 item 12) show posted/returned as a visible state of the gradebook; ours has no "Finalized" marker near the title, only a sentence below the toolbar.
- Running "Letter" (B-) and "Final" (F) for lucas.moreau sit side by side with no explanation; Canvas separates Current and Total with labels.
- Eight "Change" ghost buttons in accent per row repeat purple down the column (pack (g) 3 asks at most one inline action; this is that one, but the label could carry the student).
- TA note says "reopen grades to recompute", an action the TA cannot take.
- TA "Published" Badge uses the accent tone: purple as status, outside Spec 3 §3.1 (actions, links, current location, band).

Improvements:
1. Page header when finalized: add `Badge tone="success"` "Finalized" next to the description (or in `PageHeader` action group beside Reopen). M/S/A.
2. Finalized table: rename the column headers to "Running %" and "Running letter" while Final exists, so the two letters read as different things. M/S/A. Copy change: check e2e.
3. TA note when finalized: "Final grades are recorded. Score changes do not change them until an instructor reopens grades." L/S/A.
4. TA publish Badge: `tone="neutral"` for both Published and Draft (text carries the state), removing purple-as-status. L/S/A.
5. Equalise header height: the TA header stack uses the same line structure as the instructor (name line, points line, state line). L/S/A.

### 1.5 My grades (student)

Captures: 65-grades-my-grades-student-light-1440, -light-390.

Scores (pack §4 (j)):
1. Term and cumulative GPA at top with credits labelled - 3: "Cumulative GPA 1.85 / Credits 8.0" top card; "Credits 8.0" counts only finalized sections (5 sections, 19 credits enrolled) with no label saying so.
2. Final letter or running percent labelled - 5: "B / 3.00 grade points · 83.00%" vs "Current D+ / 69.20% so far".
3. Assessments expandable with points/max/weight - 4: "Published scores (2)" disclosure per card (contents not captured).
4. Hidden grades say why - 2: a section with unpublished assessments shows only the published count; nothing says scores are still to come.
5. Number emphasis - 4: large bold final letter, muted grade points and percent; running letter is 16 px bold, so final vs running differ by size.
6. Phone - 5: cards stack, nothing truncates, document width 390.

Works:
- The summary card plus per-term heading "Term GPA 1.85 · 8.0 credits" matches Canvas's single-total-on-top pattern.
- "so far" and "Current" label running grades in words (H19).
- "How letters and grade points are assigned" disclosure answers the scale question in place (Canvas weights table equivalent).

Weaker than references:
- Canvas (pack (j)) shows Current and Total and the weights per section; our card shows no weight breakdown until the disclosure opens, and no count of unpublished work.
- Google Classroom (pack (j)) explains a hidden total; ours leaves "Credits 8.0" unexplained next to 19 enrolled credits.
- Final-letter cards are 8 px taller than running cards, so card bottoms in a row do not align (pack (c) 2 rule).
- The summary card spans 1296 px for two numbers.
- Disclosures use the native triangle marker; the rest of the app uses Lucide (Spec 3 §3.3 bans Unicode symbols as icons; the native marker is borderline).

Improvements:
1. Summary card: label "Graded credits" and add a muted line "Counts finalized sections only. 11.0 credits in progress." M/S/A (in-progress total is derivable client-side from the cards; if not, C).
2. Running cards: add muted "{n} assessments not yet published" under the percent when the section has unpublished assessments. M/M/C (needs the count from `/me/grades`).
3. Card body: give the letter line a fixed two-line block (`min-h`) so final and running cards align. L/S/A.
4. Summary card: constrain to content width (`inline-flex`, not full width) and add Term GPA for the current term beside Cumulative. L/S/A.
5. Disclosures: replace the native marker with Lucide `ChevronRight` rotating to `ChevronDown` via `[open]` (colour only motion rule: swap icons, no rotation transition). L/S/A.

### 1.6 Grade scale editor (admin)

Captures: 66-grades-grade-scale-admin-light-1440.

Scores (pack §4 (g) and (h), relevant rows):
- (g)1 title, count, primary in fixed spot - 2: presets top left, "Add row" under the table, "Save scale" alone at the bottom right below the fold on short screens.
- (g)2 columns aligned, numbers right-aligned - 2: Minimum percent and Grade points are left-aligned in 333 px wide inputs.
- (g)3 row actions - 3: one inline "Remove" per row, but 12 red labels form a red column.
- (h)2 labels, errors, Save never silently disabled - 4: by design client validation before PUT.
- Shell current location (H20) - 1: no header nav item is marked on this page.

Works:
- Description states the consequence for already-recorded grades.
- Two presets load real data in one click.
- Table is a plain editable grid, no modal per row.

Weaker than references:
- Gaurav Mathur 11064516 keeps cell inputs at content width with numbers right-aligned; ours are three equal 333 px boxes for 1-4 character values.
- Pack §3 item 5 (one explicit Save near the form) and Primer saving: our Save sits 800 px below the presets with no unsaved-changes indicator.
- No derived range shown ("93-100", "90-92.99"), which Canvas-style scales display so gaps or overlaps are visible.

Improvements:
1. Inputs: Letter `w-20`, Minimum percent and Grade points `w-28 text-right`; Remove column right-aligned. M/S/A.
2. Add a read-only "Range" column computed from the next row's minimum ("90 to 92.99"). M/S/A.
3. Move "Save scale" (primary) into the `PageHeader` action slot with presets and Add row as secondaries in the toolbar; show `StatusMessage` "Unsaved changes" when the draft differs. M/S/A.
4. Remove: `Button variant="ghost"` (neutral) with Lucide `Trash2` and `aria-label="Remove {letter}"`, keeping ghost-danger for confirmation steps only. L/S/A.
5. Header nav: mark "Admin" current on `/{slug}/grade-scale` for admins, or add the Grade scale admin tab (P7-L2 E5 follow-up). M/S/A.

### 1.7 Loading, error and empty states

Captures: 70-states-loading-skeleton-admin-light-1440, 71-states-error-banner-admin-light-1440, 12-courses-empty-admin-light-1440, 32-catalog-empty-result-student-light-1440.

Scores (pack §4 (k)):
1. Skeleton shaped like the result - 2: three generic grey bars (`Loading` renders 3/4, full, 5/6 widths) under a page whose result is a table.
2. Error copy plain with retry - 3: "Something went wrong / Something went wrong on our side. Try again in a moment." plus "Try again"; title and body repeat.
3. Empty copy states what is missing, action echoes it - 4: "No courses yet" + "New course"; but "New course" also sits in the header, so two identical primaries are on screen.
4. Filtered-empty distinct from first-use - 4: "No sections match / Try another search or department." with no create action; no Clear filters and no echo of the query.
5. Never shown together - 5: 70 and 71 show one state each; shell, tabs, filters and actions stay (Mavenlink 4032645 pattern).
6. Announced once - 4: `role="status"` on Loading, `role="alert"` on ErrorBanner.

Works:
- Shell, tabs, filter and header actions render first; only the content region waits or fails (Mavenlink 4032645).
- Error banner uses the danger-soft surface, Lucide `CircleAlert`, and a retry button; no codes, no "Oops".
- Empty-state copy follows Stripe's "No customers yet." form and the action is Verb + Noun (Geist).

Weaker than references:
- Gopinathan 4361464: real column headers over bars sized per column; ours shows no header row and bars of arbitrary width.
- Primer empty/error guidance: no "There was a problem" phrasing; ours says "Something went wrong" twice.
- Stripe filtered-empty: plain message plus Clear filters; Geist: quote the query. Ours does neither.
- Hashemi 17206944 keeps table headers over an empty body; our empty state replaces the table.
- During loading, "New course" is disabled (pale lilac) with no reason; H5 asks disabled-with-reason or enabled.

Improvements:
1. `Loading`: add a `variant="table"` prop rendering the real `Table` header row (caller passes column names) over 5 rows of `ui-skeleton` bars sized per column; `variant="cards"` rendering 3 `Card`-shaped blocks for Catalog and My sections. Keep the default for small regions. H/M/A (new pattern in `components/` with a test and a line in the design doc).
2. `ErrorBanner` default: title "Couldn't load {thing}" passed by each caller (e.g. "Couldn't load sections"), body kept; drop the default title duplication. M/S/A. Copy change: check e2e.
3. Catalog filtered-empty: title `No sections match "{q}"` when a query is set; action `Button variant="secondary"` "Clear filters" resetting department and query. M/S/A.
4. First-use empty with a header primary: omit the EmptyState action when the same action is in `PageHeader` (or the reverse), so one primary is visible. L/S/A.
5. Header "New course" during loading: keep it enabled (the dialog does not need the list). L/S/A.

## 2. Dark mode across the app

Captures: 01, 20, 30, 45, 52, 60 dark-1440.

Scores (pack §4 (l)):
1. Text 4.5:1, borders/focus 3:1 - 4 (by eye): body and muted text read well; the sign-in "or" divider lines and table row dividers are faint but decorative; input borders visible. Spec 3 §1.3 script measured tokens; not re-measured here.
2. Elevation through surface steps - 5: canvas about #1C1D1F, cards, header and table one step lighter, dialogs per Spec 3; no shadows needed (Shutler 15266296, Linear pattern).
3. Purple links legible with lighter tint - 5: back links, "Change role", assessment names in the lighter lilac; primary buttons lilac fill with charcoal text.
4. No pure black, no section staying light - 5: none seen; native selects and date input follow `color-scheme`.
5. Status colours keep meaning and contrast - 4: danger "Remove" salmon on surface reads; no success or warning instance captured in dark.
6. Images and badges adapt - 4: cover bands become a muted dark purple with mist code text; "Admin" badge accent-soft with light text; neutral "Member"/"Active"/"Student" outlined.

What holds:
- One token set, no visible light islands; the gradebook inputs, checkboxes and lilac Finalize button carry over unchanged (60-dark).
- Cover band in dark is quiet enough not to read as decoration; card body one step lighter than canvas (20, 30).
- Header current item (accent underline) visible on dark (20, 30, 52).

What breaks or is weaker:
- The gradebook sticky Mean/Median cell is darker than the footer row (`bg-canvas` vs surface), more visible in dark than light (60-dark).
- Secondary buttons have a 1 px mist border at near-full contrast; on dark it draws louder than the lilac primary next to it (60-dark toolbar, 20 "Schedule"). Linear/Vercel (pack (l), product knowledge) use a mid-grey border on dark.
- "List"/"Schedule" toggle: the selected option is a lilac primary fill, the same look as a primary action (20-dark); in light the same holds. A segmented control with accent-soft selected state would mark current location without competing with actions.
- Status "Active" on Members is a neutral outline badge; fine, but success has no dark instance in any capture to verify.

Improvements:
1. Gradebook footer sticky `th`: `bg-surface` (same as 1.1 item 5). M/S/A.
2. Secondary button border on dark: map to `line-strong` instead of ink in the dark token set only (token change in `index.css`, measured at 3:1 per Spec 3 §7). M/S/A.
3. List/Schedule toggle: render selected as `bg-accent-soft text-ink` with `aria-pressed`, unselected ghost; one shared `SegmentedControl` in `components/` (also usable for the theme switch look). M/M/A.
4. Capture one success and one warning state in dark (StatusMessage, weight warning Badge) in the P8-L1 capture project so (l) 5 can be scored. L/S/A.
5. Run the Spec 3 §1.3 contrast script on the dark `ink-muted` over `surface` and the dark row hover, since the sign-in helper text looks close to the floor by eye. L/S/A.

## 3. Phone width across the app

Captures: 01, 20, 21, 30, 45, 52, 60, 65 light-390, 62 light-390; widths of every light-390 capture.

Document width at 390 (full-page capture width): 390 for 01, 20, 22, 30, 31, 32, 40-42, 50, 51, 54, 64, 65, 66, 70, 71; wider for 21 and 23 (755), 43 and 44 (693), 45-47 (555), 52 (561), 53 (679), 60-63 (550). Eleven of 37 phone captures scroll sideways.

Scores (pack §4 (m)):
1. No horizontal page scroll - 1: eleven screens overflow (list above); the white header stops at 390 while the canvas extends.
2. Targets 44 px - 3: buttons 40 px; nav links are bare text about 20 px tall (20, 30); theme switch segments about 32 px.
3. Nav and search within one tap - 4: nav links, theme, Sign out and search all visible without a menu, at the cost of a 205 px three-row header.
4. Tables collapse by rule - 1: Members (4 fields + actions) and Roster scroll with actions off-screen and no frozen column; gradebook's frozen column covers everything (60, 62).
5. Dialogs fit - 2: Finalize dialog fits the viewport but its table shows one column; backdrop does not cover the overflowed area (62-390).
6. Type readable - 5: nothing below 13 px seen.

What holds:
- Sign-in, My sections cards, Catalog cards and My grades cards stack to one column with no overflow (01, 20, 30, 65), matching the Honeycutt 15436288 card collapse.
- Header search is reachable on every page without a menu.
- Filters stack vertically with labels above (30).

What breaks:
- Week grid (21): a squeezed grid with Mon-Wed visible, Thu/Fri off-page, block text truncated to "Introductio..."; pack (d) 5 and §3 item 11 ask for an agenda list under a day strip (Tsang 8051313, Shough 25829121).
- Gradebook (60, 62): sticky Student column wider than the phone (email names), so scores are unreachable; Vass 9226089 freezes a narrow name column with values beside it.
- Section detail (45): the "Add by email" row does not wrap; label, input (62 px wide) and helper text are crushed into a 60 px column beside "Add instructor" / "Add TA", which also pushes the page to 555 px. Roster's Email column clips.
- Members (52): Role and Departments partly visible, Status and actions off-screen; Admin tabs overflow ("Audit log" off-screen) with no "More" (Primer UnderlineNav).
- Header: three rows (wordmark+university, nav, theme+Sign out, then search) take 205 px of 844; Spec 3 §10 defers the drawer, so this is scored as reachable, not as a failure.
- Cover bands stay 96 px tall per card at 390, so two cards per screen (20, 30); Honeycutt cards lead with the key value.

Improvements:
1. Find and fix the overflow source per screen: every flex row with fixed-width children gets `flex-wrap` and `min-w-0` (section detail add-staff row: stack input full width above the two buttons below `sm`); add an e2e or Playwright check that `document.documentElement.scrollWidth <= 390` on every 390 capture. H/M/A.
2. Gradebook sticky column width cap (1.1 item 1). H/S/A.
3. Week grid below 768 px: render the existing meetings as an agenda list grouped by weekday (weekday as `h3`, time stacked over code and room, same left accent stripe), switching on a media query; grid unchanged above. H/M/A.
4. Members and Roster below 640 px: rows become stacked `Card`-like list items (email bold, role Badge top right, label-value lines, actions full width at the foot), per pack §3 item 11. M/M/A.
5. Admin `TabNav` overflow: allow horizontal scroll inside the tab strip with `overflow-x-auto` and `scroll-snap`, keeping the page at 390; "More" menu later. M/S/A.
6. Header nav links: `min-h-11` (44 px) and inline-flex centring below `md`. M/S/A.
7. Card cover band: reduce to 56 px below `sm`. L/S/B (band size is a D57/Spec 3 card anatomy call).
8. Mobile navigation (drawer or bottom tabs) to shrink the header: deferred by Spec 3 §10. M/M/B.

## 4. Direction questions for the owner (B items)

- Tabular figures (D56, H21): gradebook, Finalize preview, grade scale and My grades align decimals poorly with proportional digits (60, 62, 63, 65, 66). Accept, or allow a numerals-only fallback?
- Mobile navigation (Spec 3 §10 deferral): the 205 px three-row header at 390 on every page (20, 30, 45, 52, 60, 65). Drawer, bottom tabs, or keep?
- Cover band height on phone (D57 card anatomy): 96 px bands on 20, 30 at 390.
- Purple as status: should "Published" and "Admin" badges use accent-soft (64, 52), or is accent strictly actions/links/location (Spec 3 §3.1 reading)? Recommendation A item 1.4.4 assumes strict.
- Press motion and transitions (RD scale(0.98), 200-300 ms vs Spec 3 §3.5 150 ms colour only): affects every button; report only.
- Lucide (H23), DM Sans and 600 weight (H24), bordered cards and 4 px radius: report only, all screens.
- Gradebook draft/publish model (pack §5 q6): our per-assessment publish plus section finalize did not confuse in the captures; the only confusion is running vs final letter (1.4), solvable in A.

## 5. Top 10 for this group (impact, then effort)

1. Gradebook and Finalize dialog sticky Student column: width cap and truncate so scores show at 390 (1.1.1, 1.3.3). H/S/A.
2. Show the email once when there is no display name, gradebook and Finalize (1.1.2); real names via API (C). H/S/A.
3. Flag missing scores in the gradebook ("Missing" under the empty cell, "(n missing)" after Percent) (1.1.3). H/S/A.
4. Fix horizontal page overflow on eleven phone screens, starting with section detail's add-staff row, plus a scrollWidth check in the capture run (3.1). H/M/A.
5. Week grid agenda list below 768 px (3.3). H/M/A.
6. Table-shaped and card-shaped `Loading` variants (1.7.1). H/M/A.
7. "Finalized" badge plus "Running %"/"Running letter" headers after finalize (1.4.1, 1.4.2). M/S/A.
8. Catalog filtered-empty quotes the query and offers Clear filters; ErrorBanner titles name what failed (1.7.3, 1.7.2). M/S/A.
9. Grade scale: content-width right-aligned inputs, Range column, Save in the header with unsaved indicator, Admin marked current (1.6.1-1.6.3, 1.6.5). M/S/A.
10. My grades: "Graded credits" label with in-progress credits, aligned card bottoms (1.5.1, 1.5.3). M/S/A.
