# P19-L7-review - review of P19-L7 (web for Spec 12)
Status: done
## Doing
Verdict: approve with one medium follow-up (editor 400 items[i].id path). Gates green on a git-archive snapshot of cfd8650.
## Found
- Gates (snapshot): lint 0, typecheck (app + e2e) 0, vitest 75 files / 591 tests pass (incl. no-literal-colors), build 0.
- Medium: question-editor.tsx onError handles only 409. The API checks missing ids (400 items[i].id) BEFORE structure (409), so "a student started while the form was open" after the editor added a question arrives as 400: banner "Please fix the highlighted fields. / Is required once a student has started." with nothing highlighted, no reload, no regrade mode. Fix: treat a 400 with an items[i].id detail like the 409 (setStructLocked, invalidate attempts, reload).
- Copy matches design §7 everywhere checked; scope is features/quiz/**, features/assignment/**, their tests, e2e/quizzes.spec.ts:185-190 and the lane log.
- Owner-to-confirm choices (worker): staff also see the warning "Past due" badge; regrade count = submitted attempts at save time.
## Requests
none
## Blockers
none
