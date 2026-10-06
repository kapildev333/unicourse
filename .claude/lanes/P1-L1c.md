# P1-L1c - Auth HTTP surface and wiring
Status: done
## Doing
Finished; awaiting orchestrator verification.
## Found
- `app.Deps{ Auth auth.Deps }` (one field, additive for merges); `app.NewRouter` calls `auth.Register(r, deps.Auth)`. `auth.Deps{DB, Sessions, Magic, Providers, Secret, WebBase, EmailLimit, IPLimit}` (moved to handler.go). Zero `Deps` registers fine (handlers deref at request time only).
- `cmd/api`: `newHandler(logger, ready, api http.Handler)` (api nil = not mounted) mounts `/api/v1/` on the main mux; `authDeps(ctx, cfg, db, logger)` builds the wiring (empty SMTP URL -> `LogMailer` on the JSON logger; unset provider client id -> provider absent -> `startOidc` 404).
- Landing: session is created and cookie set BEFORE the redirect, also for `no_membership` (design 6 order). `next` wins over slug/first membership only when the user has at least one active membership. Failure redirects: errOIDC -> unauthenticated, ErrNotAllowed -> forbidden, `*httpx.Error` validation_failed/rate_limited -> its code; anything else -> 500.
- Magic link: GET renders the confirm page (form action = same path, `?next=` carried in the action only when `safeNext` passes); POST consumes. No rate limit on consume.
- `httpx`: `Router.ServeHTTP` records the matched pattern; `LogMiddleware` logs `path` = pattern path (e.g. `/api/v1/auth/magic-link/{token}`) when a route matched, raw path otherwise. Request log line (JSON slog): `{"msg":"request","method","path","status","durationMs","requestId","userId"?,"universityId"?}`.
- Signed-in test client for other handler tests: copy `hx`/`newHX`/`signedIn` from `internal/auth/handler_test.go`. `signedIn(email)` = `SignIn` + `Sessions.Create`, returns `&http.Cookie{Name: auth.SessionCookie, Value: tok}`; add it with `req.AddCookie` (never a cookie jar: cookies are Secure). Other packages can do the same with exported `auth.SignIn` + `(&auth.SessionStore{DB: db}).Create`.
- Deviation: handler tests live in `package auth` and use `httpx.NewRouter` + `auth.Register` (not `app.NewRouter`): `auth` tests cannot import `app` (import cycle). `app.NewRouter(app.Deps{})` + mount is covered by `cmd/api` `TestAPIMountedUnderV1` and the contract test.
- `/me` reads memberships through `pg.WithUserTx(user, "")`; department ids come from `to_json(array(...))`, so `departmentIds` is `[]` never null.
## Requests
- to orchestrator: `app.go` and `pending_routes.txt` edits are minimal (import, `Auth` field, one `Register` call; 7 lines removed) for the merge with the tenant lane.
## Blockers
None.
