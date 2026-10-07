# P9-L9a - Phase 9 closing audit, group a: entry pages, shell, My sections, schedule, Catalog

Lane P9-L9a (designer, read-only audit). Re-scores screens 01-04, 10, 11, 20-23 and 30-32 after Phase 9 against the pack rubric (`docs/design/P8-L2.md` §4) and the taste-skill §4 and §9 checks, with the Phase 8 scores of `docs/design/P8-L3a.md` as the "before". No product code. Sibling lanes: P9-L9b (12, 13, 40-54), P9-L9c (grading, states, dark mode and phone width across the app).

## 0. Method

- **Read:** pack §0, §1 (H1-H25), §3, §4; Spec 5 §1, §3, §4.1-§4.9, §5, §6, §8; P8-L3a in full; PLAN.md D79-D98; taste-skill `skills/taste-skill/SKILL.md` §4.1-§4.7, §4.11 and §9.A-§9.G (the clone the Phase 8 lanes used, in the Phase 8 session scratchpad; not vendored). Code read only to make fixes exact: `features/my-sections/schedule.tsx`, `my-sections.tsx`, `features/catalog/catalog.tsx`, `routes/$slug/route.tsx`, `features/auth/sign-in-page.tsx`, the e2e selectors, all on `lane/P9-L6-entry-search-separators` (the build the AFTER set was taken from).
- **AFTER captures read (24):** light-1440 of all 13 bases; light-390 of 01, 04, 20, 21, 23, 30, 31; dark-1440 of 01, 20, 21, 30. **BEFORE captures opened:** none; P8-L3a describes the before.
- **Measured with `sips`:** every 390 capture of the group, light and dark, AFTER: all 26 are 390 px wide (before: 21 and 23 were 755). Heights at 390 AFTER vs BEFORE: 20 1740 vs 2022, 21 1627 vs 1511 (agenda), 22 1370 vs 1452, 23 1203 vs 1511, 30 1712 vs 1877, 31 1037 vs 1133.
- **Quoting:** time and date ranges quoted from captures write the en dash as "to" (the en dash itself is discussed in §9 and Q1).
- **Scoring:** 1-5 per pack §4, lower of both themes and both widths where captured. "n/s" is not observable in a still; "-" not applicable. Before scores are copied from P8-L3a (Spec 5 §3 table). An S5 item is "delivered" only when the capture shows it.
- **Capture state notes:** the run was taken on a Wednesday at 14:12 to 14:13, inside the term, so Today and the current-time line are in frame. The 390 captures use a fine pointer, so the `pointer-coarse:` 44 px targets (P6-L1 §0) do not apply in them; (m)2 is n/s here. The seed gives no user a display name, so every person shows as an email (see R4).

## 1. Shell (header, nav, search, theme switch, Sign out)

Captures: 10, 11, 20, 21, 22, 23, 30, 31, 32 light-1440; 20, 21, 30 dark-1440; 20, 21, 23, 30, 31 light-390.

| # | Criterion (b) | Before | After | Reason |
|---|---|---|---|---|
| 1 | Current page unmistakable | 4 | 4 | Bold label plus purple underline on My sections and Catalog in every capture; `aria-current` n/s, so not 5 |
| 2 | Search by `/`, query in URL | 3 | 4 | 32 shows "zzzz" in the header field, so the field carries `q`; the `/` shortcut and its title hint are n/s in a still |
| 3 | Header at most 72 px, no wrap at 1024 | 4 | 4 | 72 px with a hairline at 1440; 1024 not captured |
| 4 | User menu, theme, university grouped and labelled | 3 | 3 | Theme group and Sign out sit together; still no name or role in the bar (B2 waits on C1, D87) |
| 5 | Skip link, focus not hidden | n/s | n/s | Not observable |
| 6 | Phone keeps search and nav reachable, no sideways scroll | 3 | 5 | Three rows (wordmark, theme, icon Sign out · nav · search), about 156 px instead of 205; every group capture is 390 wide |

S5 items for the shell seen in this group:

| Item | Status | What the capture shows |
|---|---|---|
| S5-01 one search | delivered | 30-32 have no page Search field; the header holds "zzzz" in 32 |
| S5-02 `/` shortcut | not visible | Behaviour and `title` are not in a still |
| S5-03 search slot | delivered as ruled by D93 | The field fills the slot between nav and theme group; its left edge is x 544 for the student (20), x 673 for the admin (10), accepted by D93 |
| S5-04 phone header | delivered | 20-390: wordmark, theme group and icon-only Sign out on row one, nav row two, search row three; university name hidden (D93) |
| S5-05, S5-06 | not in this group | Routes 45, 66 and the Admin tab strip belong to P9-L9b and c |
| S5-66 `line-control` | delivered | 20-dark and 30-dark: Sign out and the List/Schedule group have a grey border, quieter than the lilac primary in 01-dark |

## 2. Entry pages

### 2.1 Sign-in (01)

Captures: 01 light-1440, light-390, dark-1440.

| # | Criterion (a) | Before | After | Reason |
|---|---|---|---|---|
| 1 | One column, one primary path | 2 | 5 | One field "University email or university ID", primary "Email me a sign-in link" under it, then Google and Microsoft, all full width with mark and text |
| 3 | Inline errors keep value | n/s | n/s | No error capture |
| 4 | Wordmark and form only | 5 | 5 | Unchanged quiet canvas |
| 5 | Paste, `autocomplete=email`, 48 px | 4 | 4 | Buttons 48 px; the input is 40 px (y 297-337 at 1440) |
| 6 | Contrast both themes | 5 | 5 | Provider borders visible in light and dark; dark primary is lilac with dark text; not re-measured (P9-L9c owns measurement) |

| Item | Status | What the capture shows |
|---|---|---|
| S5-07 one field | delivered | Second "Email" field and "or" divider gone |
| S5-08 hint | delivered | "For example alice@example.edu, or your university's short name such as northfield" |
| S5-09 Last used | not visible | A first visit has no stored method; the badge cannot appear in this capture |
| B3 brand marks (D87) | delivered | Monochrome Google "G" and Microsoft four squares in ink, white in dark |

### 2.2 Check your email (02)

Captures: 02 light-1440.

| # | Criterion (a) | Before | After | Reason |
|---|---|---|---|---|
| 2 | Link-sent: address, resend, different email | 3 | 5 | Address in the body, secondary "Send the link again" disabled with "You can send it again in 30 s." under it, "use a different email" as an inline link |
| 4 | Canvas | 5 | 5 | Unchanged |
| 6 | Contrast | 5 | 5 | Same tokens; disabled button text is muted (WCAG exempts disabled controls) |

| Item | Status | What the capture shows |
|---|---|---|
| S5-10 resend | delivered (disabled state) | Button and reason text; "Sent again." is a later state, not captured |
| S5-11 inline link | delivered | Underlined purple link inside the sentence, no gap after "or" |

### 2.3 Invite link (03)

Captures: 03 light-1440.

| # | Criterion | Before | After | Reason |
|---|---|---|---|---|
| (a)1 | One primary path | 2 | 5 | Same single field as 01, prefilled once |
| H14 | Nothing asked twice | 4 | 5 | Address prefilled in the one field |
| (a)4 | Canvas | 5 | 5 | Unchanged |

| Item | Status | What the capture shows |
|---|---|---|
| S5-12 invitation row | delivered | Lucide `mail-open`, ink text "You've been invited. Sign in as ingrid.larsen@northfield-muyfdo34.edu to accept.", the address kept whole on its own line (no break at the hyphen) |

### 2.4 No access (04)

Captures: 04 light-1440, light-390.

| # | Criterion (k) | Before | After | Reason |
|---|---|---|---|---|
| 2 | Plain copy, a way on | 4 | 4 | Way on is clear, but the body reads "isn't a member of this university's address", which is not a sentence a person would write (R3) |
| 3 | States what is missing, action echoes it | 3 | 5 | Primary "Go to Harbor College", Sign out secondary with icon; the slug is gone from the copy |

| Item | Status | What the capture shows |
|---|---|---|
| S5-13 lead and primary | delivered | Single membership becomes a primary button; the "this university's address" wording is in, but joined to "member of" (R3) |

## 3. First-use empty (10 My sections, 11 Catalog)

Captures: 10, 11 light-1440.

| # | Criterion (k) | Before | After | Reason |
|---|---|---|---|---|
| 1 | Skeleton | n/s | n/s | No loading capture in this group |
| 3 | Missing thing and the action | 3 | 4 | Admins get secondary "Set up terms"; the body still says "once your university sets up its terms" to the one person who sets them up (R8) |
| 5 | One state at a time | 5 | 5 | One `EmptyState` per page |

| Item | Status | What the capture shows |
|---|---|---|
| S5-14 Set up terms | delivered | Secondary `ButtonLink` under the body on 10 and 11 |

## 4. My sections list (20 student, 22 instructor)

Captures: 20 light-1440, light-390, dark-1440; 22 light-1440.

| # | Criterion (c) | Before | After | Reason |
|---|---|---|---|---|
| 1 | Fixed anatomy | 4 | 4 | Band with code, bold title, muted lines, text badge; the instructor line is still an email for students (R4) |
| 2 | Card bottoms align | 3 | 5 | Badges sit on one line across each row (y 544 row one, y 777 row two in 20) whatever the meeting count |
| 3 | One link target, hover shadow only | 4 | 4 | Main card link unchanged; the Gradebook foot link in 22 sits outside it; focus n/s |
| 4 | Term picker visible, says which term | 4 | 4 | Option shows the name only; h2 keeps the dates, but at 390 the date wraps as "Jan 1, 2026 to Dec 31," then "2026" alone (R5) |
| 5 | Empty state names reason, one fixing action | 3 | 4 | See §3 (action present, copy speaks to someone else) |
| 6 | 3 / 2 / 1 columns, 4 px grid | 4 | 5 | 3 at 1440, 1 at 390; a card is about 232 px tall at 390 (was about 280) |

| Item | Status | What the capture shows |
|---|---|---|
| S5-18 card badge row and 64 px band | delivered | Band 64 px (y 341-405); badge pinned to the card foot |
| S5-19 "You" | delivered | 22: the instructor line reads "You" |
| S5-20 merged meeting lines | delivered | "Mon, Wed 10:00 to 11:20 · Hall B 204" on one line; BIO150 keeps two lines because times differ |
| S5-21 term option name only | delivered | Select reads "Academic Year 2026" |
| S5-22 Gradebook foot link | delivered | 22: hairline foot with Lucide `table` and "Gradebook" on every instructor card |
| S5-23 SegmentedControl | delivered | Selected segment `accent-soft` with ink text; in 20-dark it no longer reads as a primary button |

## 5. Schedule (21 student, 23 instructor)

Captures: 21 light-1440, light-390, dark-1440; 23 light-1440, light-390.

| # | Criterion (d) | Before | After | Reason |
|---|---|---|---|---|
| 1 | Days and hours scannable, today marked | 3 | 3 | 1440: "Today" badge on Wed (4 on its own), but the time-line label "14:12" prints over the "14:00" hour label and the line is hidden under the MATH201 block (R2); 390: the agenda's "Wednesday" heading has no Today mark (R1). Lower width scores 3 |
| 2 | Code, time, room without truncation at 1440 | 3 | 5 | Code bold, time `nowrap` on the same line, room muted, all whole; the title still truncates ("Introduction to Computer Sci...", R9) but the criterion names code, time and room |
| 3 | Clash by text | n/s | n/s | No clash in the data |
| 4 | Block identity matches card | 3 | 3 | Same accent left border on every block; B1 (per-course colour) was declined in D87, so this is outside the success criterion |
| 5 | Phone: agenda, not a squeezed grid | 1 | 5 | 21-390 and 23-390 are agenda lists by weekday, 390 wide, only days with meetings |
| 6 | Same information for keyboard and screen reader | 4 | 4 | Region per weekday kept (code: `section aria-labelledby` with the long day name); not verifiable in a still |

| Item | Status | What the capture shows |
|---|---|---|
| S5-24 phone agenda | delivered | Day headings, one bordered link per meeting: code and time, title, room |
| S5-25 narrow empty weekend | delivered | Sat and Sun about 103 px against 206 px for weekdays |
| S5-26 block content | delivered | Instructor line gone from blocks and agenda rows; time on the code line |
| S5-27 Today | partly | Badge and bold header on the grid; nothing on the agenda (R1) |
| S5-28 current-time line | partly | Label present but collides with the hour label; the line is drawn under blocks, so it disappears while a class is in progress (R2) |

## 6. Catalog (30 results, 31 filtered, 32 no match)

Captures: 30 light-1440, light-390, dark-1440; 31 light-1440, light-390; 32 light-1440.

| # | Criterion (e) | Before | After | Reason |
|---|---|---|---|---|
| 1 | Filters visible, labelled, one row, in URL | 4 | 4 | Term and Department labelled on one row, `q` in the header; the row's "Clear filters" sits about 4 px below the selects' centre line (R7); term and department in the URL n/s |
| 2 | Count announced | 4 | 5 | "Showing 5 sections" / "Showing 2 sections" |
| 3 | Cards match My sections | 4 | 5 | Same `Card` with a text badge "Enrolled" |
| 4 | Filtered-empty: no match plus Clear filters, no create | 3 | 5 | 32: `No sections match "zzzz"` with secondary "Clear filters", no create action |
| 5 | Load more keeps position | n/s | n/s | Five results |
| 6 | Phone: filters collapse, cards stack | 3 | 4 | Both selects `w-full`, "Clear filters" on its own row, cards stack; no disclosure, which two controls do not need |
| (k)4 (32) | Filtered-empty distinct from first-use | 4 | 5 | Own title quoting the query, `SearchX`-style icon, Clear filters |

| Item | Status | What the capture shows |
|---|---|---|
| S5-01 one search | delivered | No page Search field; header carries `q` |
| S5-29 Clear filters and quoted title | delivered | Ghost "Clear filters" in the row (31, 32) and secondary in the empty state (32) |
| S5-30 Enrolled badge | delivered, weakly shown | Every card says "Enrolled" because the seeded student takes all five sections; the capture cannot show a card without it (R10) |
| S5-31 count noun | delivered | "Showing 5 sections" |
| S5-32 phone filters full width | delivered | 30-390, 31-390 |
| S5-70 separators | delivered | Department option "CS · Computer Science" |

## 7. Dark mode for this group (01, 20, 21, 30 dark-1440)

Scored for the group only; P9-L9c owns the app-wide (l) row and the contrast measurements.

| # | Criterion (l) | Before | After | Reason |
|---|---|---|---|---|
| 1 | Text 4.5:1, borders 3:1 | 4 | 4 | Muted meta and hour labels legible; not measured here |
| 2 | Elevation by surface steps | 5 | 5 | Header and cards one step above canvas |
| 3 | Purple links legible | 5 | 5 | Lilac "Schedule" segment text and nav underline |
| 4 | No pure black, no light island | 5 | 5 | None |
| 6 | Badges adapt | 4 | 5 | Student, Enrolled, Today and the selected segment all redrawn for dark; the selected segment is a dark violet, not a primary-looking fill |

## 8. Remaining findings, ranked by impact

Each has a fix in Spec 3 tokens and components and the criterion it serves. "Unit test" means the owning lane updates the named test with the copy; "e2e" means a selector D80 protects.

| Id | Impact | Screens | Finding | Fix | Serves |
|---|---|---|---|---|---|
| R1 | H | 21, 23 at 390 | The agenda never marks today, so the phone schedule scores (d)1 = 3 and S5-27 is only half delivered | `schedule.tsx` `Agenda`: pass `todayDay` and `inTerm` from `Schedule`; when `inTerm && d === todayDay` give the `h3` `aria-current="date"` and append `<Badge tone="accent">Today</Badge>` after the day name inside a `flex items-center gap-2` wrapper, exactly as the grid header does. Unit test: one case asserting the badge in the agenda at a phone width (`useWide` false) | (d)1 |
| R2 | M | 21, 23 at 1440 | Current-time label "14:12" overprints the "14:00" hour label; the 1 px line is painted under the blocks, so it vanishes during a class | (a) On the now-line `li` add `z-10` so it paints over the block `li`s (they are `relative`, later in the DOM). (b) In the `hours.map` gutter, add `invisible` to an hour label when `inTerm && nowMin >= h * 60 - 10 && nowMin < h * 60 + 15`, so the now label never shares the slot. No colour change: line stays `border-line-strong`, label `text-ink-muted` on `bg-surface` | (d)1 |
| R3 | M | 04 | "Your account (...) isn't a member of this university's address." An account is a member of a university, not of an address | `routes/$slug/route.tsx:43`: "Your account ({email}) isn't a member of the university at this address." e2e reads the heading only (Spec 5 §4.5); update any unit test that reads the body | (k)2 |
| R4 | M | 20, 21, 30, 31 | Students see instructors as emails on every card although P9-L8 (C1) shipped names; the captures never show a name because the seed sets none | `e2e/screens.ts` (capture script, not the e2e suite): after seeding, sign in as each seeded person and `PATCH /api/v1/me` with `{displayName}` ("Elena Marsh", "David Mensah", "Amara Okafor", ...), or put names in the roster CSV `displayName` column for imported students. Re-score (c)1 after. No product change | (c)1 |
| R5 | L | 20, 22 at 390 | h2 date range wraps leaving "2026" alone on a line | `my-sections.tsx:33`: date `span` gets `whitespace-nowrap`, and below `sm` it moves under the name: `block sm:inline sm:ml-2 mt-1 sm:mt-0` (replacing `ml-2`) | (c)4 |
| R6 | L | 10 (admin) | Body "Your schedule appears here once your university sets up its terms." is addressed to the admin who must do it, under a "Set up terms" button | `my-sections.tsx:101` and the Catalog's matching empty state: when the admin action renders, body "Add a term to start scheduling sections."; non-admins keep today's copy. Unit tests that read the body change with it | (k)3, (c)5 |
| R7 | L | 31, 32 at 1440 | "Clear filters" ghost `sm` (32 px) under `md:items-end` sits about 4 px off the 40 px selects' centre | `catalog.tsx` filter row: drop `size="sm"` so the ghost is the 40 px default and aligns with the selects; keep `variant="ghost"` | (e)1 |
| R8 | L | 32 | Two "Clear filters" on one screen (row and empty state): taste-skill §4.5 "no duplicate CTA intent" | Render the row button only when `filtered && sections.length > 0`; the `EmptyState` action carries it when nothing matches. `catalog.test.tsx:145` expects two and changes to one | (e)4, taste §4.5 |
| R9 | L | 21, 23 at 1440 | Block titles truncate ("Introduction to Computer Sci...") although each block has about 70 px empty below the room | Block title `span`: `truncate` becomes `line-clamp-2`; no height change (blocks keep their duration height) | (d)2 legibility |
| R10 | L | 30, 31 | The capture seed enrolls the student in every catalog section, so "Enrolled" is on every card and the screen cannot show the difference S5-30 makes | `e2e/screens.ts`: seed one more section in the term (for example MATH201-002) without Amara Okafor in it | (e)3 evidence |
| R11 | M, C | 30, 31 | Results are not in code order (CS101, CS210, MATH201, ECON110, BIO150): the API orders by `s.created_at, s.id` (`apps/api/internal/course/section.go:162`) | API lane: order the section list by course code, section number, id with a matching keyset cursor; a client sort would break load more. Spec 5 C3, not Phase 9 scope | (e) reading order, H6 |
| R12 | L | 04 at 390 | Primary and Sign out stack at their own widths (about 174 and 114 px) | Both buttons `w-full sm:w-auto` in the no-access action row | (k)3, (m) |
| R13 | L, owner | 01, 03 | Label says "university ID" while the hint says "short name"; P8-L3a noted users read "university ID" as a student number | Label "University email or short name"; error copy `identifierCopy` to match. Breaks e2e `helpers.ts:47`, `screens.ts:97,143`, `staffing.spec.ts:38` and `sign-in-page.test.tsx`, so it needs a D80-style ruling before any lane takes it (see §11 Q2) | (a)1, H2 |

Owner direction is not changed by any of R1-R12: no palette, type or density change. R13 needs a selector ruling, not a direction call.

## 9. Taste-skill pass for the group (AFTER captures)

| Check | Result | Evidence |
|---|---|---|
| §9.A no neon or glow | pass | No glow, gradient or blur on any screen; the schedule's hour rules are a line pattern (P6-L1 allows it) |
| §9.A no pure black | pass | Dark canvas and surfaces are grey steps (01, 20, 21, 30 dark) |
| §9.A no oversaturated accent | DIR | Udemy purple is the owner's pick (D57); it stays on actions, links, current nav, band, Today and selected segment |
| §9.B no Inter, no oversized H1, no serif | pass | DM Sans only; h1 32 px; no serif |
| §9.C no awkward gaps | fail (minor) | 31, 32 "Clear filters" off the select centre line (R7); 20-390 orphan "2026" (R5) |
| §9.C no 3 equal feature cards | pass (DIR read as question) | 20, 22, 30 grids are data grids, not marketing rows |
| §9.D no generic names or brands | pass, one note | Seed names are specific (Amara Okafor, Elena Marsh, Northfield University, Harbor College); the sign-in hint keeps "alice@example.edu", a stock placeholder, acceptable for an example address |
| §9.D no filler verbs | pass | Copy is plain ("Set up terms", "Send the link again", "Go to Harbor College") |
| §9.E icons | DIR plus approved exception | Lucide (D53); the two brand marks are hand-drawn SVG, allowed by D87 B3 |
| §9.F middle dot at most one per line | pass | Card meta "Mon, Wed 10:00 to 11:20 · Hall B 204", option "CS · Computer Science": one each |
| §9.F no decorative dots, eyebrows, scroll cues, version stamps | pass | None on any screen |
| §9.G no em dash | pass | None visible in the group |
| §9.G no en dash as a range separator | fail | Times "10:00–11:20" (en dash) on cards, blocks and agenda; term range "Jan 1, 2026 – Dec 31, 2026" in the h2. e2e asserts the en dash (`staffing.spec.ts:201, 203, 216, 221`). Owner question Q1 |
| §4.1 type | DIR | DM Sans, 400/500/700 (D58) |
| §4.2 one accent, locked | pass | One purple everywhere; status colours absent from these screens; role badges `accent-soft` (Instructor), states neutral (Enrolled, Student) per B11 |
| §4.3 anti-centre | pass (override) | Entry pages centre a utility form, which is the override case |
| §4.4 one radius scale | pass | 4 px everywhere, the header search the one documented `rounded-full` exception (P6-L1 §0 #5) |
| §4.4 cards only for hierarchy | DIR | Bordered cards with hover shadow (D57) |
| §4.5 full state cycle | pass | Empty (10, 11), filtered-empty (32), resend disabled-with-reason (02); loading is P9-L9c's |
| §4.5 no duplicate CTA intent | fail (minor) | Two "Clear filters" in 32 (R8) |
| §4.5 CTA wrap, button contrast | pass | No wrapped button label; dark primary uses dark text on lilac |
| §4.6 label above, helper below, no placeholder label | pass | 01, 03, 30 filters |
| §4.7 nav one line at desktop, at most 80 px | pass | 72 px one row at 1440; 1024 n/s |
| §4.11 theme lock | pass | No light island in dark captures |

## 10. Screens still below 4

| Screen | Criteria below 4 | Single fix that lifts it |
|---|---|---|
| Shell | (b)4 = 3 | B2 user menu (name, role, theme, Sign out), already approved to follow P9-L8 (D87); not an item Phase 9 addressed, so outside the success criterion |
| Schedule (21, 23) | (d)1 = 3 | R1: Today badge and `aria-current="date"` on the agenda's day heading (R2 lifts the 1440 side to 5) |
| Schedule (21, 23) | (d)4 = 3 | B1 per-course colour, declined in D87; stays 3 by decision, outside the success criterion |
| 01, 02, 03, 04, 10, 11, 20, 22, 30, 31, 32 | none | - |

Against Spec 5 §1: the only addressed criterion still under 4 in this group is (d)1, because S5-27 left the agenda unmarked. Every 390 capture in the group is 390 wide.

## 11. Owner questions

- **Q1 (copy, owner to confirm):** taste-skill §9.G bans the en dash as a range separator; the app uses it for every time and date range and the e2e suite asserts it. Recommendation: keep the en dash for numeric ranges (it is the typographic range mark, the pack's own H22 bans only the em dash) and record it as a named deviation from the rubric. If the owner prefers the hyphen, one formatter change in `lib/schedule.ts` and `my-sections.tsx` plus four e2e lines under a D80-style ruling.
- **Q2 (selector ruling, orchestrator):** R13 changes the sign-in field label, which four e2e lines read. Take it only with an explicit ruling naming those lines.
