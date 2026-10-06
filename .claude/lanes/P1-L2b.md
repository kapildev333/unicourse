# P1-L2b - departments and terms API (apps/api/internal/tenant/{department,term,register}.go, app wiring)
Status: done
## Doing
Finished; awaiting orchestrator verification.
## Found
- Resource layout (copy for P1-L2c, P2-L2): one file per resource with the JSON struct (unexported `createdAt` for the cursor), a validator taking the full merged value (collects all FieldErrors, body order), a `xWriteErr` mapping `pg.MapError` conflict to the design's message, and one `func(r, tx, s) (status, body, error)` per route: admin check first, then `isUUID(id)` else `httpx.NotFound`, decode, validate, SQL, `audit.Write`, return. `register.go` has `Register(r, d)` with `Wrap(d, h)` per route and `const base`. PATCH = `select ... for update`, merge, validate merged, update (never set `updated_at`).
- List pattern: `page(r)`, query with keyset + `limit p.Limit+1`, loop: when `len(items)==Limit` set `NextCursor` from the last kept row and break. Cursor on a deleted row works (keyset compares values).
- `app.Deps{DB, Sessions, Mailer, BaseURL}`; `app.NewRouter` passes them to `tenant.Register`; zero Deps is fine at registration.
- Test helpers added (package tenant): `regTenant`, `(*env).adminCookie()` (seeded AdminA session), `(*env).tx(uni, fn)`, `(*env).auditRow(action, entityID) (n, before, after)`, `keys(t, raw)`, `idOf(t, raw)`, `repeat(s, n)`. Seed already has one department (CS) and one term (Fall 2026) in A, so lists have +1 row. Slugs are `a` and `b`.
- Design §4.1/§4.4 SQL sets `updated_at = now()` on the invitations trim; omitted per the trigger ruling.
## Requests
None.
## Blockers
None.
