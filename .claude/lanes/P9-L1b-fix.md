# P9-L1b-fix - fix round for P9-L1b
Status: done
## Doing
Finished; waiting for orchestrator verification.
## Found
Changes (app-shell.tsx, app-shell.test.tsx, $slug/route.tsx, e2e/screens.ts):
1. Finding 1: flat flex-wrap header. Below 768 the university name and divider are hidden, Sign out is icon-only (aria-label "Sign out", title "Sign out (email)", text from md), the select (2+ memberships) leaves row one and sits at the start of the nav row with max-w-full/min-w-0. Row one = wordmark + theme + Sign out. Brand is shrink-0 (without it the wordmark was overlapped by the nav at xl). Test: button named "Sign out".
2. Finding 2: one-row layout at xl (1280) again; 768-1279 keeps main's two rows (three when the nav wraps); below 768 three rows. Search is min-w-64 at xl, full width below.
3. Finding 3: S5-03 met by the slot (search flex-1, max-w-xl, mx-auto in its own space); "left edge never moves with nav length" claim dropped.
4. Finding 4: Go to / Sign out wrapped in `mt-6 flex flex-wrap gap-3`.
5. Finding 6: regexes anchored `^\/[^/]+\/`; test with slug `admin` on /admin/me.
6. Findings 5 and 7 accepted as they are (button-that-navigates; keyboard order nav, search, theme, Sign out at 390).
7. Coordinator item: screens.ts roster Remove locator now a regex accepting "Remove <name>" or "Remove <email>" from the roster (esc helper). Lint/typecheck ok; screens run got past that step (only the soft WIDE failures remain). Not run against L8 names.
- Sign out title kept the email hint ("Sign out (a@a.edu)") rather than the bare "Sign out" in the brief; name is exactly "Sign out".
- Header height at 390 (bottom border y+1, captures 20, 30, 45, light and dark): 157 px (was 173; main 205). Trimmed py-2/gap-y-2 below md to get there; 66 not re-measured, same header as 45. Opened 45 dark: no overlap; 20 light earlier: none. 
- Live measures (vite preview, mocked /me, admin): 1 membership header 157/169/117/117/73/73 at 390/768/1024/1279/1280/1440; search width 342/576/576/576/362/426; no overlap, no scrollWidth growth. 2 memberships: 205 at 390 (select + 4 admin links cannot share one row at 342 px, nav wraps to its own row; non-admin fits), 169/117/117 at 768-1279, xl search 349 at 1280, 413 at 1440 (always >= 256).
- 768 with one membership wraps to three rows (name + nav + trail do not fit), same content as main.
- WIDE list is exactly 21, 23, 43-47, 52, 53, 60-63, 73.
- Gates (apps/web): lint clean; typecheck clean; test 37 files / 253 tests passed; build ok; e2e 2 passed (no "address already in use"); screens 1 failed (soft WIDE assertions only, by design).
## Requests
## Blockers
