# P5-L3-review - review of P5-L3 (Spec 2 success criterion e2e) at 17813f4
Status: done
## Doing
Finished. Verdict: changes requested (2 defects), both fixed in the follow-up commit on the lane branch.
## Found
Verdict: changes requested.
Defects:
1. `apps/web/e2e/staffing.spec.ts:213`: the catalog search check `expect(getByRole('row', {name: /CS101-001/})).toHaveCount(0)` passes vacuously on the loading flash (0 rows) and fails otherwise because the MA201 row's name contains "Clashes with CS101-001"; scratch copy failed 2/2, worktree passed 2/2 from identical sources. Lines 209-212 can also pass on the unfiltered list. Fix: after `fill('MA201')` assert `expect(p.getByRole('row')).toHaveCount(2)` (header + MA201) and drop the count-0 line; 3/3 green on a scratch copy.
2. `apps/web/node_modules` symlink committed in 17813f4 (`.gitignore`'s `node_modules/` does not match a symlink). Fix: `git rm --cached`, ignore `apps/web/node_modules`.
Coverage of spec §1: overlapping Monday meetings (CS101 10:00-11:15, CS102 10:30-12:00, MA201 11:00-12:00); times pinned (:197, :200); clash lines naming the other section (:199, :201; mutation run failed as it should); instructor on the schedule (:202, CS101 only; mutation failed); catalog professor (:210; mutation failed), times (:211), clash flag (:212; mutation failed); student gets no roster (:218); "searched by course code" not reliably pinned (defect 1). Term spans the UTC year so the default pick is always it; clash is same-day wall-clock so timezone does not matter.
Flakiness: "Sunday region visible" uses toBeVisible (no scrolling dependency); no sleeps or custom timeouts; strict-mode locators fine apart from defect 1. D33: everything inside run-unique university A, nothing reset, links from the mail log. Reuse: one new sign-in context, whole test ~15-16 s.
Nits: CS102 has no instructor so "both sections with instructor names" is checked on CS101 only (add the professor to CS102); `Remove ${instr} as Instructor` matches by case-insensitive name; `year` uses the UTC year while the app picks the term by local date (New Year in UTC+ only; predates P5-L3); `mondayMeeting` reused well.
Gates (worktree, after the node_modules link was repointed): typecheck ok 3.0 s; lint ok 2.6 s; e2e run 1: 1 passed (15.5 s); run 2: 1 passed (15.2 s). Scratch copy of the same commit: e2e failed 2/2 at :213; with the fix 3/3 passed (~16.5 s each). Mutation runs on the fixed copy: clash line removed → failed at :199; catalog instructor removed → :210; catalog clash note removed → :212; schedule instructor removed → :202.
## Requests
## Blockers
