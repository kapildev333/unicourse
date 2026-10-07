# P4-L5b - small web items
Status: done
## Doing
All three items committed.
## Found
- Cancel invitation: control on each pending chip for viewers who may remove that row; DELETE pending-staff/{invitationId}; success notice, 404 shows "Already cancelled" (and the section refetches so the row disappears).
- Catalog nit: fixed by syncing `text` to `q` during render when `q` changes; no effect. Section dialog: `useAllCourses` got an `enabled` arg, off in edit mode.
review fixes applied: 53cccb4

## Requests
(none)
## Blockers
None.
