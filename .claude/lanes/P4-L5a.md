# P4-L5a - cancel pending staff; exact-case JSON keys
Status: blocked
## Doing
Item 1 committed. Item 2 stopped on an open design choice (see Blockers).
## Found
- For P4-L5b: `DELETE /api/v1/universities/{slug}/sections/{id}/pending-staff/{invitationId}` -> 204. Errors: 401, 403 (ta/other dept_admin; instructor on an `instructor` row), 404 `not_found` (section, bad/unknown invitationId, or no pending row on that section). Invitation stays; other sections' rows stay. Audit `pending_staff.delete`. operationId `deleteSectionPendingStaff`; types.ts regenerated.
## Requests
- to orchestrator: decide how DecodeJSON treats untagged struct fields (below).
## Blockers
The ruling matches keys against `json` struct tags. Every DecodeJSON caller (13 sites in tenant, course, auth) uses untagged structs such as `struct{ Email, Role string }` or `DepartmentID`, relying on case-insensitive matching ("departmentId" -> DepartmentID). Byte-equal-to-tag would reject every valid request. Options: (a) add camelCase json tags to all ~13 request structs (files outside my scope), then the ruling applies as written; (b) for untagged fields accept the Go name with first rune lowercased plus acronym handling (fragile: "departmentId" vs "departmentID"). Recommend (a).
