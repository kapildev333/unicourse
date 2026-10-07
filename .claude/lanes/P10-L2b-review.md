# P10-L2b-review - review of P10-L2b Materials page
Status: done
## Doing
Review finished: changes needed (one finding).
## Found
- materials.tsx:60 Move up never returns focus: `focusAfter` is "up w1", slice(0,4) gives "up w" and slice(5) gives "", so no button id matches. Move down works (4-letter dir). Fix: `const [dir, id] = focusAfter.current.split(' ')`; add a "Move Week 2 up" focus test.
- Nit: week-dialog.tsx:75 uses size="lg"; design §5.6 says md. Matches the other textarea dialogs; orchestrator's call.
- Upload loop, allowlist (matches apps/api/internal/content fileTypes), FormData field "file", credentials include, uploadReason messages (match material.go), sizes, delete-week count, move-to-week filter, invalidation, beforeunload add/remove, per-role rendering, 403 state: all correct.
- Scope clean; tokens only; no font-semibold or dark:. The route.test.tsx router-plugin warning predates this lane (P9-L1b).
- Gates: lint 0, typecheck 0, 342/342 tests in 50 files, build ok, routeTree.gen.ts unchanged after build.
- P10-L2a-review.md does not exist in this worktree or main; nothing to answer there.
## Requests
- to P10-L2b (via orchestrator): the focus fix above.
## Blockers
