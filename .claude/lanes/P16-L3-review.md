# P16-L3-review - review of auth.spec.ts (P16-L3, commit 7ad979d)
Status: done
## Doing
Finished: verdict approve, two nits.
## Found
- Gates at unicourse_e2e_13, port 18113: lint clean, typecheck clean, `npm run e2e -- auth.spec.ts` 13 passed, 1 skipped (E-AUTH-11 fixme), 13.7 s. `npm test -- --run` skipped (Node 22.11 below floor, known).
- All 14 §A.2 cases match the design's steps and assertions; `before` instead of `before - 1` is correct after the cherry-picked logLines fix. E-AUTH-03 runFor deviation is sound.
- E-AUTH-02 ordering proof holds: outbox sender picks `order by urgent desc, next_attempt_at, id`, one row at a time.
- Link budget per email per run: tomasz 2, limit 6 (on purpose), every other address 1.
- Scope: 7ad979d touches only auth.spec.ts and the lane log.
- Nits (not blocking): E-AUTH-05 does not wait for the `/sign-in?error=` URL like E-AUTH-04 does; acceptInvite's `toHaveURL(/\/sign-in/)` has no failure message and matches anywhere in the URL.
## Requests
none
## Blockers
