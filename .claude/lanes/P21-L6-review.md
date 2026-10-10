# P21-L6-review - review of P21-L6 (e2e notifications and Upcoming)
Status: done
## Doing
Review finished. Verdict: changes needed (small test additions, no product code).
## Found
- R1 notifications.spec.ts:32-48: focus 1 "a student who is not enrolled gets none" is not asserted. Jonas (P.jonas) is a student not on CS210-001; open his Notifications and expect no "New announcement: Lab moved to Friday" link.
- R2 notifications.spec.ts:61: "never reaches students for an unpublished grade" is not asserted. Before the first publish, check that Priya has no "Grades released: Notify quiz" link.
- R3 notifications.spec.ts:102-103: Priya is not a sound fence for Amara. One outbox sender claims rows `order by next_attempt_at, id`, and ids are random uuids, so a wrongly sent Amara mail can land just after Priya's and the read at :103 can miss it. A broken opt-out can then pass the test, though it would not make it fail falsely. Fix: after "Mail off", post a third announcement ("Mail fence", email: true) and poll for Priya's fence mail. Its rows have a later next_attempt_at, so every "Mail off" row is logged before them. Then assert on Amara.
- R4 upcoming.spec.ts:76: the token-in-log check runs before the rotation and 404 fetches, so it never covers the old token on a 404 or the fresh token. Move it after :93 and check both tokens.
- R5 (optional) upcoming.spec.ts:74: add `not.toContain('Done essay')` to the feed body (focus 2: the feed omits submitted work).
- Gates: e2e on both files twice, 9/9 each time (21.6 s, 21.8 s); typecheck clean; lint clean.
## Requests
## Blockers
