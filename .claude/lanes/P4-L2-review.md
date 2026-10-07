# P4-L2-review - review of P4-L2a (database role split)
Status: done
## Doing
Finished. Verdict: approve.
## Found
Verdict: **approve**. Diff matches docs/design/P4-L2.md §0-§9 with no deviations; scope is exactly the §9 file list (plus lane logs and the design doc).

Defects: none.

Nits (none block):
1. apps/web/README.md:18 (outside the lane's scope) still says "Override the database with `UNICOURSE_E2E_DATABASE_URL`". Overriding only that one now sends the CLI's migrate to the default unicourse_e2e while the API reads the override, so the API exits with "database is missing migrations". Add `UNICOURSE_E2E_OWNER_DATABASE_URL` (orchestrator or the next web lane).
2. apps/api/internal/pg/migrate.go:13 (outside scope): the lock comment "two API processes starting together must not race" is stale now that only the CLI and testdb migrate.
3. apps/api/README.md §"Database roles and migrations", local-volume block: the first line `create role unicourse_owner ...` aborts the block under ON_ERROR_STOP on any cluster where the role already exists. The shared local cluster is one of them (P4-L2a created the role). Fine for an old developer volume. On the shared cluster, leave that line out (commands below).
4. internal/pg/check_test.go:28-32: assertGrants ignores `rows.Scan` and `rows.Err` errors.
5. internal/pg/check.go:42: `sort.Strings` is redundant because fs.Glob already returns sorted names. It mirrors Migrate, so leave it.
6. Mutation check: deleting only the `nspowner` half, or only the `relowner` half, of the CheckAPIRole query still passes every test, because on a migrated database the owner matches both. Deleting both fails TestAPIRoleCannotEscalate. Real processes confirmed both halves work (see probes). Optional: also test against an empty schema.
7. CLI with UNICOURSE_OWNER_DATABASE_URL unset exits 1 (`database: missing environment: UNICOURSE_OWNER_DATABASE_URL`). That was already the behaviour before this change, and the design does not ask for exit 2.
8. If you start the API as role `unicourse` on a database that has not been cut over, it refuses with "... must use the API role (unicourse), not the owner". That is correct, but it reads oddly when the role named is unicourse. Acceptable.

Cut-over validated: I rebuilt a pre-split state on a fresh PG16 container. main's binary migrated 0001-0003 as `unicourse`, and the _test copy had public recreated by `unicourse`, the same as the shared unicourse/unicourse_test. Shared unicourse_e2e has public owned by pg_database_owner, which I covered separately. Then, as postgres, with `unicourse_owner` already present:
```
docker compose exec -T postgres psql -U postgres -v ON_ERROR_STOP=1 <<'SQL'
\c unicourse
reassign owned by unicourse to unicourse_owner;
\c unicourse_test
reassign owned by unicourse to unicourse_owner;
\c unicourse_e2e
reassign owned by unicourse to unicourse_owner;
SQL
for d in unicourse unicourse_test unicourse_e2e; do UNICOURSE_OWNER_DATABASE_URL=postgres://unicourse_owner:unicourse_owner@localhost:5432/$d?sslmode=disable go run ./cmd/unicourse-admin migrate; done
```
Results: all databases are now owned by unicourse_owner, as are every table and set_updated_at. 0004 applied. The API role has DML on all tables, no TRUNCATE, and USAGE on public. Data was kept. The new API passes CheckAPIRole and /readyz returns 200. `alter table ... disable row level security` as unicourse returns "must be owner". main's old CLI as unicourse returns "permission denied for schema public", so rebase the other lanes. Run it while no lane has a transaction open on those databases: reassign takes exclusive locks.
## Requests
## Blockers
