# P21-L3-review - review of the P21-L3 landing (calendar, Upcoming, feed, ICS)
Status: done
## Doing
Review finished at c5aa390. Verdict: changes needed, one cross-scope item (audit entity calendar_feed); everything else approves.
## Found
- Gates on a fresh git-archive snapshot (scratchpad/snap-c5aa390): gofmt clean, go vet ok, go test calendar/auth/quiz/app ok, web typecheck 0; regenerated types.ts and Postman show no drift.
- Mutations (calendar package): dropping the active-membership join, the not-submitted filter, the 74-octet continuation limit, the UTF-8 back-off, the campus `at time zone`, and the rotate delete all fail a test.
- F1 (medium): the audit entity `calendar_feed` is missing from openapi AuditEntry.entityType enum (openapi.json ~5054), tenant/handler.go entityTypes (:115), and web audit-log.tsx entityLabels (:23). Admin Audit log renders "undefined created" and Target "undefined" for each feed create/rotate/delete. Fixing it needs files outside L3's §8 scope (tenant, audit-log.tsx), so the orchestrator decides.
- F2 (low): two concurrent POST /me/calendar-feed (double click) -> second insert hits unique (university_id, user_id) -> 500. Fix: lock the membership row first (`select 1 from memberships where user_id = $1 for update`) before the delete.
- Process: I wrote into the shared scratchpad `snap/` and `gates.txt` before I noticed they already existed (possibly the orchestrator's). `snap/` now holds HEAD c5aa390 extracted over its old contents.
## Requests
- to orchestrator: decide F1 (add `calendar_feed` to the enum, the tenant filter list and the audit-log labels, e.g. in L5 or a follow-up lane).
## Blockers
