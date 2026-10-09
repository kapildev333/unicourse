# P18-L6a-review - review of P18-L6 sub-lane 6a (main...653a07b)
Status: done
## Doing
Finished: verdict changes needed (small). Gates green on a snapshot of 653a07b.
## Found
- Gates (snapshot of 653a07b): lint ok, typecheck ok, vitest 67 files / 486 tests pass (no-literal-colors included), build ok, routeTree.gen.ts unchanged by the build.
- M1 Assignments tab added by 6a (design §2/§10 give it to lane 7; 6a's tab test should list six tabs): dead link until P18-L7 lands. Drop it from 6a, or keep only if the PR lands together with 7.
- M2 quiz-list.tsx:55-71: any non-404 error on attempts/me shows Start; removing the 404 -> null mapping in queries.ts:65 fails no test.
- L: question-editor.tsx errors keyed by index survive add/move/remove; QuestionsPage ignores access.canEdit (students can GET questions while open, so they see an editable form that 403s on Save); no test for "guard released after a successful save" or "staff do not fetch attempts/me"; quiz-list.tsx:242 delete text says "No student has started it" when GET attempts fails.
## Requests
- to P18-L6a (or the orchestrator's fix brief): see Found.
## Blockers
