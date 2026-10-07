# P14-L3-review - review of web grading backlog items
Status: done
## Doing
Finished: approve with one out-of-scope follow-up (screens.ts selector).
## Found
- Landing (main...HEAD): gradebook.tsx, gradebook.test.tsx, lane log. Gates re-run: lint clean, typecheck clean, 55 files / 409 tests pass.
- e2e/staffing.spec.ts gradebook selectors all still match.
- e2e/screens.ts:306 `getByText('Must total 100% to finalize')` will now hit two elements (badge + new reason line "Weights must total 100% to finalize. They total 110%."; Playwright text match is case-insensitive substring) -> strict-mode failure in the screens run. Fix outside this lane: add `{ exact: true }`.
- Plain "(n missing)" wording (all unscored are published) has no unit test.
- P14-L1b confirms: no gradebook field needed; `MyGradesSection.unpublishedCount` (required int) added on its branch, so item 1 can follow once L1b lands.
## Requests
- to orchestrator: e2e/screens.ts:306 add `{ exact: true }` (not P14-L3's path).
## Blockers
