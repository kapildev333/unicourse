# P8-L3b - Benchmark comparison: Courses, Section detail, dialogs, Admin tables

Lane P8-L3b, Phase 8 (D75-D78). Compares our captures against the reference pack (P8-L2) and the owner's taste rubric (redesign-skill, D78). No product code. Input for the synthesis lane P8-L3d.

## 0. Method

- **Read:** CLAUDE.md UI conventions; Spec 3 §3, §5, §6, §7; P6-L1 §9.4-9.6; pack §0, §1, §2 (f) (g) (h), §3, §4 (f) (g) (h) and (m), §5; redesign-skill Design Audit and Fix Priority. To confirm two findings I also read `components/dialog.tsx` (initial focus), `components/table.tsx`, `components/tab-nav.tsx` and the e2e selectors in `e2e/staffing.spec.ts`.
- **Captures judged:** `light-1440` of 40-47 and 50-54 (13 images); `dark-1440` of 41, 45, 46, 52; `light-390` of 40, 44, 45, 52.
- **Skipped:** all Dribbble preview images (the pack's descriptions were enough); the other dark and phone captures. Phone scores for 41, 42, 43, 46, 47, 50, 51, 53, 54 are inferred from the shared components and marked "inferred".
- **Capture facts:**
  - The 390 captures for 44, 45 and 52 are 693, 555 and 561 px wide. The document scrolls sideways at 390 px, because the white header stops at 390 while the content runs past it.
  - In the tall captures (45-47, 54) the modal backdrop covers only the first viewport. That is a full-page screenshot artifact, not a finding.
  - Row hover and focus rings cannot be judged from still images and are not scored.
- **Scores** use the pack's §4 scale (5 = matches the row's description, 3 = present but inconsistent, 1 = absent or broken). Each score is the lower of the two themes and the two widths. "n/a" means the criterion does not apply to that screen.
- **DIR rules** (Lucide, DM Sans, purple, bordered cards, 4 px radius, tabular figures) are not findings; they are listed in §3.

## 1. Patterns

### 1.1 Tables (Courses, Sections, Roster, Departments, Terms, Members, Invitations, Audit log)

What works, everywhere:
- Every table renders through the one `Table` (`ui-table`). Rows are 48 px (56 px where a cell stacks two lines), headers bold on the mist header row, hairline rules and a bordered 4 px container. That matches Primer "normal" density and pack §3.3.
- Empty cells show a dash (Members and Invitations Departments, Sections TAs), as in the Mercury log.
- Row actions are text buttons with verbs ("Edit", "Delete", "Change role", "Revoke"), right-aligned in the last column, and they appear only where an action is possible (Revoke only on pending rows). Accessible names carry the object (e2e uses "Edit CS101 section 001").
- Dark mode: the header row, surface and hairlines step correctly, and the danger text is lightened to stay legible (41, 45, 52 dark).

Weaker than references:
- **Phone collapse is broken.** At 390 the Courses table (40) scrolls inside its container with Edit/Delete off-screen and no frozen first column. Members (52) and the Roster (45) do the same, and the page itself overflows. Pack §3.11 (Shyaula, Honeycutt) stacks rows of 3-5 fields into cards. Rubric (g)7 and (m)1/(m)4 fail.
- **The count sits below the table** ("Showing n", bottom left) instead of beside the title or toolbar (Untitled UI "Team members 48 users"; Primer title/actions slot). There is no position for long lists: the Audit log reads "Showing 50" with Load more, not "50 of N" (Sergushkin "1-8 of 32").
- **Filters are full-width selects** (Courses 40, Sections 43): a 1296 px select holding 5 options. Stripe and the Ralabs log put sized filter controls in one row, and our own Audit log (54) already does.
- **Destructive red on every row**, at rest: a column of 5-14 red "Delete"/"Remove" words. Primer puts destructive actions in a row menu, and Untitled UI uses muted icons. It is the loudest colour on every admin page.
- **No bulk selection** anywhere (GitHub People, Dhendup "2 Selected"). This is a product gap, not a style gap (C).
- **Dates are left-aligned.** Spec 3 §5.5 asks only that numeric columns be right-aligned, so dates as text are acceptable. Line numbers in the import preview are right-aligned. No finding.

Improvements (pattern level; screens refer back by number):
- **T1.** Fix the horizontal page overflow at 390 px on 43, 44, 45 and 52 (likely a flex child without `min-w-0` around `Table`'s `overflow-x-auto` wrapper, plus the non-wrapping rows named in 2.6 and 2.5). Acceptance: `document.documentElement.scrollWidth === 390` on every route. Impact H, effort S, A.
- **T2.** Phone tables become stacked cards below 640 px. `Table` gains a `stack` mode: each `td` carries `data-label`, CSS in `ui-table` renders rows as blocks, with the label before the value, the first cell bold, badges top right and the actions row full width at the foot. Apply it to every admin table except the Audit log, which keeps horizontal scroll with a sticky first column. Add a test for the component and a line in the design doc. Impact H, effort M, A.
- **T3.** Move "Showing n" into the toolbar row above the table: right-aligned, on the same line as the filter controls, and wrapping to its own line on phone. Same copy. Impact L, effort S, A.
- **T4.** Constrain filter selects to `w-full sm:w-72` inside a `flex flex-wrap items-end gap-4` row, the Audit log's layout. Impact M, effort S, A.
- **T5.** `ghost-danger` renders `text-ink-muted` at rest and `text-danger` on hover and focus-visible. The word stays, so meaning is never by colour alone. This changes one shared component and needs the owner, see §3 B2. Impact M, effort S, B.
- **T6.** Bulk select and remove on Roster and Members (tri-state header checkbox, "n selected" in the toolbar, one confirm listing the people). Impact M, effort L, C (scope; needs either a bulk endpoint or N calls plus an audit decision).

### 1.2 Dialogs (41, 42, 44, 46, 47)

What works, everywhere:
- One rhythm on every dialog: bold `text-xl` title, optional muted description, body `mt-5`, then a hairline footer (`DialogFooter`) with Cancel (secondary) first and the verb primary on the right ("Save changes", "Create section", "Import 2 rows", "Delete", "Remove"). Rubric (h)1 holds on all five.
- The destructive confirm (42, 47) asks a question that names the object ("Delete BIO150?") and states the consequence. Red is reserved for the confirm button. Initial focus lands on Cancel, because native `showModal()` focuses the first focusable element and Cancel comes first in the DOM. This matches Primer ConfirmationDialog and the Hubacek modal, and beats the Fastly counter-example.
- Native `<dialog>`: focus is trapped, Escape closes, and focus returns to the trigger (`dialog.tsx`). Rubric (h)5 holds.
- Labels sit above inputs and helper text below ("Up to 20 characters, e.g. CS101"). There are no placeholder labels.
- Dark (41, 46): the dialog reads one surface step above the dimmed page, and the primary button uses the lighter tint with charcoal text per Spec 3.

Weaker than references:
- **No close control in the header.** Sergushkin's invite modal and Primer dialogs carry an X. On touch there is no Escape key, but Cancel is always present, so the impact is low.
- **Phone presentation:** at 390 the `max-w-md`/`max-w-2xl` dialog fills the width edge to edge with no margin (44 light-390). The meeting row inside is unusable (2.5).
- **Long object names wrap the title** (47: a two-line email). This comes from the email-as-name gap (C1).

Improvements:
- **D1.** Below 640 px, `Dialog` gets `w-[calc(100%-2rem)]` and `max-h-[calc(100dvh-2rem)]` with the body scrolling, so the footer stays visible. One change in `dialog.tsx`. Impact M, effort S, A.
- **D2.** Optional: an icon-only close button (Lucide `X`, `aria-label="Close"`, `title="Close"`, `ghost`) at the top right of `Dialog`. Impact L, effort S, A.

### 1.3 People shown as emails (all screens in this group)

- **C1 (known C item).** No API sets a display name, so every person shows as an email: Sections staff chips, the detail band, Staff, Roster (Name and Email columns are identical), Members, Invitations, Audit "Who". The roster CSV already carries `displayName` (46 shows "Amara Okafor"), but existing users keep showing their email, so names supplied by import are dropped or never read back. A way to set and return names (import, invite accept, profile) is needed. Impact H, effort L, C.
- **P1.** Until C1 lands, use one fallback rule through `personName`:
  - Show the email once, in ink.
  - Members: drop the dash line above the email, which brings rows back to 48 px.
  - Roster: the Name cell shows a muted dash when no name exists, instead of repeating the email.

  Impact M, effort S, A.

## 2. Screens

### 2.1 Courses list
Captures: `40-courses-list-admin-light-1440`, `40-courses-list-admin-light-390`.
Scores (g): 1. Title, count, action: 3. Title, description and "Import courses"/"New course" sit in a fixed spot, but the count is below the table. 2. Alignment and rows: 4. 48 px rows with aligned columns, and no numeric columns. 3. Row actions: 4. Two inline verbs with accessible names. Primer would pull one into a menu. 4. Bulk: n/a (absent). 5. Paging: 3. "Showing 5", no position for long lists. 6. Audit: n/a. 7. Phone: 1. Actions are cut off at 390 and the columns squeeze.

Works: Page header with a secondary Import and a primary New, with Lucide icons, matches the Untitled UI header (Download CSV + Add user). Code, Title and Department columns read in scan order.

Weaker: Untitled UI shows "48 users" beside the title. Ours puts the count under the table. The Stripe filter row uses sized controls. Ours is a full-width select.

Improvements:
1. T4 on the Department filter. Impact M, effort S, A.
2. T3. Impact L, effort S, A.
3. T2 (Code bold as the card head, Department top right, actions at the foot). Impact H, effort M, A.
4. Show each course's section count in a right-aligned column. Impact L, effort M, C (the list endpoint does not return it).

### 2.2 Course dialog
Captures: `41-courses-course-dialog-admin-light-1440`, `-dark-1440`.
Scores (h): 1. Rhythm: 5. 2. Labels and errors: 4. Label above with a helper under Code. The error state was not captured. 3. Destructive: n/a. 4. Import preview: n/a. 5. Focus: 5. The first field is focused on open. 6. Phone: 3 (inferred). Edge-to-edge width (D1).

Works: Three fields in a sensible order (Department, Code, Title), with "Save changes" as the verb. The title names the object ("Edit CS101").

Weaker: Option text "CS — Computer Science" uses an em dash (H22, taste-skill 9.G). None of the other references show anything this dialog lacks.

Improvements:
1. Options read "CS · Computer Science" (middle dot, as on the meeting lines). The text is an e2e selector: update `staffing.spec.ts` lines 140-141 and 159-160 together. Impact L, effort S, A (copy change, orchestrator checks selectors).

### 2.3 Remove dialogs (course 42, roster 47)
Captures: `42-courses-remove-dialog-admin-light-1440`, `47-section-remove-dialog-admin-light-1440`.
Scores (h): 1. Rhythm: 5. 2. Labels and errors: n/a. 3. Destructive: 4. The object and consequence are named, focus starts on Cancel and the red is reserved. 42 does not say what happens to the course's sections. 4. Import preview: n/a. 5. Focus: 5. 6. Phone: 3 (inferred, D1).

Works: The question titles name the object. 47's body states the exact blast radius ("Their account and other sections are not affected"), which is better than Hubacek's modal, whose body never names the product. Cancel first, then a red verb button.

Weaker: Primer's body states every consequence. 42 says "permanently deletes 'Cell Biology'" and leaves out the course's sections. 47's title wraps over two lines because the person shows as an email (C1).

Improvements:
1. 42's body adds the effect on sections ("Its sections are deleted too" or "Courses with sections cannot be deleted"), whichever the API does. Impact M, effort S, A (copy change; the worker confirms the API behaviour first).
2. 47's title uses `personName` once C1 lands. Covered by C1.

### 2.4 Sections list
Captures: `43-sections-list-admin-light-1440`.
Scores (g): 1. Title, count, action: 3. As 2.1. 2. Alignment and rows: 4. Course cell is the code link over a muted title (Sergushkin name-over-email) and rows are 56 px. 3. Row actions: 4. 4. Bulk: n/a. 5. Paging: 3. 6. Audit: n/a. 7. Phone: 1 (inferred from 44's background at 390: staff chips run past the container).

Works: The code-over-title cell gives each row one link target and a quiet second line. An empty TA cell shows a dash. The term filter shows the term's date range.

Weaker: Staff shown as bordered chips holding full emails (a 280 px pill each) read as inputs or badges. Olga Sipatova's table keeps rows one height with "3 regions" plus a popover. Untitled UI shows a name. The full-width term select (T4).

Improvements:
1. Instructors and TAs cells become plain `text-sm` text (names joined by commas, pending names muted with the "Invited" word), not bordered chips. With more than two people, show "First name +2". Impact M, effort S, A.
2. T4 on Term. Impact M, effort S, A.
3. T2. Impact H, effort M, A.

### 2.5 Section dialog (with meeting row)
Captures: `44-sections-section-dialog-admin-light-1440`, `44-...-light-390`.
Scores (h): 1. Rhythm: 5. 2. Labels and errors: 3. Labels sit above, but the Location input is cut off at 1440 ("eering Building 215"). 3. Destructive: n/a. 4. Import preview: n/a. 5. Focus: 5. 6. Phone: 1. The meeting row squeezes Location to about 20 px, "Remove" overlaps it, and the page overflows to 693 px.

Works: The term is stated as context ("Term: Academic Year 2026") rather than as a field. "Add meeting time" is secondary with a Plus icon, and per-row Remove sits on the row.

Weaker: The Sergushkin invite modal gives each input its full width. Our five-column meeting grid in a 672 px dialog truncates the location at desktop width and breaks on phone.

Improvements:
1. The meeting row grid becomes `grid-cols-2 sm:grid-cols-[8rem_7rem_7rem_minmax(0,1fr)_auto]`. On phone, Day spans 2 columns, Starts and Ends share a row, Location spans 2, and Remove sits at the right of a row under them. Open the section dialog `wide` (max-w-4xl) so Location gets at least 16 rem at desktop. Impact H, effort S, A.
2. Course options lose the em dash (2.2 improvement 1). Impact L, effort S, A.

### 2.6 Section detail (staff, meetings, roster)
Captures: `45-section-detail-admin-light-1440`, `-dark-1440`, `-light-390`.
Scores (f): 1. Title band: 4. Code, title, term and instructor appear before any section, but the instructor shows as an email. 2. Tabs with counts: 2. Three stacked sections with no tab row and no counts (deliberate, P6-L1 §9.4). 3. Tables follow (g): 3. The Roster inherits 1.1, and its Name equals its Email. 4. Back link: 5. "Sections" with ArrowLeft in the band. 5. Edit next to what it edits: 3. Staff add and remove sit inline, but meeting times are read-only here: editing them means going back to the list. 6. Phone: 1. The Add-by-email input squeezes to 60 px with its helper wrapping word by word, and the page overflows to 555 px.

Works: The title band matches Coursera's ("what is this, who teaches it" before anything else). The meeting lines are plain and complete ("Monday 10:00-11:20 · Hall B 204"). Staff lists sit at 48 px rows, with pending invitees inline. Dark mode holds: band, lists and table step correctly.

Weaker: Coursera and Primer UnderlineNav offer anchor tabs with counts ("Roster 42"). Ours has no count, and a long roster pushes Roster below the fold with no jump. The Shayoli Singh course detail puts key facts in one row under the title. Ours shows meetings as a section of its own, which is fine for two lines. The Import roster button floats alone on its own row under the heading, where every other page puts the action on the header row (Untitled UI, Primer title/actions slot). The global nav shows no current item on this route ("Courses" is not marked; H20, pack §3.1). This is a shell issue, also reported to P8-L3a.

Improvements:
1. The Add-by-email form becomes `flex flex-wrap items-start gap-2`. The input becomes `min-w-64 flex-1`, with the helper below the whole row. The buttons align to the input's top (`mt-[26px]`, the label height, as the meetings editor does with `mt-7.5`). Impact H, effort S, A.
2. Put the Roster `h2` and "Import roster" in one row (`flex items-center justify-between mb-4`). Impact M, effort S, A.
3. The section headings carry their count in muted text: "Roster" then `<span class="text-ink-muted font-normal">8</span>`, and the same for Staff. e2e `getByRole('heading', { name: 'Roster' })` is a substring match, so it still passes. Impact M, effort S, A.
4. The band gets an "Edit section" secondary button (Lucide `Pencil`) as the `PageHeader` action for roles that may edit. It opens the existing `SectionDialog`. Impact M, effort S, A.
5. The band's description replaces "—" with " · " ("Introduction to Computer Science · Academic Year 2026"). Impact L, effort S, A (copy change).
6. Mark "Courses" current in the global nav on `/sections/{id}` for staff-side viewers. Impact M, effort S, A (shell file; owned by P8-L3a's area).

### 2.7 CSV import dialog (roster, with preview)
Captures: `46-section-csv-import-dialog-admin-light-1440`, `-dark-1440`.
Scores (h): 1. Rhythm: 5. 2. Labels and errors: 4. "CSV file" label sits above, and the footer carries the disabled reason (P6-L1 §9.5). 3. Destructive: n/a. 4. Import preview: 4. Per-row outcome badges with text (Exists, Invite) and a summary before commit. The summary is shown twice. 5. Focus: 5. 6. Phone: 3 (inferred). A five-column preview inside an edge-to-edge dialog.

Works: "Download template" sits next to the instructions, with a Lucide icon (NING batch import). A preview before commit, with the primary button stating the count ("Import 2 rows"), matches GitHub's review-before-remove. Line numbers are right-aligned, and the header names match the CSV header exactly.

Weaker: NING and Attio show one status strip. Ours repeats the same numbers as a sentence ("4 rows checked: 0 to add, 2 to invite...") and then as a counter row ("Add 0 Invite 2 Exists 2 Errors 0"). The Message column is empty on every row yet takes about 110 px. The NING drop zone versus our native "Choose File": a minor difference, not worth a new pattern.

Improvements:
1. Keep the sentence (it is the `role="status"` announcement) and delete the counter row. Impact M, effort S, A.
2. Render the Message column only when at least one row has a message. Impact L, effort S, A.
3. On phone, the preview table uses T2 stacking (Line and Result on the head line, email below). Impact M, effort S once T2 exists, A.

### 2.8 Departments
Captures: `50-admin-departments-admin-light-1440`.
Scores (g): 1. Title, count, action: 3. 2. Alignment and rows: 5. 3. Row actions: 4. 4. Bulk: n/a. 5. Paging: 4. Short list. 6. Audit: n/a. 7. Phone: 2 (inferred; T1/T2).

Works: A bold TabNav with the current tab underlined in purple and `aria-current` (Primer UnderlineNav). A one-sentence description that says what departments are for. A plain two-column table at 48 px.

Weaker: Untitled UI shows a count by the title (T3). Nothing else; this is at reference level.

Improvements:
1. T3. Impact L, effort S, A.
2. A course count per department as a right-aligned number column. Impact L, effort M, C.

### 2.9 Terms
Captures: `51-admin-terms-admin-light-1440`.
Scores (g): 1. Title, count, action: 3. 2. Alignment and rows: 4. Dates as text, left. 3. Row actions: 4. 4. Bulk: n/a. 5. Paging: 4. 6. Audit: n/a. 7. Phone: 2 (inferred).

Works: Starts and Ends are separate, readable "Jan 12, 2027" dates. A primary "New term" with Plus.

Weaker: None of the pack's references show a term list. By H6 ("current term shown"), nothing says which term is in progress.

Improvements:
1. Add a Status column with a `Badge` "Current", "Upcoming" or "Past", computed client-side from the dates (neutral tone; "Current" may use accent-soft like the Admin role badge). Impact M, effort S, A.

### 2.10 Members
Captures: `52-admin-members-admin-light-1440`, `-dark-1440`, `-light-390`.
Scores (g): 1. Title, count, action: 2. No action (by design: invites live in Invitations) and the count is below. 2. Alignment and rows: 3. The Name cell is a dash line over the email, giving 56 px rows of mostly empty space. 3. Row actions: 4. "Change role" and "Remove" with names. 4. Bulk: n/a (absent; GitHub People has it, T6). 5. Paging: 3. 6. Audit: n/a. 7. Phone: 1. Only Name and Role are visible, actions are off-screen, and the page overflows to 561 px.

Works: Role and status as text badges (Admin on accent-soft, Member and Active neutral), so status is never colour alone. The description points to where invites happen. Dark contrast holds for badges, links and danger text.

Weaker: Sergushkin and Untitled UI show name over muted email on one row of about 45 px. Ours shows an empty dash line over the email (P1/C1). Untitled UI and Sergushkin have a primary "Add user" on the header. Ours makes the reader go to the Invitations tab without a link.

Improvements:
1. P1 (email alone, 48 px rows). Impact M, effort S, A.
2. The word "Invitations" in the description becomes a link to the Invitations tab (same copy). Impact L, effort S, A.
3. T2 (email head, Role and Status badges top right, actions at the foot). Impact H, effort M, A.

### 2.11 Invitations
Captures: `53-admin-invitations-admin-light-1440`.
Scores (g): 1. Title, count, action: 3. 2. Alignment and rows: 4. 3. Row actions: 5. Revoke appears only on pending rows. 4. Bulk: n/a. 5. Paging: 3. 6. Audit: n/a. 7. Phone: 2 (inferred).

Works: Status as text badges (Pending on warning-soft, Accepted neutral). A primary "Invite someone" with UserPlus. Accepted invitations stay visible as a record.

Weaker: The actionable rows (2 Pending) sit in the middle of 12 Accepted ones. GitHub and Stripe lists are filterable, and the Dhendup toolbar has a filter. Expires shows a date on accepted invitations, where it no longer means anything. Mercury uses a dash for no value.

Improvements:
1. Sort Pending first, then by expiry. Impact M, effort S, A (client sort).
2. The Expires cell shows a muted dash for accepted invitations. Impact L, effort S, A.
3. Optional: a Status filter select ("All", "Pending", "Accepted") in the T4 row. Impact L, effort S, A.

### 2.12 Audit log
Captures: `54-admin-audit-admin-light-1440`.
Scores (g): 1. Title, count, action: 3. The description says "newest first", and the count sits at the foot. 2. Alignment and rows: 4. Muted timestamps per Spec 3 §6. 3. Row actions: 4. "Show changes" disclosure. 4. Bulk: n/a. 5. Paging: 3. "Showing 50" plus Load more, no total. 6. Audit sentence: 3. Newest first, the action is a word, timestamps are muted, and a date-range filter sits above, but there is no target column, so "Assessment set" cannot be read as a sentence. 7. Phone: 2 (inferred; horizontal scroll, no frozen column).

Works: The filter row (entity, actor, from, to) is the set the Ralabs log shows and matches GitHub's qualifiers. The action is a word, not colour alone (Gajara audit log). Load more with the count on the left and the button on the right.

Weaker: GitHub's audit entry is actor, action, target and time. Ours lacks the target, so 20 identical "Assessment set" rows cannot be told apart without opening each one. Action copy is uneven ("Invitation accept" next to "Membership created"). The disclosure uses the native triangle marker, a Unicode-style glyph (Spec 3 §3.3). Gajara stacks date over a muted time. Ours prints "Oct 7, 2026, 11:32 AM" on one line, which is fine.

Improvements:
1. Add a Target column after Action ("CS101 001 · Midterm", "jonas.berg@..."). Impact H, effort M, C (needs a target label per entry from the API).
2. Action labels in the past tense throughout ("Invitation accepted"; "Assessment set" becomes "Scores set" if that is what the event means). Impact M, effort S, A (copy change; check `audit-log` tests).
3. The `summary` drops the native marker (`list-none`) and shows a Lucide `ChevronRight` or `ChevronDown` by open state, 16 px, `aria-hidden`. Impact L, effort S, A.
4. "Showing 50 of N" once the API returns a total. Impact L, effort S, C.
5. At 390: horizontal scroll with When as a sticky first column (`sticky left-0 bg-surface`). Impact M, effort S, A.

## 3. Direction questions for the owner (B)

- **B1. Section detail as real tabs?** Coursera anchor tabs and Primer UnderlineNav with counts, against P6-L1's stacked sections (the e2e reads all three headings on one page). Affects 45, 46, 47. 2.6 improvement 3 (counts in headings) is the A-level substitute.
- **B2. Quiet destructive row actions (T5).** `ghost-danger` muted at rest, red on hover and focus, against Spec 3's ghost-danger as built. Affects 40, 43, 45, 50, 51, 52, 53.
- **B3. Phone dialogs as bottom sheets.** This is the mobile pattern question deferred in Spec 3 §10 and pack §5 Q3. D1 (margins and scroll) is the A-level substitute. Affects 41, 42, 44, 46, 47.
- **B4. Row actions in a kebab menu (Primer, Sergushkin) versus inline verbs (Untitled UI).** It needs a new Menu component. I recommend keeping inline. Affects every table.
- **DIR, not scored:**
  - Tabular figures (H21, D56) affect date and number columns in 51, 53, 54. Proportional digits did not misalign anything in these screens.
  - Lucide (H23), DM Sans 400/700 only (H24), purple (H25), bordered cards and 4 px radius. Nothing in this group argues for reopening them.

## 4. Top 10 for this group

Ordered by impact, then effort.

1. **T1.** Remove horizontal page overflow at 390 px (43, 44, 45, 52). Impact H, effort S, A.
2. **2.5 improvement 1.** Responsive meeting row and a `wide` section dialog (44). Impact H, effort S, A.
3. **2.6 improvement 1.** Wrapping, aligned Add-by-email row (45). Impact H, effort S, A.
4. **T2.** Stacked-card phone mode in `Table` (all tables; Audit log keeps scroll with a sticky When). Impact H, effort M, A.
5. **2.12 improvement 1.** Audit Target column (54). Impact H, effort M, C.
6. **C1.** Display names through the API, so people stop showing as emails (every screen). Impact H, effort L, C.
7. **P1.** One email fallback (Members without the dash line, Roster without the duplicate column). Impact M, effort S, A.
8. **2.6 improvements 2 to 4.** Roster heading row with Import, counts in the section headings, "Edit section" in the band (45). Impact M, effort S, A.
9. **T4 plus T3.** Sized filter row with the count on it (40, 43, later 53). Impact M, effort S, A.
10. **2.11 improvement 1 plus 2.7 improvement 1.** Pending invitations first, and a single import summary (53, 46). Impact M, effort S, A.
