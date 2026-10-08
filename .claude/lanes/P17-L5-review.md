# P17-L5-review - review of P17-L5 (D131 overlap, D132 generic 409, F4 F5 F6 F10)
Status: done
## Doing
Finished; verdict changes needed (one item: section CSV import skips the overlap rule).
## Found
- Must fix: apps/api/internal/course/meeting.go parseSlots (used by import.go:514) never calls overlaps, so a sections CSV row "Mon 10:00-11:00; Mon 10:30-12:00" imports overlapping meetings and bypasses D131. Fix: `if o := overlaps(ms); o != nil { return nil, o[0].Message }` before sortMeetings, plus one parseSlots test case.
- validateMeetings overlap logic, touching allowed, 1-based sent order, skip-on-row-errors: correct; skipping only delays the overlap to the next submit.
- 21-row copy: API "add at most 20 meeting times" renders via sentence() as "Add at most 20 meeting times." (unit test asserts it), same as before.
- Side effect, harmless: staff.go:189 reuses tenant.ErrAlreadyMember, so the staff POST 409 text changed too; web staff-editor has its own conflict copy.
- F6 no flash: UniversityLayout renders only after me loads. Nits: empty sr-only Actions column and "Add a term..." empty-state body still shown to dept admins.
- e2e selectors changed: exactly 3 (E-COURSES-01 duplicate code, E-ADMIN-07 invitation duplicate, E-ADMIN-08 Changed by option), each tied to a copy change.
- Gates: gofmt clean, vet ok, go test course+tenant ok (0 SKIP), lint ok, typecheck ok, vitest 428 passed, e2e 24 passed on 17115/unicourse_e2e_15.
## Requests
- to P17-L5 (via orchestrator): add the overlap check to parseSlots.
## Blockers
