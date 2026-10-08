# P16-L5-review - review of courses.spec.ts (P16-L5)
Status: done
## Doing
Review complete: changes needed (one finding), two nits.
## Found
- Gates at port 18115: lint clean, typecheck clean, `npm run e2e -- courses.spec.ts` 12 passed (13.4 s). `npm test -- --run` skipped (Node 22.11 below the 22.22.2 floor, known).
- Finding 1 (fix): courses.spec.ts:221-223, Elena's `/from the roster$/` count 0 can pass before the roster has loaded, because the `Roster` heading renders while the list is still pending (roster.tsx:139-150). Wait for `/^Roster 8/` first.
- Nits: line 124 `m2 Starts fill('')` does nothing (the row is already empty); lines 227 and 269 inline the header that helpers.ts exports as `H`.
- Scope: the diff touches only apps/web/e2e/courses.spec.ts and the lane log.
## Requests
- to P16-L5: change line 221 to `name: /^Roster 8/`.
## Blockers
none
