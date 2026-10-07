# P9-L9b - Phase 9 closing audit, group b: Courses and Sections lists, dialogs, section detail, admin

Lane P9-L9b. Screens 12, 13, 40-47, 50-54 re-scored after Phase 9 against the pack rubric (`docs/design/P8-L2.md` §4) and the taste-skill rubric (P8-L2 §0, §1, §3; taste-skill §4 and §9). "Before" is `docs/design/P8-L3b.md` as tabled in Spec 5 §3. No product code.

## 0. Method

- **AFTER set:** `scratchpad/after-l6/` (the P9-L6 branch, which carries P9-L1c, P9-L2 to P9-L5, P9-L8 and Phase 10's section tabs). **BEFORE set:** `scratchpad/before/`, used only to confirm widths.
- **Read:** light-1440 for all 15 bases. Light-390 for 13, 40, 43, 44, 45, 46 (dialog area), 52, 53, 54. Dark-1440 for 41, 46, 52, 54. That is 24 images.
- **Widths (`sips -g pixelWidth`):** all 25 AFTER 390 captures in this group are 390 px wide, light and dark. The BEFORE widths were 693 for 43 and 44, 555 for 45, 46 and 47, 561 for 52, and 679 for 53. The width criterion of Spec 5 §1 holds for the whole group.
- **Source checks:** I read the source on `lane/P9-L6-entry-search-separators`, read only, to find the cause of two findings: `index.css` (the `ui-table-stack` block) and the `font-medium` and `data-label` usages in `features/courses` and `features/admin`. I also grepped the e2e suite for the en dash and the admin headings.
- **Scoring:** the scale is the pack's §4: 5 matches the row, 3 is present but inconsistent, 1 is absent. Each score is the lower of the themes and widths I saw. "inferred" means the capture was not opened, but its width is 390 and the shared component is shown working on a sibling screen. "n/s" means the criterion cannot be seen in a still image. "-" means it does not apply.
- **Artifacts, not findings:** in tall dialog captures (45 to 47, 54) the backdrop dims only one viewport, and in 46-dark-1440 the band above the dialog is left undimmed. Both come from taking a full-page screenshot.

## 1. Screens

### 1.1 First-use empty: Courses (12) and Sections (13) · rubric (k)

| Criterion | Before | After 12 | After 13 | Reason |
|---|---|---|---|---|
| 1 Skeleton | n/s | n/s | n/s | not in a still |
| 3 Empty copy, action echoes it | 3 | 5 | 4 | 12: "No courses yet" with one primary "New course" and a secondary "Import courses", and the header pair is hidden. 13: "No terms yet" with "Set up terms", but the body tells an admin that "An administrator creates terms under Admin → Terms", and the disabled lilac "New section" competes with it |
| 4 Filtered-empty distinct | 4 | 4 | 4 | the filter is hidden while the list is empty, so first-use and filtered-empty cannot look alike; the filtered-empty view itself is not captured |
| 5 Never shown together | 5 | 5 | 5 | one state per page |
| (m) Phone | - | 5 | 5 | 390 px wide, actions wrap under the description (13-light-390) |

S5 items:

| Item | Status | What the capture shows |
|---|---|---|
| S5-14 "Set up terms" for admins | delivered | 13 shows a secondary button in the empty state at both widths |
| S5-15 New section disabled with reason, Import and Term hidden | delivered | 13 shows "New section" disabled with the muted "Add a term first" beside it. There is no Import button and no Term select |
| S5-16 No filter, one primary while empty | delivered | 12 has no Department select and no header buttons. "New course" is the primary and "Import courses" the secondary |
| S5-17 Title above tabs | delivered | 12, 13, 40, 43: the h1 "Courses" sits above the Courses and Sections TabNav |

Remaining:
1. **13 copy (L, (k)3).** Change the body to "Sections belong to a term. Add one in Admin, Terms, then come back." This removes the arrow glyph (Spec 3 §3.3) and the third person. The text is in `section-list.tsx:134` and no e2e test asserts it.
2. **13 disabled primary reads as a lilac button (L, (k)3).** With no terms, render the header "New section" as `secondary` and keep it disabled, so that "Set up terms" is the only thing that looks actionable. The e2e test at `staffing.spec.ts:247` only checks that the button is disabled, so it holds.

### 1.2 Courses list (40) · rubric (g)

| Criterion | Before | After | Reason |
|---|---|---|---|
| 1 Title, count, action | 3 | 4 | "Import courses" and "New course" sit on the description row, and "Showing 5" sits right-aligned on the filter row above the table. The count is not beside the title, but it is in the same place on every list |
| 2 Alignment, rows | 4 | 4 | 48 px rows, three text columns, no numbers |
| 3 Row actions | 4 | 4 | Edit is in accent and Delete is muted (B6). The ceiling is B8: inline verbs were kept |
| 5 Paging, position | 3 | 4 | the count shows above the table, and the list is short |
| 7 Phone | 1 | 4 | 390 px wide; each row stacks into a block with Edit and Delete at its foot (40-light-390). The anatomy is short of the pack's §3.11, see F1 |

S5 items: S5-33 is delivered (the Department select is 288 px). S5-34 is delivered. S5-35 is partly delivered: the rows stack, but the Code head is not bold and every field repeats its label. S5-70 is delivered in the dialog options (41).

### 1.3 Course dialog (41), Remove course (42), Remove from roster (47) · rubric (h)

| Criterion | Before 41 / 42-47 | After 41 | After 42 | After 47 | Reason |
|---|---|---|---|---|---|
| 1 Rhythm | 5 / 5 | 5 | 5 | 5 | bold title, body, hairline footer, Cancel first, plus the new close X at the top right |
| 2 Labels, errors | 4 / - | 4 | - | - | label above and helper below ("Up to 20 characters, e.g. CS101"); no error state captured |
| 3 Destructive | - / 4 | - | 5 | 5 | 42 states the API rule ("A course that still has sections can't be deleted: delete its sections first"). 47 names the person ("Remove Lucas Moreau from this section?") and the blast radius. Red is used only on the confirm button |
| 5 Focus | 5 / 5 | 5 | 5 | 5 | 41 opens with the focus ring on Department; native `<dialog>` |
| 6 Phone | 3 / 3 | 5 inferred | 5 inferred | 5 inferred | the 390 captures are 390 px wide. The same `Dialog` fits with margins and keeps its footer visible in 44-light-390 and 46-light-390 |

Dark (41-dark-1440): the dialog sits one surface step above the dimmed page, and the primary is the light tint with dark text.

S5 items: S5-37 is delivered, as seen in 44 and 46 at 390. S5-38 is delivered: an X sits on every dialog, in accent. S5-39 is delivered in 42. S5-69 and C1 are delivered in 47's title. S5-70 is delivered: "CS · Computer Science".

Remaining: F7 (the close X is a second purple in the dialog).

### 1.4 Sections list (43) · rubric (g)

| Criterion | Before | After | Reason |
|---|---|---|---|
| 1 Title, count, action | 3 | 4 | as 40 |
| 2 Alignment, rows | 4 | 4 | 56 px rows with the code over a muted title, and staff as plain text. The Term select cuts its own label ("Academic Year 2026 (Jan 1, 2026 – D") |
| 3 Row actions | 4 | 4 | as 40 |
| 5 Paging | 3 | 4 | as 40 |
| 7 Phone | 1 | 4 | 390 px wide (693 before); rows stack. The Course block shows the code link in regular weight and the title in bold, which inverts the hierarchy. Each block is about 278 px tall |

S5 items: S5-36 is delivered: plain `text-sm` emails and a dash for an empty TA list (the "+2" and "Invited" variants have no data in the capture). S5-33 is partly delivered: the select is sized, but its option text overflows (F4). S5-35 is partly delivered (F1).

### 1.5 Section dialog (44) · rubric (h)

| Criterion | Before | After | Reason |
|---|---|---|---|
| 1 Rhythm | 5 | 5 | same as the other dialogs, opened `wide` |
| 2 Labels, errors | 3 | 5 | Location shows "Engineering Building 215" in full at 1440. Every input has its label above |
| 5 Focus | 5 | 5 | focus ring on Location (the last field the capture filled) |
| 6 Phone | 1 | 5 | 390 px wide (693 before). Day is full width, Starts and Ends share a row, Location is full width, and Remove sits under them. The footer is visible with 16 px side margins |

S5 items: S5-40 is delivered at both widths. S5-37 is delivered. S5-70 is delivered ("CS210 · Data Structures and Algorithms").

Remaining: F5 (the meeting row's "Remove" is accent purple while every other row removal is muted).

### 1.6 Section detail (45) · rubric (f)

| Criterion | Before | After | Reason |
|---|---|---|---|
| 1 Title band | 4 | 4 | "CS101 · Section 001", then the title and term, then the instructor, all before the tabs. The instructor is still an email (F2) |
| 2 Tabs, counts | 2 | 4 | Phase 10 added Home, Announcements, Materials and Grades with an accent underline. The counts sit on the "Staff 2" and "Roster 8" headings, not on the tabs |
| 3 Tables follow (g) | 3 | 4 | the Roster shows names (P9-L8) over emails, 48 px rows and a muted Remove. Staff rows are emails |
| 4 Back link | 5 | 5 | "Sections" with an arrow in the band |
| 5 Edit beside what it edits | 3 | 5 | "Edit section" in the band, "Add description" beside About, staff add and remove inline, "Import roster" on the Roster heading row |
| 6 Phone | 1 | 4 | 390 px wide (555 before). The add-by-email input is full width with both buttons under it, the tabs fit, and the Roster stacks. Each roster block repeats "Name", "Email" and "Added" labels in regular weight (F1) |

S5 items: S5-42, S5-43, S5-44 and S5-45 are delivered. S5-35 is partly delivered: the Roster stacks without a bold head. S5-69 and C1 are partly delivered: students carry names, but staff and the band still show emails. S5-70 is delivered on the band line.

### 1.7 Roster CSV import (46) · rubric (h)

| Criterion | Before | After | Reason |
|---|---|---|---|
| 1 Rhythm | 5 | 5 | same as the other dialogs |
| 2 Labels, errors | 4 | 4 | "CSV file" label above; no error row captured |
| 4 Import preview | 4 | 5 | one `role="status"` sentence ("4 rows checked: 0 to add, 2 to invite, 2 already exist, 0 with errors"), per-row text badges, no counter row, no empty Message column, and a primary that says "Import 2 rows" |
| 5 Focus | 5 | 5 | native dialog |
| 6 Phone | 3 | 4 | 390 px wide (555 before). The dialog fits, the body scrolls and the footer stays visible. In the stacked preview, Line is the bold head but right-aligned, so the block opens with a number floating at the right edge |

Dark (46-dark-1440): the badges stay legible, the "Invite" warning tint reads on the dark surface, and the template link is the light tint.

S5 items: S5-41 is delivered. S5-37 is delivered. S5-35 is partly delivered (F1, the `text-right` on Line).

Remaining: F8 ("Invite" badge on `warning-soft`).

### 1.8 Departments (50), Terms (51) · rubric (g)

| Criterion | Before 50 / 51 | After 50 | After 51 | Reason |
|---|---|---|---|---|
| 1 Title, count, action | 3 / 3 | 4 | 4 | h1 with a description, the primary on the right, and "Showing n" on its own line above the table. The admin area puts the tabs above the h1, while Courses puts the h1 above the tabs (F3) |
| 2 Alignment, rows | 5 / 4 | 5 | 5 | 51 gains a Status column ("Current", "Upcoming") as neutral badges |
| 3 Row actions | 4 / 4 | 4 | 4 | Edit in accent, Delete muted (B6) |
| 5 Paging | 4 / 4 | 4 | 4 | short lists with the count shown |
| 7 Phone | 2 / 2 | 4 inferred | 4 inferred | 390 px wide, and the same stack mode as 52 |

S5 items: S5-46 is delivered. S5-34 is delivered through `ListCount`. S5-35 is delivered, inferred from width and the shared mode.

### 1.9 Members (52) · rubric (g)

| Criterion | Before | After | Reason |
|---|---|---|---|
| 1 Title, count, action | 2 | 4 | "Invite new people from Invitations." is now a link, and the count sits above the table. There is no primary on this page by design |
| 2 Alignment, rows | 3 | 4 | named people show the name over a muted email, and unnamed people show one email line at 48 px. In this data Departments is all dashes and Status is all "Active" |
| 3 Row actions | 4 | 4 | "Change role" in accent, "Remove" muted |
| 5 Paging | 3 | 4 | "Showing 12" |
| 7 Phone | 1 | 4 | 390 px wide (561 before). Rows stack with a bold head. The page is 3844 px tall for 12 people, because each block carries four labelled fields, two of them dashes (F1) |

Dark (52-dark-1440): the Admin badge sits on the dark `accent-soft`, the neutral badges are outlined, and the link tint and the muted Remove stay legible.

S5 items: S5-47 is delivered. S5-69 is delivered (no dash line, 48 px email-only rows). S5-35 is partly delivered (F1). C1 is partly delivered: students are named, staff and the admin are not (F2).

### 1.10 Invitations (53) · rubric (g)

| Criterion | Before | After | Reason |
|---|---|---|---|
| 1 Title, count, action | 3 | 4 | "Invite someone" primary, count above the table |
| 2 Alignment, rows | 4 | 4 | a muted dash in Expires on accepted rows. The two Pending rows sit at rows 5 and 6 of 14 (C9 waits) |
| 3 Row actions | 5 | 5 | Revoke only on pending rows, muted |
| 5 Paging | 3 | 4 | "Showing 14" |
| 7 Phone | 2 | 3 | 390 px wide (679 before), but 4589 px tall. Each block lists five labelled fields, and three of them (Departments, Expires, often Status "Accepted") carry nothing actionable. The two pending rows start about 1500 px down |

S5 items: S5-48 is delivered for the dash, and the order is moved to C9. S5-35 is partly delivered (F1).

### 1.11 Audit log (54) · rubric (g)

| Criterion | Before | After | Reason |
|---|---|---|---|
| 1 Title, count, action | 3 | 4 | the description says "newest first", and "Showing 50" sits above the table |
| 2 Alignment, rows | 4 | 4 | muted timestamps, 48 px rows |
| 3 Row actions | 4 | 4 | "Show changes" with a Lucide chevron |
| 5 Paging | 3 | 3 | "Showing 50" plus Load more, with no total (C6) |
| 6 Audit sentence | 3 | 3 | the actions read in plain past tense ("Score set", "Assessment created", "Invitation accepted", "Member joined", "Student enrolled") and the filters sit in one sized row. There is still no Target, so the 20 "Score set" and "Assessment created" rows cannot be told apart (C5) |
| 7 Phone | 2 | 4 | When is sticky and the table scrolls inside its container (54-light-390), and the page is 390 px wide. Only When and a two-line wrapped Who are visible, and Action is off screen (F6) |

Dark (54-dark-1440): the header row, the sticky When cell and the chevron links step correctly.

S5 items: S5-50 is delivered. S5-51 is delivered. S5-71 is delivered (the chevron replaces the triangle). S5-34 is delivered. C5 and C6 are not built (§7, not part of the success criterion).

## 2. Remaining findings, ranked

Each finding gives impact, effort, the rubric criterion it serves and a fix inside Spec 3's tokens and components.

**F1. Stacked rows are tall, label everything, and lose their bold head (H, S; (g)7, (f)6, (m)4).**
- **Symptoms:** 53 at 390 is 4589 px tall, and 52 is 3844 px. In 40 the Code head and in 45 the Name head render in regular weight. In 43 the code link is regular while the muted course title turns bold. In 46 the Line head is right-aligned.
- **Cause, from the source:**
  - The stack rules live in `@layer components` in `index.css`.
  - `course-list.tsx:134`, `roster.tsx:95` and the link at `section-list.tsx:187` carry `font-medium`. A utility always wins over the components layer, so these cells stay at 500 instead of 700.
  - `csv-import-dialog.tsx:221` keeps `text-right` on the Line cell.
- **Fix, all inside the existing stack media query in `index.css`:**
  1. Move the `@media (max-width: 639px)` stack block from `@layer components` to `@layer utilities`, so that `.ui-table-stack td:first-child` (higher specificity) wins over a single utility class.
  2. Bold the head's first line only: `.ui-table-stack td:first-child, .ui-table-stack td:first-child > :first-child { font-weight: 700 }` and `.ui-table-stack td:first-child > :not(:first-child) { font-weight: 400 }`.
  3. Give the head no caption: `.ui-table-stack td:first-child::before { content: none }`.
  4. Lay out the other fields as label-value rows: `.ui-table-stack td:not(:first-child):not(:last-child) { display: grid; grid-template-columns: 7rem minmax(0,1fr); gap: 0 0.75rem; }`, and make `::before` `display: inline` in that grid.
  5. Reset alignment: `.ui-table-stack td { text-align: left }`.
  6. Hide empty fields: add `.ui-table-stack td[data-empty] { display: none }`. In the features, set `data-empty` on cells whose value renders the dash: Departments on 52 and 53, Expires on accepted rows in 53, TAs on 43, Department when unknown on 40.
- **Expected result:** about 110 to 140 px per block instead of 280. Add a unit test in `components/` that renders a `Table stack` row with a `data-empty` cell and asserts the cell is present in the DOM (CSS hides it). Add a line in `docs/design/P6-L1.md`.
- **Lifts:** 53 (g)7 from 3 to 4, and moves 40, 43, 45, 52 toward 5.

**F2. Staff and admins still show as emails (H, S web plus the approved B2; (f)1, (g)2).**
- **Where:** the 45 band and Staff list, 43 Instructors and TAs, the first four Members rows in 52, and the Audit "Who" column.
- **Cause:** P9-L8 names people through import and OIDC sign-in, and `PATCH /api/v1/me` exists, but nothing in the web app calls it.
- **Fix:** this is D87 B2, the user menu after C1, already approved for P9-L7. The menu holds a "Your name" field that calls `PATCH /me`, Sign out, and the theme. Until it ships, the seed in `e2e/screens.ts` should give staff a display name, so the captures show the named state real OIDC users will see. That is an orchestrator call, because it is an e2e file under D80.

**F3. The two tabbed areas order title and tabs differently (M, S; (g)1, H4).**
- **What differs:** Courses renders the h1 above its TabNav (S5-17). Admin renders the TabNav above a per-tab h1 (50 to 54).
- **Fix:** the Admin layout route renders `<h1>Admin</h1>` above the TabNav. Each admin tab's `PageHeader` becomes the description and actions row, without an h1, the same change P9-L3 made for Courses. No e2e test asserts the admin headings (grep of `getByRole('heading'`). The admin unit tests change with it.

**F4. The Term filter cuts its own label (M, S; (g)2, H6).**
- **What happens:** at 288 px, "Academic Year 2026 (Jan 1, 2026 – Dec 31, 2026)" is cut in 43 at both widths, and the date range uses a spaced en dash.
- **Fix:** the option text becomes the term name alone (`section-list.tsx:255`). The selected term's dates show as the `FormField` hint under the select ("Jan 1, 2026 to Dec 31, 2026"). No e2e test selects by this label.

**F5. The meeting row's "Remove" is purple (M, S; (h)1, H4, D87 B6).**
- **What happens:** 44 shows "Remove" in accent next to a meeting row. Every other row removal is now muted.
- **Fix:** `meetings-editor.tsx` uses `variant="ghost-danger"` for Remove.

**F6. On phone, the Audit log hides the Action column (M, S; (g)6, (g)7).**
- **What happens:** at 390, only When and a wrapped Who fit.
- **Fix:** use the `Table stack` mode for the Audit log as well. It has four fields, and the pack's (m)4 asks for cards up to five. The head is When, muted, over "Who · Action", with "Show changes" at the foot. This reverses S5-51, so it is an orchestrator decision; see the report. A smaller alternative keeps the scroll and puts Action second (When, Action, Who), at the cost of the desktop actor-action sentence order.

**F7. The dialog close X is a second purple (L, S; (h)1).**
- **What happens:** the X uses `ghost` (accent) in every dialog, beside the purple primary.
- **Fix:** the X uses `ghost-neutral`, so the verb button stays the dialog's only accent.

**F8. The import preview's "Invite" badge uses the caution colour (L, S; H19, Spec 3 status meaning).**
- **What happens:** "Invite" is an outcome, not a warning.
- **Fix:** use the neutral `Badge` tone, as "Exists" already does. Pending on 53 keeps warning, because the invitation can expire.

**F9. Copy (L, S; H2).**
- 13's empty body: see §1.1 item 1.
- "Catalog courses by department." on 40 reads as a noun phrase or as an instruction. Change it to "Courses in the catalog, by department." Nothing in e2e asserts it.

## 3. Taste-skill pass for the group

| Check | Result | Evidence |
|---|---|---|
| 9.A No glow, no pure black, no gradient text | pass | dark captures 41, 46, 52, 54 are charcoal, not black; no gradients |
| 9.A Saturation of the accent | DIR | the purple is the owner's pick (D57, H25) |
| 9.B No oversized h1 | pass | h1 is 32 px on every screen, and the band title on 45 is the largest |
| 9.B No serif, no Inter | pass | DM Sans only |
| 9.C Even spacing, no floating gaps | pass with note | admin "Showing n" sits alone on a line above the table, while on 40 and 43 it shares the filter row. F3 would bring the areas closer |
| 9.D Realistic names and data | pass | "Amara Okafor", "Northfield University", "Hall B 204"; no Acme or John Doe |
| 9.E Icon set | DIR | Lucide (D53, H23) |
| 9.F Middle dot at most one per line | pass | "CS101 · Section 001", "Introduction to Computer Science · Academic Year 2026" and "Monday 10:00–11:20 · Hall B 204" each carry one |
| 9.F No decorative status dots | pass | statuses are text badges |
| 9.F No section-number eyebrows or step labels | pass | none |
| 9.F Hairline under every row of a long list | DIR / not applicable | the rule targets marketing spec sheets. Data tables use Spec 3's `Table` |
| 9.G No em dash | fail by rule, kept by decision | lone dash placeholders in empty cells (43, 52, 53) were kept on purpose by S5-70. No em dash remains as a separator |
| 9.G No en dash as a separator or range | fail | time ranges "10:00–11:20" (45; e2e-asserted at `staffing.spec.ts:201, 203, 217, 222`), the Term option (43, F4), and the "2–10" helper in the department dialog (not captured). Owner question Q1 |
| No Unicode glyph icons (Spec 3 §3.3) | fail, one | "Admin → Terms" in 13 (F9) |
| 4.2 One accent, colour lock | pass with note | purple only on actions, links, the current nav item and tab. F5 and F7 are the two places it lands on a remove or close control. Status colours keep their meaning except F8 |
| 4.4 One radius, cards only for hierarchy | pass | 4 px throughout; the only bordered card that is not a table is the 45 band |
| 4.5 Full states, button contrast, no CTA wrap | pass | 12 and 13 empty states are composed; no wrapped button at 1440 or 390 |
| 4.5 No duplicate CTA intent | pass | 13's disabled "New section" and "Set up terms" are different intents. P9-L3 noted that an empty term shows two "New section" primaries; that state is not captured |
| 4.6 Label above, error below, no placeholder labels | pass | 41, 44, 46 |
| 4.7 Nav single line and at most 80 px; explicit phone collapse | pass | the header is 72 px at 1440; every 390 capture is 390 px wide |
| 4.9 Long lists and copy self-audit | pass with note | the 50-row Audit log is a data view, not marketing; copy notes are in F9 |

## 4. Summary table

| Screen | Criteria still below 4 | The single fix that lifts each |
|---|---|---|
| 12 Courses empty | none | - |
| 13 Sections empty | none | - |
| 40 Courses list | none | - |
| 41 Course dialog | none | - |
| 42 Remove course | none | - |
| 43 Sections list | none | - |
| 44 Section dialog | none | - |
| 45 Section detail | none | - |
| 46 Roster import | none | - |
| 47 Remove from roster | none | - |
| 50 Departments | none | - |
| 51 Terms | none | - |
| 52 Members | none | - |
| 53 Invitations | (g)7 phone, 3 | F1: stack-mode label-value grid with dash cells hidden |
| 54 Audit log | (g)5 paging, 3; (g)6 audit sentence, 3 | (g)5: C6 "Showing 50 of N" (API total); (g)6: C5 Target column. Both are product gaps outside the Spec 5 §1 criterion |

## 5. Questions for the owner and the orchestrator

- **Q1 (owner, direction).** Taste-skill §9.G bans the en dash even in ranges ("10:00–11:20"). It is correct typography, and the e2e suite asserts it (`staffing.spec.ts:201, 203, 217, 222`), so a change means an e2e edit outside D80's list. P9-L9a raises the same question. Recommendation: keep the en dash in numeric time and date ranges, and drop it only where it acts as a spaced separator (the Term option, F4).
- **Q2 (orchestrator).** F6 reverses S5-51 by putting the Audit log into stack mode on phones. Recommendation: yes, because the pack's (m)4 asks for cards when a row has at most five fields.
- **Q3 (orchestrator).** Give staff and the admin display names in the capture seed (`e2e/screens.ts`), so that C1 shows in the captures (F2). P9-L9a raises it as R4. This touches an e2e file under D80.
