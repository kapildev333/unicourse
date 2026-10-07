# P9-L10b-review - review of P9-L10b closing fixes b
Status: done
## Doing
Done. Verdict: approve with three minor findings (none blocking).
## Found
- schedule.tsx:150 hour-label hide is not gated on nowMin >= start: 07:50-07:59 hides 08:00 with no now label drawn.
- finalize-dialog.tsx:168 first stacked table with a tbody th: th keeps .ui-table th canvas background and capName max-w-48, so a grey 192 px box at 390 (capture 62).
- finalize-dialog.tsx:171, scale-editor.tsx:163 text-right cells stay right-aligned when stacked (zigzag labels at 390, captures 62/66).
- Gates: lint, typecheck, test 393/393, build, e2e 3/3, screens 1 passed, 0 WIDE. staffing.spec.ts/helpers.ts untouched.
## Requests
## Blockers
