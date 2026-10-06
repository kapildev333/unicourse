# P1-L4a - Web shared layer, app shell, sign-in
Status: done
## Doing
Finished; awaiting orchestrator verification.
## Found
- Exports as designed: `PageHeader`, `EmptyState`, `ErrorBanner` (pass `title=""` for body-only; forwardRef), `StatusMessage`, `Badge`/`RoleBadge`/`roleLabel`, `LoadMore`, `Loading`, `inputClass` (from `form-field.tsx`), `FormErrors/useFieldErrors/useFocusFirstError/errorMessage/FieldCopy` (`form-errors.tsx`), `useCursorList`, `formatDate/formatDateTime` (`lib/format.ts`), `useMe/useUniversity/Me/MyMembership` (`features/auth/use-me.ts`, types come from `components['schemas']`).
- `src/test-utils.tsx`: `mockApi(routes)` returns the vi.fn (read bodies with `await fn.mock.calls[i][0].json()`; request is cloned so it is readable), `renderWithQuery(ui)` returns RTL result plus `client`. Call `vi.unstubAllGlobals()` in `afterEach` yourself. `test-setup.ts` polyfills dialog showModal/close and stubs `window.scrollTo` (router scroll restoration spams jsdom errors otherwise).
- `FormField` error text now has an sr-only "Error: " prefix, so `toHaveAccessibleDescription('required')` became `/required/` in the existing form-field test. Use regex in new tests.
- P1-L4b/c MUST: `src/features/admin/admin-layout.tsx` (`adminTo`) and `src/routes/$slug/admin/index.tsx` cast admin sub-route paths `as '/$slug/admin'` because the route files do not exist yet (typed router rejects unknown paths). Once departments/terms/members/invitations/audit route files exist and routeTree.gen.ts is regenerated, replace the casts with plain `to="/$slug/admin/<x>"` and delete `adminTo`.
- `/$slug/courses` and `/$slug/sections/$id` are still skeleton `<h1>` pages (Task 11).
- Sign-in `next` goes through `safeNext` once; magic-link and OIDC start both send it only when safe.
- `AppShell` uses `useQueryClient`/`useNavigate`, so render it under QueryClientProvider and a router (see app-shell.test.tsx).
## Requests
- to orchestrator: `npm ci` was needed in the worktree (no node_modules); no lockfile change.
## Blockers
