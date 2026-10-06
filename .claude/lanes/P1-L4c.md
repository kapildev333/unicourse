# P1-L4c - Web: members, invitations, audit log
Status: done
## Doing
Finished; awaiting orchestrator verification.
## Found
- `DepartmentPicker({ slug, value, onChange, error })` in `features/admin/department-picker.tsx` is reusable by P2-L3 (checkbox group, first checkbox gets focus via `useFocusFirstError`).
- `useRefresh(slug, resource)` is now in `features/admin/queries.ts` (returns `(...extraKeys) => Promise`, also invalidates the audit log); `departments.tsx`/`terms.tsx` still use their local copies (out of this lane's scope).
- `useAllMembers(slug)` added to `queries.ts`; used only by the audit-log "Changed by" filter. The audit "Who" column uses `entry.actor` ("System" when absent).
- `adminTo` is gone; all admin links are typed.
- Members page reads the university name and invalidates `['me']` on every role change/removal (simpler than comparing userId to the caller).
## Requests
## Blockers
