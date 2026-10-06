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

## Fix round
- Self-demotion navigates to the admin index; `['me']` refetched only when the caller's own membership changes or is removed. Shared `useRefresh` used in departments/terms. Audit form `onSubmit` preventDefault. Tests tightened (PATCH bodies, audit instants, DELETE paths, tab aria-current).
- Tab indicator root cause: base `border-transparent`/`text-ink-muted` and active `border-accent`/`text-accent` were both applied (TanStack merges className); stylesheet order let the base colors win. Fixed with `inactiveProps` carrying the inactive colors.
- Admin feature files reformatted to repo style (single quotes, no semis, 120 cols) in a separate commit.
