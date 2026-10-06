# P1-L2c - memberships, invitations, audit-log route
Status: done
## Doing
Finished; waiting for orchestrator verification.
## Found
- Exports (package tenant, `invitation.go`): `NewInvitation{UniversityID, ActorUserID, Email, Role authz.Role, DepartmentIDs}`; `CreateInvitation(ctx, tx, in, m auth.Mailer, baseURL) (Invitation, error)`; `ErrAlreadyMember`, `ErrInvitationPending` (both `*httpx.Error` 409, compare with `errors.Is`). No authorisation inside (callers use `authz.CanInvite`). tx must be `pg.WithTx(in.UniversityID)`. Validates email (400 `email`), role and `departmentIds` (400, same rules as membership PATCH, so `departmentIds` on a non-dept_admin role is 400), department existence inside the tx (unknown/foreign id -> 404 "department"). Mail is sent inside the tx before commit.
- Audit actions written: `membership.update`, `membership.delete`, `invitation.create`, `invitation.delete`. Entity types `membership`, `invitation`.
- `audit.List` now left-joins `users` and fills `Row.Actor` (`{id,email,displayName}`, omitted when no actor or the user is not visible under the tenant's RLS); `Filter.To` is now exclusive (was inclusive), test updated.
- `trimLen` now rejects NUL (`must not contain NUL`), same wording as the courses branch. Other tenant strings (email/ids) are validated by regexp/NormalizeEmail so cannot carry NUL.
- Shared in `membership.go`: `roleDepts(role, ids)`, `checkDepts`, `lockMembers`, `guardLastAdmin`, `rowScanner` (reuse for staff/enrollment role checks).
- `pending_routes.txt`: removed the seven routes; `GET .../me/sections` left for the courses lane. `Register` gained the 7 routes; `createInvitation` is a closure over Deps (needs Mailer/BaseURL).
- Test helpers added: `(*env).memberID`, `activeAdmins`, `deptAdminRows`, `mails`, const `pathA`.
## Requests
None.
## Blockers
None.
