# P16-F1-review - review of P16-F1 (D128, accepted outside-domain invitee signs in again)
Status: done
## Doing
Finished. Verdict: approve (two nits, none blocking).
## Found
- Security: the tx-local app.user_id set in `allowed` (set_config(..., true)) widens only the select-only memberships_self and department_admins_self policies, and only while app.university_id is unset. Send inserts magic_links and outbox only (no RLS reads). SignIn runs only on a verified email (magic-link consume or OIDC), so the user id is the caller's own. No widening, no impersonation. Send still returns nil for every email; the only extra work is one indexed users lookup for every email.
- Confirmed: the new test fails on the reverted invite.go (invite_test.go:727), and it also fails when only the set_config line is dropped, so the RLS dependency is real and exercised.
- Nit: magic.go:25 Send doc still says "(no matching domain, no pending invitation)"; add "no active membership".
- Nit (hardening): invite.go:28 `exists (select 1 from memberships where status = 'active')` is scoped only by RLS; add `and user_id = nullif(current_setting('app.user_id', true), '')::uuid`, or join users on email, so a role that bypasses RLS cannot make every email allowed. Covered by the documented NOBYPASSRLS invariant (0001_init.sql:4), so not blocking.
- Orchestrator: Spec 1 §5 (docs/specs/2026-10-05-platform-core-and-staffing.md:62) still states the old rule; D128 supersedes it.
- Gates: gofmt clean, vet ok, auth -v 90 PASS 0 SKIP 0 FAIL, go test ./... all ok, lint ok, typecheck ok, e2e auth.spec.ts 14 passed. npm test skipped (Node 22.11, known).
## Requests
none
## Blockers
