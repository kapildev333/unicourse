# P4-L5a-review - review of pending-staff DELETE and exact-case JSON keys
Status: done
## Doing
Verdict: changes requested.
## Found
Verdict: changes requested.

Defects
1. apps/api/internal/course/staff.go:286-287 - an accepted invitation's pending row is still deletable (204, plus a misleading `pending_staff.delete` audit row). auth.SignIn sets accepted_at but leaves pending_section_staff rows in place, and the query does not filter them. Probe: invite x@o.org to SectionA (ta) and s2, auth.SignIn("x@o.org"), DELETE s2/pending-staff/{inv} -> 204, want 404 (brief item 3). Expired invitations behave the same way, and Section.pendingStaff already hides both. Smallest fix: add `and i.accepted_at is null and i.expires_at > now()` to the select (same filter as section.go:61), and add one test line (sign in, then delete -> 404).
2. apps/api/internal/course/section.go:24,60 + openapi `PendingStaff` - the route takes an invitationId that no client can read back. PendingStaff is only {email, role}, and GET /invitations is admin/dept_admin only, so an instructor (the main ta-row caller) can only use the id from the original 202 response. P4-L5b is blocked on exactly this (see its lane log). Smallest fix: add `'invitationId', i.id` to the json_build_object, `InvitationID string `json:"invitationId"`` to PendingStaff, the openapi property (uuid, required), and regenerate types.ts. The masking (D41) already hides pendingStaff from non-managers, so this exposes nothing new. The orchestrator decides which lane does it, but it belongs with this API change.

Nits
- httpx/json.go:71 `name == "-" && f.Tag.Get("json") == "-"`: the first clause is redundant.
- httpx/json.go:66 reflect.VisibleFields also admits the name of an embedded struct field and ignores json's tag-dominance rules. No request struct embeds today, so this is a ceiling worth one more line in the ponytail comment.
- staff.go:286-300 `select ... for update` + `delete` could be one `delete from pending_section_staff p using invitations i ... returning i.email, p.role` (the 403 still rolls back the tx).
- json_test.go:93 `_ = dst{}.priv` exists only to silence an unused-field lint.
- .claude/lanes/P4-L5a.md: Blockers section garbled ("The ruling matches keys against `json` stru## Blockers").
- Nested objects stay case-insensitive (`meetings[].Weekday` is accepted, verified). This matches the documented top-level-only ceiling.

Verified OK: route roles (ta, student, other, daOut -> 403; instructor on an instructor row -> 403; instructor on a ta row, daIn, admin -> 204; no cookie -> 401). Admin B via slug b with A's section -> 404 (D19). Admin B via slug a -> 403, which is the existing Wrap convention, the same as deleteStaff. The other section's row and the invitation survive; SignIn afterwards adds staff only on the remaining section. The audit row before is {invitationId,email,role} with entity_type section. The 13 json tags match the openapi request schemas exactly (checked by script). Wrong case is rejected: NAME (tenant), CourseId (course), EMAIL (auth). Duplicate key name+NAME -> NAME is rejected; name+name is OK; null is OK; fullwidth n, long s and Kelvin K -> unknown field. A 1 MiB body is OK, 1 MiB + 1 -> body too large. The reader is consumed once and capped at limit+1 bytes. Contract: operationId, Who: line, 204/401/403/404/500, invitationId uuid param; generate leaves no diff. Mutations: dropping the instructor-row 403 fails TestPendingStaffDelete; dropping exactKeys fails TestDecodeJSONExactCaseKeys and TestDepartmentsRejectWrongCaseKey.
## Requests
- to orchestrator: defect 2 is the same request as P4-L5b's (add invitationId to PendingStaff).
## Blockers
None.
