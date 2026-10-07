# P8-L3a - Benchmark: entry, shell, My sections, Catalog

Lane P8-L3a, Phase 8 (D75-D78). Compares our captures against the reference pack P8-L2 (`scratchpad/P8-L2.md`) and the redesign-skill audit (D78). No product code. Category: A fits Spec 3 tokens and components, B direction question for the owner, C product or API gap. Impact H/M/L, effort S/M/L. "Copy change" means the orchestrator checks the new string against unit and e2e selectors before a worker starts (Spec 3 §5.8).

## 0. Method

- Read: CLAUDE.md UI conventions; Spec 3 §3, §5, §6, §7; P6-L1 §9.0-§9.3; pack §1, §2 (a) (b) (c) (d) (e) (k) (m), §3, §4, §5; redesign-skill Design Audit and Fix Priority. Code glanced at for accuracy only: `components/card.tsx`, `empty-state.tsx`, `load-more.tsx`, `features/auth/sign-in-page.tsx` (hint string).
- Captures read (16 light-1440): 01, 02, 03, 04, 10, 11, 12, 13, 20, 21, 22, 23, 30, 31, 32. Dark-1440: 01, 20, 21, 30. Light-390: 01, 20, 21, 30.
- Skipped: every Dribbble preview image (the pack's words were enough for these screen types); dark and 390 captures of 02-04, 10-13, 22, 23, 31, 32. Focus rings, skip link, live regions, hover and the 1024 px header are not observable in still PNGs; those criteria are marked n/s (not scored) or scored from P6-L1's stated DOM with that caveat.
- Capture artifact: in 21-light the pressed Schedule button is deep purple (hover colour) while List in 20-light is the base accent. 21-dark shows the base tint, so this is the pointer resting on the button, not a bug.
- DIR rules (Lucide, DM Sans, purple, bordered cards, 4 px radius, motion) are not scored; where they touch these screens they are in §3.

## 1. Shell (header, nav, search, theme switch, sign-out, page header rhythm)

Captures: every signed-in capture above; mainly 10, 20, 30 (light and dark 1440), 20 and 30 at 390.

Scores (pack §4 (b)):
1. Current page: 4 - bold label plus purple underline (10, 20, 30); `aria-current` not observable.
2. Search by keyboard and in URL: 3 - `q` reaches the Catalog URL (D59); no `/` shortcut and no hint of one.
3. Header height: 4 - 72 px at 1440 with a hairline; 1024 not captured; at 390 it stacks into four rows, about 205 px.
4. Grouping and labels: 3 - theme group and Sign out sit together on the right, but nothing shows who is signed in or in what role (email only in the Sign out tooltip, Spec 3 §6).
5. Skip link and focus under sticky header: n/s.
6. Phone: 3 - nav and search reachable without a menu, but 205 px of header before the page title; schedule page scrolls sideways (see §2.6).

Works:
- Udemy-like bar is clean: bold wordmark, hairline divider, muted university name, text nav with underline; nothing decorative (Spec 3 §5.1 holds).
- Theme control is a labelled three-way system/light/dark group, not a sun/moon toggle; it already does what RD asks ("system preference detection").
- Dark header is the surface step above the canvas, so elevation reads without shadow (20-dark, 30-dark).
- Page header rhythm is identical across My sections and Catalog: 32 px bold h1, muted one-line description, 24 px to the filter row.

Weaker than references:
- Linear and Stripe search (pack (b)): a shortcut (`/` or Cmd+K) focuses search; ours has none, and the header field does not show the current query when the Catalog has one (32: header empty, page field "zzzz").
- Dribbble 23125072 (karthikeyan): the signed-in user's name and role are visible in the bar. Ours shows no identity; a user with several roles cannot tell which hat they are wearing.
- Primer navigation: "minimize navigational elements". The Catalog page has two search inputs (header "Find a course" and page "Search"), which is the same control twice.
- The search field's left edge moves with the nav length: x 544 for student (20) and x 673 for admin (10). Linear and Stripe keep search in a fixed slot.
- Courses and Sections (12, 13) put the sub-tabs above the h1, so the page title sits under its own tabs. Primer underline nav and the pack's (f) put the title first, tabs under it.

Improvements:
1. Header search shows the current `q` when the route is the Catalog, and the Catalog page's own "Search" input is removed so the header field is the one search; the filter row keeps Term and Department. Impact H, effort M, A. Copy and selector check: e2e `staffing.spec.ts` line 212 uses `getByLabel('Search')`; it must move to the header field "Find a course".
2. `/` focuses the header search when focus is not in an input; add `aria-keyshortcuts="/"` and the hint text "Press / to search" as the input's `title`. Impact M, effort S, A.
3. Give the search a fixed slot: `app-shell.tsx` grid `grid-cols-[auto_1fr_auto]` with nav inside the first column and search `max-w-xl` centred in the second, so its left edge is the same for every role. Impact L, effort S, A.
4. At under 768 px put wordmark, theme group and Sign out on one row (theme group icon-only is already named), nav on row two, search on row three: header drops to about 150 px. Impact M, effort S, A.
5. Courses layout: render `PageHeader` (h1 and description) above `TabNav`, not below. Impact L, effort S, A. Note for P8-L3b, which owns the Courses screens.
6. A user menu (name, role in this university, theme, Sign out) replacing the bare Sign out button. Impact M, effort M, B (Spec 3 §6 places the email in a tooltip; pack §5 Q5) and C (no display name).

## 2. Screens

### 2.1 Sign-in

Captures: 01-light-1440, 01-dark-1440, 01-light-390.

Scores (pack §4 (a)):
1. One column, one primary path: 2 - two inputs on one card ("University email or university ID" for providers, then "Email" for the magic link); the user must decide which field is theirs before acting.
2. Link-sent state: see 2.2.
3. Inline errors keep value: n/s (no error capture).
4. Wordmark and form only: 5 - quiet canvas, wordmark, card; no marketing panel.
5. Paste, autocomplete, 48 px: 4 - provider and submit buttons are 48 px; identifier has `autocomplete="email"`; the second field's attributes not checked.
6. Contrast both themes: 5 - charcoal provider borders and the lilac dark primary measured in P6-L1 §1.2; not re-measured here.

Works:
- Full-width provider buttons with icon and "Continue with ..." text, equal width (matches Vercel login and the Duwal social-login sheet).
- Labels above inputs, hint below, no placeholder-as-label (H16).
- One purple element on the page: the primary submit (D57 respected).
- 390 layout keeps the same card with 16 px side margin; nothing truncates.

Weaker than references:
- Vercel login and Talebook 4080832: one field, one button, then the rest. Ours asks for the same address twice in two fields with two different labels; 03 shows the same email prefilled in both.
- Hint "For example alice@example.edu or example" reads as a typo; "example" is a university identifier, which the label calls "university ID" (users read that as a student number).
- Provider icons are the generic Lucide `log-in` arrow on both buttons; the Duwal sheet and Vercel show the provider's mark, so Google and Microsoft are told apart by text only.
- No "last used" hint (Duwal sheet, TikTok "Last login: Email").

Improvements:
1. One field: keep "University email or university ID" as the only input; below it the two provider buttons and the primary "Email me a sign-in link", all acting on that one value (magic link enabled only when the value is an email; otherwise inline error under the field "Enter your email to get a sign-in link"). Removes the second Email field and the "or" divider. Impact H, effort M, A, copy change: e2e `signIn` helper uses `getByLabel('Email', { exact: true })`; it must move to the single field.
2. Hint copy: "For example alice@example.edu, or your university's short name such as northfield". Impact M, effort S, A, copy change (`sign-in-page.test.tsx` types "example", reads no hint).
3. Remember the last method in `localStorage` (`unicourse:last-sign-in`) and show a `Badge` "Last used" inside that button. Impact L, effort S, A.
4. Provider brand marks (Google G, Microsoft squares) as inline SVG beside the label. Impact L, effort S, B (second icon source vs D53).

### 2.2 Check your email

Captures: 02-light-1440.

Scores (pack §4 (a)):
2. Link-sent state: 3 - shows the address and "use a different email"; no resend.
4. Canvas: 5 - wordmark and card only.
6. Contrast: 5 - same tokens as 01.

Works:
- States the address, single use and 15-minute expiry in plain words (superdesign "One Link, No Passwords" pattern).
- Neutral wording ("If ... can sign in") avoids revealing which emails exist.

Weaker than references:
- superdesign magic-link card: offers resend; ours offers only "use a different email", so a user who waited must retype the address.
- "use a different email" is a ghost button sitting in a sentence; its padding leaves a visible 16 px gap after "or" (02 at x 714-730) and it is lowercase bold purple, unlike any other link (H4).

Improvements:
1. Add a secondary `Button` "Send the link again" under the text that re-posts the same email and shows `StatusMessage` "Sent again." inline; disabled for 30 s with the reason as text. Impact M, effort S, A, copy change.
2. Render "use a different email" as an inline text link (`text-accent underline hover:text-accent-hover`, the link style from 04) so the sentence has no gap. Impact L, effort S, A.

### 2.3 Invite link

Captures: 03-light-1440.

Scores (pack §4 (a) plus H14):
1. One primary path: 2 - same two-field card as 01, both prefilled with the invited address.
H14 nothing asked twice: 4 - the address is prefilled, so the user types nothing.
4. Canvas: 5.

Works:
- Email prefilled from the invitation (H14).
- The invitation line names the address the user must sign in as.

Weaker than references:
- GitHub and Vercel invitation flows (pack (a), product knowledge) name the inviting organisation and role before sign-in. Ours says "You've been invited" with no university or role, in muted 14 px text below the subtitle, so it reads as secondary.
- The address breaks at its hyphen ("northfield-" / "muy9n4im.edu").

Improvements:
1. Invitation line becomes an info block above the form: `StatusMessage`-style row with Lucide `mail-open`, ink text (not muted), "You've been invited. Sign in as {email} to accept." Impact M, effort S, A.
2. Keep the address on one line: wrap the email in `<span className="whitespace-nowrap">` inside that paragraph. Impact L, effort S, A.
3. Name the university and role ("Join Northfield University as an instructor"). Impact M, effort M, C (the invite route only redirects to sign-in; no API returns invitation details to a signed-out visitor).
4. With improvement 2.1 #1, the single field is prefilled once instead of twice. A.

### 2.4 No access

Captures: 04-light-1440.

Scores (pack §4 (k)):
2. Error copy plain, offers a way on: 4 - says which account and what it is not a member of; offers own university and Sign out.
3. States what is missing and the action: 3 - the reason uses the URL slug "northfield-muy9n4im", and "Harbor College" is a bare link with no lead-in.

Works:
- Heading says exactly what happened in the product's words; no "Oops", no code.
- Two ways out: the user's own university and Sign out.

Weaker than references:
- Primer empty states: title, secondary text, one primary action, optional secondary link. Ours has the right parts but the user's university link has no label and looks like stray text; Sign out (the less likely action) is the only button.
- Quoting the slug in typographic quotes is internal data shown as copy (H2).

Improvements:
1. Lead the link with text: "Go to your university:" then the link list; make the single-membership case a primary `Button` "Go to Harbor College" and keep Sign out secondary. Impact M, effort S, A, copy change (e2e reads only the heading).
2. Replace `"northfield-muy9n4im"` with "this university's address" (the name is deliberately unknown to non-members). Impact L, effort S, A, copy change.

### 2.5 Empty states (10 My sections, 11 Catalog, 12 Courses, 13 Sections)

Captures: 10, 11, 12, 13 light-1440.

Scores (pack §4 (k)):
1. Skeleton: n/s.
3. Copy states what is missing; action echoes: 3 - titles are right ("No terms yet", "No courses yet"); 10, 11 and 13 are seen by an admin who can fix it, yet offer no action; 13 names the fix as plain text "Admin → Terms".
4. Filtered-empty distinct from first-use: 4 - 32 uses "No sections match" with `SearchX`; first-use uses its own icons.
5. Never shown together: 5 - one state per page in every capture.

Works:
- One shared `EmptyState` everywhere: icon, bold title, one muted line, centred in a bordered surface (Primer anatomy).
- 12 offers "New course" in the empty state, echoing the title (Stripe "action echoes the title").
- Copy says when data appears ("once your university sets up its terms").

Weaker than references:
- Stripe empty state: "offers the one action that fixes it"; Spec 3 §5.4 says the same. Admin on 10, 11, 13 gets a sentence, not a link to Admin > Terms.
- 13: "New section" is disabled (pale purple) with no visible reason, and the Term select is empty and full width (H5 disabled-with-reason).
- 12: the Department filter (1296 px wide) is shown over an empty list and "New course" appears twice (header and empty state); Primer: one primary action. The empty body mentions import but its button only creates.
- Stripe "section-level empty state is compact": ours is a 176 px tall box at full width for a page-level state; fine, but 12 stacks a full-width select above it, so the page reads as a form.

Improvements:
1. `EmptyState` `action` for admins on "No terms yet" (10, 11, 13): secondary `Button` link "Set up terms" to `/$slug/admin` terms tab; non-admins keep no action. Impact H, effort S, A, copy change.
2. 13: hide "New section" and "Import sections" when there are no terms (the empty state carries the action from #1), and hide the empty Term select. Impact M, effort S, A.
3. 12: hide the Department filter while there are no courses and no filter is set; in the empty state show both actions, `Button` primary "New course" and secondary "Import courses", and hide the header pair while empty. Impact M, effort S, A.
4. Filter selects on Courses and Sections get `w-64` like the Catalog filters instead of full width. Impact L, effort S, A (P8-L3b screens; listed for the empty case only).

### 2.6 My sections list (student 20, instructor 22)

Captures: 20 light and dark 1440, 20-light-390, 22-light-1440.

Scores (pack §4 (c)):
1. Fixed anatomy: 4 - band with code, bold title, muted meta lines, text badge; the instructor line is an email (C).
2. Card bottoms align: 3 - aligned here only because every card has two meetings; `card.tsx` puts the badge right after the meta (`mt-2`), so a card with one meeting ends its badge 24 px higher.
3. Whole card is one link, hover shadow only: 4 - one `Link`; focus not observable.
4. Term picker visible and says which term: 4 - picker plus h2 "Academic Year 2026"; the option text truncates at 390 ("Dec 31, 20").
5. Empty state names term and reason: 3 - see 2.5 (no admin action).
6. 3 / 2 / 1 columns, 4 px grid: 4 - 3 at 1440, 1 at 390; each card is about 280 px tall at 390, so under three cards per screen.

Works:
- Card grid with code band, bold title and role badge matches the Udemy card concept (Dribbble 2941299) and Canvas card anatomy without stock images.
- Role badge carries text; instructor badge uses `accent-soft`, student neutral (H19).
- Dark mode: band, surface and text all hold (20-dark).
- List/Schedule view switch is visible and pressed state is clear.

Weaker than references:
- Dribbble 2941299 (Udemy card concept): a divider pins the footer row to the card bottom. Ours lets the badge float with the content (RD "buttons not bottom-aligned").
- Same shot and Coursera results: the cover is proportionate to the content. Our band is 96 px (`h-24`) of flat tint holding only the code, which is repeated as the link name; at 390 a third of each card is empty tint.
- Canvas cards (pack (c)): one muted meta line, counts of what needs attention. Ours stacks the instructor email plus one line per meeting; the instructor in 22 sees her own email on every card (H8).
- Google Classroom: role-specific action on the card ("Open gradebook" for teachers). Instructor cards in 22 offer nothing beyond the card link.
- The term appears three times: picker option with dates, h2 with the same dates, and the list `aria-label`.

Improvements:
1. `card.tsx`: badge wrapper `mt-auto pt-3` and body `flex-1`, so badges sit on one line across a row. Impact M, effort S, A.
2. `card.tsx`: band `h-24` becomes `h-16` (64 px); code stays `text-xl font-bold`. Impact M, effort S, A.
3. Instructor line: when the viewer is one of the instructors, show "You" in place of their own address ("You, david.mensah@..."). Impact M, effort S, A, copy change.
4. Merge each section's meeting lines into one line when room and time repeat ("Mon, Wed 10:00-11:20 · Hall B 204") with a `formatMeetings` helper beside `formatMeeting` in the same file. Impact M, effort M, A (pure formatting; unit tests read "Mon 10:30-11:45 · Room 101" in catalog tests, so check).
5. Term picker option label shows the name only; the h2 keeps the dates. Impact L, effort S, A, copy change (e2e may select by option label; check).
6. Instructor and TA cards: a secondary link row at the card foot, "Gradebook" (Lucide `table`) to the section's grading tab. Impact M, effort M, A if the card stays one link target (needs a second link outside the card `Link`: card markup change, `card.test.tsx` updates).
7. Show display names instead of emails. Impact H, effort M, C (no API sets a display name).
8. Per-course colour on the band and the schedule block. Impact M, effort M, B (pack §5 Q7, D57).

### 2.7 Schedule (student 21, instructor 23)

Captures: 21 light and dark 1440, 21-light-390, 23-light-1440.

Scores (pack §4 (d)):
1. Days and hours scannable, current day marked: 3 - clean grid, 08:00-18:00; no today marker, no current-time line.
2. Code, time, room without truncation at 1440: 3 - code and room show; title and instructor email truncate ("Introduction to Comput..."), and "13:00-14:20" wraps after the dash.
3. Clash by text: n/s (no clash in data; P6-L1 keeps the "clash" text line).
4. Identity matches card: 3 - same purple accent left border for every course; the card band is the same tint, so no per-course link.
5. Phone: 1 - 390 capture is 755 px wide: the 7-column grid overflows the page, which scrolls sideways under a 390 px header.
6. Same information by keyboard and screen reader: 4 - region per weekday and a link per block (P6-L1 §9.2), not verified in the still.

Works:
- Week grid with blocks sized by duration, matching the Framer "Aisol" model and Kamushken 5715370 layout.
- Each block is a link to the section; bold code first (Polkovnikov 2930553 anatomy: title, time, room).
- Dark grid keeps hairlines and block borders legible (21-dark).

Weaker than references:
- Kamushken 5715370: today's header bold and a labelled current-time line. Ours has neither.
- Kamushken and Szabaciuk 16249040: weekend columns greyed or narrow when empty. Ours gives Sat and Sun full width with no classes, about 28 percent of the grid empty in both 21 and 23.
- Canvas Calendar, Michalczyk 3737562, Nikitin 5182836 (pack (m)): at phone width an agenda list grouped by day replaces the grid. Ours squeezes the grid and breaks the page width (pack §4 (m) 1).
- Blocks carry four lines in a 120 px tall box; the instructor email is what truncates the title. Michalczyk keeps title, time, room only.

Improvements:
1. Below 768 px render the schedule as an agenda: one `section` per weekday that has meetings (the existing weekday regions), `h3` day name, then a list of rows "10:00-11:20  CS101-001 Introduction to Computer Science  Hall B 204", each a link; the grid only from `md`. Impact H, effort M, A.
2. Until #1 lands, wrap the grid in `overflow-x-auto` so only the grid scrolls, not the page. Impact H, effort S, A.
3. Drop weekend columns when no meeting in the term falls on them (grid template from the days present). Impact M, effort S, A.
4. Block content: line 1 code bold, line 2 time (no wrap, `whitespace-nowrap`), line 3 title, line 4 room; drop the instructor line from the block (it stays on the card and detail). Impact M, effort S, A.
5. Today's column header: `font-bold` plus a `Badge` "Today" and `aria-current="date"`; no colour fill. Impact M, effort S, A.
6. Current-time line in today's column (1 px `line-strong` with a muted "10:28" label), updated each minute. Impact L, effort M, A.
7. Per-course block colour: B, same question as 2.6 #8.

### 2.8 Catalog results and filtered (30, 31)

Captures: 30 light and dark 1440, 30-light-390, 31-light-1440.

Scores (pack §4 (e)):
1. Filters visible, labelled, one row, in URL: 4 - Term, Department, Search on one labelled row; `q` in URL; term and department in URL not observable.
2. Count announced: 4 - "Showing 5" is `role="status"`; it has no noun and no total.
3. Cards match My sections: 4 - same `Card`; no badge, so enrolled sections look like any other.
4. Filtered-empty: see 2.9.
5. Load more: n/s (five results).
6. Phone: 3 - filters stack into three rows at different widths (Term full, Department about 205 px, Search about 172 px); cards stack correctly.

Works:
- Labelled filter controls above the grid, the Stripe filter row pattern without chips.
- Department filter narrows the grid and the count follows (31: "Showing 2").
- Same card as My sections (Spec 3 §6), so a section looks the same wherever it appears.
- Clash note exists in the card for overlapping sections (P6-L1 §9.3; not in this data).

Weaker than references:
- Stripe filter controls: a "Clear filters" link appears when any filter is set. 31 has none; the user must reset the department select by hand.
- Coursera search: the heading reflects the query ("Results for 'python'") and each card has one meta line plus a text badge ("Free trial"). Our student sees the five sections they are already enrolled in with no "Enrolled" mark.
- Coursera and Udemy: results ordered for the reader. Ours lists CS101, CS210, MATH201, ECON110, BIO150, neither by code nor by department.
- Department option label joins code and name with an em dash ("CS", em dash, "Computer Science"; 31) (H22).
- The Search input is narrower than the two selects and holds the placeholder only; with improvement 1.1 the field goes away.

Improvements:
1. A `Button` ghost sm "Clear filters" at the end of the filter row, shown only when department or search is set; resets both, keeps the term. Impact H, effort S, A, copy change.
2. `Badge` "Enrolled" (or the viewer's role: "Instructor", "TA") on cards whose id is in the viewer's sections; the page already loads `mine` for clashes. Impact H, effort S, A.
3. Count text "Showing 5 sections"; "of N" needs a total. Impact L, effort S, A copy change for the noun; C for the total.
4. Department option label "CS - Computer Science" or "Computer Science (CS)". Impact L, effort S, A, copy change (e2e may select departments by label; check).
5. Order results by class code. Impact M, effort S if the API already sorts and only the seed is unsorted, else C (sort param on the catalog endpoint; client sort breaks load-more paging).
6. At under 768 px the filter controls are all `w-full`; Term and Department side by side from 390 is too narrow, so stacked full width. Impact L, effort S, A.

### 2.9 Catalog empty result (32)

Captures: 32-light-1440.

Scores (pack §4 (e) 4, (k) 3-4):
- Filtered-empty says no match, offers Clear filters, no create: 3 - "No sections match" and no create action, but no Clear filters and the query is not repeated.
- Distinct from first-use: 4 - own title and `SearchX` icon.

Works:
- Correct state: no "create first" call to action (Stripe, H18).
- Title in the product's word ("sections").

Weaker than references:
- Vercel Geist: "a no-match message quotes the user's query". Ours does not repeat "zzzz".
- Stripe: filtered-empty plus Clear filters. Ours says "Try another search or department." and leaves the work to the user.
- Hashemi 17206944 is the counter-example we match only halfway: no "Search again" button, but no Clear filters either.

Improvements:
1. Title `No sections match "{q}"` when a query is set (department name when only a department is set); body unchanged; `EmptyState` `action` = secondary `Button` "Clear filters" (same handler as 2.8 #1). Impact H, effort S, A, copy change (e2e and unit tests reading "No sections match" use the prefix; check exact matches).

## 3. Direction questions for the owner (B)

- Q1 Course identity colour: one colour per course (or department) on the card band and the schedule block, as Canvas and Dissanayake 25058951 do, against D57 "purple only on actions, links, location, band". Screens: My sections, Schedule, Catalog. Pack §5 Q7.
- Q2 User menu: replace the bare Sign out with a menu holding name, role, theme and Sign out (pack §5 Q5, GitHub theme in settings). Conflicts with Spec 3 §6 (email in tooltip, theme switch in the bar). Screens: shell everywhere. Needs display names (C) to be worth it.
- Q3 Provider brand marks on sign-in buttons: inline SVG logos beside the Lucide set (D53 says no second icon set). Screens: sign-in, invite.
- Q4 Global search scope: keep the header search as Catalog search (D59) or make it app-wide by role (pack §5 Q4). Screens: shell, Catalog. Affects 1 #1.
- Not raised as findings (DIR, owner already decided): Lucide, DM Sans 400/500/700, purple accent, bordered cards with hover shadow, 4 px radius, 150 ms colour-only motion, 3-column card grid.

## 4. Top 10 for this group

Ordered by impact, then effort.

1. Schedule at phone width: grid scrolls inside `overflow-x-auto` now (S), agenda list per weekday below 768 px (M). 2.7 #2, #1. H, A.
2. Catalog "Clear filters" in the filter row and in the no-match state, query quoted in the title. 2.8 #1, 2.9 #1. H, S, A.
3. "Enrolled" or role badge on Catalog cards from the already loaded `mine`. 2.8 #2. H, S, A.
4. Admin "Set up terms" action on the "No terms yet" empty states; hide disabled New section and empty Term select. 2.5 #1, #2. H, S, A.
5. One search: header field mirrors `q` and replaces the Catalog page Search input. 1 #1. H, M, A (one e2e line moves).
6. Sign-in single field for all three methods. 2.1 #1. H, M, A (e2e `signIn` helper moves).
7. Card: badge pinned to the bottom, band 96 to 64 px. 2.6 #1, #2. M, S, A.
8. Schedule blocks: drop instructor line, no-wrap time, drop empty weekend columns, Today marker. 2.7 #3, #4, #5. M, S, A.
9. "You" in place of the viewer's own email on cards; resend link on Check your email. 2.6 #3, 2.2 #1. M, S, A.
10. Display names instead of emails on cards, schedule and future user menu. 2.6 #7. H, M, C (API).
