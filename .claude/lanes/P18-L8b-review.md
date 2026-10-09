# P18-L8b-review - review of P18-L8b e2e (quizzes, assignments)
Status: done
## Doing
Finished; verdict changes needed (one criterion gap).
## Found
- Reviewed at 4c2f73f (assignments moved to CS210-001). First run at 402aa96 failed E-ASG-01 on the #172 seed collision (known, fixed by 4c2f73f).
- At 4c2f73f: two runs of both files on unicourse_e2e_11, port 17111: 17 passed (1.7m) each; typecheck and lint clean. Scope: two specs + lane log; helpers.ts untouched.
- Gap: Spec 11 §1 "A second university sees none of it" is asserted nowhere in either file.
- Minor: quizzes.spec.ts:104 second "Saved" assertion is vacuous (status still "Saved" from the first save during the 1 s debounce); assignments side never asserts the grade is hidden before release.
- No other dependence on the demo's assessments, attempts or submissions: both files now use CS210-001, which #172 leaves without quizzes or assignments.
## Requests
- to P18-L8b: add a cross-university case (see review report).
## Blockers
