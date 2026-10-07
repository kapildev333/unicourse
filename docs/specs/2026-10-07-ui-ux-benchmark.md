# Spec 5: UI/UX benchmark - every screen against references, and the plan to improve it

Status: written by the orchestrator's synthesis lane (P8-L3d) under D75-D78, owner to confirm · Date: 2026-10-07 · Builds on Spec 3 (binding) and Spec 4 (grading screens)

Sources: the three comparison notes `docs/design/P8-L3a.md` (entry, shell, My sections, Catalog), `docs/design/P8-L3b.md` (Courses, section detail, dialogs, admin tables), `docs/design/P8-L3c.md` (grading, states, dark mode, phone width); the reference pack and rubric `docs/design/P8-L2.md`. Every finding below cites a capture by its base name (files are `<base>-<light|dark>-<1440|390>.png` in `apps/web/e2e/.tmp/screens/`) and a reference from the pack. "Note a 2.6" means P8-L3a §2.6.

## 1. Intent and success criterion

Owner's ask (2026-10-07): "utilize Dribbble and professional websites for UI/UX, compare each and every screen of ours, generate a plan to make it better". Phase 8 delivers this document and its plan; Phase 9 builds the plan once the owner has read it and answered §6 (D75).

**Success criterion.** After Phase 9 (lanes P9-L1 to P9-L6), a fresh `npm run screens` run produces captures in which:
- every criterion a planned item addresses scores at least 4 on the pack §4 rubric, re-scored by the reviewer from the new captures against the "before" scores in §3;
- no 390 px capture, light or dark, is wider than 390 px (asserted by the capture script, §9);
- the web gate (`lint`, `typecheck`, `test`, `build`) and `npm run e2e` are green with no e2e edit outside the list in §5.7.

Criteria scored "n/s" (not observable in a still image) and items blocked on §6 or §7 are not part of the criterion.

## 2. Out of scope

- New product features beyond the C items in §7, and C items the owner does not pick.
- A change of visual direction (palette, type, density, icon set, radius, motion) without the owner's pick from screenshots (D57 lesson, D75, D77). Those questions are §6.
- Everything Spec 3 §10 deferred that this spec does not touch: theme stored on the account, high-contrast theme, print stylesheet, axe in CI. The mobile drawer stays deferred (§6 B10).
- Bulk actions, global app-wide search, per-student grade release.

## 3. Method

| Step | What | Source |
|---|---|---|
| Captures | `npm run screens` (Playwright project `screens`, not run by `e2e` or CI) seeds one university with every role, then captures 37 screens at 1440x900 and 390x844 in light and dark (dialogs skip dark 390): 141 files | D76, P8-L1, `apps/web/e2e/screens.ts` |
| References | Pack P8-L2: 56 Dribbble preview images seen (descriptions only, no asset copied), Primer, Stripe, Vercel Geist, GitHub, Google Classroom, Coursera, Canvas, Linear, Notion pages; claims not read this session are marked "product knowledge" | D77 |
| Rubric | Pack §4: 13 screen types, 6-7 criteria each, 1-5 (5 matches the row, 3 present but inconsistent, 1 absent or broken); the lower of both themes and both widths | D78 |
| Scoring | Three lanes split the screens (P8-L3a, b, c); each read the light-1440 captures of its screens, the dark-1440 and light-390 of a subset, and gave a score only where the still image shows it | notes §0 |
| Measurement | The orchestrator measured the 390 captures: schedule 755 px, gradebook 550 px, Members 561 px wide; P8-L3c measured all 37 light-390 captures with `sips`: 11 are wider than 390 | §5.5 |

Scores as the lanes gave them ("n/s" not scored from a still, "-" not applicable):

| Screen | Captures | Rubric | 1 | 2 | 3 | 4 | 5 | 6 | 7 |
|---|---|---|---|---|---|---|---|---|---|
| Shell | 10, 20, 30, 45, 66 | (b) | 4 | 3 | 4 | 3 | n/s | 3 | - |
| Sign-in | 01 | (a) | 2 | see 02 | n/s | 5 | 4 | 5 | - |
| Check your email | 02 | (a) | - | 3 | - | 5 | - | 5 | - |
| Invite link | 03 | (a) | 2 | - | - | 5 | - | - | - |
| No access | 04 | (k) | - | 4 | 3 | - | - | - | - |
| First-use empty | 10, 11, 12, 13 | (k) | n/s | - | 3 | 4 | 5 | - | - |
| My sections list | 20, 22 | (c) | 4 | 3 | 4 | 4 | 3 | 4 | - |
| Schedule | 21, 23 | (d) | 3 | 3 | n/s | 3 | 1 | 4 | - |
| Catalog | 30, 31 | (e) | 4 | 4 | 4 | 3 | n/s | 3 | - |
| Catalog no match | 32 | (e)4, (k)4 | - | - | - | 3 | - | - | - |
| Courses list | 40 | (g) | 3 | 4 | 4 | - | 3 | - | 1 |
| Course dialog | 41 | (h) | 5 | 4 | - | - | 5 | 3 | - |
| Remove dialogs | 42, 47 | (h) | 5 | - | 4 | - | 5 | 3 | - |
| Sections list | 43 | (g) | 3 | 4 | 4 | - | 3 | - | 1 |
| Section dialog | 44 | (h) | 5 | 3 | - | - | 5 | 1 | - |
| Section detail | 45 | (f) | 4 | 2 | 3 | 5 | 3 | 1 | - |
| Roster CSV import | 46 | (h) | 5 | 4 | - | 4 | 5 | 3 | - |
| Departments | 50 | (g) | 3 | 5 | 4 | - | 4 | - | 2 |
| Terms | 51 | (g) | 3 | 4 | 4 | - | 4 | - | 2 |
| Members | 52 | (g) | 2 | 3 | 4 | - | 3 | - | 1 |
| Invitations | 53 | (g) | 3 | 4 | 5 | - | 3 | - | 2 |
| Audit log | 54 | (g) | 3 | 4 | 4 | - | 3 | 3 | 2 |
| Gradebook | 60 | (i) | 4 | 3 | 4 | 4 | 4 | 5 | 1 |
| Gradebook finalized, TA | 63, 64 | (i) | 4 | 2 | - | - | 4 | - | 1 |
| Assessment dialog | 61 | (h) | 5 | 5 | 4 | - | 4 | n/s | - |
| Finalize preview | 62 | (h), (i)5 | 5 | - | - | - | 4 | 1 | - |
| My grades | 65 | (j) | 3 | 5 | 4 | 2 | 4 | 5 | - |
| Grade scale | 66 | (g) | 2 | 2 | 3 | - | - | - | n/s |
| Loading, error | 70, 71 | (k) | 2 | 3 | 4 | 4 | 5 | 4 | - |
| Dark mode | 01, 20, 30, 45, 52, 60 dark | (l) | 4 | 5 | 5 | 5 | 4 | 4 | - |
| Phone width | all light-390 | (m) | 1 | 3 | 4 | 1 | 2 | 5 | - |

The grade-scale page also scores 1 on shell (b)1: no nav item is current on `/{slug}/grade-scale`. Phone scores for 41-43, 46, 47, 50, 51, 53, 54 are inferred from shared components (note b §0).

## 4. Findings per screen

Columns: impact H/M/L, effort S/M/L, category A (inside Spec 3 tokens and components), B (owner direction, §6), C (product or API gap, §7). "Copy" marks a copy change whose selectors are listed in §5.7. Lane is the Phase 9 lane of §8.

### 4.1 Shell (10, 20, 30, 45, 66; 20 and 30 at 390)

Works: Udemy-like bar with bold wordmark, hairline, text nav with underline; labelled three-way theme switch; dark header one surface step above the canvas; identical page-header rhythm on every page.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-01 | One search: the header field shows the current `q` on the Catalog; the Catalog page's own "Search" input is removed, its filter row keeps Term and Department. Copy | H | M | A | Primer navigation ("minimize navigational elements"); Linear, Stripe search | L6 |
| S5-02 | `/` focuses the header search when focus is not in a field; input gets `aria-keyshortcuts="/"` and `title="Press / to search"` | M | S | A | Linear, Stripe Dashboard search | L1 |
| S5-03 | Search in a fixed slot: shell grid `grid-cols-[auto_1fr_auto]`, search `max-w-xl` centred, so its left edge does not move with the nav length (x 544 student, x 673 admin) | L | S | A | Linear, Stripe | L1 |
| S5-04 | Below 768 px: wordmark, theme group and Sign out on row one, nav on row two, search on row three (about 150 px instead of 205); nav links `min-h-11`. Merges note a 1 #4 and note c 3 #6 | M | S | A | pack (m)2, (b)6 | L1 |
| S5-05 | Current nav item on every route: "Courses" on `/{slug}/sections/{id}` for staff-side viewers, "Admin" on `/{slug}/grade-scale` for admins. Merges note b 2.6 #6 and note c 1.6 #5 | M | S | A | Primer UnderlineNav, heuristic H20 | L1 |
| S5-06 | `TabNav` strip scrolls inside itself (`overflow-x-auto`, scroll-snap) so "Audit log" never pushes the page wider at 390 (52) | M | S | A | Primer UnderlineNav overflow | L1 |

### 4.2 Sign-in (01)

Works: full-width provider buttons with icon and text; labels above inputs; one purple element; the 390 layout holds.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-07 | One field: "University email or university ID" is the only input; the provider buttons and the primary "Email me a sign-in link" act on it; a non-email value gives the inline error "Enter your email to get a sign-in link". Removes the second "Email" field and the "or" divider. Copy | H | M | A | Vercel login; Talebook 4080832 | L6 |
| S5-08 | Hint "For example alice@example.edu, or your university's short name such as northfield" (today's "or example" reads as a typo). Copy | M | S | A | heuristic H2 | L6 |
| S5-09 | Remember the last method in `localStorage` key `unicourse:last-sign-in`; `Badge` "Last used" inside that button | L | S | A | Duwal social-login sheet; TikTok "Last login" | L6 |

### 4.3 Check your email (02)

Works: states the address, single use and 15-minute expiry; neutral wording does not reveal which emails exist.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-10 | Secondary `Button` "Send the link again" re-posts the same email, shows `StatusMessage` "Sent again.", disabled for 30 s with the reason as text. Copy | M | S | A | superdesign magic-link card | L6 |
| S5-11 | "use a different email" becomes an inline text link (the link style of 04), removing the 16 px gap after "or" | L | S | A | heuristic H4 | L6 |

### 4.4 Invite link (03)

Works: the email is prefilled; the line names the address to sign in as.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-12 | Invitation line becomes an info row above the form: Lucide `mail-open`, ink text, "You've been invited. Sign in as {email} to accept.", the email in `whitespace-nowrap` (it broke at its hyphen). Merges note a 2.3 #1, #2, #4 (single field via S5-07) | M | S | A | GitHub and Vercel invitations (product knowledge) | L6 |

### 4.5 No access (04)

Works: the heading says what happened in product words; two ways out.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-13 | Lead the link list with "Go to your university:"; with one membership a primary `Button` "Go to Harbor College" and Sign out secondary; the quoted slug becomes "this university's address". Merges note a 2.4 #1, #2. Copy (e2e reads the heading only) | M | S | A | Primer empty states | L1 |

### 4.6 First-use empty states (10, 11, 12, 13)

Works: one shared `EmptyState` (icon, bold title, one muted line); 12 echoes its title with "New course"; copy says when data appears.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-14 | Admins on "No terms yet" (10, 11, 13) get `EmptyState` action, secondary `Button` link "Set up terms" to the admin Terms tab; others get no action. Copy | H | S | A | Stripe empty state; Spec 3 §5.4 | L2 (10, 11), L3 (13) |
| S5-15 | 13: with no terms, hide "New section", "Import sections" and the empty Term select | M | S | A | heuristic H5 | L3 |
| S5-16 | 12: hide the Department filter while there are no courses and no filter; one primary visible (empty state holds "New course" primary and "Import courses" secondary, header pair hidden while empty); header "New course" stays enabled while the list loads (70). Merges note a 2.5 #3, note c 1.7 #4, #5 | M | S | A | Primer empty states (one primary action) | L3 |
| S5-17 | Courses layout renders `PageHeader` above `TabNav` (title first, tabs under it) | L | S | A | Primer UnderlineNav; pack (f) | L3 |

### 4.7 My sections list (20, 22)

Works: card grid with code band, bold title and text role badge; dark band and surfaces hold; List/Schedule switch visible.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-18 | `Card`: body `flex-1`, badge row `mt-auto pt-3` so badges align across a row; band `h-24` becomes `h-16` at every width. Merges note a 2.6 #1, #2 and note c 3 #7 (note c called the phone band a B item; Spec 3 fixes no band height, so it is A here) | M | S | A | Dribbble 2941299 (Udemy card concept) | L1 |
| S5-19 | Instructor line shows "You" in place of the viewer's own address. Copy | M | S | A | heuristic H8 | L2 |
| S5-20 | `formatMeetings` in `lib/schedule.ts` merges lines whose time and room repeat ("Mon, Wed 10:00-11:20 · Hall B 204") | M | M | A | Canvas course cards (one meta line) | L2 |
| S5-21 | Term picker options show the term name only; the h2 keeps the dates (option text truncated at 390). Copy | L | S | A | rubric (c)4 | L2 |
| S5-22 | Instructor and TA cards get a foot link "Gradebook" (Lucide `table`) to the section's grades page, outside the card's main link (`Card` gains a `footer` slot in L1) | M | M | A | Google Classroom class cards | L1 slot, L2 use |
| S5-23 | List/Schedule becomes a `SegmentedControl` (new shared component): selected `bg-accent-soft text-ink` with `aria-pressed`, so it no longer looks like a primary action (20-dark) | M | M | A | Linear, Vercel (product knowledge) | L1 component, L2 use |

### 4.8 Schedule (21, 23)

Works: week grid with blocks sized by duration; each block a link with the code first; dark hairlines legible.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-24 | Below 768 px an agenda: one region per weekday with meetings, `h3` day name, rows "10:00-11:20 CS101-001 Introduction to Computer Science Hall B 204", each a link; the grid only from `md`, inside `overflow-x-auto`. Fixes the 755 px page. Merges note a 2.7 #1, #2 and note c 3 #3 | H | M | A | Canvas Calendar; Michalczyk 3737562; Nikitin 5182836; Tsang 8051313 | L2 |
| S5-25 | Drop Sat and Sun columns when no meeting in the term falls on them (28 percent of the grid is empty in 21, 23) | M | S | A | Kamushken 5715370; Szabaciuk 16249040 | L2 |
| S5-26 | Block content: code bold, time `whitespace-nowrap`, title, room; the instructor line leaves the block | M | S | A | Michalczyk 3737562 | L2 |
| S5-27 | Today's column header bold with `Badge` "Today" and `aria-current="date"`; no fill | M | S | A | Kamushken 5715370 | L2 |
| S5-28 | Current-time line in today's column (1 px `line-strong`, muted time label), updated each minute | L | M | A | Kamushken 5715370 | L2 |

### 4.9 Catalog (30, 31, 32)

Works: labelled filter row; announced count; same `Card` as My sections; filtered-empty has its own title and icon and no create action.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-29 | `Button` ghost sm "Clear filters" at the end of the filter row when department or query is set (keeps the term); no-match title `No sections match "{q}"` (department name when only a department is set) with a secondary "Clear filters" action. Merges note a 2.8 #1, 2.9 #1 and note c 1.7 #3. Copy | H | S | A | Stripe filter controls; Vercel Geist (quote the query) | L2 |
| S5-30 | `Badge` "Enrolled" (or "Instructor", "TA") on cards in the viewer's sections, from the `mine` query the page already loads | H | S | A | Coursera search results (text badge) | L2 |
| S5-31 | Count reads "Showing 5 sections". Copy | L | S | A | rubric (e)2 | L2 |
| S5-32 | Below 768 px every filter control is `w-full` (today three widths) | L | S | A | rubric (e)6 | L2 |

### 4.10 Courses and Sections lists (40, 43)

Works: one `Table` everywhere, 48 px rows, bold header on mist, verbs as row actions with accessible names; code-over-title cell; dashes for empty cells.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-33 | Filter selects `w-full sm:w-72` in a `flex flex-wrap items-end gap-4` row (the Audit log's layout) instead of 1296 px wide. Merges note b T4 and note a 2.5 #4 | M | S | A | Stripe filter controls; Ralabs log | L3 |
| S5-34 | "Showing n" moves to the toolbar row above the table, right-aligned (`LoadMore` gains a prop to omit its count). Same copy. Note b T3; applies to 40, 43, 50-54 | L | S | A | Untitled UI table header ("48 users") | L1 prop, L3, L4 |
| S5-35 | `Table` `stack` mode below 640 px: each `td` has `data-label`, rows render as blocks (first cell bold, badges top right, actions full width at the foot). Used by 40, 43, 45 roster, 46 preview, 50-53; the Audit log keeps scroll (S5-51). Merges note b T2, 2.7 #3 and note c 3 #4 | H | M | A | Shyaula, Honeycutt 15436288 (pack §3.11) | L1 mode, L3, L4 |
| S5-36 | 43: Instructors and TAs cells are plain `text-sm` text joined by commas, pending names muted with "Invited", more than two shown as "{first} +2"; no bordered chips | M | S | A | Olga Sipatova table; Untitled UI | L3 |

### 4.11 Dialogs (41, 42, 44, 46, 47)

Works: one rhythm (bold title, muted description, hairline footer, Cancel first, verb primary); destructive confirms name the object, focus starts on Cancel; native `<dialog>` traps focus and returns it.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-37 | Below 640 px `Dialog` is `w-[calc(100%-2rem)]`, `max-h-[calc(100dvh-2rem)]`, body scrolls, footer stays visible (today edge to edge, 44-390) | M | S | A | rubric (h)6 | L1 |
| S5-38 | Icon-only close button in the dialog header: Lucide `X`, `ghost`, `aria-label="Close"`, `title="Close"` | L | S | A | Sergushkin invite modal; Primer dialog | L1 |
| S5-39 | 42: the body states what happens to the course's sections, worded from the API's actual behaviour (worker checks first) | M | S | A | Primer ConfirmationDialog | L3 |
| S5-40 | 44: meeting row `grid-cols-2 sm:grid-cols-[8rem_7rem_7rem_minmax(0,1fr)_auto]` (on phone Day and Location span 2, Starts and Ends share a row, Remove under them); the section dialog opens `wide` so Location gets at least 16 rem ("eering Building 215" is cut at 1440). Decision proposed in §5.7 (amends D69) | H | S | A | Sergushkin invite modal (full-width inputs) | L3 |
| S5-41 | 46: keep the summary sentence (it is the `role="status"`), delete the counter row; render the Message column only when a row has a message | M | S | A | NING batch import; Attio | L3 |

### 4.12 Section detail (45)

Works: title band answers code, title, term and instructor first; back link; plain complete meeting lines; dark holds.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-42 | Add-by-email row: `flex flex-wrap items-start gap-2`, input `min-w-64 flex-1`, helper under the row, buttons top-aligned; below `sm` input full width above the two buttons. Source of the 555 px page. Merges note b 2.6 #1 and note c 3 #1 | H | S | A | rubric (m)1 | L3 |
| S5-43 | Roster `h2` and "Import roster" on one row (`flex items-center justify-between`) | M | S | A | Untitled UI; Primer title/actions slot | L3 |
| S5-44 | Staff and Roster headings carry a muted count (`<span class="text-ink-muted font-normal">8</span>`); e2e heading matches are substring, so they still pass | M | S | A | Coursera tabs; Primer UnderlineNav counters | L3 |
| S5-45 | "Edit section" secondary button (Lucide `Pencil`) as the band's `PageHeader` action for roles that may edit; opens the existing `SectionDialog` | M | S | A | rubric (f)5 | L3 |

### 4.13 Admin (50, 51, 52, 53, 54)

Works: bold `TabNav` with `aria-current`; one-sentence descriptions; role and status as text badges; Revoke only on pending rows; the Audit log's entity, actor and date filters match GitHub's qualifiers.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-46 | 51: Status column with `Badge` "Current", "Upcoming" or "Past" from the dates (neutral tone) | M | S | A | heuristic H6 | L4 |
| S5-47 | 52: "Invitations" in the description becomes a link to the Invitations tab, same copy | L | S | A | Untitled UI "Add user" on the header | L4 |
| S5-48 | 53: Pending first, then by expiry; Expires shows a muted dash on accepted rows | M | S | A | GitHub people list; Mercury log (dash for no value) | L4 |
| S5-49 | 53: Status filter select ("All", "Pending", "Accepted") in the S5-33 row | L | S | A | Dhendup toolbar | L4 |
| S5-50 | 54: action labels in the past tense throughout ("Invitation accepted"), worded from what each event means. Copy (audit-log unit tests) | M | S | A | GitHub audit log | L4 |
| S5-51 | 54 at 390: keeps horizontal scroll with When as a sticky first column (`sticky left-0 bg-surface`) | M | S | A | rubric (g)7 | L4 |

### 4.14 Gradebook (60, 63, 64)

Works: header cell carries title, max, weight and publish state (richer than Revelski 10832693); right-aligned server-computed percent; one primary action; Final column pairs bold letter with muted points; TA toolbar shows only TA tools; dark holds.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-52 | Sticky Student cell (header and body `th`) capped at `max-w-48` with `truncate` below `sm`, full text in `title`, in the gradebook and the Finalize dialog, so scores show beside it at 390 (today 550 px page, no score visible). Merges note c 1.1 #1, 1.3 #3, 3 #2 | H | S | A | Vass 9226089 (narrow frozen column) | L5 |
| S5-53 | Empty cell of a published assessment shows a muted "Missing" caption; Percent gains "(1 missing)" as the Finalize dialog already does (lucas.moreau reads 81.75% B- running, 32.70% F at finalize) | H | S | A | Google Classroom (missing labelled) | L5 |
| S5-54 | Assessment header stack `text-right` to the input edge; Student `w-auto` (400 px gap at 1440) | M | S | A | Canvas gradebook; Gaurav Mathur 11064516 | L5 |
| S5-55 | Sticky footer `th` uses `bg-surface` (two-tone strip, louder in 60-dark); "Letters:" omits zero counts. Merges note c 1.1 #5 and 2 #1 | M | S | A | heuristic H8 | L5 |
| S5-56 | When final grades exist: `Badge tone="success"` "Finalized" beside the description, and headers "Running %" and "Running letter". Copy | M | S | A | Canvas Current/Total; Google Classroom returned state | L5 |
| S5-57 | TA note when finalized: "Final grades are recorded. Score changes do not change them until an instructor reopens grades."; TA header stack uses the instructor's three-line structure. Copy | L | S | A | heuristic H5 | L5 |

### 4.15 Assessment dialog (61) and Finalize preview (62)

Works: helper copy explains max points and the 100 rule; destructive action separate and never purple; the Finalize consequence sentence names both effects; the confirm label counts ("Finalize 8 grades").

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-58 | 61: Max points and Weight `w-32`, Due date `w-48` (P7-L2 §3.4 asked this); Weight helper appends "Other assessments total 60%." | M | S | A | Ben Low 7486471 | L5 |
| S5-59 | 62: Grade select default reads "Keep {letter}" and the separate Letter column goes; rows with missing scores carry `Badge tone="warning"` "1 missing" and sort first; the scroll area gets a bottom border and "Showing 8 of 8". Copy (`finalize-dialog.test.tsx` line 67) | M | S | A | Primer ConfirmationDialog; Google Classroom | L5 |

### 4.16 My grades (65)

Works: summary plus per-term GPA line; "so far" and "Current" label running grades; the scale explained in place; nothing overflows at 390.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-60 | Summary "Credits" becomes "Graded credits" with a muted line "Counts finalized sections only. 11.0 credits in progress." (derived from the cards) | M | S | A | Google Classroom (explains a hidden total) | L5 |
| S5-61 | Letter line a fixed two-line block so final and running cards align; summary card at content width with Term GPA beside Cumulative | L | S | A | pack (c)2; Canvas grades | L5 |

### 4.17 Grade scale (66)

Works: the description states the effect on recorded grades; presets load in one click; an editable grid without a dialog per row.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-62 | Letter `w-20`, Minimum percent and Grade points `w-28 text-right`, Remove column right-aligned; read-only "Range" column from the next row's minimum ("90 to 92.99") | M | S | A | Gaurav Mathur 11064516; Canvas scale | L5 |
| S5-63 | "Save scale" in the `PageHeader` action slot, presets and Add row as secondaries in the toolbar, `StatusMessage` "Unsaved changes" while the draft differs | M | S | A | Primer saving; pack §3 item 5 | L5 |

### 4.18 Loading and error (70, 71)

Works: shell, tabs, filters and actions render first and only the content region waits or fails; error banner uses danger-soft, an icon and retry, no codes.

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-64 | `Loading` gains `variant="table"` (real header row from the caller's column names over 5 rows of bars sized per column) and `variant="cards"` (3 card-shaped blocks); default kept for small regions | H | M | A | Gopinathan 4361464 | L1, adopted by L2-L5 |
| S5-65 | Load-failure banners name what failed: callers pass `title="Couldn't load sections"` (and so on), so title and body no longer both say "Something went wrong". Copy (check e2e per lane) | M | S | A | Primer error messaging | L2-L5 |

## 5. Cross-cutting findings

### 5.1 Shell
Covered by S5-01 to S5-06. The shell shows no identity (email only in the Sign out tooltip, D65); that waits on B2 and C1.

### 5.2 Tables
One `Table` is the strength; the weaknesses are the phone collapse (S5-35), the count below the table (S5-34), full-width filters (S5-33) and a red word on every row (B6). Dates stay left-aligned (Spec 3 §5.5 asks right alignment for numbers only).

### 5.3 Dialogs
Rhythm and focus are at reference level everywhere (rubric (h)1 and 5 score 5). Phone fit (S5-37) and the meeting row (S5-40) are the gaps. Backdrops covering only the first viewport in tall captures (45-47, 54, 62) are a full-page screenshot artifact, not a finding.

### 5.4 Dark mode
Elevation, link tint and badges hold (rubric (l) 4-5). Items: S5-55 (footer strip), S5-23 (selected toggle reads as a primary), and:

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-66 | Secondary button border (`border-ink` today, near-white mist in dark, louder than the lilac primary in 60-dark and 20-dark) moves to a new token `line-control`: the ink value in light (no visible change), the `line-strong` value in dark, measured at 3:1 (Spec 3 §7); CLAUDE.md's token list gains it | M | S | A | Linear, Vercel dark (product knowledge) | L1 |
| S5-67 | The capture script adds one success and one warning state in dark (a `StatusMessage`, the weight warning badge) so (l)5 can be scored; the orchestrator runs the P6-L1 design §1.3 contrast script on dark `ink-muted` over `surface` | L | S | A | rubric (l)5 | L1 |

### 5.5 Phone width
Eleven of 37 light-390 captures are wider than 390 px: 21 and 23 (755), 43 and 44 (693), 45, 46, 47 (555), 52 (561), 53 (679), 60-63 (550). Each has a named fix: 21, 23 by S5-24; 43 by S5-35 and S5-36; 44 by S5-40; 45-47 by S5-42 and S5-35; 52, 53 by S5-35 and S5-06; 60-63 by S5-52. The rule and its gate:

| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-68 | No page wider than the viewport: `Table`'s scroll wrapper sits in a `min-w-0` parent; the capture script asserts `document.documentElement.scrollWidth <= 390` for every 390 capture with `expect.soft`, so one run lists every offender | H | M | A | rubric (m)1; Spec 3 §2 ("nothing overflows") | L1 |

### 5.6 People and copy
| Id | Change | Impact | Effort | Cat | Reference | Lane |
|---|---|---|---|---|---|---|
| S5-69 | Until C1: a person without a display name shows the email once, in ink. Members drops the dash line (rows back to 48 px); Roster's Name cell shows a muted dash instead of repeating the email; gradebook and Finalize drop the second email line. Merges note b P1 and note c 1.1 #2 | M | S | A | Sergushkin and Untitled UI (name over email) | L3, L4, L5 |
| S5-70 | Separator em dashes become " · " (or ", " in sentences): Catalog and course dialog department options, course list filter options, section dialog course options and context line, section detail band, gradebook description, My grades assessment lines, schedule blocks. Lone dash placeholders in empty cells stay. Copy | L | S | A | heuristic H22; taste-skill §9 | L6 |
| S5-71 | One disclosure pattern: a shared `Disclosure` (`summary` with `list-none` and Lucide `ChevronRight`/`ChevronDown` swapped by open state, no rotation) replaces the native triangle in the Audit log and My grades. Merges note b 2.12 #3 and note c 1.5 #5 | L | S | A | Spec 3 §3.3 (no Unicode glyph icons) | L1, used by L4, L5 |

### 5.7 Copy changes that break selectors, and the ruling needed

| Item | Selector today | New copy or target |
|---|---|---|
| S5-07 | `e2e/helpers.ts:47` and `e2e/screens.ts:91` `getByLabel('Email', { exact: true }).fill`; `e2e/staffing.spec.ts:38` and `e2e/screens.ts:137` `toHaveValue`; `staffing.spec.ts:81` `getByLabel('Email')` | `getByLabel('University email or university ID')` |
| S5-01 | `staffing.spec.ts:212` `getByLabel('Search').fill('MA201')` | header field `getByRole('searchbox', { name: 'Find a course' })` (or its label) then Enter |
| S5-70 | `staffing.spec.ts:140-141` (department option "CS, em dash, Computer Science"), `159-160` (course option "CS102, em dash, Data Structures"), used at 145 and 164 by `selectOption({ label })`; `385-386` ("Midterm, em dash, 95 / 100 · 40%"); `screens.ts:209` (course option) | the same strings with " · " in place of the em dash |
| S5-59 | `finalize-dialog.test.tsx:67` `'Computed: F'` (unit only) | `'Keep F'` |
| S5-29 | `catalog.test.tsx` "No sections match" (unit only; check exact matches) | `No sections match "{q}"` |
| S5-08 | `sign-in-page.test.tsx` (types "example", reads no hint) | none expected |

**Decision proposed (like D60):** in Phase 9 the e2e suite and unit tests may change exactly the selectors in this table, for the same assertions, in the lane that changes the copy (P9-L6, or P9-L5 for unit tests in `features/grades`); any other e2e edit stops the lane and returns to the orchestrator.

**Decision proposed:** the section dialog opens `wide` instead of `lg` (S5-40), amending D69, because `lg` truncates Location at 1440.

**Decision proposed:** Spec 3 §2 ("a mobile layout below 768 px beyond nothing overflows") is extended for the schedule agenda (S5-24) and stacked table rows (S5-35), and "no capture wider than 390" becomes a gate of every Phase 9 lane.

## 6. Direction questions for the owner (B)

Nothing in the A plan depends on these. Each line: question, screens, recommendation and reason.

| Id | Question | Screens | Recommendation |
|---|---|---|---|
| B1 | One colour per course (or department) on the card band and schedule block (pack Q7) | 20-23, 30 | No for now: D57 keeps purple single; several colours each need a contrast check in two themes. Revisit when students ask to tell courses apart |
| B2 | User menu with name, role, theme and Sign out instead of the bare Sign out (pack Q5; GitHub keeps theme in settings) | every signed-in page | Yes, after C1: without names it would show an email, which D65 already moved out of the bar |
| B3 | Provider brand marks (Google G, Microsoft squares) as inline SVG beside the Lucide set (D53) | 01, 03 | Yes as a named exception for two marks: providers are told apart by text alone today, cost is S |
| B4 | Header search stays Catalog search (D59) or becomes app-wide by role (pack Q4) | shell, 30 | Keep D59: app-wide search is a feature, not a restyle; S5-01 and S5-02 cover the reachable part |
| B5 | Section detail as real tabs with counts (Coursera, Primer) instead of stacked sections | 45-47 | No: S5-44 puts counts in the headings and the e2e reads all three headings on one page |
| B6 | Quiet destructive row actions: `ghost-danger` muted at rest, red on hover and focus (note b T5); also the grade-scale Remove as neutral ghost with Lucide `Trash2` (note c 1.6 #4, listed A there, moved here because it changes the same rule) | 40, 43, 45, 50-53, 66 | Yes: a red column is the loudest thing on every admin page; the word stays, so meaning never depends on colour |
| B7 | Phone dialogs as bottom sheets (pack Q3) | 41-47, 61, 62 | No: S5-37 (margins, scrolling body) first; a sheet is a new pattern for little gain |
| B8 | Row actions in a kebab menu instead of inline verbs | every table | No: inline verbs are fewer clicks and need no new Menu component |
| B9 | Tabular figures (D56, pack Q2): proportional digits misalign decimals in 60, 62, 63, 65, 66 | grading | First check whether the bundled DM Sans variable exposes `tnum` (one CSS line to test); if not, keep D56 rather than add a numerals font against D58 |
| B10 | Mobile navigation (drawer or bottom tabs, Spec 3 §10) | every page at 390 | Wait: S5-04 cuts the header to about 150 px; keep the Spec 3 §10 trigger |
| B11 | Purple as status: "Published" (TA, 64) and "Admin" (52) badges on `accent-soft` (note c 1.4 #4, listed A there, moved here because Spec 3 §3.1 allows accent-soft badges) | 52, 64 | Neutral for states (Published, Draft), accent-soft kept for roles: purple then means "you" or "act", not a state |
| B12 | Press motion `scale(0.98)` and 200-300 ms transitions (taste-skill) against Spec 3 §3.5 (150 ms colour only) | every button | Keep Spec 3: no finding shows motion as the problem |
| B13 | Taste-skill rules that contradict agreed decisions: Phosphor instead of Lucide, a 600 weight, borderless cards, varied radius (pack Q1, D78) | all | Keep the decisions (D53, D58, D57): no capture argues for reopening them |
| B14 | Gradebook draft and return per student (Google Classroom) instead of per-assessment publish plus finalize (pack Q6) | 60-64 | No: the only confusion seen is running vs final letter, fixed by S5-56 |

Not a direction item: pack Q8 (is one Dribbble preview frame enough to cite). No A item rests on a Dribbble shot alone; each also cites a product page or a rubric row.

## 7. Product gaps (C)

| Id | Gap | Screens | Backend or data work | Recommendation |
|---|---|---|---|---|
| C1 | Display names: people show as emails everywhere; `users.display_name` exists and every read returns it, but only an OIDC sign-in fills it (`auth.SignIn`); the roster CSV's `displayName` column is parsed and dropped | 20-23, 43-47, 52-54, 60-64 | The roster import (`/sections/{id}/enrollments/import`) writes the CSV name when the user's name is empty (same "fill only if empty" rule as `SignIn`); `PATCH /api/v1/me` (beside the existing `GET`) with `{displayName}` for the user's own name. No web change: `personName` already prefers the name. The self-edit field waits for B2 | Now, as P9-L8 (small API lane, owner to confirm) |
| C2 | Invite page names the university and role before sign-in | 03 | Public `GET /api/v1/invitations/{token}` returning university name and role for a valid unexpired token | Wait: the token in the URL is the secret, so exposure needs a security review first |
| C3 | Catalog sort by class code and a total ("of N") | 30, 31 | Order by code on the catalog query if not already; a total needs a count query beside keyset paging | Sort: check the API first, it may be the seed order only; total waits |
| C4 | Section count per course, course count per department | 40, 50 | Count columns on two list endpoints | Wait |
| C5 | Audit log Target column ("CS101 001 · Midterm") | 54 | Each entry returns a target label resolved by entity type | Next after C1: 20 identical "Assessment set" rows cannot be told apart (H impact) |
| C6 | Audit log "Showing 50 of N" | 54 | Total count | Wait |
| C7 | My grades "{n} assessments not yet published" | 65 | Count of unpublished assessments per section from `/me/grades` | Wait; fold into C5's lane if it runs |
| C8 | Bulk select and remove on Roster and Members | 45, 52 | Bulk endpoint or N calls plus an audit decision | Wait |

## 8. Ranked improvement plan

Waves: L1 lands first (shared pieces the feature lanes use); L2-L5 then run in parallel with disjoint scopes; L6 runs alone after them because it owns the copy that the e2e suite asserts. L7 waits on §6, L8 on the owner's yes. Exact scopes are in the plan.

| Lane | Scope (summary) | Items, by impact then effort | What the owner will see |
|---|---|---|---|
| P9-L1 shared components and shell (sub-lanes a components, b shell and capture script) | `components/`, `index.css`, `__root.tsx`, `$slug/route.tsx`, `e2e/screens.ts` | a: S5-68, S5-35, S5-64, S5-18, S5-37, S5-66, S5-22 slot, S5-23, S5-34 prop, S5-38, S5-71; b: S5-68 assertion, S5-04, S5-05, S5-06, S5-13, S5-02, S5-03, S5-67 | A shorter phone header, the right nav item always marked, aligned card badges with a slimmer band, dialogs with margins on phones, and a capture run that names every page that scrolls sideways |
| P9-L2 my sections and catalog | `features/my-sections/`, `features/catalog/`, `lib/schedule.ts`, their routes | S5-24, S5-29, S5-30, S5-14, S5-27, S5-26, S5-25, S5-19, S5-23 use, S5-20, S5-22 use, S5-65, S5-64 use, S5-28, S5-32, S5-31, S5-21 | The phone schedule as a day list, Clear filters and "Enrolled" in the Catalog, Today marked, a "Set up terms" button where admins hit an empty page |
| P9-L3 courses, sections and dialogs | `features/courses/`, their routes | S5-42, S5-40, S5-35 use, S5-14, S5-69, S5-16, S5-15, S5-33, S5-36, S5-41, S5-43, S5-44, S5-45, S5-39, S5-65, S5-64 use, S5-17, S5-34 | Section detail and the section dialog usable on a phone, tables as stacked rows, slim filters, one summary in the import |
| P9-L4 admin | `features/admin/`, their routes | S5-35 use, S5-51, S5-69, S5-48, S5-46, S5-50, S5-65, S5-64 use, S5-71 use, S5-47, S5-49, S5-34 | Members and Invitations readable on a phone, pending invitations on top, term status, audit actions in plain past tense |
| P9-L5 grades | `features/grades/`, their routes | S5-52, S5-53, S5-56, S5-55, S5-54, S5-59, S5-58, S5-60, S5-62, S5-63, S5-69, S5-65, S5-64 use, S5-71 use, S5-61, S5-57 | Scores visible beside names on a phone, missing scores flagged before finalize, "Finalized" stated, a compact grade scale with Save at the top |
| P9-L6 entry pages, one search, separators, e2e selectors | `features/auth/`, entry routes, `app-shell.tsx` search, the S5-70 lines, `e2e/helpers.ts`, `e2e/staffing.spec.ts`, `e2e/screens.ts` | S5-07, S5-01, S5-10, S5-12, S5-08, S5-70, S5-11, S5-09 | One field on sign-in, one search box, a resend link, no em dashes in labels |
| P9-L7 direction items | per the owner's answers | B1-B14 the owner picks | blocked on owner §6 |
| P9-L8 display names (API) | `apps/api/internal/course` import, `internal/auth` or a `me` handler, `routes.go`, `openapi.json` | C1 | Names instead of emails on every screen once imported or set (owner to confirm) |

Counts: 71 A items (S5-01 to S5-71, several merged from two or three lanes as marked), 14 B items, 8 C items.

## 9. Constraints and gates

- Spec 3 stays binding: semantic tokens only, no literal colours (`no-literal-colors.test.ts` is the gate), shared components only and a new pattern goes into `components/` with a test and a line in `docs/design/P6-L1.md` (new here: `SegmentedControl`, `Disclosure`, `Table` stack mode, `Loading` variants, `Card` footer), DM Sans only, Lucide only (B3 would be the one exception), purple only on actions, links, current location and the band.
- The e2e suite is the copy arbiter: only the selectors in §5.7 may change, in the lane named there.
- Every PR: `npm run lint`, `npm run typecheck`, `npm test`, `npm run build`, `npm run e2e`, then `npm run screens` with the S5-68 width assertion; the lane's screens are checked in light and dark at 1440 and 390 and contain no width offender.
- `npm run screens` uses the real API and the shared local database: one integration run at a time (PLAN.md Method).
- No API change in L1-L6; L8 is the only API lane.

## 10. Deferred

| Item | Trigger to pick it up |
|---|---|
| All B items not picked | the owner's answer to §6 |
| C2-C8 | as each row of §7 says |
| Dark-390 and 1024 px header captures | a phone-dark or tablet complaint, or B10 |
| Hover, focus and skip-link scoring (n/s) | an accessibility audit or axe in CI (Spec 3 §10) |
| Re-scoring the Dribbble multi-screen shots from their full pages | the owner asks (pack Q8) |
