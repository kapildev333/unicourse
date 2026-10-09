# P18-L6b-review - review of sub-lane 6b (take-quiz page, countdown, results)
Status: done
## Doing
Finished: verdict changes needed (one medium bug, two missing guard tests). Reviewed 653a07b..9e60629 on a snapshot.
## Found
- Gates (snapshot of 9e60629): lint ok, typecheck ok, vitest 503/503 (no-literal-colors included), build ok (exit 0 on rerun; first chained run printed a vite error stack, not reproduced), routeTree.gen.ts unchanged by build.
- M1 take-quiz.tsx:351-353 + 393-396: Submit pending when the countdown hits zero makes atZero return (finishing set); if the submit then fails (5xx/network) finishing resets but the [isZero] effect never fires again: no final save, no polling, form stuck on "Time is up" with the dialog error. Reproduced with fake timers. Fix: in submit's catch, after resetting, `if (closesAt && Date.parse(closesAt) <= Date.now()) void zeroRef.current()`.
- M2 tests: removing the submit `finishing` guard (take-quiz.tsx:379) and treating 409 "already submitted" as a non-results error (take-quiz.tsx:330) both keep all tests green. Add: double confirm while PUT pending -> one PUT; autosave 409 "already submitted" -> results, no banner; 400 -> "Reload the page" banner.
- L: atZero does not wait for an in-flight autosave (older snapshot can land after the final PUT when the browser clock is early); resume() does not reschedule edits made while stopped; test "upcoming and unstarted-closed states" only tests upcoming; quiz-list.tsx take link still `params as never`.
- Close path otherwise per D147: one PUT without submit at zero, 3 s polling until submittedAt; early clock -> final PUT 200 then polling, no error; late clock -> 409 -> results (verified by scratch tests).
## Requests
- to orchestrator: fix brief for M1, M2 (and L items at your discretion).
## Blockers
none
