# P9-L4-review - review of P9-L4 (admin, Spec 5 Task 4)
Status: done
## Doing
Review finished: changes needed (one copy-correctness fix), rest low/nit.
## Found
- 1 (medium) audit-log.tsx:43 `'membership.update': 'Member role changed'` is wrong for most emitters: PATCH /me name change (api auth/handler.go:265), roster import name fill (course/import.go:250), invite accept onto an existing membership (auth/invite.go:163), and department-only edits (tenant/membership.go:206). Fix: "Member updated".
- 2 (low) audit-log.tsx:173 sticky When td `bg-surface` (utilities layer) beats `.ui-table tbody tr:hover td` (components layer): on hover the row turns canvas but the When cell stays surface. Verified with built CSS in dark. Fix: add `[tr:hover>&]:bg-canvas`. Scrolled sticky cell does not overlap (verified dark, 390).
- 3 (low) list-count.tsx is a one-off recipe (Review focus 4); P9-L3 inlined the same "Showing n" markup twice (courses/course-list.tsx:196, section-list.tsx:260). Fine to land; consolidate into components/ (or a LoadMore placement prop) in one follow-up for L3/L4/L5.
- 4 (nit) table skeletons show a visible "Actions" header; real tables hide it (sr-only). Pass '' as the last column.
- 5 (nit) terms.tsx:26 re-implements todayIso (features/courses/queries.ts:27); semantics match lib/schedule.ts currentTerm (inclusive, local day). No boundary test (startsOn == today, endsOn == today).
- Deviations accepted: Actions cells without data-label (CSS :not([data-label]) drops the caption anyway); th bg-canvas matches `.ui-table th`.
- Gates: lint ok, typecheck ok, unit 272/272, build ok, e2e 2 passed (after two port collisions), screens red elsewhere, no WIDE for 50-54.
## Requests
## Blockers
