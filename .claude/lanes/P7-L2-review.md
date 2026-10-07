# P7-L2-review - review of P7-L2 web (a, b, c) at 8158385
Status: done
## Doing
Verdict: changes requested (3 defects, 7 nits).
## Found
Gates (apps/web, branch): lint ok; typecheck ok; test 36 files / 242 passed; build ok. Merged with main (git merge-tree, clean): lint, typecheck, 244 passed, build ok.
Scope: only features/grades/**, the three route files, routeTree.gen.ts (consistent: a build leaves it unchanged), lane logs, design doc. No package changes. Conventions: no hex / palette / dark: / toast / fetch outside src/api / semicolons; no-literal-colors.test.ts is on the branch and passes.
Mutation checks: letterFor `<` breaks grade.test + finalize test; save-on-keystroke breaks the gradebook Escape test (only that one); My grades computing a letter breaks the Review Focus 3 test.
Probes (scratch tests) passing: letterFor 92.99/93.00; finalPercent with missing=0 and 0-weight; parsePoints 93, 93.5, abc, '', -5, 9.125; validateScale >2 decimals; Review Focus 3 with an unpublished assessment cached in the gradebook (nothing leaks, no gradebook request); server error keeps the typed value; ArrowLeft/ArrowUp; TA has no Edit/Finalize; export revokes the URL; finalize preview at 92.995 = A- and missing = 0.

Defects
1. grade.ts:11-19 (and tests grade.test.ts:25-26, gradebook.test.tsx:229-235, 280-290): gradebookAccess gives admin / own-dept dept_admin {view, finalize} only. D65 (spec 4 §4 as amended) gives them every instructor power. Failing input: admin, not staff -> scores false, assessments false: no score inputs, no Import scores, no New assessment/Edit/publish, although the API (`teach` = CanManageTAs) accepts those writes. Fix: `scores: manager || !!staffRole, assessments: manager || instructor`; flip the three tests.
2. gradebook.tsx:268-277 and 293-303: the PUT result overwrites the cell unconditionally. Scenario: type 34, blur (PUT pending), type 35 in the same cell, response arrives -> input shows "34 Saved"; 35 is silently discarded and never sent. Reproduced in a scratch test. Fix: in both patches skip the cell when `c[k]?.draft !== b.text` (only settle cells still holding what was sent).
3. gradebook.tsx:378: description built by `code.replace('-', ' · Section ')`; a course code with a hyphen ("CS-101") renders "CS · Section 101-001". Fix: `${sec.course.code} · Section ${sec.number} — ...`.

Nits
- queries.ts:83-88: revokeObjectURL right after click on a detached anchor; Safari can abort the download. `setTimeout(() => URL.revokeObjectURL(url))`.
- gradebook.tsx:216-222: finalize reasons differ from design §3.1 copy ("No students to grade.", "Add assessments before you finalize.", "...They total {n}%.") and sit under the button, not the toolbar.
- gradebook.tsx:520: "Change" shows for a student with no final (enrolled after finalizing) -> PATCH on a missing final grade.
- finalize-dialog.tsx:270-276: a server error on field `status` hides the FormErrors banner (fields non-empty) but no FormField shows it; same pattern in assessment-dialog.tsx:134 for unknown fields.
- scores-import-dialog.tsx:212: font-semibold (copied); Spec 3 says bold.
- gradebook.test.tsx: export test does not assert revokeObjectURL; save-per-keystroke is caught by one test only (add a 3-change, one-PUT assertion).
- After rebase on main: Card/DialogFooter now exist (PLAN.md note); design §0 #14 hand-built cards can move to them.
## Requests
- to orchestrator: D65 vs design §0 #1/§3.9 - the design table and L2c log still say admins have no score entry; the brief's D65 governs, so defect 1 stands.
## Blockers
