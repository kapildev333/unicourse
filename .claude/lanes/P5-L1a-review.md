# P5-L1a-review - review of P5-L1a (section meetings in the API)
Status: done
## Doing
Verdict: changes requested (one small defect). Review of ecda7f8 against design P5-L1 §1, §2.1, §3, §4.1, §4.2, §5 (L1a), §7.
## Found
Defects:
1. apps/api/internal/course/course.go:31-40 (likeParam): `q` holding invalid UTF-8 reaches Postgres and gives a 500 (SQLSTATE 22021, logged as internal error). Probe: `GET /courses?q=%FF` and `GET /sections?q=%FF` as admin -> 500. `q` is the first free-text query parameter in the API; request bodies are protected by encoding/json, query strings are not. Fix: add `case !utf8.ValidString(v): return nil, &httpx.FieldError{Field: "q", Message: "must be valid UTF-8"}` (or fold it into the NUL case) plus one row in TestCourseListQ.
Nits (no change required):
- meeting.go sortMeetings vs meetingsJSON order: Go compares location bytewise, SQL uses the DB collation, so audit `meetings` and response `meetings` can differ in order only for rows equal on weekday/start/end that differ in location. Cosmetic.
- section.go createSection builds the `secs` slice by hand, duplicating replaceMeetings' loop; calling replaceMeetings (a no-op delete on a fresh section) would drop 4 lines.
- No test pins the DB `check (starts_at < ends_at)` (Go validation always catches first); probe confirms the constraint holds (23514). No test covers `q=\`; probe confirms it matches literally.
Probes run (scratch copy of apps/api, real handler + SQL, all as expected unless listed above):
- RLS: tx scoped to A inserting a meeting for B's section with university A -> 23503 FK; with university B -> 42501 RLS; update/delete of B's rows -> 0 rows; A sees 0 of B's meetings; B's row intact; A admin PATCH B's section -> 404.
- PATCH: `{number, meetings:null}` keeps meetings; `[]` clears; absent keeps; `{meetings:null}` alone -> 400 field "" (design §0.3).
- Errors follow sent order (meetings[1].endsAt with bad row second, also when sort would move it).
- Times: 00:00-23:59 ok, 10:00-10:01 ok; 9:30, 24:00, 10:00:00, 09:60, " 9:00", -1:00, +9:00 -> 400 on the right field.
- 20 rows accepted and stored, 21 -> field meetings.
- q: `%`, `_`, `\` match literally; leading/trailing spaces trimmed; foreign-tenant departmentId -> 200 empty (no leak); Stud still 403 on lists (L1b scope).
- Cascade on section delete; audit after.meetings on create/patch (and on number-only patch); /me/sections carries meetings (worker tests, re-run).
- Mutations: removing the endsAt check, the Go sort, the SQL order, or the LIKE escape each fails a test.
Gates (run by me at ecda7f8): gofmt -l . empty; go vet ./... ok; go test ./internal/course/... ./internal/pg/... ./internal/app/... -count=1 ok (course 9.1s, pg 4.8s, app 0.8s); apps/web npm ci, npm run generate no diff, typecheck ok, lint ok, npm test -- --run 20 files / 121 tests passed.
Scope: matches design §6 L1a file list; nothing beyond it. No requests addressed to this lane in other lane files.
## Requests
- to P5-L1a (or its fix lane): defect 1 above.
## Blockers
