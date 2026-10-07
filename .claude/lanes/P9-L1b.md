# P9-L1b - shell, no-access page, capture script
Status: done
## Doing
Finished; waiting for orchestrator verification.
## Found
- Gates (apps/web): lint clean; typecheck clean; `npm test -- --run` 37 files, 251 tests passed; build ok; `npm run e2e` 2 passed; `npm run screens` red as expected (soft width assertion).
- WIDE list (390 px, scrollWidth): 21 (755 light+dark), 23 (755), 43 (693 light+dark), 44 (693 light), 45 (555 light+dark), 46 (555 light), 47 (555 light), 52 (561 light+dark), 53 (679 light+dark), 60 (546 light+dark), 61, 62 (546 light), 63 (546 light+dark), and 73 (740 light+dark). That is all eleven spec bases except none new, PLUS 73.
- 04 and 72 have no WIDE line. 73 does: the weight warning badge exists only in the gradebook (`gradebook.tsx:380`), whose table is wide until S5-52 (Task 5). The brief's gate for 73 cannot hold before then; the capture itself is correct (CS210 weights 110, warning badge visible). Orchestrator: expect 73 to turn clean with the 60-63 fix.
- No unit tests for the grid layout (S5-03/04): verify captures 20 and 30 at 390 (header height ~150 px target) and 1440.
- Header breakpoint is md (768) for the three-column grid; search sits in the 1fr slot, `max-w-xl`, `mx-auto`. Centring means its left edge still shifts a little with nav length (spec S5-03 wording followed literally); measure on 20 vs 45 at 1440.
- Courses/Admin links only render for admin and dept_admin, so S5-05 tests use an admin ("staff-side viewer" in the brief); a plain section staff member has no Courses link to mark.
- Prettier has no config in the repo; used `--no-semi --single-quote --print-width 110` to match the existing style.
- S5-13: one membership shows a primary "Go to {university}" Button (navigates; replaces the list, no lead-in); several keep the link list led by "Go to your university:". Slug sentence: "has no membership at this university's address."
- New file `routes/$slug/route.test.tsx` (outside the listed scope only in being the new test for `route.tsx`).
- Screens index lists WIDE lines as `- WIDE <file> <width>` (they pass through the index writer).
## Requests
## Blockers
