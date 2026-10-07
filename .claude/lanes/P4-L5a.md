# P4-L5a - cancel pending staff; exact-case JSON keys
Status: done
## Doing
Both items committed.
## Found
- For P4-L5b: `DELETE /api/v1/universities/{slug}/sections/{id}/pending-staff/{invitationId}` -> 204. Errors: 401, 403 (ta/other dept_admin; instructor on an `instructor` row), 404 `not_found` (section, bad/unknown invitationId, or no pending row on that section). Invitation stays; other sections' rows stay. Audit `pending_staff.delete`. operationId `deleteSectionPendingStaff`; types.ts regenerated.
- DecodeJSON now rejects wrong-case top-level keys (`{"NAME":..}` -> 400 validation_failed, field NAME, "unknown field"). Any new request struct needs explicit json tags; untagged fields only accept the exact Go name. Nested objects still case-insensitive (ponytail ceiling).
## Requests
(none)
## Blockers
The ruling matches keys against `json` stru## Blockers
None. Resolved by orchestrator ruling (a): 13 request structs now carry explicit camelCase json tags.
