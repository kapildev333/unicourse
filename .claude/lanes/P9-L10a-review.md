# P9-L10a-review - review of P9-L10a (closing fixes a)
Status: done
## Doing
Done. Verdict: approve, minor findings only (none blocking).
## Found
- csv-import-dialog.tsx:225 Invite special case: tone belongs in components/badge.tsx:44 (`invite` tone 'neutral'), then revert to plain ActionBadge. Follow-up lane owning components/badge.tsx.
- features/admin/tab-header.tsx: near-copy of PageHeader with h2 and no test; CLAUDE.md says new patterns go in components/ with a test. Brief named the file, so accepted; fold into PageHeader (heading level prop) later.
- section-list.tsx:197 only TAs get data-empty; an empty Instructors cell still shows a bare dash on phones (probably intended, "unstaffed" is information). data-empty also hides StaffCell's sr-only "none".
- gradebook.tsx:504 sticky Student th keeps bg-surface on row hover (utility beats the components-layer hover rule), pre-existing; audit-log uses [tr:hover>&]:bg-canvas for this. Out of scope.
- Cross-lane with P9-L10b: the stack rules now in @layer utilities force text-align:left and tbody th surface, which should resolve P9-L10b-review findings 2 (grey th box, except capName max-w-48) and 3 (right-aligned stacked cells). Re-check captures 62/66 at 390 after both land.
- Gates: lint, typecheck clean; test 54 files 393/393; build ok; e2e 3/3; screens 1 passed, no WIDE (soft expect green). e2e diff empty.
## Requests
## Blockers
