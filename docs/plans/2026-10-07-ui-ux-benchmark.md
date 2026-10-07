# UI/UX benchmark - Implementation plan (Phase 9)

> **For agentic workers:** executed by `/orchestrate` (orchestrator + `designer` → `worker` → `reviewer` lanes, one branch and worktree per lane). Each task is one lane. Global constraints, gates and git rules are those of `docs/plans/2026-10-05-platform-core-and-staffing.md` and Spec 3's plan, and are not repeated.

**Goal:** every screen scores at least 4 on each rubric criterion an item addresses, and no 390 px capture is wider than 390 px (spec §1).

**Spec:** `docs/specs/2026-10-07-ui-ux-benchmark.md` - every task cites S5 ids; the id's row in spec §4-§5 is the build instruction. A worker that finds a row ambiguous stops and reports.

**Order:**
0. Before Phase 9 the orchestrator runs `npm run screens` once on `main` and copies `apps/web/e2e/.tmp/screens/` to its scratchpad as `before/` (captures are gitignored and each run overwrites them).
1. Task 1 (P9-L1, sub-lanes a and b in parallel) lands first: it owns `components/`, `index.css`, the shell and the capture script.
2. Tasks 2-5 then run in parallel, one feature directory each.
3. Task 6 runs alone after Tasks 2-5 land: it owns the copy the e2e suite asserts, and the e2e files.
4. Task 7 waits on the owner's §6 answers; Task 8 waits on the owner's yes and may run any time (API only).

`npm run e2e` and `npm run screens` share the local database: one integration run at a time, orchestrator included; workers ask the orchestrator for a slot.

## Review focus

1. **Width** - the S5-68 assertion is the gate; the reviewer opens the lane's 390 captures in both themes, not only the assertion output.
2. **Both themes** - each lane's screens checked light and dark at 1440 and 390.
3. **Copy and selectors** - only the e2e selectors in spec §5.7 change, only in the lane named there; any other e2e edit is a defect. Unit tests inside the lane's own scope change with its copy and need no listing.
4. **No one-off recipe** - new patterns live in `components/` with a test and a line in `docs/design/P6-L1.md`; feature files compose them.
5. **No literal colours** - `no-literal-colors.test.ts` green; the reviewer greps the diff for `dark:`, hex and palette utilities.

## File structure

```
apps/web/src/components/{card,dialog,table,loading,load-more,button}.tsx (+ tests)   Task 1a
apps/web/src/components/segmented-control.tsx, disclosure.tsx (+ tests)             Task 1a (new)
apps/web/src/index.css                                                               Task 1a (token line-control)
apps/web/src/components/{app-shell,tab-nav}.tsx, routes/__root.tsx, routes/$slug/route.tsx   Task 1b
apps/web/e2e/screens.ts                                                              Task 1b, then Task 6
apps/web/src/features/{my-sections,catalog}/*, lib/schedule.ts, routes/$slug/{me,catalog,index}.tsx   Task 2
apps/web/src/features/courses/*, routes/$slug/{courses.tsx,courses/*,sections.$id.tsx}   Task 3
apps/web/src/features/admin/*, routes/$slug/{admin.tsx,admin/*}                       Task 4
apps/web/src/features/grades/*, routes/$slug/{grades,grade-scale,sections_.$id.grades}.tsx   Task 5
apps/web/src/features/auth/*, routes/{sign-in,invite.$token}.tsx, e2e/{helpers,staffing.spec}.ts   Task 6
apps/api/migrations/ (if needed), internal/course/import.go, sign-in, a me handler, httpx/routes.go, openapi.json   Task 8
```

---

## Phase 9 - UI/UX improvements

### Task 1a (P9-L1a, W): shared components

Spec §4.7, §4.10, §4.11, §4.18, §5.4-§5.6. No design lane: the rows are concrete; the worker adds one line per new pattern to `docs/design/P6-L1.md`.

**Scope:** `apps/web/src/components/card.tsx`, `card.test.tsx`, `dialog.tsx`, `dialog.test.tsx`, `table.tsx`, `table.test.tsx` (new), `loading.tsx`, `loading.test.tsx` (new), `load-more.tsx`, `button.tsx`, `segmented-control.tsx` and test (new), `disclosure.tsx` and test (new), `apps/web/src/index.css`, `docs/design/P6-L1.md` (appended lines only).

**Items:** S5-68 (component part), S5-35, S5-64, S5-18, S5-37, S5-66, S5-22 (slot), S5-23, S5-34 (prop), S5-38, S5-71.

- [ ] S5-35: failing `table.test.tsx`: `<Table stack>` renders `data-label` on cells and the `ui-table-stack` class; then the mode (CSS in `ui-table` under a `max-width: 639px` media rule, label before value, first cell bold, actions row full width). Table's scroll wrapper gets a `min-w-0` parent (S5-68).
- [ ] S5-64: failing `loading.test.tsx`: `variant="table"` renders the given column names as `th` and 5 rows of `ui-skeleton`; `variant="cards"` renders 3 card-shaped blocks; default unchanged; then the variants.
- [ ] S5-18, S5-22: extend `card.test.tsx`: badge row has `mt-auto`; a `footer` prop renders a link outside the card's main link (two links, not nested); then the change, band `h-16`.
- [ ] S5-37, S5-38: extend `dialog.test.tsx`: a button named "Close" closes the dialog and returns focus; then the close button and the phone width and height classes.
- [ ] S5-23: `segmented-control.test.tsx`: options render as buttons with `aria-pressed`, arrow keys move, `onChange` fires; then the component (selected `bg-accent-soft text-ink`, others ghost).
- [ ] S5-71: `disclosure.test.tsx`: `summary` toggles `open`, the chevron icon swaps (`ChevronRight` closed, `ChevronDown` open), `aria-hidden` on the icon; then the component (`list-none`, no rotation transition).
- [ ] S5-34: `LoadMore` gains `showCount` (default `true`); a test that `false` omits the `role="status"` text.
- [ ] S5-66: token `--color-line-control` in all three token blocks of `index.css` (light: the ink value; dark and `prefers-color-scheme` dark: the `line-strong` value), `@theme` mapping, `button.tsx` secondary uses `border-line-control`; run the P6-L1 §1.3 contrast script and add the measured ratio to `docs/design/P6-L1.md`.
- [ ] Gates: `npm run lint`, `typecheck`, `test`, `build`, `e2e`. Commit.

**Orchestrator verification:** gates re-run; `npm run screens`; compare `before/` and new 20, 30, 41, 44 (card band and badge row, dialog margins at 390) and 20-dark (secondary border) at 1440 and 390, light and dark; contrast of `line-control` on dark `surface` re-measured; `CLAUDE.md` UI conventions gain `line-control`, `SegmentedControl`, `Disclosure` (orchestrator edit).

### Task 1b (P9-L1b, W): shell, no-access page and capture script

Spec §4.1, §4.5, §5.4, §5.5.

**Scope:** `apps/web/src/components/app-shell.tsx`, `app-shell.test.tsx`, `tab-nav.tsx`, `apps/web/src/routes/__root.tsx`, `apps/web/src/routes/$slug/route.tsx`, `apps/web/e2e/screens.ts`.

**Items:** S5-68 (assertion), S5-04, S5-05, S5-06, S5-13, S5-02, S5-03, S5-67 (captures).

- [ ] S5-68: in `screens.ts` `snap`, at width 390 read `document.documentElement.scrollWidth` and `expect.soft(width, file).toBeLessThanOrEqual(390)`; also append `WIDE` lines to the index file. Run `npm run screens` and record the failing list (expected: the eleven bases of spec §5.5). The full run is red until Task 6 lands; it is not in CI. Each lane's gate is no WIDE line for the lane's own captures.
- [ ] S5-67: `screens.ts` captures one success `StatusMessage` and the weight warning badge in dark (new bases `72-states-success-admin`, `73-states-weight-warning-instructor`).
- [ ] S5-05: failing `app-shell.test.tsx`: on `/{slug}/sections/{id}` as staff "Courses" has `aria-current="page"`; on `/{slug}/grade-scale` as admin "Admin" has it; then the match rules.
- [ ] S5-02: failing test: pressing `/` with focus on `body` focuses the search; with focus in an input it types; then the handler and `aria-keyshortcuts`, `title`.
- [ ] S5-03, S5-04: grid slot for search; below 768 px three rows with nav links `min-h-11`. No unit test (layout); verified by captures.
- [ ] S5-06: `TabNav` list `overflow-x-auto` with scroll-snap; existing `admin-layout.test.tsx` stays green.
- [ ] S5-13: failing test for the no-access page: the link list is led by "Go to your university:"; with one membership a primary "Go to {university}" button and Sign out secondary; the slug text reads "this university's address"; then the page. Existing route tests green; e2e reads the heading only.
- [ ] Gates as Task 1a, plus `npm run screens`: no WIDE line for the lane's own captures (04, 72, 73). Commit.

**Orchestrator verification:** gates; `npm run screens`; the WIDE list is a subset of the eleven bases in spec §5.5, never a new one; compare 20, 30, 45, 52, 66 header at 1440 and 390 (header height at 390 measured on the capture, target about 150 px), 04 light and dark; nav current on 45 and 66.

### Task 2 (P9-L2, W): my sections and catalog

Spec §4.6 (10, 11), §4.7, §4.8, §4.9, §4.18. After Task 1.

**Scope:** `apps/web/src/features/my-sections/**`, `apps/web/src/features/catalog/**`, `apps/web/src/lib/schedule.ts`, `apps/web/src/lib/schedule.test.ts`, `apps/web/src/routes/$slug/me.tsx`, `catalog.tsx`, `index.tsx`. Not the Catalog's search input or the department option separator (Task 6).

**Items:** S5-24, S5-29, S5-30, S5-14 (10, 11), S5-27, S5-26, S5-25, S5-19, S5-23 (use), S5-20, S5-22 (use), S5-65, S5-64 (cards), S5-28, S5-32, S5-31, S5-21.

- [ ] S5-24: failing `my-sections.test.tsx`: with `matchMedia('(min-width: 768px)')` false the schedule renders weekday regions as lists with one link per meeting and no grid; true renders the grid inside an `overflow-x-auto` wrapper; then the agenda (same regions and link names, so e2e `region Monday` holds).
- [ ] S5-25, S5-26, S5-27, S5-28: failing tests: the grid keeps seven day regions, an empty Sat or Sun gets the narrow track; a block shows no visible instructor line but its link's accessible name still contains the instructor (`sr-only`), and the agenda row has none; the untimed list uses " · " (`schedule.tsx:124`); today's header has `aria-current="date"` and "Today" (fake timers); the time line exists only in today's column; then the changes.
- [ ] S5-20: failing `schedule.test.ts` for `formatMeetings` (same time and room on Mon and Wed gives one line; differing rooms give two); then the helper; existing `formatMeeting` tests untouched.
- [ ] S5-29, S5-30, S5-31: failing `catalog.test.tsx`: "Clear filters" appears with a department set and resets it, keeping the term; no-match title quotes `q`; a card in `mine` shows "Enrolled"; count reads "Showing 5 sections"; update the existing "No sections match" assertion (spec §5.7); then the changes.
- [ ] S5-14: failing tests: admin on "No terms yet" sees a "Set up terms" link to the Terms tab; a student sees none; then the action.
- [ ] S5-19, S5-21, S5-22, S5-23: failing tests: own address reads "You"; option text is the name only; instructor card has a "Gradebook" link; List/Schedule is a `SegmentedControl` with `aria-pressed`; then the changes.
- [ ] S5-64, S5-65, S5-32: `Loading variant="cards"` while loading; load banners pass `title="Couldn't load your sections"` / `"Couldn't load sections"`; filter controls `w-full` below 768 px.
- [ ] Gates: lint, typecheck, test, build, e2e (`staffing.spec.ts:200`, `205-206` unchanged and green), screens: no WIDE line for the lane's own captures (20-23, 30-32). Commit.

**Orchestrator verification:** gates; `npm run screens`; compare `before/` and new 10, 11, 20, 21, 22, 23, 30, 31, 32 at 1440 and 390 in both themes; `sips -g pixelWidth` on 21 and 23 light-390 and dark-390 gives 390; open the app at 390 in a real browser and tap a schedule row through to the section.

### Task 3 (P9-L3, W): courses, sections and dialogs

Spec §4.6 (12, 13), §4.10, §4.11, §4.12, §5.6. After Task 1.

**Scope:** `apps/web/src/features/courses/**`, `apps/web/src/routes/$slug/courses.tsx`, `apps/web/src/routes/$slug/courses/**`, `apps/web/src/routes/$slug/sections.$id.tsx`. Not the em dash separators (Task 6).

**Items:** S5-42, S5-40, S5-35 (use), S5-14 (13), S5-69, S5-16, S5-15, S5-33, S5-36, S5-41, S5-43, S5-44, S5-45, S5-39, S5-65, S5-64 (table), S5-17, S5-34.

- [ ] S5-42: failing `staff-editor.test.tsx`: the add row has the wrapping classes and the helper is outside the flex row; then the layout. Verified at 390 by `screens` (45).
- [ ] S5-40: failing `section-dialog.test.tsx`: the dialog uses `wide`; the meeting group uses the two-column phone grid; then the change (needs the D69 amendment, spec §5.7; stop if not ruled).
- [ ] S5-35: every table here passes `stack` and `data-label` per cell (course list, section list, roster, import preview); one test per table that cells carry labels.
- [ ] S5-69: failing `roster.test.tsx`: a student without a name shows the email once and a muted dash in Name; then the change in `roster.tsx` and the other person cells (`access.ts` `personName` unchanged).
- [ ] S5-14, S5-15, S5-16, S5-17: failing `section-list.test.tsx` and `course-list.test.tsx`: no terms keeps New section visible and disabled with its reason ("Add a term first"), hides Import and the Term select and shows "Set up terms" to admins (e2e `staffing.spec.ts:247` holds); no courses shows one primary; New course enabled while loading; then the changes; `PageHeader` above `TabNav` in `courses-layout.tsx`.
- [ ] S5-33, S5-34, S5-36: failing tests: staff cells have no chip border and show "+2" past two names; then the filter row, count in the toolbar (`LoadMore showCount={false}`), staff text.
- [ ] S5-41, S5-43, S5-44, S5-45, S5-39: failing tests: import shows one summary and no Message column when no row has a message; Roster heading row holds Import; headings carry counts; "Edit section" opens the dialog for editors only; remove-course body names the effect on sections (worker reads the API's delete behaviour first); then the changes.
- [ ] S5-64, S5-65: table skeletons with the real column names; banners titled "Couldn't load courses" / "sections" / "section".
- [ ] Gates as Task 2 (no WIDE line for the lane's own captures, 12, 13, 40-47). Commit.

**Orchestrator verification:** gates; `npm run screens`; compare 12, 13, 40-47, 70, 71 before and after, both themes, both widths; `sips` width 390 on 43-47 light-390; in a browser at 390 add a TA by email on section detail and add a meeting in the section dialog.

### Task 4 (P9-L4, W): admin

Spec §4.13, §5.6. After Task 1.

**Scope:** `apps/web/src/features/admin/**`, `apps/web/src/routes/$slug/admin.tsx`, `apps/web/src/routes/$slug/admin/**`.

**Items:** S5-35 (use), S5-51, S5-69, S5-48, S5-46, S5-50, S5-65, S5-64 (table), S5-71 (use), S5-47, S5-34. S5-49 and the S5-48 order are C9 (API), not here.

- [ ] S5-35, S5-51: departments, terms, members and invitations tables pass `stack` with labels; the audit table keeps scroll with a sticky When column; one test per table.
- [ ] S5-69: failing `members.test.tsx`: a member without a name shows one email line and no dash; then the change.
- [ ] S5-48: failing `invitations.test.tsx`: accepted rows show a muted dash in Expires; row order stays the API's; then the change.
- [ ] S5-46: failing `terms.test.tsx` with fixed dates: Current, Upcoming, Past badges; then the column.
- [ ] S5-50, S5-71: failing `audit-log.test.tsx`: past-tense labels for each action the API emits (list them from the API's audit calls); details use `Disclosure`; then the changes.
- [ ] S5-47, S5-34, S5-64, S5-65: description link to Invitations; count in the toolbar; table skeletons; banners titled "Couldn't load members" and so on.
- [ ] Gates as Task 2 (no WIDE line for the lane's own captures, 50-54). Commit.

**Orchestrator verification:** gates; `npm run screens`; compare 50-54 before and after, both themes, both widths; `sips` width 390 on 52 and 53 light-390 and dark-390; in a browser at 390 change a member's role from the stacked row.

### Task 5 (P9-L5, W): grades

Spec §4.14-§4.17, §5.6. After Task 1.

**Scope:** `apps/web/src/features/grades/**`, `apps/web/src/routes/$slug/grades.tsx`, `grade-scale.tsx`, `sections_.$id.grades.tsx`. Not the em dash separators in `gradebook.tsx` and `my-grades.tsx` (Task 6).

**Items:** S5-52, S5-53, S5-56, S5-55, S5-54, S5-59, S5-58, S5-60, S5-62, S5-63, S5-69, S5-65, S5-64 (table), S5-71 (use), S5-61, S5-57.

- [ ] S5-52: failing `gradebook.test.tsx` and `finalize-dialog.test.tsx`: the row header and the Student `th` carry `max-w-48` and `truncate` below `sm`, full name in `title`; then the classes. Verified at 390 by `screens` (60-63).
- [ ] S5-53, S5-69: failing tests: an empty cell of a published assessment shows "Missing"; Percent shows "(1 missing)"; a student without a name shows one email line; then the changes.
- [ ] S5-56, S5-57: failing tests: with final grades, "Finalized" badge and "Running %", "Running letter" headers; TA note text; then the changes.
- [ ] S5-54, S5-55: header `text-right`; footer `th` `bg-surface`; letters with count 0 omitted (test the tally string).
- [ ] S5-59: change `finalize-dialog.test.tsx:67` to "Keep F" (spec §5.7) and add: no Letter column; missing rows first with "1 missing"; "Showing 8 of 8"; then the changes.
- [ ] S5-58: failing `assessment-dialog.test.tsx`: weight helper shows the other assessments' total; inputs carry `w-32`/`w-48` after `inputClass`; then the change.
- [ ] S5-60, S5-61, S5-71: failing `my-grades.test.tsx`: "Graded credits" and the in-progress line from the cards; disclosures use `Disclosure`; then the changes and the card alignment.
- [ ] S5-62, S5-63: failing `scale-editor.test.tsx`: Range column values from the next row; "Unsaved changes" after an edit and gone after save; Save in the header; then the changes.
- [ ] S5-64, S5-65: table skeleton on the gradebook; banners titled "Couldn't load the gradebook" and so on.
- [ ] Gates as Task 2 (no WIDE line for the lane's own captures, 60-66). Commit.

**Orchestrator verification:** gates; `npm run screens`; compare 60-66 before and after, both themes, both widths; `sips` width 390 on 60-63; in a browser at 390 enter a score for the last assessment column and finalize from the dialog.

### Task 6 (P9-L6, W): entry pages, one search, separators and e2e selectors

Spec §4.1 (S5-01), §4.2-§4.4, §5.6 (S5-70), §5.7. After Tasks 2-5 land; needs the e2e selector ruling (spec §8, decision proposal 2) in PLAN.md.

**Scope:** `apps/web/src/features/auth/**`, `apps/web/src/routes/sign-in.tsx`, `apps/web/src/routes/invite.$token.tsx`, `apps/web/src/components/app-shell.tsx` (search only), `apps/web/src/features/catalog/catalog.tsx` and `catalog.test.tsx`, the separator lines in `features/courses/{course-dialog,course-list,section-dialog,section-detail}.tsx`, `features/grades/{gradebook,my-grades}.tsx` and their tests, `features/courses/section-list.test.tsx` (it reads the section dialog's course options), `apps/web/e2e/helpers.ts`, `apps/web/e2e/staffing.spec.ts`, `apps/web/e2e/screens.ts`.

**Items:** S5-07, S5-01, S5-10, S5-12, S5-08, S5-70, S5-11, S5-09.

- [ ] S5-07: change the e2e selectors of spec §5.7 to the single field first (suite red), then failing `sign-in-page.test.tsx`: one input; providers and magic link act on its value; non-email value with the magic link shows the inline error; then the page. e2e green.
- [ ] S5-01: change `staffing.spec.ts:212` to the header field (red); failing tests: on the Catalog the header field shows `q`; the Catalog has no Search input; then the change. e2e green.
- [ ] S5-70: change the e2e strings at `staffing.spec.ts` 140-141, 159-160, 385-386 and `screens.ts:209` (red); then each separator in the listed files and their unit tests (not `schedule.tsx`, which Task 2 owns). e2e green.
- [ ] S5-10, S5-11, S5-12, S5-08, S5-09: failing tests per row (resend re-posts and disables for 30 s with fake timers; inline link; invite info row; hint text; "Last used" badge from `localStorage`); then the changes.
- [ ] Gates: lint, typecheck, test, build, e2e, screens: the WIDE list is empty. Commit.

**Orchestrator verification:** gates; `npm run screens` with an empty WIDE list (the spec §1 width criterion); compare 01, 02, 03, 30, 31, 32, 41, 44, 45, 60, 65 before and after in both themes; sign in through the single field by magic link and by a provider in a real browser; reviewer re-scores every screen against spec §3 for the §1 criterion and the orchestrator records the after-scores in PLAN.md.

### Task 7 (P9-L7): direction items - blocked on owner §6

Spec §6. Scope and steps are cut from the owner's answers; a designer lane first for any answer that changes Spec 3 (screenshots to the owner before re-tokenising, D57). Nothing in Tasks 1-6 waits on it.

### Task 8 (P9-L8, D then W): display names (C1) - owner to confirm

Spec §7 C1. API only; may run in parallel with any web task (disjoint `apps/api` scope). Designer writes `docs/design/P9-L8.md` (the import write rule, `PATCH /api/v1/me` body, validation and length, audit entry, RLS). Roster rows for people without an account become invitations (D21: `pending_enrollments`, no `users` row), so the CSV name must be stored on the invitation and applied at sign-in with the same fill-only-if-empty rule; likely a migration, which the design covers.

**Scope:** `apps/api/migrations/` (new file, if the design adds the invitation name column), `apps/api/internal/course/import.go` and tests, the sign-in path that applies pending invitations and its tests, the package that serves `GET /api/v1/me` and its tests, `apps/api/internal/httpx/routes.go`, `apps/api/openapi.json` (hot spots: single lane), `apps/web/src/api/types.ts` (regenerated only).

- [ ] Failing test: a roster import row with a name fills an empty `users.display_name` and never overwrites a set one; then the write in the import transaction.
- [ ] Failing test: a roster row with a name for an email without an account stores the name on the invitation, and signing in with that email fills the new user's `display_name`; then the change.
- [ ] Failing test: `PATCH /api/v1/me` sets the caller's name (trimmed, 1-100 characters, 400 otherwise), audited; then the handler and route; regenerate `types.ts`.
- [ ] Gates: gofmt, `go vet`, `go test` against Postgres; web gates unchanged and green. Commit.

**Orchestrator verification:** re-run the API suite; over real HTTP import a roster with names and read them back on `GET /api/v1/universities/{slug}/sections/{id}/enrollments`; `npm run screens` shows names on 45, 52, 60 instead of emails.
