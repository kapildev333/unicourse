# P8-L3-fix - apply review findings to Spec 5 and its plan
Status: done
## Doing
Done: all eleven findings applied to the spec and plan, committed on lane/P8-L3-benchmark.
## Found
- §5.7 is now e2e only and the exclusive list; staffing.spec.ts:81, :200, :205-206, :247 listed as assertions kept intact (S5-25 narrow weekend track, S5-26 instructor as sr-only in the block link, S5-15 New section disabled with reason).
- S5-49 moved to new C9 (API status filter and pending-first order); S5-48 keeps only the Expires dash. Counts now 70 A, 14 B, 9 C.
- §8 ends with eight "Decisions proposed" for PLAN.md (owner to confirm); §5.7 no longer carries proposals.
- Task 6 scope gained `features/courses/section-list.test.tsx` (it asserts the em-dash course option).
- Task 8 scope gained a possible migration and the sign-in path for invitee names (D21).
## Requests
## Blockers
