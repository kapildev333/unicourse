# P16-L4-review - review of admin.spec.ts (P16-L4)
Status: done (verdict: approve, two low findings, one design gap for the orchestrator)
## Doing
Review finished; report sent to the orchestrator.
## Found
- Gates at 18114 / unicourse_e2e_14: lint clean, typecheck clean, admin.spec.ts 11/11 twice (20.9 s, 18.7 s). `npm test -- --run` skipped (Node 22.11 below the 22.22.2 floor).
- All 11 cases match §A.3 step for step; copy, confirmations, the blocked deletes, mail via mails(logLines()) with the fixed semantics, revoke, dept admin view and phone checks are present. Negatives follow a positive wait. Roles/labels/text only; no waitForTimeout, test ids or CSS. Cherry-pick 6eacc35 is content-identical to 8f663f1.
- Low: admin.spec.ts:203-207 the Entity=Department filter is never proven: the three newest audit entries overall are already the Physics ones, so rows 1-3 pass with a broken or not-yet-applied filter (design-level; fix: select Entity "Term" and expect row 1 to contain "Term deleted" from E-ADMIN-03).
- Low/nit: admin.spec.ts:200,254 inline `{ 'Sec-Fetch-Site': 'same-origin' }` although helpers exports `H`; the API does not check it on writes anyway.
- Design gap (orchestrator): Spec 9 §3.3 asks for audit entries "for each of the above"; §A.3 only checks department entries (no term/member/invitation audit rows).
## Requests
none
## Blockers
