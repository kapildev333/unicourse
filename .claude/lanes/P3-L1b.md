# P3-L1b - Playwright staffing flow and CI e2e job
Status: done
## Doing
Single serial-in-one-test staffing flow green locally; CI `e2e` job added.
## Found
- `git merge main` conflicted on already-squashed P1-L2c files; resolved by taking main, except `tenant/invitation.go` (+test) kept from the branch because main lacks `tenant.CreateInvitations` (used by course/import.go). Orchestrator: check this when squashing.
- Added devDependency `@types/node@22` (not in the brief): required to typecheck the node-side e2e files; `types` in tsconfig now includes `node`.
- Roster page shows the email twice per student (name column falls back to the email); e2e uses `.first()`. Cosmetic, not a bug.
- Machine was slow at times (one vite build took minutes); not a product issue.
## Requests
## Blockers
- Run time: 13 s warm; twice a first e2e run took 15+ min, all of it inside `vite build` ("built in 16m 49s") on this machine, never in the tests. Not reproduced on the second run.
- `npm test` now excludes `e2e/**` (vitest otherwise loads the Playwright spec); package.json script changed.
