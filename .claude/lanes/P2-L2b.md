# P2-L2b - staff, enrollments and my-sections API (apps/api/internal/course)
Status: done
## Doing
Finished; awaiting orchestrator verification.
## Found
- Reuse for P2-L2c (package `course`, unexported, `staff.go`): `resolveEmails(ctx, tx, emails []string) (map[string]emailState, error)` (emails must already be `auth.NormalizeEmail`'d; missing keys never occur: every input email has an entry, zero value = no user/membership/invitation); `emailState{UserID, DisplayName, Status, InvitationID}`; `sectionExists(r, tx, id)` (404 "section"); `allow(ok, err)` (predicate result -> 403); `readInvitation(ctx, tx, id) tenant.Invitation` (no `tenant` change needed); `staffAudit`, `StaffInvited`, `enrollmentAudit`, `Enrollment` types.
- Routes registered in `handler.go`: six; `createStaff(d)` is a closure over `tenant.Deps` (mailer, base URL). `pending_routes.txt` now holds only the six import routes.
- `fixture_test.go` now builds the env with a recording mailer (`e.Mails.count()`), plus helpers `e.count(sql, args...)`, `e.newSection(termID, number)`, `e.count0` (in staff_test.go).
- Audit entity types: `staff` (id = section_staff.id), `enrollment` (id = enrollments.id); actions `staff.create|delete`, `enrollment.delete`.
- `tenant` untouched (no `/me/sections` placeholder existed).
- DB suite waits on the advisory lock for minutes when other lanes run; ~5 min waits seen.
## Design gaps (reported, not picked)
- Brief says staff list "ordered instructors first then email"; design §5 specifies keyset on `(ss.created_at, ss.id)` (cursor carries a time+id), which cannot give that order. Built the design SQL; embedded `Section.staff` is instructors-first. Needs a ruling if the list order must match.
## Requests
None.
## Blockers
None.
