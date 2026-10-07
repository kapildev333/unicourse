# P9-L9a - Phase 9 closing audit, group a (screens 01-04, 10, 11, 20-23, 30-32), docs/design/P9-L9a.md
Status: done
## Doing
Done: docs/design/P9-L9a.md written (not committed; orchestrator commits)
## Found
- All 26 group 390 captures (AFTER, light and dark) are 390 wide; 21/23 were 755 before.
- Only addressed criterion still below 4 in group a: schedule (d)1 = 3. The phone agenda has no Today mark (S5-27 half done); at 1440 the now-label "14:12" overprints the "14:00" hour label and the now-line paints under blocks (needs `z-10`). Fixes R1, R2 in the design.
- No-access body copy reads "isn't a member of this university's address" (`routes/$slug/route.tsx:43`), R3.
- The capture seed sets no display names, so C1 (P9-L8) is never visible in captures; and the student is enrolled in every catalog section, so "Enrolled" shows on all cards (R4, R10, both in `e2e/screens.ts`).
- Taste-skill §9.G: en dash in every time/date range; e2e asserts it (`staffing.spec.ts:201,203,216,221`). Raised as owner question Q1, recommendation keep.
- Captures at 390 use a fine pointer, so `pointer-coarse` 44 px targets are not visible in them: P9-L9c, score (m)2 as n/s rather than 40 px.
## Requests
- to P9-L9c: for the app-wide (l) and (m) rows, group a dark/phone scores are in P9-L9a §7 and §0 (widths); none of my screens is wide.
## Blockers
