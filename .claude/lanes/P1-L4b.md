# P1-L4b - Web: L4a review fixes, departments and terms
Status: done
## Doing
Finished; awaiting orchestrator verification.
## Found
- Reusable from `features/admin/queries.ts`: `useAllDepartments(slug)` (key `[slug,'departments','all']`, limit 200, staleTime 60s; data is `{ items }`). L4c adds `useAllMembers` the same way.
- Page pattern (copy it for members/invitations): `useCursorList` + `PageHeader` / `Loading` / `ErrorBanner onRetry` / `EmptyState` / `Table` + `LoadMore`; one `Dialog` always mounted with `open={target !== null}` (keeps focus-return to the opener) whose child form is rendered only while a target is set and `key`ed by target id (resets state and mutation error); inner `useMutation` form with `FormErrors ref={bannerRef}` (shown only when there are no field errors, so a detail-less 409 does not show twice) + `useFieldErrors` + `useFocusFirstError(formRef, error, bannerRef)`. Delete dialog: separate small form, Cancel first in DOM (opens on Cancel), 409 shown via `ErrorBanner title=""` using a `{ conflict: ... }` copy.
- Every mutation success invalidates `[slug, <resource>]` and `[slug, 'audit-log']` (helper `useRefresh` is local to `departments.tsx`/`terms.tsx`; lift it to `queries.ts` if L4c wants it).
- Delete button style: `Button variant="ghost" className="text-danger hover:bg-danger-soft"` (no ghost-danger variant exists).
- `FormErrors` is now a forwardRef (ref goes to the `ErrorBanner` div, which has `tabIndex={-1}`). `signOut(queryClient)` lives in `features/auth/sign-out.ts`. `/$slug` redirects to `/$slug/me`.
- `adminTo` casts remain only for members, invitations, audit (`admin-layout.tsx`) and the invitations redirect in `routes/$slug/admin/index.tsx`; L4c must remove them after adding its routes (and delete `adminTo`).
- Do not run `prettier --write` over existing files: repo has no prettier config and defaults reformat everything. New departments/terms files were prettier-formatted (80 cols); other files use ~120.
## Requests
## Blockers
