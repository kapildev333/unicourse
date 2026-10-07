# P9-L1a-fix - fixes for P9-L1a review
Status: done
## Doing
Finished; waiting for orchestrator.
## Found
1. High: removed `sm:max-h-none` from dialog.tsx; regression test asserts no `max-h-none` and the dvh cap.
2. Low: DialogFooter now `sticky -bottom-6 -mb-6 pb-6` (bg kept).
3. Low: index.css stack rules cover `tbody th` like `td`; Table comment updated.
4. Nit: `::before` label `font-weight: 400`; deleted `width: 100%` (block td is already full width).
5. Nit: token line indented; card.test.tsx imports merged.
Gates (apps/web): lint clean; typecheck clean; test 41 files, 258 passed; build ok; e2e 2 passed (no bind collision; waited for port 18180).
## Requests
## Blockers
