# P4-L2a - database role split implementation
Status: done
## Doing
Finished; awaiting orchestrator verification. Scratch databases dropped (role unicourse_owner stays).
## Found
- Built per docs/design/P4-L2.md without deviations. `0004_role_split.sql`, `initdb.sql`, `pg.CheckAPIRole`/`CheckMigrated`, `testdb.OpenWithOwner`, CLI reads `UNICOURSE_OWNER_DATABASE_URL`, API no longer migrates, `/readyz` uses CheckMigrated, env.ts/CI/README updated.
- Cut-over (design §8) still to do by the orchestrator: reassign ownership on unicourse, unicourse_test, unicourse_e2e, migrate each with the owner URL, then tell lanes to rebase. Other lanes' testdb fails until then.
- New env for tests: `UNICOURSE_TEST_OWNER_DATABASE_URL`; e2e: `UNICOURSE_E2E_OWNER_DATABASE_URL`.
- P4-L3 conflict: only the `newHandler(...)` call line in cmd/api/main.go.
## Requests
## Blockers
