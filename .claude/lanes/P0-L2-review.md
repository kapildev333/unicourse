# P0-L2-review - review of pg, schema, RLS, testdb, audit
Status: done (verdict: changes needed)
## Doing
Finished. Gates green; RLS and cascade behaviour probed in unicourse_test under lock 7263551; probe rows removed.
## Found
- RLS holds: with only app.user_id or app.email set, UPDATE, DELETE and INSERT ... ON CONFLICT on the self rows touch 0 rows or fail with 42501. Empty settings left over from a pooled connection match nothing. A garbage tenant id raises 22P02. Deleting a university cascades through the composite and restrict FKs without errors.
- Cross-tenant existence oracle: the uniques keyed on a parent id (enrollments, section_staff, department_admins, sections, pending_section_staff) raise 23505 before the composite FK check runs. So in university A, inserting with B's section id gives ErrConflict instead of ErrNotFound. Fix: put university_id first in those uniques.
- The self policies are OR-ed with tenant_isolation. If both app.university_id and app.user_id are set, the user's memberships in other universities become visible.
- testdb has no `_test` database-name guard, so it can truncate the dev database.
- The table owner is the API role, which can turn off FORCE RLS. Plan-level, for the orchestrator.
- Nothing maintains updated_at. Later lanes must set `updated_at = now()` in every UPDATE.
## Requests
- to orchestrator: findings 1-3 for the P0-L2 worker; finding 5 (table ownership) and the updated_at expectation for PLAN.md.
## Blockers
