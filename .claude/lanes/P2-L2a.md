# P2-L2a - courses and sections API (apps/api/internal/course, tenant helper exports)
Status: done
## Doing
Finished; awaiting orchestrator verification.
## Found
- `tenant` exports added (renames only, tenant tests green): `IsUUID`, `Page`, `TrimLen`, `NoFields`, `ScanIDs`. Nothing else in `tenant` changed.
- `course.Register(r, tenant.Deps)` wired in `app.NewRouter`; 9 lines removed from `pending_routes.txt`.
- P2-L2b reuse (package `course`, unexported): `sectionCols` + `sectionFrom` (the one select, `where s.id = $1` etc. appended by caller; my-sections adds `my_role` after `sectionCols` and passes a dest via `scanSection(sc, extra ...any)`), `scanSection`, `readSection(r, tx, id)` (404 "section" on malformed/unknown), `loadForWrite` (locks the section, returns `sectionAudit` + course department), `plainMember(s)`, `invalidUUID(field)`, types `Section`, `Staff`, `PendingStaff`, `Course` (Staff/PendingStaff already declared in section.go; P2-L2b must not redeclare them). `const base` is in handler.go; `Register` in handler.go needs the staff/enrollment routes added there.
- Fixture (`fixture_test.go`): `newEnv(t)` returns `*env` with `Fx`, `DeptA2`, `TermA2`, `CourseA` (CS101, DeptA), `SectionA` (CS101/TermA/001 with inst + ta staff and stud enrolled), `DeptB/TermB/CourseB` (university B), cookies `Admin, DaIn, DaOut, Inst, TA, Stud, Other`, `UserID[email]`; helpers `call(cookie, method, pathUnderUniA, body)`, `user(email, role, deptIDs...)`, `tx(uni, fn)`, `audit(action, id)`, `jmap`, `errDetails`. `newEnv` calls `Register`, so P2-L2b's routes appear once registered in handler.go.
- Course PATCH moving a course: both the old and the new department must be manageable (403 otherwise); `updated_at` never set (trigger).
- `createSectionStaff` 409 text: added to the operation description, not as a response override (the shared `Conflict` $ref stays; no shape change, no generator risk).
- No N+1: staff and pendingStaff are correlated subqueries in the single select (design §4); verified by construction, not by counting statements.
## Requests
None.
## Blockers
None.
