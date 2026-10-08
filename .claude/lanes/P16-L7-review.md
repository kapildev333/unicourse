# P16-L7-review - review of catalog.spec.ts (P16-L7)
Status: done
## Doing
Nothing; verdict approve.
## Found
- Verdict: approve. All eight E-CATALOG cases match design §A.6 step for step: quoted copy, exact counts, and negative checks (Enrolled absent on ECON110/BIO150, Set up terms count 0 for a member, Fri-Sun count 0 in the agenda, no Clashes). Each negative check comes after a positive one on the same page.
- Selectors are roles, labels and text only; no waitForTimeout, test ids or CSS. Seeding follows §0.4 (seedDemo in beforeAll, an api() invitation as root). Sign-ins: 6 personas, 1 each.
- Nit: catalog.spec.ts:65 `expect(showing(page, 5)).toBeVisible()` has no §0.7 message. Fix: add `'student, Catalog: Showing 5 sections'`.
- Nit (no change needed): E-CATALOG-04 checks Elena's and Sofia's badges with a substring toContainText('Instructor'/'TA'). It is real today because the card meta lists instructor names, never the word "Instructor", and nothing in the CS101 card contains an uppercase "TA".
- Gates at port 18117: lint clean; typecheck (app and e2e) clean; `npm run e2e -- catalog.spec.ts` 8 passed (12.0 s). `npm test -- --run` skipped (Node 22.11 is below the 22.22.2 floor, known).
- Scope: the diff 4cb1bed...HEAD touches only apps/web/e2e/catalog.spec.ts and .claude/lanes/P16-L7.md.
## Requests
none
## Blockers
none
