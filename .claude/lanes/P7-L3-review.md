# P7-L3-review - review of P7-L3 e2e grading steps (e2874d1)
Status: done
## Doing
Verdict: changes requested (one defect, small fix).
## Found
Gates (apps/web, this worktree): typecheck clean 3.7 s; lint clean 3.4 s; e2e 2 passed 21.8 s (staffing 8.3 s), again 2 passed 21.1 s (staffing 8.4 s).
GPA: API `round(sum(points*credits)/sum(credits), 2)` on numeric (half away from zero) = 20.8/6 -> 3.47; UI fmt2 renders "Term GPA 3.47 · 6.0 credits" and "3.47" in the Cumulative GPA dt/dd pair (the parent locator holds only that pair). Credits are patched before finalize, so the snapshot credits (D66) are 4 and 2.
Mutations (scratch copy of apps/web + apps/api):
- finalize preview tally A- shown as B+ -> FAIL line 340 (caught)
- /me/grades without cumulativeGpa -> FAIL line 379 "Cumulative GPA—" (caught)
- GPA unweighted avg(points) (3.35) -> FAIL line 378 (caught; proves the credit weight matters)
- CSV drops `final` column -> FAIL line 368 (caught)
- CSV drops BOM -> PASS (e2e strips it optionally; Go TestGradebookCSV pins the BOM, so acceptable)
- /me/grades published assessments without points -> FAIL line 384 (caught)
- /me/grades with no published assessments -> FAIL, but only at the 120 s test timeout (nit 2)
- student card renders B- instead of B on MA201 -> PASS (defect 1)
Scope: commit touches only apps/web/e2e/staffing.spec.ts and .claude/lanes/P7-L3.md. D33: new runId slugs/emails, no reset, magic-link sign-in. No sleeps. credits PATCH asserts 200 per course, cookie never printed.
### Defects
1. apps/web/e2e/staffing.spec.ts:388 `expect(ma).toContainText('B')` passes for B+, B- or any card containing a capital B; the spec's "student sees final letters" is not pinned for the second section (mutation survived). Fix: `await expect(ma).toContainText('B3.00 grade points · 83.00%')` (or `ma.getByText('B', { exact: true })`); same hardening at :382 (`'A-3.70 grade points · 92.00%'`).
### Nits
1. No GPA before the MA201 finalize (3.70) to show the second section "changes" it; 3.47 already differs from 3.70 and the unweighted 3.35, so optional.
2. :383 `cs.getByText(/Published scores/).click()` waits the full test timeout when the list is missing; `getByText('Published scores (2)')` with an expect first fails fast and pins the count.
3. Student "No access" (:390) checks the UI only; `expect((await p.request.get(`${api}/sections/${ids.cs101}/gradebook`)).status()).toBe(403)` would pin the API too.
4. For the orchestrator: PLAN.md has two rows labelled D66.
## Requests
- to P7-L3: defect 1.
## Blockers
none
