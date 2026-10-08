# P16-L8-review - review of content-edit.spec.ts (34afc89)
Status: done
## Doing
Finished: verdict approve, nits only.
## Found
- All 9 E-CONTENT cases match design §A.7 step for step; no weakened or skipped assertion; no fixme.
- Gates at 18118: lint clean, typecheck clean, `npm run e2e -- content-edit.spec.ts` 9 passed (13.2 s). `npm test -- --run` skipped (Node 22.11 below floor, known).
- Scope: only apps/web/e2e/content-edit.spec.ts and the lane log changed.
- Nits: helpers already exports `H` (Sec-Fetch-Site header), the file re-inlines it 3 times (:94, :173, :206); `(x.items ?? x)` fallbacks at :171 and :205 are dead (both list endpoints return `items`); E-CONTENT-08 anchors ECON110 on any h1 rather than "About this section"; E-CONTENT-03 could assert the "Quiz moved" article is visible before the "Emailed" count 0 (safe today because the dialog awaits the list refresh before the status).
## Requests
none
## Blockers
