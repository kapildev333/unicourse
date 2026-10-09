# P19-L8-review - review of P19-L8 (E-QUIZ-11, E-QUIZ-12, E-ASG-09)
Status: done
## Doing
Review complete: approve with two small should-fixes.
## Found
- Gates on a git-archive snapshot (port 17121, unicourse_e2e_11): e2e quizzes+assignments 21 passed (2.8m, E-QUIZ-11 1.1m); typecheck and lint clean.
- quizzes.spec.ts:294 `/Time left:/` passes even if the countdown used the 24 h window (timed quizzes always say "Time left: ..."); assert `/^Time left: (1 min|\d+ s)$/` to prove the attempt deadline drives it.
- Spec 12 §1 says e2e asserts the regrade sentence including "the audit log records a regrade per attempt"; E-QUIZ-12 does not check it. Admin audit page renders it as "Assessment regrade" (entity assessment, action attempt.regrade, no label in actionLabels).
- Scope clean: only the two spec files and the lane log. Only E-QUIZ-11 / E-ASG-09 setup goes through api(); everything graded/released/viewed is UI.
## Requests
none
## Blockers
none
