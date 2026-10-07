# P8-L3d - Spec 5 and plan from the benchmark notes
Status: done
## Doing
Done: Spec 5 `docs/specs/2026-10-07-ui-ux-benchmark.md` (scores, 71 A / 14 B / 8 C items) and plan `docs/plans/2026-10-07-ui-ux-benchmark.md` (P9-L1a/b to P9-L8), committed on lane/P8-L3-benchmark.
## Found
- For the orchestrator, decisions proposed for the PLAN.md table:
  - Spec 5 and its plan become Phase 9: P9-L1 (a components, b shell and capture script) first, P9-L2..L5 in parallel, P9-L6 alone last; P9-L7 blocked on owner §6; P9-L8 owner to confirm.
  - Selector ruling like D60 (spec §5.7): only the listed e2e and unit selectors change, in the lane named there.
  - Section dialog opens `wide` instead of `lg` (S5-40), amending D69.
  - Spec 3 §2 phone scope extended for the schedule agenda and stacked table rows; "no 390 capture wider than 390" (S5-68, `expect.soft` in `screens.ts`) gates every Phase 9 lane.
  - C1 display names as a small API lane P9-L8: roster import fills an empty `users.display_name` from the CSV; `PATCH /api/v1/me`.
  - New token `line-control` (S5-66) and new shared components `SegmentedControl`, `Disclosure`, `Table` stack mode, `Loading` variants, `Card` footer: CLAUDE.md UI conventions need a line after P9-L1a.
- Reclassified from the notes: card band height (note c B) is A in S5-18; TA "Published" badge (note c A) and grade-scale Remove styling (note c A) moved to B11 and B6.
- Before Phase 9, keep a copy of today's captures as `before/` (each run overwrites them).
## Requests
## Blockers
