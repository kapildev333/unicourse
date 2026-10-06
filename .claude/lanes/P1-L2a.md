# P1-L2a - authz and the tenant request pipeline (apps/api/internal/{authz,tenant})
Status: done
## Doing
Finished; awaiting orchestrator verification.
## Found
Exported surface as built:
- `authz`: `Role` (`Admin`, `DeptAdmin`, `Member`); `Scope{UniversityID, UserID string; Role; DeptIDs []string}`; `WithScope(ctx, s)`, `ScopeFromContext(ctx) (Scope, bool)`; `IsAdmin(s)`, `CanManageDepartment(s, deptID)`, `CanInvite(s, role)`; `CanManageSection|CanManageTAs|CanReadSection|CanReadRoster(ctx, tx, s, sectionID) (bool, error)` (false, nil for a missing/foreign section; admin short-circuits to true without a query, so an admin gets true even for an unknown id - load the entity first).
- `tenant`: `Deps{DB, Sessions, Mailer, BaseURL}`; `type Handler func(r, tx, s authz.Scope) (status int, body any, err error)`; `Wrap(d Deps, h Handler) httpx.Handler`; `UniversityID(ctx) string`.
- `tenant` unexported helpers (handler.go): `isUUID`, `scanIDs`, `trimLen`, `noFields`, `page(r) (httpx.Page, after any, afterID any, err)` (after/afterID both nil without a cursor; `after` is `time.Time` inside the `any`).
- Test helpers (package `tenant`, helpers_test.go): `newServer(t, reg func(*httpx.Router, Deps)) *env` (real router + RequestID/Log/Recover + httptest; `env{DB, Fx, Deps, Srv, Log}`; Log has request-log and mail JSON lines); `(*env).user(uni, email, role, status, deptIDs...) (userID, *http.Cookie)`; `(*env).do(cookie, method, path, body) (status, raw)`; `errBody(raw) (code, details)`. Probe routes in middleware_test.go (`probeEnv`).
- Reuse note: the same email can get memberships in two universities via repeated `e.user` calls (user row is reused).
## Requests
None.
## Blockers
None.
