# P4-L5b - small web items
Status: blocked
## Doing
Items 2 and 3 done and committed. Item 1 (cancel pending staff) blocked, see Requests.
## Found
- Item 1 root cause: `PendingStaff` (Go `course.PendingStaff`, `types.ts`) is only `{email, role}`; no `invitationId`, so the UI cannot build `DELETE .../pending-staff/{invitationId}`. The invitations list is not usable (instructors cannot read it).
- Catalog nit: fixed by syncing `text` to `q` during render when `q` changes; no effect. Section dialog: `useAllCourses` got an `enabled` arg, off in edit mode.
## Requests
- to orchestrator: add `invitationId` to `PendingStaff` in the API (struct, query, openapi, types.ts regen); then item 1 is a small follow-up.
## Blockers
Item 1 needs the API field above.
